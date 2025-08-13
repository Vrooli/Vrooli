# Extension Icons

Place your extension icons here. Chrome requires icons in the following sizes:

## Required Icons

- `icon-16.png` - 16x16 pixels (toolbar icon)
- `icon-32.png` - 32x32 pixels (Windows computers)
- `icon-48.png` - 48x48 pixels (extension management page)
- `icon-128.png` - 128x128 pixels (Chrome Web Store)

## Optional Icons

- `icon-19.png` - 19x19 pixels (toolbar icon on some devices)
- `icon-38.png` - 38x38 pixels (toolbar icon on Retina displays)

## Icon Guidelines

1. **Format**: PNG format with transparency
2. **Design**: Simple, recognizable at small sizes
3. **Colors**: Should work on both light and dark backgrounds
4. **Padding**: Include some padding around the icon edges

## Generating Icons

If you have a high-resolution source image (512x512 or larger), you can use ImageMagick to generate all sizes:

```bash
# Install ImageMagick if not already installed
# brew install imagemagick  # macOS
# apt-get install imagemagick  # Ubuntu/Debian

# Generate all icon sizes from source.png
convert source.png -resize 16x16 icon-16.png
convert source.png -resize 32x32 icon-32.png
convert source.png -resize 48x48 icon-48.png
convert source.png -resize 128x128 icon-128.png
```

## Placeholder Icons

For testing, you can use placeholder icons:
- Create solid color squares with the app's primary color
- Add a letter or symbol to represent the app
- Use online tools like https://placeholder.com/ for quick placeholders

## Icon Usage in Manifest

Icons are referenced in `manifest.json`:

```json
"icons": {
    "16": "icons/icon-16.png",
    "32": "icons/icon-32.png",
    "48": "icons/icon-48.png",
    "128": "icons/icon-128.png"
}
```

**AGENT NOTE**: When adapting this template, replace these placeholder icons with the actual app's branding.