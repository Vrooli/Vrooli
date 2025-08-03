# Whisper Test Audio Files

This directory contains test audio files for verifying Whisper functionality across different scenarios and edge cases.

## Test Files

### Original Test Files
- **test_short.mp3** (101KB) - Short music/audio sample for basic functionality testing
- **test_silent.wav** (862KB) - 5-second silent audio to test edge cases and no-speech detection
- **test_speech.mp3** (940KB) - Martin Luther King Jr. speech excerpt for real speech transcription testing

### Extended Test Coverage
- **test_very_short.wav** (32KB) - 1-second tone for minimal processing time testing
- **test_noise.mp3** (9.3KB) - White noise sample to test handling of non-speech audio
- **test_format.flac** (20KB) - FLAC format test to verify format compatibility
- **test_quiet.wav** (63KB) - Extremely quiet audio (1% volume) to test low-volume handling
- **test_corrupted.wav** (1KB) - Corrupted/invalid audio file to test error handling

## Test Coverage Areas

- **Format Support**: MP3, WAV, FLAC formats
- **Content Types**: Speech, music, noise, silence
- **Volume Levels**: Normal, very quiet, silent
- **Duration**: Very short (1s), short (2-5s), medium (60s+)
- **Error Handling**: Corrupted files, invalid formats
- **Edge Cases**: No speech detection, noise handling

## Usage

These files are automatically used by the Whisper test command:

```bash
./manage.sh --action test
```

The test suite will:
1. Process each valid audio file
2. Verify transcription results
3. Test error handling with corrupted files
4. Validate API responses and JSON parsing
5. Check language detection capabilities

## Adding New Test Files

When adding new test audio files:
1. Use descriptive names that indicate the content and purpose
2. Keep files reasonably small (< 10MB for efficient testing)
3. Include a variety of audio types, formats, and edge cases  
4. Ensure files are not copyrighted or have appropriate licenses
5. Document the purpose of each file in this README

## File Generation

Extended test files were generated using ffmpeg:
- Tones: `sine=frequency=440:duration=N`
- Noise: `anoisesrc=duration=N:color=white`
- Volume adjustment: `volume=0.01` for quiet files
- Format conversion: Various `-acodec` options