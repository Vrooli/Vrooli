# Audition narration delivery

From the repository root run:

```bash
python3 scenarios/audio-tools/scripts/audition-narration.py recipe.json takes
```

The script uses the Audio Tools CLI and FFmpeg. Supply ordered `lines` (a JSON object mapping line IDs to exact spoken copy) and a `variants` array. Each variant has an `id`, canonical `voice`, and `hypothesis`. Optional `delivery` entries keyed by line ID specify `speed` and `pause_ms`. Optional `adapter_voice` selects a stock backend voice for that call using `tier:provider=voice`, independently of saved settings. An `expect` object can require the actual `provider_id`, `model_id`, or `voice_used`. Set these when a comparison depends on who actually synthesized the audio.

The result is a mono WAV per variant at 24 kHz and a `takes.json` receipt with line timing and provider traces. Synthesis is reused for identical inputs within a batch. The pause is added after the provider's audio for each line, including any tail already emitted by the provider. Keep the words fixed when testing delivery, compare at matched listening levels, and choose the take against the picture.

This workflow controls voice choice, line pace and silence. Natural-language emotion and prosody require a model and adapter with native direction support. Qualify that support with the same text in multiple requested deliveries and blind review of intelligibility, character consistency, and audible adherence before using it in production.

To audition a single backend voice:

```bash
audio-tools tts synthesize --text 'Which way will she turn?' \
  --adapter-voice local:kale-tts=owen --format wav --out take.wav --json --json-metadata-only
```

## Local native-direction trial

`scripts/audition-qwen-direction.py` is an experimental checkpoint qualification runner.
In an isolated environment with `qwen-tts==0.1.1`, pass a locally staged
Qwen3-TTS 1.7B CustomVoice checkpoint directory, a JSON recipe, and an output
directory. The recipe supplies `speaker`, optional `seed`, and ordered
`lines` with `id`, `text`, and `direction`. The checkpoint receives
direction separately from transcript text. The runner emits individual PCM
takes and a shared timing receipt. Compare direction variants at fixed text
and speaker. Use CPU BF16 where supported, or `--device cuda:0` when enough
GPU memory is available. Run with `PYTHONNOUSERSITE=1` when user-site
PyTorch extensions are incompatible with the environment.

This is a model trial seam. The shipping scenario provider chain continues
to use its catalog-backed routes. Promotion requires qualification across
speakers, instructions, latency, memory, and intelligibility.
