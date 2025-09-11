# GGWave Resource

Air-gapped data transmission via FSK-modulated audio signals. Transfer data securely through sound waves without any network connectivity.

## Quick Start

```bash
# Install and start the resource
vrooli resource ggwave manage install
vrooli resource ggwave manage start --wait

# Check status
vrooli resource ggwave status

# Test transmission
vrooli resource ggwave content execute --data "Hello World" --mode normal
```

## Features

- **Multiple Transmission Modes**:
  - Normal: 8-64 bytes/sec (audible, reliable)
  - Fast: 32-64 bytes/sec (audible, faster)
  - DT (Dual-Tone): 64-500 bytes/sec (audible, highest speed)
  - Ultrasonic: 8-64 bytes/sec (inaudible to humans)

- **Error Correction**: Reed-Solomon error correction for reliable transmission
- **Cross-Platform**: Works on iOS, Android, Linux, Arduino
- **Standard Hardware**: Uses any microphone/speaker - no special equipment needed
- **Range**: 1-5 meters typical indoor range

## Use Cases

1. **Secure Credential Exchange**: Transfer API keys and passwords air-gapped
2. **IoT Device Pairing**: Configure devices without network access
3. **Offline Payments**: Send payment data in network-dead zones
4. **Emergency Communication**: Critical data when networks are down
5. **Interactive Installations**: Creative audio-based experiences

## API Endpoints

```bash
# Encode data to audio
curl -X POST http://localhost:8196/api/encode \
  -H "Content-Type: application/json" \
  -d '{"data": "Hello World", "mode": "normal"}'

# Decode audio to data
curl -X POST http://localhost:8196/api/decode \
  -H "Content-Type: application/json" \
  -d '{"audio": "<base64_audio>", "mode": "auto"}'

# Health check
curl http://localhost:8196/health
```

## Configuration

Edit `config/defaults.sh` to customize:

```bash
GGWAVE_MODE="auto"           # Transmission mode
GGWAVE_SAMPLE_RATE="48000"   # Audio sample rate
GGWAVE_VOLUME="0.8"          # Output volume (0.0-1.0)
GGWAVE_ERROR_CORRECTION="true" # Enable Reed-Solomon
```

## Testing

```bash
# Run all tests
vrooli resource ggwave test all

# Quick health check
vrooli resource ggwave test smoke

# Test specific transmission mode
vrooli resource ggwave test integration --mode ultrasonic
```

## Requirements

- Docker
- Audio input/output devices (microphone/speaker)
- 48kHz sample rate support for ultrasonic mode

## Documentation

- [API Reference](docs/API.md)
- [Transmission Modes](docs/MODES.md)
- [Integration Guide](docs/INTEGRATION.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

## License

MIT License - Based on [ggwave](https://github.com/ggerganov/ggwave) by Georgi Gerganov