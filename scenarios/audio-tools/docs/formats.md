# Audio Formats Guide

## Supported Formats

Audio Tools supports all major audio formats through FFmpeg. This guide provides details on each format, when to use them, and optimal settings.

## Quick Reference

| Format | Extension | Type | Best For | Quality | File Size |
|--------|-----------|------|----------|---------|-----------|
| MP3 | .mp3 | Lossy | General use, streaming | Good | Small |
| WAV | .wav | Lossless | Professional editing | Excellent | Large |
| FLAC | .flac | Lossless | Archival, high quality | Excellent | Medium |
| AAC | .aac, .m4a | Lossy | Modern streaming | Very Good | Small |
| OGG | .ogg | Lossy | Open source projects | Good | Small |

## Format Details

### MP3 (MPEG-1 Audio Layer 3)

**Overview:**
- Most widely supported audio format
- Lossy compression with good quality/size ratio
- Variable and constant bitrate options

**Specifications:**
- Sample rates: 8-48 kHz
- Bitrates: 32-320 kbps
- Channels: Mono, Stereo, Joint Stereo

**Recommended Settings:**
```bash
# High quality (music)
audio-tools convert input.wav output.mp3 --bitrate 320k

# Standard quality (podcasts)
audio-tools convert input.wav output.mp3 --bitrate 192k

# Low quality (voice only)
audio-tools convert input.wav output.mp3 --bitrate 96k

# Variable bitrate (best quality/size ratio)
audio-tools convert input.wav output.mp3 --vbr --quality 0
```

**Use Cases:**
- Music distribution
- Podcast publishing
- General audio sharing
- Web streaming

### WAV (Waveform Audio File Format)

**Overview:**
- Uncompressed audio format
- Industry standard for professional audio
- No quality loss

**Specifications:**
- Sample rates: 8-192 kHz
- Bit depths: 8, 16, 24, 32 bit
- Channels: Up to 18 channels

**Recommended Settings:**
```bash
# CD quality
audio-tools convert input.mp3 output.wav --sample-rate 44100 --bit-depth 16

# Studio quality
audio-tools convert input.mp3 output.wav --sample-rate 48000 --bit-depth 24

# High resolution
audio-tools convert input.mp3 output.wav --sample-rate 96000 --bit-depth 24
```

**Use Cases:**
- Professional audio production
- Audio mastering
- Intermediate processing format
- Archival (when FLAC not suitable)

### FLAC (Free Lossless Audio Codec)

**Overview:**
- Lossless compression (50-60% of WAV size)
- Perfect quality preservation
- Open source format

**Specifications:**
- Sample rates: 8-192 kHz
- Bit depths: 8, 16, 24 bit
- Compression levels: 0-8

**Recommended Settings:**
```bash
# Standard compression (balanced)
audio-tools convert input.wav output.flac --compression 5

# Maximum compression (smallest file)
audio-tools convert input.wav output.flac --compression 8

# Fast compression (for speed)
audio-tools convert input.wav output.flac --compression 0

# With metadata preservation
audio-tools convert input.wav output.flac --preserve-metadata
```

**Use Cases:**
- Music archival
- High-quality distribution
- Professional audio storage
- Audiophile releases

### AAC (Advanced Audio Coding)

**Overview:**
- Successor to MP3
- Better quality at same bitrates
- Standard for Apple devices

**Specifications:**
- Sample rates: 8-96 kHz
- Bitrates: 8-512 kbps
- Profiles: LC, HE, HE v2

**Recommended Settings:**
```bash
# High quality (Apple Music quality)
audio-tools convert input.wav output.m4a --bitrate 256k

# Standard quality
audio-tools convert input.wav output.aac --bitrate 192k

# Low bitrate with HE-AAC
audio-tools convert input.wav output.aac --profile he --bitrate 64k
```

**Use Cases:**
- iTunes/Apple Music
- YouTube audio
- Mobile streaming
- Modern podcasts

### OGG Vorbis

**Overview:**
- Open source lossy format
- Better than MP3 at low bitrates
- Free from patents

**Specifications:**
- Sample rates: 8-192 kHz
- Bitrates: 45-500 kbps
- Quality levels: -1 to 10

**Recommended Settings:**
```bash
# High quality
audio-tools convert input.wav output.ogg --quality 8

# Standard quality
audio-tools convert input.wav output.ogg --quality 5

# Low bitrate
audio-tools convert input.wav output.ogg --quality 2
```

**Use Cases:**
- Open source projects
- Game audio
- Web audio (Firefox/Chrome)
- Streaming services

## Codec Comparison

### Quality Comparison at 128 kbps
1. AAC (best)
2. OGG Vorbis
3. MP3 (acceptable)

### File Size Comparison (3-minute song)
- WAV (16-bit/44.1kHz): ~30 MB
- FLAC: ~15-20 MB
- MP3 (320k): ~7 MB
- MP3 (192k): ~4.5 MB
- AAC (256k): ~6 MB
- OGG (q5): ~4 MB

## Conversion Guidelines

### Lossy to Lossy
**Warning:** Avoid converting between lossy formats when possible.
```bash
# Bad: MP3 → AAC (quality loss)
# Good: Original → MP3 and Original → AAC
```

### Upsampling
**Note:** Cannot improve quality by upsampling.
```bash
# Pointless: 128k MP3 → 320k MP3
# Pointless: 44.1kHz → 96kHz (unless required)
```

### Recommended Conversion Paths
```
Original Recording
    ├── WAV/FLAC (archival)
    ├── MP3 320k (distribution)
    ├── MP3 192k (streaming)
    └── AAC 256k (mobile)
```

