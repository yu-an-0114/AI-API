# Swift Troubleshooting

## `'nil' is incompatible with return type 'String'`

If you see this error while working on the `RecipeService` class in the iOS client, it means the function is declared to return a `String` but one of the code paths is returning `nil`. Swift does not allow `nil` to be returned from a non-optional `String` function. Update the implementation to either return a non-empty fallback value (for example `""`) or change the signature to return `String?` and handle the optional where the method is called.

```swift
func generatePrompt(...) -> String {
    guard let name = recipe.name else {
        return ""
    }

    return name
}
```

Alternatively:

```swift
func generatePrompt(...) -> String? {
    guard let name = recipe.name else {
        return nil
    }

    return name
}
```

Be sure to adjust any callers accordingly.
