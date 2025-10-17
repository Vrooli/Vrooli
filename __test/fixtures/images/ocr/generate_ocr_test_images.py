#!/usr/bin/env python3
"""
Generate various test images with text for OCR testing.
Creates images with different text characteristics to test OCR capabilities.
"""

import os
import random
from PIL import Image, ImageDraw, ImageFont, ImageFilter
from datetime import datetime

# Create output directory
OUTPUT_DIR = "images"
os.makedirs(OUTPUT_DIR, exist_ok=True)

def get_font(size=40, font_type="default"):
    """Get a font with fallback to default if specific font not available."""
    try:
        if font_type == "serif":
            return ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf", size)
        elif font_type == "sans":
            return ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf", size)
        elif font_type == "mono":
            return ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf", size)
        else:
            return ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf", size)
    except:
        # Fallback to default font
        return ImageFont.load_default()

def get_text_dimensions(text, font):
    """Get the bounding box of text for proper sizing."""
    # Create a temporary image to measure text
    temp_img = Image.new('RGB', (1, 1))
    temp_draw = ImageDraw.Draw(temp_img)
    bbox = temp_draw.textbbox((0, 0), text, font=font)
    return bbox[2] - bbox[0], bbox[3] - bbox[1]

def create_simple_text():
    """1. Simple clear text on white background"""
    font = get_font(48)
    text = "Hello World! This is a simple OCR test."
    
    # Calculate required dimensions with padding
    text_width, text_height = get_text_dimensions(text, font)
    padding = 50
    img_width = text_width + (padding * 2)
    img_height = text_height + (padding * 2)
    
    img = Image.new('RGB', (img_width, img_height), color='white')
    draw = ImageDraw.Draw(img)
    
    # Center the text
    x = padding
    y = padding
    draw.text((x, y), text, fill='black', font=font)
    
    img.save(os.path.join(OUTPUT_DIR, "1_simple_text.png"))
    print("Created: 1_simple_text.png")

def create_multi_font_text():
    """2. Text with multiple fonts and sizes"""
    texts = [
        ("Large Serif Text", get_font(60, "serif")),
        ("Medium Sans-Serif Text", get_font(40, "sans")),
        ("Small Monospace Text", get_font(25, "mono")),
        ("UPPERCASE TEXT", get_font(35)),
        ("lowercase text", get_font(35)),
        ("MiXeD CaSe TeXt", get_font(35))
    ]
    
    # Calculate total height needed
    padding = 50
    line_spacing = 20
    total_height = padding * 2
    max_width = 0
    
    text_dimensions = []
    for text, font in texts:
        width, height = get_text_dimensions(text, font)
        text_dimensions.append((width, height))
        max_width = max(max_width, width)
        total_height += height + line_spacing
    
    img_width = max_width + (padding * 2)
    img = Image.new('RGB', (img_width, total_height), color='white')
    draw = ImageDraw.Draw(img)
    
    # Draw each text
    y_offset = padding
    for i, (text_font_pair, dims) in enumerate(zip(texts, text_dimensions)):
        text, font = text_font_pair
        draw.text((padding, y_offset), text, fill='black', font=font)
        y_offset += dims[1] + line_spacing
    
    img.save(os.path.join(OUTPUT_DIR, "2_multi_font_text.png"))
    print("Created: 2_multi_font_text.png")

def create_rotated_text():
    """3. Text at different angles (rotated)"""
    img = Image.new('RGB', (1000, 1000), color='white')
    
    texts = [
        ("Horizontal Text", 0),
        ("Slightly Tilted", 15),
        ("45 Degree Angle", 45),
        ("Vertical Text", 90),
        ("Upside Down", 180)
    ]
    
    y_offset = 100
    for text, angle in texts:
        font = get_font(36)
        text_width, text_height = get_text_dimensions(text, font)
        
        # Create a temporary image with enough space for the text
        temp_img = Image.new('RGBA', (text_width + 40, text_height + 40), (255, 255, 255, 0))
        temp_draw = ImageDraw.Draw(temp_img)
        temp_draw.text((20, 20), text, fill='black', font=font)
        
        # Rotate the temporary image
        rotated = temp_img.rotate(angle, expand=1)
        
        # Calculate position to center on main image
        x = (img.width - rotated.width) // 2
        img.paste(rotated, (x, y_offset), rotated)
        y_offset += 180
    
    img.save(os.path.join(OUTPUT_DIR, "3_rotated_text.png"))
    print("Created: 3_rotated_text.png")

