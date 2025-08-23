# ComfyUI Example Workflows

This directory contains example ComfyUI workflows demonstrating various features and techniques for AI image generation with the Vrooli resource system.

## Available Examples

### 1. Basic Text-to-Image (`basic_text_to_image.json`)
A simple workflow demonstrating the fundamental ComfyUI text-to-image generation process.

**Features:**
- Single text prompt to image generation
- SDXL Base 1.0 model usage
- Basic positive and negative prompting
- Standard sampling parameters

**Usage:**
```bash
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow \
  --workflow ./scripts/resources/automation/comfyui/examples/basic_text_to_image.json
```

### 2. Advanced Comic Book Workflows

Two sophisticated workflows for creating comic book-style pirate rabbit illustrations:

## Workflow 1: `pirate_rabbit_comic_advanced.json`
### Single-Page Comic Layout with Multi-Stage Refinement

**Concept**: Creates a complete comic book page with multiple panels in a single image using advanced refinement techniques.

### Technical Features:
- **Multi-stage generation pipeline** with 3 refinement passes
- **Progressive quality enhancement** from base → refinement → upscaling → final polish
- **Advanced prompting** with comic book terminology and style specifications
- **Resolution progression**: 768x1024 → 1536x2048 final output
- **Multiple sampling algorithms** optimized for different stages
- **Comprehensive negative prompting** to avoid common comic book generation issues

### Workflow Stages:
1. **Base Generation** (Node 4): Initial comic page layout with DPM++2M sampler (30 steps, CFG 8.5)
2. **Refinement Pass** (Node 10): Character consistency and panel cleanup with Euler Ancestral (25 steps, CFG 7.0, 60% denoise)
3. **Upscaling** (Node 12): 2x resolution increase using nearest-exact interpolation
4. **Final Polish** (Node 16): Detail enhancement with DPM++SDE (20 steps, CFG 6.5, 35% denoise)

### Key Prompting Strategy:
```
Positive: "comic book page layout, multiple panels, heroic pirate rabbit adventure story, 
sequential art panels showing: panel 1 - spyglass scene, panel 2 - treasure map discovery, 
panel 3 - sword fighting, panel 4 - treasure finding, bold outlines, vibrant colors, 
professional comic illustration"

Negative: "inconsistent character design, mixed art styles, messy panels, poor panel layout, 
photorealistic, manga style, amateur comic art"
```

### Output Files:
- `pirate_rabbit_comic_stage1_*.png` - Initial generation
- `pirate_rabbit_comic_stage2_*.png` - After refinement
- `pirate_rabbit_comic_upscaled_*.png` - After upscaling
- `pirate_rabbit_comic_advanced_*.png` - Final polished result

---

## Workflow 2: `pirate_rabbit_comic_composite_v2.json` 
### Individual Panel Generation with Automatic Composition

**Concept**: Generates four separate comic panels as individual high-quality images, plus includes a Python script for automatic composition into a professional comic book page layout.

### Technical Features:
- **Four independent generation nodes** with unique seeds for variety
- **Character consistency prompting** across all panels
- **Story progression** with logical narrative flow
- **Panel-specific compositions** optimized for comic book storytelling
- **Upscaling capability** for each individual panel
- **Modular design** allowing individual panel regeneration

### Panel Breakdown:

#### Panel 1 - Discovery (Seed: 111)
**Scene**: Pirate rabbit with spyglass on ship deck
**Focus**: Establishing character and setting
**Prompt**: "looking through brass spyglass on ship deck, ocean horizon"

#### Panel 2 - Plot Development (Seed: 222)  
**Scene**: Finding treasure map in ship cabin
**Focus**: Story progression and excitement
**Prompt**: "excited expression, holding treasure map, ship interior cabin"

#### Panel 3 - Action Climax (Seed: 333)
**Scene**: Sword fighting with skeleton pirate
**Focus**: Dynamic action and conflict
**Prompt**: "dynamic action scene, sword fighting duel, crossed cutlass swords, action lines"

#### Panel 4 - Resolution (Seed: 444)
**Scene**: Triumphant discovery of treasure
**Focus**: Victory and conclusion
**Prompt**: "triumphant pose over treasure chest, tropical island beach, victory pose"

### Character Consistency Strategy:
- Same model checkpoint (SDXL Base 1.0) for all panels
- Consistent character description: "pirate rabbit captain with tricorn hat and eye patch"
- Comic book style specification in every prompt
- Shared negative prompts to avoid style inconsistencies

