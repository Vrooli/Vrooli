#!/usr/bin/env python3
"""
Comic Panel Compositor for ComfyUI
Automatically combines individual comic panels into a professional comic book page layout.

Usage:
    python3 composite_comic_panels.py [output_directory]

This script looks for the latest generated comic panels and combines them into:
- A 2x2 grid layout with proper spacing
- Comic book style borders
- Professional presentation

Dependencies: PIL (Pillow)
Install with: pip install Pillow
"""

import os
import sys
import glob
from PIL import Image, ImageDraw, ImageFont
from datetime import datetime

def find_latest_panels(output_dir="/home/matthalloran8/.comfyui/outputs"):
    """Find the most recent comic panel files."""
    panel_patterns = [
        "comic_panel_1_spyglass_*.png",
        "comic_panel_2_map_*.png", 
        "comic_panel_3_battle_*.png",
        "comic_panel_4_treasure_*.png"
    ]
    
    panels = []
    for pattern in panel_patterns:
        files = glob.glob(os.path.join(output_dir, pattern))
        if files:
            # Get the most recent file for this panel
            latest = max(files, key=os.path.getctime)
            panels.append(latest)
        else:
            print(f"Warning: No files found for pattern {pattern}")
            return None
    
    if len(panels) == 4:
        print(f"Found all 4 panels:")
        for i, panel in enumerate(panels, 1):
            print(f"  Panel {i}: {os.path.basename(panel)}")
        return panels
    else:
        print(f"Error: Only found {len(panels)} panels, need 4")
        return None

def create_comic_page(panels, output_path):
    """Create a professional comic book page layout."""
    
    # Load all panel images
    panel_images = []
    for panel_path in panels:
        try:
            img = Image.open(panel_path)
            panel_images.append(img)
            print(f"Loaded panel: {os.path.basename(panel_path)} ({img.size})")
        except Exception as e:
            print(f"Error loading {panel_path}: {e}")
            return False
    
    # Get panel dimensions (assuming all are same size)
    panel_width, panel_height = panel_images[0].size
    
    # Define layout parameters
    border_width = 8  # Black border around each panel
    gutter_width = 20  # Space between panels
    page_margin = 30  # Margin around entire page
    
    # Calculate page dimensions
    page_width = (panel_width * 2) + (border_width * 4) + gutter_width + (page_margin * 2)
    page_height = (panel_height * 2) + (border_width * 4) + gutter_width + (page_margin * 2)
    
    # Create white background
    comic_page = Image.new('RGB', (page_width, page_height), 'white')
    draw = ImageDraw.Draw(comic_page)
    
    # Panel positions: [top-left, top-right, bottom-left, bottom-right]
    positions = [
        (page_margin, page_margin),  # Top-left
        (page_margin + panel_width + border_width * 2 + gutter_width, page_margin),  # Top-right
        (page_margin, page_margin + panel_height + border_width * 2 + gutter_width),  # Bottom-left
        (page_margin + panel_width + border_width * 2 + gutter_width, 
         page_margin + panel_height + border_width * 2 + gutter_width)  # Bottom-right
    ]
    
    # Place each panel with border
    for i, (panel_img, (x, y)) in enumerate(zip(panel_images, positions)):
        # Draw black border
        border_rect = [
            x - border_width, 
            y - border_width, 
            x + panel_width + border_width, 
            y + panel_height + border_width
        ]
        draw.rectangle(border_rect, fill='black')
        
        # Paste the panel image
        comic_page.paste(panel_img, (x, y))
        
        print(f"Placed panel {i+1} at position ({x}, {y})")
    
    # Add title/credits area at bottom
    title_y = page_height - 60
    draw.rectangle([0, title_y, page_width, page_height], fill='black')
    
    try:
        # Try to use a comic-style font, fall back to default
        font = ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf", 24)
    except:
        font = ImageFont.load_default()
    
    # Add title text
    title_text = "PIRATE RABBIT ADVENTURES"
    subtitle_text = f"Generated with ComfyUI • {datetime.now().strftime('%Y-%m-%d')}"
    
    # Calculate text position for centering
    title_bbox = draw.textbbox((0, 0), title_text, font=font)
    title_width = title_bbox[2] - title_bbox[0]
    title_x = (page_width - title_width) // 2
    
    draw.text((title_x, title_y + 5), title_text, fill='white', font=font)
    
    # Subtitle with smaller font
    try:
        small_font = ImageFont.truetype("/usr/share/fonts/truetype/liberation/LiberationSans.ttf", 14)
    except:
        small_font = ImageFont.load_default()
    
    subtitle_bbox = draw.textbbox((0, 0), subtitle_text, font=small_font)
    subtitle_width = subtitle_bbox[2] - subtitle_bbox[0]
    subtitle_x = (page_width - subtitle_width) // 2
    
    draw.text((subtitle_x, title_y + 35), subtitle_text, fill='white', font=small_font)
    
    # Save the composite image
    comic_page.save(output_path, 'PNG', quality=95)
    print(f"\n✅ Comic page saved: {output_path}")
    print(f"   Dimensions: {page_width}x{page_height}")
    print(f"   File size: {os.path.getsize(output_path) / (1024*1024):.1f} MB")
    
    return True

def main():
    """Main function to create comic page from panels."""
    print("🎨 Comic Panel Compositor")
    print("=" * 40)
    
    # Get output directory from command line or use default
    if len(sys.argv) > 1:
        output_dir = sys.argv[1]
    else:
        output_dir = "/home/matthalloran8/.comfyui/outputs"
    
    if not os.path.exists(output_dir):
        print(f"Error: Output directory does not exist: {output_dir}")
        sys.exit(1)
    
    print(f"Looking for panels in: {output_dir}")
    
    # Find the latest comic panels
    panels = find_latest_panels(output_dir)
    if not panels:
        print("\n❌ Could not find all required comic panels")
        print("Make sure you've run the multi-panel comic workflow first:")
        print("curl -X POST http://localhost:8188/prompt -H 'Content-Type: application/json' \\")
        print("  -d @/home/matthalloran8/.comfyui/workflows/pirate_rabbit_comic_composite_v2.json")
        sys.exit(1)
    
    # Create output filename with timestamp
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    output_filename = f"pirate_rabbit_comic_page_{timestamp}.png"
    output_path = os.path.join(output_dir, output_filename)
    
    # Create the composite comic page
    print(f"\n🔧 Creating comic page layout...")
    success = create_comic_page(panels, output_path)
    
    if success:
        print(f"\n🎉 Success! Your comic book page is ready!")
        print(f"   Open: {output_path}")
        print(f"   Or view in browser: file://{output_path}")
    else:
        print(f"\n❌ Failed to create comic page")
        sys.exit(1)

if __name__ == "__main__":
    main()