def create_noisy_text():
    """4. Text with noise/distortion"""
    font = get_font(48)
    text = "Noisy Text with Distortion"
    
    # Calculate dimensions
    text_width, text_height = get_text_dimensions(text, font)
    padding = 100
    img_width = text_width + (padding * 2)
    img_height = text_height + (padding * 2)
    
    img = Image.new('RGB', (img_width, img_height), color='white')
    draw = ImageDraw.Draw(img)
    
    # Draw text centered
    x = padding
    y = padding
    draw.text((x, y), text, fill='black', font=font)
    
    # Add salt and pepper noise
    for _ in range(3000):
        x = random.randint(0, img_width - 1)
        y = random.randint(0, img_height - 1)
        color = random.choice(['black', 'white', 'gray'])
        draw.point((x, y), fill=color)
    
    # Add random lines
    for _ in range(20):
        x1, y1 = random.randint(0, img_width - 1), random.randint(0, img_height - 1)
        x2, y2 = random.randint(0, img_width - 1), random.randint(0, img_height - 1)
        draw.line([(x1, y1), (x2, y2)], fill='lightgray', width=1)
    
    # Apply slight blur
    img = img.filter(ImageFilter.GaussianBlur(radius=0.5))
    
    img.save(os.path.join(OUTPUT_DIR, "4_noisy_text.png"))
    print("Created: 4_noisy_text.png")

def create_complex_background_text():
    """5. Text on complex/noisy background"""
    font = get_font(52)
    text = "Text on Complex Background"
    
    # Calculate dimensions
    text_width, text_height = get_text_dimensions(text, font)
    padding = 100
    img_width = text_width + (padding * 2)
    img_height = text_height + (padding * 2)
    
    img = Image.new('RGB', (img_width, img_height), color='white')
    draw = ImageDraw.Draw(img)
    
    # Create gradient background
    import math
    for y in range(img_height):
        r = int(255 * (1 - y/img_height))
        g = int(128 + 127 * math.sin(y/50))
        b = int(128 + 127 * math.cos(y/50))
        draw.line([(0, y), (img_width, y)], fill=(r, g, b))
    
    # Add random shapes
    for _ in range(30):
        x = random.randint(0, img_width - 50)
        y = random.randint(0, img_height - 50)
        shape_type = random.choice(['rectangle', 'ellipse'])
        color = (random.randint(100, 255), random.randint(100, 255), random.randint(100, 255))
        
        if shape_type == 'rectangle':
            draw.rectangle([x, y, x+50, y+50], fill=color, outline=None)
        else:
            draw.ellipse([x, y, x+50, y+50], fill=color, outline=None)
    
    # Draw text with outline for visibility
    x = padding
    y = padding
    
    # Draw outline
    for dx in [-2, -1, 0, 1, 2]:
        for dy in [-2, -1, 0, 1, 2]:
            if dx != 0 or dy != 0:
                draw.text((x+dx, y+dy), text, fill='white', font=font)
    
    # Draw main text
    draw.text((x, y), text, fill='black', font=font)
    
    img.save(os.path.join(OUTPUT_DIR, "5_complex_background_text.png"))
    print("Created: 5_complex_background_text.png")