### Output Files:
- `comic_panel_1_spyglass_*.png` - Panel 1: Discovery scene
- `comic_panel_2_map_*.png` - Panel 2: Treasure map scene  
- `comic_panel_3_battle_*.png` - Panel 3: Battle scene
- `comic_panel_4_treasure_*.png` - Panel 4: Treasure discovery scene
- `pirate_rabbit_comic_page_*.png` - Complete comic page (via Python compositor)

### Automatic Composition Features:
- **Professional comic book layout** with 2x2 grid arrangement
- **Comic book style borders** (8px black borders around each panel)
- **Proper spacing and gutters** (20px between panels, 30px page margins)
- **Title and credits area** with comic book styling
- **High-quality output** optimized for print and digital viewing
- **Automatic file detection** finds the most recent panel files

---

## Usage Instructions

### Method 1: Complete Multi-Panel Comic with Composition
```bash
# Generate 4 comic panels
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "comic-panels",
    "prompt": '$(cat ./scripts/resources/automation/comfyui/examples/pirate_rabbit_comic_composite_v2.json)'
  }'

# Wait for generation to complete, then compose into comic page
python3 ./scripts/resources/automation/comfyui/examples/composite_comic_panels.py
```

### Method 2: Via Management Script
```bash
# Execute single-page comic workflow
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow \
  --workflow ./scripts/resources/automation/comfyui/examples/pirate_rabbit_comic_advanced.json

# Execute multi-panel workflow  
./scripts/resources/automation/comfyui/manage.sh --action execute-workflow \
  --workflow ./scripts/resources/automation/comfyui/examples/pirate_rabbit_comic_composite_v2.json
```

### Method 2: Via ComfyUI Web Interface
1. Open http://localhost:8188
2. Click "Load" button
3. Select desired workflow JSON file
4. Adjust parameters if needed
5. Click "Queue Prompt"

### Method 3: Via API
```bash
# Single-page comic
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "comic-generation",
    "prompt": '$(cat /home/matthalloran8/.comfyui/workflows/pirate_rabbit_comic_advanced.json)'
  }'

# Multi-panel comic  
curl -X POST http://localhost:8188/prompt \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "comic-panels",
    "prompt": '$(cat /home/matthalloran8/.comfyui/workflows/pirate_rabbit_comic_multi_panel.json)'
  }'
```

## Advanced Customization

### Modifying Panel Content
Edit the text prompts in nodes 2, 8, 12, 16 for different story elements:
```json
"text": "YOUR_CUSTOM_PANEL_DESCRIPTION_HERE"
```

### Adjusting Art Style
Modify the style keywords in prompts:
- Change "comic book art style" to "manga style", "graphic novel style", etc.
- Add specific artist references: "Jim Lee style", "Jack Kirby style"
- Adjust color schemes: "noir comic", "watercolor comic", "vintage comic"

### Seed Variation
Change seed values for different character poses and compositions:
```json
"seed": 42  // Change to any number for variations
```

### Resolution Scaling
Adjust dimensions in EmptyLatentImage and ImageScale nodes:
```json
"width": 1024,   // Increase for higher resolution
"height": 1024   // Maintain aspect ratio
```

## Performance Considerations

- **Single-page workflow**: ~2-3 minutes on RTX 4070 Ti SUPER
- **Multi-panel workflow**: ~4-5 minutes (4 separate generations)
- **Memory usage**: ~6-8GB VRAM for SDXL at high resolution
- **Recommended settings**: 16GB+ system RAM, 8GB+ VRAM

## Troubleshooting

### Common Issues:
1. **Inconsistent character across panels**: Ensure same checkpoint and consistent character description
2. **Low quality results**: Check model integrity with status command
3. **Out of memory**: Reduce resolution or batch size
4. **Slow generation**: Verify GPU acceleration is working

### Quality Optimization:
- Use CFG 7-8.5 for optimal detail/coherence balance
- Keep denoise values: 1.0 for initial, 0.3-0.6 for refinement
- Use DPM++2M or Euler Ancestral for best comic book results

## Integration with External Tools

### Post-Processing with External Software:
1. **Panel borders**: Add using GIMP, Photoshop, or Krita
2. **Speech bubbles**: Use comic creation software like Comic Life or Clip Studio Paint
3. **Text overlay**: Add dialogue and sound effects in post-production
4. **Page composition**: Arrange individual panels into full page layouts

### Automation Possibilities:
- Create n8n workflows to automatically generate comic series
- Use batch processing to create multiple character variations
- Integrate with print-on-demand services for physical comic creation

---

**Created by**: Claude Code AI Assistant  
**Version**: Advanced Comic Generation v1.0  
**Compatible with**: ComfyUI + SDXL Base 1.0 model