## Quality Settings

### Bitrate Recommendations

**Music:**
- Lossless: FLAC or WAV
- High: 320 kbps MP3 or 256 kbps AAC
- Standard: 192 kbps MP3 or 192 kbps AAC
- Acceptable: 128 kbps MP3 or 128 kbps AAC

**Speech/Podcasts:**
- High: 128 kbps MP3
- Standard: 96 kbps MP3
- Low: 64 kbps MP3 or HE-AAC

**Audiobooks:**
- Stereo: 64-96 kbps MP3
- Mono: 32-64 kbps MP3

### Sample Rate Guidelines

**Standard Rates:**
- 44.1 kHz: CD audio, music distribution
- 48 kHz: Video production, broadcasting
- 96 kHz: High-resolution audio
- 192 kHz: Studio mastering

**Conversion Rules:**
- Maintain original sample rate when possible
- Downsample by integer factors (96→48, 88.2→44.1)
- Use high-quality resampling algorithms

## Metadata Support

### ID3 Tags (MP3)
```bash
audio-tools metadata input.mp3 --set \
  title="Song Title" \
  artist="Artist Name" \
  album="Album Name" \
  year="2024" \
  genre="Rock"
```

### Vorbis Comments (FLAC/OGG)
```bash
audio-tools metadata input.flac --set \
  TITLE="Song Title" \
  ARTIST="Artist Name" \
  ALBUM="Album Name" \
  DATE="2024"
```

### iTunes Tags (M4A/AAC)
```bash
audio-tools metadata input.m4a --set \
  title="Song Title" \
  artist="Artist Name" \
  album="Album Name" \
  year="2024" \
  artwork="cover.jpg"
```

## Container Formats

### Common Containers
- **M4A**: AAC audio in MPEG-4 container
- **MP4**: Can contain AAC audio
- **MKA**: Matroska audio container
- **WEBM**: Can contain Opus or Vorbis

### Extraction from Video
```bash
# Extract audio from video
audio-tools extract video.mp4 audio.mp3

# Extract without re-encoding
audio-tools extract video.mp4 audio.aac --copy
```

## Advanced Encoding Options

### MP3 Encoding
```bash
# Constant bitrate
audio-tools convert input.wav output.mp3 --cbr 256

# Variable bitrate (recommended)
audio-tools convert input.wav output.mp3 --vbr --quality 0

# Average bitrate
audio-tools convert input.wav output.mp3 --abr 192
```

### FLAC Encoding
```bash
# Maximum compression
audio-tools convert input.wav output.flac \
  --compression 8 \
  --verify \
  --exhaustive-model

# Fast encoding
audio-tools convert input.wav output.flac \
  --compression 0 \
  --fast
```

### AAC Encoding
```bash
# Apple AAC encoder (best quality)
audio-tools convert input.wav output.m4a \
  --encoder aac_at \
  --bitrate 256k

# FDK-AAC (good open source)
audio-tools convert input.wav output.aac \
  --encoder libfdk_aac \
  --profile aac_he_v2
```

## Special Formats

### Opus (Modern codec)
```bash
# High efficiency for speech
audio-tools convert input.wav output.opus --bitrate 32k
```

### AC3 (Dolby Digital)
```bash
# For video production
audio-tools convert input.wav output.ac3 --bitrate 384k
```

### DTS
```bash
# For surround sound
audio-tools convert input.wav output.dts --channels 6
```

## Format Detection

### Automatic Detection
```bash
# Let the tool detect format
audio-tools info unknown_file

# Force format detection
audio-tools convert mystery.audio output.mp3 --input-format auto
```

### Check Format Support
```bash
# List all supported formats
audio-tools formats --list

# Check specific format
audio-tools formats --check opus
```

## Troubleshooting

### Common Issues

**"Unsupported format" error:**
```bash
# Check FFmpeg support
audio-tools doctor --check-codecs

# Update FFmpeg
audio-tools doctor --update-ffmpeg
```

**Quality loss after conversion:**
```bash
# Use higher bitrate
audio-tools convert input.mp3 output.mp3 --bitrate 320k

# Or use lossless
audio-tools convert input.mp3 output.flac
```

**Large file sizes:**
```bash
# Optimize for size
audio-tools optimize input.wav output.mp3 --target-size 5MB
```

## Best Practices

### For Archival
1. Use FLAC or WAV
2. Preserve original sample rate
3. Keep metadata intact
4. Store checksums

### For Distribution
1. Provide multiple formats
2. Use standard bitrates
3. Include proper metadata
4. Test on target devices

### For Streaming
1. Use AAC or MP3
2. Consider adaptive bitrate
3. Optimize for bandwidth
4. Test latency

### For Editing
1. Use WAV or FLAC
2. Work at highest quality
3. Convert only at final step
4. Keep project files

## Performance Considerations

### Encoding Speed
Fastest to Slowest:
1. WAV (no encoding)
2. MP3 (fast encoder)
3. AAC
4. OGG Vorbis
5. FLAC (max compression)

### Decoding Speed
Fastest to Slowest:
1. WAV
2. FLAC
3. MP3
4. AAC
5. OGG Vorbis

### Memory Usage
- WAV: High (uncompressed in memory)
- FLAC: Medium
- MP3/AAC/OGG: Low

## Future Format Support

Planned additions:
- Opus (efficient modern codec)
- WMA (Windows Media Audio)
- APE (Monkey's Audio)
- ALAC (Apple Lossless)
- DSD (Direct Stream Digital)