# 🎮 QR Code Generator

A fun retro-style QR code generator with customization options and batch processing capabilities.

## Purpose
Fun retro-style QR code generator with customization, batch processing, and extensible architecture. Demonstrates Vrooli's ability to create practical utilities with modern web interfaces.

## Key Features
- **Standard QR Generation**: Create QR codes for URLs, text, and any data
- **Batch Processing**: Process multiple QR codes via API or CLI
- **Customization**: Control size, error correction level
- **Retro UI**: 80s arcade aesthetic with neon colors
- **Multiple Interfaces**: Web UI, REST API, and CLI tool

## Quick Start

```bash
# Start the scenario
make run

# Or use Vrooli CLI
vrooli scenario run qr-code-generator
```

## Access Points

- **Web UI**: Dynamic port (shown on startup) - Retro gaming style interface
- **API**: Dynamic port (shown on startup) - REST API endpoints  
- **CLI**: `qr-generator` command with automatic port detection

## API Endpoints

### Health Check
```bash
GET /health
```
Returns service status and available features.

### Generate QR Code
```bash
POST /generate
Content-Type: application/json

{
  "text": "Hello World",
  "size": 256,
  "errorCorrection": "Medium"
}
```
Returns base64-encoded PNG image.

### Batch Generate
```bash
POST /batch
Content-Type: application/json

{
  "items": [
    {"text": "Item 1", "label": "First"},
    {"text": "Item 2", "label": "Second"}
  ],
  "options": {
    "size": 256
  }
}
```
Returns array of base64-encoded QR codes.

### Get Supported Formats
```bash
GET /formats
```
Returns supported formats, sizes, and error correction levels.

## CLI Usage

### Generate Single QR Code
```bash
# Display JSON response
qr-generator generate "Hello World"

# Save to file
qr-generator generate "Hello World" --output hello.png

# Custom size
qr-generator generate "Hello World" --size 512 --output large.png
```

### Batch Processing
```bash
# Create a file with URLs/text (one per line)
echo -e "https://vrooli.com\nhttps://github.com" > urls.txt

# Process batch
qr-generator batch urls.txt --size 256
```

### Help
```bash
qr-generator help
```

## Configuration

The scenario uses dynamic port allocation:
- `API_PORT`: API server port (allocated by Vrooli)
- `UI_PORT`: Web UI port (allocated by Vrooli)

Ports are automatically detected by the CLI and UI.

## Dependencies
- **Go 1.21+**: API server implementation
- **Node.js 18+**: UI server
- **Optional Resources**:
  - **n8n**: Advanced workflows (when available)
  - **Redis**: Caching (when available)

## Architecture
- **API**: Go HTTP server with `github.com/skip2/go-qrcode` library
- **UI**: Node.js/Express with retro gaming theme
- **CLI**: Bash wrapper with automatic port detection
- **Lifecycle**: Fully managed by Vrooli v2.0 system

## Testing

```bash
# Run tests
make test

# Check status
make status

# View logs
make logs
```

## Error Correction Levels

- `Low` (L) - ~7% correction capability
- `Medium` (M) - ~15% correction (default)
- `High` (H) - ~25% correction
- `Highest` (Q) - ~30% correction

## Performance

- **Generation Time**: <100ms per QR code
- **Batch Processing**: Parallel generation for efficiency
- **Caching**: Redis integration ready (when available)

## Troubleshooting

### API not responding
```bash
make status              # Check if running
make logs               # View recent logs
make stop && make run   # Restart scenario
```

### UI not loading
```bash
vrooli scenario logs qr-code-generator --step start-ui
```

### CLI can't find API
The CLI automatically detects the API port. Override if needed:
```bash
API_URL=http://localhost:PORT qr-generator generate "test"
```

## Development

```bash
# Build API
cd api && go build -o qr-code-generator-api .

# Install UI dependencies
cd ui && npm install

# Install CLI
cd cli && ./install.sh
```

## Business Value

This QR code generator provides:
- **Privacy**: Local generation, no data sent to third parties
- **Speed**: Sub-100ms generation time
- **Flexibility**: Multiple interfaces for different use cases
- **Scalability**: Batch processing for marketing campaigns
- **Integration**: REST API for seamless integration

Target market: Small businesses, event organizers, developers needing quick QR generation without external dependencies.

Revenue potential: $15K monthly from 300 small businesses at $50/month for unlimited generation with customization.

## Future Enhancements (P1/P2)

- **QR Art Generation**: AI-enhanced artistic QR codes
- **URL Shortening**: Integrated shortener for long URLs
- **Analytics**: Track generation statistics
- **Logo Embedding**: Add logos to QR center
- **More Formats**: SVG, PDF, JPEG export

## License

Part of the Vrooli ecosystem - see main repository for license details.