# Palette Gen - AI-Powered Color Palette Generator

## Overview
Palette Gen is a sophisticated color palette generation tool that combines AI intelligence with color theory principles to create harmonious, accessible color schemes for designers and developers. It goes beyond simple color picking by analyzing harmony relationships, ensuring WCAG compliance, and providing context-aware recommendations.

## Why It's Useful
- **AI-Driven Generation**: Uses LLMs to understand themes and contexts for more meaningful palettes
- **Accessibility First**: Built-in WCAG compliance checking ensures usable color combinations
- **Harmony Analysis**: Validates color relationships using established color theory principles
- **Export Flexibility**: Multiple export formats for seamless integration into design workflows
- **Educational**: Helps users understand why certain colors work together

## Capabilities & Features
- Theme-based palette generation (e.g., "ocean sunset", "tech startup", "vintage bookstore")
- Multiple style presets (vibrant, pastel, dark, minimal, earthy, neon)
- **Color harmony analysis** (complementary, analogous, triadic, monochromatic) ✨
- **WCAG accessibility compliance** checking with detailed contrast ratios
- **Colorblind simulation** (protanopia, deuteranopia, tritanopia) for inclusive design ✨
- **Redis caching** for performance optimization with cache headers ✨
- **Palette history tracking** with retrieval of recent generations ✨
- Export to CSS, SCSS, JSON formats
- **AI-powered suggestions** via Ollama with graceful fallback
- Quick suggestion templates for common use cases

## Dependencies on Other Scenarios
- Uses shared `ollama.json` workflow for AI inference
- Leverages `chain-of-thought-orchestrator` for complex color analysis

## Scenarios It Enhances
- **brand-manager**: Provides color palettes for complete branding packages
- **app-personalizer**: Supplies custom color schemes for personalized app experiences
- **document-manager**: Offers accessible color combinations for documentation
- **retro-game-launcher**: Generates nostalgic color palettes for game themes
- **study-buddy**: Creates calming, focus-enhancing color schemes

## UX Style & Design Philosophy
The UI embraces a clean, professional aesthetic that lets the colors take center stage. Key design elements:
- **Minimalist Interface**: Uncluttered layout that doesn't compete with generated palettes
- **Visual Feedback**: Instant preview of palettes with smooth animations
- **Interactive Elements**: Clickable color swatches with copy-to-clipboard functionality
- **Gradient Previews**: Shows color transitions and relationships visually
- **Dark/Light Toggle**: Interface adapts to showcase palettes in different contexts

## Technical Architecture
- **API**: Go-based REST API for palette generation and analysis (standalone implementation)
- **Algorithms**: Built-in color generation using HSL color space and theme-based hue mapping
- **Styles**: Vibrant, Pastel, Dark, Minimal, Earthy - each with unique generation parameters
- **Export**: Multiple format support (CSS variables, JSON array, SCSS variables)
- **Caching**: Redis-based caching with 24-hour TTL for generated palettes
- **History**: Redis sorted sets for tracking palette generation history (7-day retention)
- **AI Integration**: Ollama integration for contextual palette suggestions with fallback
- **Accessibility**: Full WCAG compliance checking and colorblind simulation

## Use Cases
1. **Web Designers**: Generate cohesive color schemes for websites
2. **Brand Designers**: Create brand-appropriate color palettes
3. **UI/UX Teams**: Ensure accessible color combinations in interfaces
4. **Artists**: Explore color relationships and harmony
5. **Developers**: Quickly generate theme colors for applications

## Recent Enhancements (2025-10-03)
- ✅ Redis caching for improved performance
- ✅ Color harmony analysis with relationship detection
- ✅ Colorblind simulation for accessibility
- ✅ Palette history tracking and retrieval
- ✅ Enhanced CLI with new commands

## Future Enhancements (P2)
- Integration with image analysis for palette extraction
- Seasonal and trending palette suggestions
- Color psychology insights
- Collaborative palette sharing
- Adobe/Figma plugin exports