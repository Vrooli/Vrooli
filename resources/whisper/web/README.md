# Whisper Web Test Tools

This directory contains web-based test interfaces for the Whisper speech-to-text service.

## Files

### whisper-microphone-test.html
A unified microphone recording interface with two connection modes:

**Features:**
- Real-time audio visualizer showing frequency spectrum
- Two connection modes with easy toggle:
  - **SSH Tunnel Mode** (Recommended): Uses localhost:8090 via SSH tunnel
  - **Direct Connection Mode**: Connects directly to server IP
- Audio playback of recordings
- Detailed error messages and debugging info
- Saves server URL preference in browser localStorage

**Usage:**

1. **SSH Tunnel Mode (Recommended):**
   ```bash
   # On your local machine, create tunnel
   ssh -L 8090:localhost:8090 user@server
   
   # Open whisper-microphone-test.html in browser
   # Select "SSH Tunnel" mode and record
   ```

2. **Direct Connection Mode:**
   ```bash
   # Serve the HTML file (to avoid CORS issues)
   python -m http.server 8000
   
   # Browse to http://localhost:8000/whisper-microphone-test.html
   # Select "Direct Connection" and enter server IP
   ```

### file-upload-test.html
A drag-and-drop file upload interface for testing Whisper transcription.

**Features:**
- Drag-and-drop or click to upload audio files
- Supports MP3, WAV, OGG, M4A, FLAC, AAC, WebM formats
- Shows file size and type information
- Audio preview player
- Works well over RDP connections (no microphone needed)

**Usage:**
```bash
# Open directly on server via RDP
xdg-open /path/to/file-upload-test.html

# Or serve locally and connect to server
python -m http.server 8000
```

## Troubleshooting

### CORS Issues
- Direct connection mode may fail due to CORS when opening HTML as `file://`
- Solution: Use SSH tunnel mode or serve HTML via HTTP server

### No Audio Detected
- Check browser microphone permissions
- Ensure microphone is not in use by another application
- For RDP users: Use file-upload-test.html instead

### Transcription Errors
- Verify Whisper is running: `docker ps | grep whisper`
- Check Whisper logs: `docker logs whisper`
- Test API directly: `curl http://localhost:8090/docs`

## Development Notes

The visualizer uses Web Audio API with:
- FFT size: 256 (128 frequency bins)
- Color gradient: Green (quiet) → Yellow (medium) → Red (loud)
- Canvas rendering at ~60fps via requestAnimationFrame

Both test files are standalone with no external dependencies.