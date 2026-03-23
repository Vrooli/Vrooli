# Speaker Verification Troubleshooting

## Startup Issues

### Service takes a long time to start

The NeMo TitaNet model needs to be downloaded on first run (~500MB). Subsequent starts use the cached model.

- Check logs: `resource-speaker-verification logs`
- The startup timeout is 180 seconds by default. Increase `SPEAKER_VERIFICATION_STARTUP_MAX_WAIT` if needed.

### Container starts but /ready never succeeds

The model may have failed to load. Check logs for Python errors:

```bash
resource-speaker-verification logs
```

Common causes:
- Insufficient memory (TitaNet needs ~2GB RAM)
- Corrupted model cache — clear with `rm -rf ~/.speaker-verification/cache`
- Missing CUDA libraries when `DEVICE=cuda` — switch to `DEVICE=cpu`

### Port already in use

```bash
# Check what's using the port
ss -tlnp | grep 8891

# Use a custom port
export SPEAKER_VERIFICATION_CUSTOM_PORT=8892
resource-speaker-verification manage install
```

## GPU Issues

### GPU detected but inference is slow

Ensure the NVIDIA Docker runtime is properly configured:

```bash
docker info | grep -i nvidia
nvidia-smi
```

### CUDA out of memory

TitaNet-Large uses ~1GB GPU memory. If other processes are using the GPU:

```bash
# Force CPU mode
export SPEAKER_VERIFICATION_DEVICE=cpu
resource-speaker-verification manage restart
```

## Verification Issues

### False rejections (legitimate speaker not matching)

- Check the similarity score in the response — if it's close to the threshold, lower it
- Ensure enrollment audio is at least 5 seconds of clear speech
- Ensure verification audio has minimal background noise
- Try re-enrolling with higher-quality audio

### False acceptances (wrong speaker matching)

- Increase the threshold (e.g., 0.8 or 0.85)
- Ensure enrollment audio contains only the target speaker
- Use longer enrollment audio for a more robust voiceprint

### "Audio too short" error

- Enrollment requires at least 3 seconds (configurable via `SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS`)
- Verification requires at least 1 second (configurable via `SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS`)

### "Audio appears to be silence" error

The audio file contains no meaningful signal. Ensure the audio file:
- Is not a silence/blank recording
- Has correct encoding (WAV 16-bit PCM recommended)
- Is not corrupt

## Permission Issues

### Profile store not writable

```bash
# Check directory permissions
ls -la ~/.speaker-verification/profiles/

# Fix permissions
chmod -R 755 ~/.speaker-verification/
```

### Docker permission denied

```bash
# Add user to docker group
sudo usermod -aG docker $USER
# Log out and back in for group changes to take effect
```

## Data Management

### Removing all profiles

```bash
rm -rf ~/.speaker-verification/profiles/*
```

### Clearing model cache

```bash
rm -rf ~/.speaker-verification/cache/*
resource-speaker-verification manage restart
```

### Full reset

```bash
resource-speaker-verification manage uninstall --force
rm -rf ~/.speaker-verification/
resource-speaker-verification manage install
```
