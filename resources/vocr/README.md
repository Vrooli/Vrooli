# VOCR (Vision OCR) Resource

Advanced screen recognition and accessibility tool with AI-powered image analysis, enabling Vrooli to "see" and interact with any screen content.

## Overview

VOCR integrates OCR (Optical Character Recognition) with AI vision capabilities to enable:
- Screen reading and interaction with any application
- Real-time OCR of screen regions
- AI-powered image understanding and question answering
- Accessibility features for automation scenarios
- Cross-platform screen capture and analysis

## Features

- **Real-time OCR**: Extract text from any screen region
- **AI Vision**: Ask questions about images and get intelligent responses
- **Screen Navigation**: Programmatically interact with UI elements
- **Accessibility Integration**: Works with VoiceOver and other screen readers
- **Format Support**: Screenshots, PDFs, images, and live screen capture
- **Multi-language**: Supports 100+ languages for OCR

## Use Cases

### Automation Scenarios
- Monitor and interact with desktop applications
- Extract data from non-API systems
- Automate legacy software interactions
- Visual testing and validation
- Screen monitoring and alerts

### Accessibility
- Screen reading for visually impaired users
- Content extraction from inaccessible applications
- Real-time translation of on-screen text

### Data Extraction
- Extract tables from screenshots
- Read text from images and PDFs
- Monitor dashboards and reports
- Capture data from video streams

## Installation

```bash
vrooli resource install vocr
```

## Configuration

VOCR requires permissions for screen capture and accessibility:

```bash
# Grant screen recording permission (macOS)
resource-vocr configure

# Test screen capture
resource-vocr test-capture
```

## Usage

### CLI Commands

```bash
# Check status
resource-vocr status

# Capture screen region
resource-vocr capture --region "100,100,500,400"

# Extract text from screen
resource-vocr ocr --region "0,0,1920,1080"

# Ask AI about current screen
resource-vocr ask "What application is currently open?"

# Monitor region for changes
resource-vocr monitor --region "100,100,200,50" --interval 5
```

### Integration Examples

```javascript
// n8n workflow node
{
  "name": "VOCR Screen Reader",
  "type": "vocr",
  "operation": "ocr",
  "region": "{{$json.region}}",
  "language": "en"
}
```

## API Endpoints

- `POST /capture` - Capture screen region
- `POST /ocr` - Extract text from image
- `POST /ask` - Ask AI about image
- `GET /monitor` - Monitor screen changes

## Architecture

VOCR uses a modular architecture:
- **Capture Engine**: Platform-specific screen capture
- **OCR Engine**: Tesseract/EasyOCR for text extraction
- **Vision Model**: AI model for image understanding
- **API Server**: REST interface for integrations

## Requirements

- Python 3.8+
- Tesseract OCR engine
- Screen capture permissions
- Optional: GPU for faster processing

## Performance

- OCR Speed: ~100ms for 1920x1080 screen
- AI Response: ~500ms for vision queries
- Memory: 200-500MB depending on models
- CPU: Minimal when idle, spikes during processing

## Security

- All processing is done locally
- No data sent to external services
- Screenshots stored temporarily and auto-deleted
- Configurable retention policies

## Troubleshooting

### Permission Issues
```bash
# Check permissions
resource-vocr diagnose

# Reset permissions
resource-vocr reset-permissions
```

### Performance Issues
```bash
# Use GPU acceleration
resource-vocr configure --gpu

# Adjust quality settings
resource-vocr configure --quality medium
```

## Related Resources

- **browserless**: For web-specific automation
- **ollama**: Local vision models
- **judge0**: Code execution from screenshots