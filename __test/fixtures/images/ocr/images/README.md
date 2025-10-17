# OCR Test Images

This directory contains test images designed to evaluate OCR (Optical Character Recognition) capabilities across various challenging scenarios.

## Image Descriptions

### 1_simple_text.png
- **Purpose**: Baseline test with optimal conditions
- **Content**: "Hello World! This is a simple OCR test."
- **Characteristics**: Black text on white background, clear font, no distortions

### 2_multi_font_text.png
- **Purpose**: Test recognition of different fonts and text styles
- **Content**: Multiple text samples with varying fonts (serif, sans-serif, monospace), sizes, and cases
- **Characteristics**: Different font families, sizes from 25px to 60px, mixed case text

### 3_rotated_text.png
- **Purpose**: Test recognition of text at various angles
- **Content**: Text samples at 0°, 15°, 45°, 90°, and 180° rotation
- **Characteristics**: Same font but different orientations

### 4_noisy_text.png
- **Purpose**: Test robustness against image noise
- **Content**: "Noisy Text with Distortion"
- **Characteristics**: Salt-and-pepper noise, random lines, slight Gaussian blur

### 5_complex_background_text.png
- **Purpose**: Test text extraction from busy backgrounds
- **Content**: "Text on Complex Background"
- **Characteristics**: Gradient background, random colored shapes, text with white outline

### 6_document_text.png
- **Purpose**: Test multi-line document recognition
- **Content**: Full document with title, date, paragraphs, special characters, URLs, email, phone numbers
- **Characteristics**: Multiple font sizes, various text types, bullet points

### 7_colored_text.png
- **Purpose**: Test color text recognition
- **Content**: Various colored text samples and text on colored backgrounds
- **Characteristics**: Red, blue, green, purple, orange text; white on black, yellow on blue

### 8_low_contrast_text.png
- **Purpose**: Test recognition in poor contrast conditions
- **Content**: Text with varying levels of contrast
- **Characteristics**: Light gray on light background, dark on dark background

### 9_handwriting_style.png
- **Purpose**: Test recognition of handwriting-style text
- **Content**: "This looks like handwritten text with slight variations"
- **Characteristics**: Oblique font, slight rotations, irregular positioning

### 10_perspective_text.png
- **Purpose**: Test recognition of perspectively transformed text
- **Content**: "PERSPECTIVE TEXT"
- **Characteristics**: Text with perspective distortion as if receding into distance

## Usage

These images are designed to test OCR systems progressively from simple to complex scenarios. Start with `1_simple_text.png` for baseline testing and progress through more challenging images.

## Generation

To regenerate these images, run:
```bash
python generate_ocr_test_images.py
```

Note: Font availability may vary by system. The script includes fallbacks for missing fonts.
