# Audio Operations

Generated from `api/internal/operationcatalog/operations.seed.json`.

This index lists every declared operation and its CLI journey. For implementation details, see [Audio Transformations](../domains/audio/transformations.md).

| Operation | Family | Transport | CLI journey | Parameters |
|---|---|---|---|---|
| `transcode` | transform | unary | `audio-tools audio transcode` | `audio`, `output_format`, `input_format`, `sample_rate`, `channels`, `bitrate` |
| `trim` | transform | unary | `audio-tools audio trim` | `audio`, `format`, `start_seconds`, `end_seconds` |
| `merge` | transform | unary | `audio-tools audio merge` | `sources`, `output_format`, `crossfade_seconds` |
| `split` | transform | unary | `audio-tools audio split` | `audio`, `format`, `chunk_seconds`, `boundaries_seconds`, `output_format` |
| `fade` | transform | unary | `audio-tools audio fade` | `audio`, `format`, `fade_in_seconds`, `fade_out_seconds`, `output_format` |
| `volume` | transform | unary | `audio-tools audio volume` | `audio`, `format`, `gain_db`, `output_format` |
| `normalize` | transform | unary | `audio-tools audio normalize` | `audio`, `format`, `method`, `target_lufs`, `output_format` |
| `extract_metadata` | transform | unary | `audio-tools audio extract-metadata` | `audio`, `format` |
| `transcribe` | stt | unary | `audio-tools stt transcribe` |  |
| `transcribe_stream` | stt | bidi | `audio-tools stt transcribe-stream` |  |
| `synthesize` | tts | unary | `audio-tools tts synthesize` |  |
| `synthesize_stream` | tts | server_stream | `audio-tools tts synthesize-stream` |  |
| `summarize` | summarize | unary | `audio-tools summarize text` |  |
