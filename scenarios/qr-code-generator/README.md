# QR Code Generator

## Purpose
Fun retro-style QR code generator with AI-enhanced customization, batch processing, and artistic themes. Demonstrates Vrooli's ability to combine traditional utilities with AI creativity.

## Key Features
- **Standard QR Generation**: Create QR codes for URLs, text, WiFi, emails, phone numbers
- **AI Art Mode**: Generate themed ASCII art QR codes with captions using Ollama
- **Batch Processing**: Process multiple QR codes via CSV upload
- **Puzzle Mode**: Create QR code puzzles for gamification
- **Retro UI**: 80s arcade aesthetic with neon colors and CRT effects

## Dependencies
- **n8n**: Workflow automation for batch processing and AI features
- **Redis**: Caching generated codes and tracking batch jobs
- **Ollama** (via shared workflow): AI text generation for captions and ASCII art

## Architecture
- **API**: Go server handling QR generation and caching
- **UI**: Retro JavaScript interface with animated effects
- **CLI**: Command-line tool for quick QR generation
- **Workflows**: Four n8n workflows for different generation modes

## Shared Resources
Uses these shared n8n workflows:
- `ollama.json`: AI text generation
- `intelligent-text-classifier.json`: Smart content type detection
- `file-list-writer.json`: Batch output handling

## UI Style
**Theme**: 80s arcade retro
- Neon green/pink color scheme
- CRT screen effects with scanlines
- Pixel fonts and glitch animations
- Sound effects for interactions

## Use Cases
- Quick QR codes for sharing URLs/contact info
- Marketing materials with branded QR art
- Educational puzzles and scavenger hunts
- Batch generation for events/inventory
- Fun retro-themed QR codes for social media

## CLI Usage
```bash
qr-generator "https://vrooli.com" --theme cyberpunk --size 512
qr-generator batch process.csv --output results/
qr-generator art "Secret Message" --caption --ascii
```

## Integration Potential
Other scenarios can leverage this for:
- Adding QR codes to generated reports
- Creating shareable links for apps
- Batch processing for inventory systems
- Gamification elements in other UIs