def create_document_text():
    """6. Multi-line document with various elements"""
    # Create a document-like image
    img = Image.new('RGB', (800, 1200), color='white')
    draw = ImageDraw.Draw(img)
    
    # Title
    title_font = get_font(48, "serif")
    body_font = get_font(24)
    small_font = get_font(18)
    
    y_offset = 50
    
    # Document content
    content = [
        ("Test Document Title", title_font, 'black', 'center'),
        ("Date: " + datetime.now().strftime("%B %d, %Y"), small_font, 'gray', 'center'),
        ("", body_font, 'black', 'left'),  # Empty line
        ("Lorem ipsum dolor sit amet, consectetur adipiscing elit.", body_font, 'black', 'left'),
        ("Sed do eiusmod tempor incididunt ut labore et dolore magna.", body_font, 'black', 'left'),
        ("", body_font, 'black', 'left'),
        ("Key Information:", get_font(28), 'black', 'left'),
        ("• First bullet point with some text", body_font, 'black', 'left'),
        ("• Second bullet point with more information", body_font, 'black', 'left'),
        ("• Third bullet point example", body_font, 'black', 'left'),
        ("", body_font, 'black', 'left'),
        ("Contact Information:", get_font(28), 'black', 'left'),
        ("Email: test@example.com", body_font, 'blue', 'left'),
        ("Phone: +1 (555) 123-4567", body_font, 'black', 'left'),
        ("Website: https://www.example.com", body_font, 'blue', 'left'),
        ("", body_font, 'black', 'left'),
        ("Special Characters: @#$%&*(){}[]<>", body_font, 'black', 'left'),
        ("Numbers: 0123456789", body_font, 'black', 'left'),
        ("Mixed: ABC123def456GHI789", body_font, 'black', 'left'),
    ]
    
    padding = 50
    for text, font, color, align in content:
        if text:  # Skip empty lines
            text_width, text_height = get_text_dimensions(text, font)
            
            if align == 'center':
                x = (img.width - text_width) // 2
            else:
                x = padding
            
            draw.text((x, y_offset), text, fill=color, font=font)
            y_offset += text_height + 10
        else:
            y_offset += 20  # Empty line spacing
    
    img.save(os.path.join(OUTPUT_DIR, "6_document_text.png"))
    print("Created: 6_document_text.png")

def create_colored_text():
    """7. Text in various colors and backgrounds"""
    img = Image.new('RGB', (1000, 800), color='white')
    draw = ImageDraw.Draw(img)
    font = get_font(40)
    
    color_combinations = [
        ("Red text on white", 'red', 'white', False),
        ("Blue text on white", 'blue', 'white', False),
        ("Green text on white", 'green', 'white', False),
        ("Purple text on white", 'purple', 'white', False),
        ("Orange text on white", 'orange', 'white', False),
        ("White text on black", 'white', 'black', True),
        ("Yellow text on blue", 'yellow', 'blue', True),
        ("Black text on yellow", 'black', 'yellow', True),
    ]
    
    y_offset = 50
    for text, text_color, bg_color, has_bg in color_combinations:
        text_width, text_height = get_text_dimensions(text, font)
        
        if has_bg:
            # Draw background rectangle
            draw.rectangle([40, y_offset - 10, 50 + text_width + 10, y_offset + text_height + 10], 
                          fill=bg_color)
        
        draw.text((50, y_offset), text, fill=text_color, font=font)
        y_offset += text_height + 30
    
    img.save(os.path.join(OUTPUT_DIR, "7_colored_text.png"))
    print("Created: 7_colored_text.png")

