# Audio Test Fixtures

This directory contains audio files for testing Whisper and other audio processing resources.

## Files

### Special Cases
- **empty.txt** (0 bytes) - Empty file for edge case testing
- **silent_5sec.wav** (862KB) - Valid WAV file with 5 seconds of silence

### Speech/Dialog
- **speech_mlk_dream.mp3** (940KB) - Martin Luther King Jr.'s "I Have a Dream" speech (1 minute)
- **speech_test_short.mp3** (101KB) - Very short speech clip (~5 seconds)
- **speech_sample.ogg** (1.6MB) - George W. Bush speech excerpt in OGG Vorbis format

### Music/Non-Speech Audio
- **music_classical.mp3** (8.6MB) - Electronic/synthesized music (stereo, 192kbps)
- **music_kalimba.mp3** (8.1MB) - Kalimba instrument music
- **sound_bell.wav** (656KB) - Bell ringing sound effect
- **sound_sample.ogg** (1.7MB) - Sound sample in OGG format

## Formats Covered
- WAV (PCM 16-bit and 24-bit)
- MP3 (various bitrates)
- OGG Vorbis (mono and stereo)
- TXT (empty file for edge cases)

## Usage in Tests

These files are used by integration tests to verify:
1. **Format Support** - Different audio codecs and containers
2. **Content Types** - Speech vs music vs silence
3. **Edge Cases** - Empty files, silent audio
4. **Quality Levels** - Various sample rates and bit depths

## Notes
- All audio files are from public domain or Creative Commons sources
- Files represent real-world scenarios for comprehensive testing
- Speech samples range from 5 seconds to 1 minute for efficient testing
- Sizes range from 0 bytes to 8.6MB to test various file size handling
