# Test Fixture Images

This directory contains a comprehensive collection of test images organized by category and use case. These images are designed to test various aspects of image processing, validation, and handling in the Vrooli application.

## Directory Structure

```
images/
├── synthetic/          # Programmatically generated test images
│   ├── colors/        # Solid color images
│   ├── patterns/      # Pattern-based images (gradients, checkerboards)
│   └── animations/    # Animated images (GIF)
├── real-world/        # Photographic images (Creative Commons)
│   ├── people/        # Human portraits
│   ├── animals/       # Animal photographs
│   ├── nature/        # Landscapes and nature
│   ├── architecture/  # Buildings and cityscapes
│   ├── food/          # Food photography
│   └── abstract/      # Abstract art
├── dimensions/        # Images categorized by size (copies of synthetic images)
│   ├── tiny/          # Very small images (1x1 to 10x10)
│   ├── small/         # Small images (50x50)
│   ├── medium/        # Medium images (100x100 to 200x200)
│   ├── large/         # Large images (500x300 to 1920x1080)
│   └── special/       # Non-square aspect ratios
├── formats/           # Special format testing
│   └── sample-image.heic  # Apple HEIC format
└── ocr/              # OCR testing images
    ├── images/  # Generated OCR test images
    └── generate_ocr_test_images.py  # Script to regenerate OCR images

```

**Note**: Synthetic images appear in both `synthetic/` (organized by type) and `dimensions/` (organized by size) for different testing needs.

## Test Categories and Use Cases

### 1. Synthetic Images (`synthetic/`)

#### Colors (`synthetic/colors/`)
**Purpose**: Test basic image handling, color detection, and format support
- **Test scenarios**:
  - Image upload and storage
  - Color space handling (RGB)
  - Format conversion between PNG, JPEG, WebP, GIF
  - Compression quality testing
  - Thumbnail generation

**Files**:
- Various solid color images in different formats
- Static GIF for format detection

#### Patterns (`synthetic/patterns/`)
**Purpose**: Test image processing algorithms and edge detection
- **Test scenarios**:
  - Gradient handling
  - Pattern recognition
  - Image analysis algorithms
  - Compression artifact testing
  - Edge and corner detection

**Files**:
- `gradient-red-blue.png` - Linear gradient
- `test-pattern.png` - Complex test pattern
- `checkerboard.png` - High contrast pattern

#### Animations (`synthetic/animations/`)
**Purpose**: Test animated image support
- **Test scenarios**:
  - GIF animation handling
  - Frame extraction
  - Animation metadata reading
  - Animated thumbnail generation

**Files**:
- `animated-rgb.gif` - 3-frame color animation

### 2. Real-World Images (`real-world/`)

**Purpose**: Test with realistic photographic content
- **Test scenarios**:
  - Face detection (people folder)
  - Object recognition
  - EXIF data handling
  - Content-aware resizing
  - Image categorization
  - Machine learning model testing

**Subcategories**:
- `people/` - Portrait photography for face detection
- `animals/` - Animal recognition testing
- `nature/` - Landscape processing
- `architecture/` - Straight line detection, perspective
- `food/` - Color vibrancy, close-up handling
- `abstract/` - Edge cases for recognition algorithms

**Note**: All images are from Unsplash with free-to-use licensing.

### 3. Dimension Testing (`dimensions/`)

#### Tiny (`dimensions/tiny/`)
**Purpose**: Test edge cases with minimal pixel counts
- **Test scenarios**:
  - Minimum size validation
  - Upscaling algorithms
  - Icon generation
  - Memory efficiency

**Files**:
- `small-image.png` - 1x1 pixel (absolute minimum)
- Images up to 10x10 pixels

#### Small to Large (`dimensions/small/`, `medium/`, `large/`)
**Purpose**: Test scaling and performance across sizes
- **Test scenarios**:
  - Progressive image loading
  - Responsive image generation
  - Memory usage scaling
  - Processing time benchmarks
  - CDN optimization

#### Special (`dimensions/special/`)
**Purpose**: Test non-standard aspect ratios
- **Test scenarios**:
  - Aspect ratio preservation
  - Banner/header image handling
  - Panoramic image support
  - Mobile vs desktop optimization

### 4. Format Testing (`formats/`)

**Purpose**: Test support for various image formats
- **Test scenarios**:
  - Format detection
  - Conversion capabilities
  - Metadata preservation
  - Browser compatibility fallbacks

**Current formats**:
- PNG (throughout other folders)
- JPEG/JPG (throughout other folders)
- WebP (in synthetic/colors/)
- GIF (static and animated)
- HEIC (Apple format)

### 5. OCR Testing (`ocr/`)

**Purpose**: Test optical character recognition capabilities
- **Test scenarios**:
  - Basic text extraction
  - Multi-language support
  - Handwriting recognition
  - Low quality/noisy image OCR
  - Document structure understanding
  - Text orientation detection

**Test images include**:
1. Simple clear text (baseline)
2. Multiple fonts and sizes
3. Rotated text (various angles)
4. Noisy/distorted text
5. Text on complex backgrounds
6. Multi-line documents
7. Colored text variations
8. Low contrast scenarios
9. Handwriting simulation
10. Perspective distortion

## Usage Examples

### Basic Upload Test
```bash
# Test with a simple color image
synthetic/colors/medium-blue.png
```

### Format Support Test
```bash
# Test each format
synthetic/colors/small-green.jpg    # JPEG
synthetic/colors/medium-purple.webp  # WebP
synthetic/animations/animated-rgb.gif # Animated GIF
formats/sample-image.heic           # HEIC
```

### Performance Test
```bash
# Small to large progression
dimensions/tiny/small-image.png     # 1x1
dimensions/small/*                  # 50x50
dimensions/medium/*                 # 100x100-200x200
dimensions/large/large-hd.png       # 1920x1080
```

### OCR Capability Test
```bash
# Progress through difficulty levels
ocr/1_simple_text.png              # Start here
ocr/4_noisy_text.png               # Medium difficulty
ocr/10_perspective_text.png        # Advanced
```

## Adding New Test Images

When adding new test images:
1. Place them in the appropriate category folder
2. Use descriptive filenames
3. Keep file sizes reasonable (< 500KB unless testing large files)
4. Document any special properties in this README
5. For generated images, add generation scripts to maintain reproducibility

## Image Sources and Licensing

- **Synthetic images**: Generated programmatically, free to use
- **Real-world images**: From Unsplash (free-to-use license)
- **OCR images**: Generated programmatically, free to use

For questions about specific image usage or licensing, refer to the source documentation.