def create_low_contrast_text():
    """8. Text with varying levels of contrast"""
    img = Image.new('RGB', (900, 800), color='white')
    draw = ImageDraw.Draw(img)
    font = get_font(36)
    
    contrast_levels = [
        ("High contrast (black on white)", (0, 0, 0), (255, 255, 255)),
        ("Medium contrast", (100, 100, 100), (255, 255, 255)),
        ("Low contrast", (200, 200, 200), (255, 255, 255)),
        ("Very low contrast", (230, 230, 230), (255, 255, 255)),
        ("Dark on dark", (50, 50, 50), (0, 0, 0)),
        ("Light on light", (240, 240, 240), (220, 220, 220)),
    ]
    
    y_offset = 50
    for text, text_color, bg_color in contrast_levels:
        text_width, text_height = get_text_dimensions(text, font)
        
        # Draw background
        draw.rectangle([40, y_offset - 10, img.width - 40, y_offset + text_height + 10], 
                      fill=bg_color)
        
        # Draw text
        draw.text((50, y_offset), text, fill=text_color, font=font)
        y_offset += text_height + 40
    
    img.save(os.path.join(OUTPUT_DIR, "8_low_contrast_text.png"))
    print("Created: 8_low_contrast_text.png")

def create_handwriting_style():
    """9. Simulated handwriting style text"""
    img = Image.new('RGB', (1000, 400), color='white')
    draw = ImageDraw.Draw(img)
    
    # Try to use an italic or script font, fallback to regular
    font = get_font(32)
    text = "This looks like handwritten text with slight variations"
    
    # Draw text with slight random variations in position
    x_base = 50
    y_base = 150
    
    words = text.split()
    x_offset = x_base
    
    for word in words:
        # Add slight random variation
        x_var = random.randint(-2, 2)
        y_var = random.randint(-3, 3)
        rotation = random.randint(-3, 3)
        
        # Create temp image for word
        word_width, word_height = get_text_dimensions(word + " ", font)
        temp_img = Image.new('RGBA', (word_width + 20, word_height + 20), (255, 255, 255, 0))
        temp_draw = ImageDraw.Draw(temp_img)
        temp_draw.text((10, 10), word, fill='darkblue', font=font)
        
        # Rotate slightly
        rotated = temp_img.rotate(rotation, expand=1)
        
        # Paste onto main image
        img.paste(rotated, (x_offset + x_var, y_base + y_var), rotated)
        x_offset += word_width
    
    img.save(os.path.join(OUTPUT_DIR, "9_handwriting_style.png"))
    print("Created: 9_handwriting_style.png")

def create_perspective_text():
    """10. Text with perspective transformation"""
    # Create base image with text
    font = get_font(60)
    text = "PERSPECTIVE TEXT"
    
    text_width, text_height = get_text_dimensions(text, font)
    padding = 100
    
    # Create larger base image
    base_img = Image.new('RGB', (text_width + padding*2, text_height + padding*2), 'white')
    draw = ImageDraw.Draw(base_img)
    draw.text((padding, padding), text, fill='black', font=font)
    
    # Apply perspective transformation
    width, height = base_img.size
    
    # Define perspective transformation
    # Making text appear to recede into distance
    coeffs = [
        1.2, 0.1, -50,    # x = a*x + b*y + c
        0.0, 1.0, 0,      # y = d*x + e*y + f
        0.0, 0.0005       # w = g*x + h*y + 1
    ]
    
    img = base_img.transform(
        (width + 200, height + 100),
        Image.PERSPECTIVE,
        coeffs,
        Image.BICUBIC,
        fillcolor='white'
    )
    
    img.save(os.path.join(OUTPUT_DIR, "10_perspective_text.png"))
    print("Created: 10_perspective_text.png")

# Create README for the OCR test images
def create_readme():
    readme_content = """# OCR Test Images

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
"""
    
    with open(os.path.join(OUTPUT_DIR, "README.md"), 'w') as f:
        f.write(readme_content)
    print("Created: README.md")

# Main execution
if __name__ == "__main__":
    print("Generating OCR test images...")
    
    # Create all test images
    create_simple_text()
    create_multi_font_text()
    create_rotated_text()
    create_noisy_text()
    create_complex_background_text()
    create_document_text()
    create_colored_text()
    create_low_contrast_text()
    create_handwriting_style()
    create_perspective_text()
    create_readme()
    
    print("\nAll OCR test images have been generated successfully!")
    print(f"Images saved in: {OUTPUT_DIR}/")