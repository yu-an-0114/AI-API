package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"recipe-generator/internal/infrastructure/config"
	"recipe-generator/internal/pkg/common"

	"go.uber.org/zap"
)

// CacheManager 緩存管理器
type CacheManager struct {
	config *config.Config
	mu     sync.RWMutex
	store  map[string]cacheEntry
	stats  cacheStats
}

// cacheEntry 緩存條目
type cacheEntry struct {
	value       string
	expiresAt   time.Time
	imageHash   string
	createdAt   time.Time
	lastAccess  time.Time
	accessCount int
}

// cacheStats 緩存統計
type cacheStats struct {
	hits      int64
	misses    int64
	evictions int64
	errors    int64
}

// NewManager 創建新的緩存管理器
func NewManager(cfg *config.Config) *CacheManager {
	if !cfg.Cache.Enabled {
		common.LogInfo("Cache disabled")
		return nil
	}

	m := &CacheManager{
		config: cfg,
		store:  make(map[string]cacheEntry),
		stats:  cacheStats{},
	}

	// 啟動清理過期緩存的協程
	go m.startCleanup()

	common.LogInfo("快取管理員已初始化",
		zap.Int("最大容量", cfg.Cache.MaxSize),
		zap.Duration("存活時間", cfg.Cache.TTL),
		zap.Duration("清理間隔", cfg.Cache.CleanupInterval),
	)

	return m
}

// Get 獲取緩存值
func (m *CacheManager) Get(ctx context.Context, prompt, imageData string) (string, error) {
	if !m.config.Cache.Enabled {
		common.LogInfo("Cache disabled, skipping lookup")
		return "", common.ErrCacheDisabled
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 生成緩存鍵
	key := m.generateKey(prompt, imageData)

	// 檢查緩存
	if entry, exists := m.store[key]; exists {
		// 檢查是否過期
		if time.Now().After(entry.expiresAt) {
			m.mu.RUnlock()
			m.mu.Lock()
			delete(m.store, key)
			m.stats.evictions++
			m.mu.Unlock()
			m.mu.RLock()
			common.LogInfo("快取已過期",
				zap.String("鍵", key),
			)
			return "", common.ErrCacheDisabled
		}

		// 檢查圖片哈希是否匹配
		if imageData != "" && entry.imageHash != m.hashImage(imageData) {
			m.stats.misses++
			common.LogInfo("快取因圖片變更未命中",
				zap.String("鍵", key),
			)
			return "", fmt.Errorf("image changed")
		}

		// 更新訪問統計
		entry.lastAccess = time.Now()
		entry.accessCount++
		m.store[key] = entry
		m.stats.hits++

		common.LogInfo("快取命中",
			zap.String("鍵", key),
		)
		return entry.value, nil
	}

	m.stats.misses++
	common.LogInfo("快取未命中",
		zap.String("鍵", key),
	)
	return "", common.ErrCacheDisabled
}

// Set 設置緩存值
func (m *CacheManager) Set(ctx context.Context, prompt, imageData, value string) error {
	if !m.config.Cache.Enabled {
		common.LogInfo("Cache disabled, skipping set")
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 檢查緩存大小
	if len(m.store) >= m.config.Cache.MaxSize {
		// 清理過期項目
		evicted := m.cleanup()
		common.LogInfo("快取清理執行",
			zap.Int("清理數量", evicted),
		)

		// 如果仍然超過大小限制，執行 LRU 清理
		if len(m.store) >= m.config.Cache.MaxSize {
			m.evictLRU()
		}

		// 如果仍然超過大小限制，返回錯誤
		if len(m.store) >= m.config.Cache.MaxSize {
			m.stats.errors++
			common.LogWarn("快取已滿",
				zap.Int("目前容量", len(m.store)),
			)
			return common.ErrCacheFull
		}
	}

	// 生成緩存鍵
	key := m.generateKey(prompt, imageData)

	// 設置緩存
	now := time.Now()
	m.store[key] = cacheEntry{
		value:       value,
		expiresAt:   now.Add(m.config.Cache.TTL),
		imageHash:   m.hashImage(imageData),
		createdAt:   now,
		lastAccess:  now,
		accessCount: 0,
	}

	common.LogInfo("快取已儲存",
		zap.String("鍵", key),
	)

	return nil
}

// generateKey 生成緩存鍵
func (m *CacheManager) generateKey(prompt, imageData string) string {
	if imageData == "" {
		return fmt.Sprintf("text:%s", m.hashString(prompt))
	}
	return fmt.Sprintf("multimodal:%s:%s", m.hashString(prompt), m.hashImage(imageData))
}

// hashString 計算字符串的 SHA-256 哈希值
func (m *CacheManager) hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// hashImage 計算圖片數據的哈希值
func (m *CacheManager) hashImage(imageData string) string {
	return m.hashString(imageData)
}

// startCleanup 啟動清理過期緩存的協程
func (m *CacheManager) startCleanup() {
	ticker := time.NewTicker(m.config.Cache.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

// cleanup 清理過期的緩存
func (m *CacheManager) cleanup() int {
	now := time.Now()
	count := 0

	for key, entry := range m.store {
		if now.After(entry.expiresAt) {
			delete(m.store, key)
			count++
			m.stats.evictions++
		}
	}

	if count > 0 {
		common.LogInfo("Cleaned up expired cache entries",
			zap.Int("count", count),
			zap.Int64("total_evictions", m.stats.evictions),
			zap.Int("remaining_size", len(m.store)),
			zap.Float64("eviction_ratio", float64(m.stats.evictions)/float64(m.stats.hits+m.stats.misses)),
		)
	}

	return count
}

// evictLRU 執行 LRU 清理
func (m *CacheManager) evictLRU() {
	var oldestKey string
	var oldestAccess time.Time
	var lowestAccessCount int

	// 找到最少訪問的項目
	for key, entry := range m.store {
		if oldestKey == "" ||
			entry.accessCount < lowestAccessCount ||
			(entry.accessCount == lowestAccessCount && entry.lastAccess.Before(oldestAccess)) {
			oldestKey = key
			oldestAccess = entry.lastAccess
			lowestAccessCount = entry.accessCount
		}
	}

	if oldestKey != "" {
		delete(m.store, oldestKey)
		m.stats.evictions++
		common.LogInfo("快取已淘汰(LRU)",
			zap.String("鍵", oldestKey),
		)
	}
}

// GetStats 獲取緩存統計信息
func (m *CacheManager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"size":      len(m.store),
		"max_size":  m.config.Cache.MaxSize,
		"hits":      m.stats.hits,
		"misses":    m.stats.misses,
		"evictions": m.stats.evictions,
		"errors":    m.stats.errors,
		"hit_ratio": float64(m.stats.hits) / float64(m.stats.hits+m.stats.misses),
	}
}

// Close 關閉緩存管理器
func (m *CacheManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 清空緩存
	m.store = make(map[string]cacheEntry)
	common.LogInfo("快取管理員已關閉",
		zap.Int64("命中次數", m.stats.hits),
		zap.Int64("未命中次數", m.stats.misses),
		zap.Int64("淘汰次數", m.stats.evictions),
	)
	return nil
}
