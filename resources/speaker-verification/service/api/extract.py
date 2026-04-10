"""Target speaker extraction endpoint."""

import io
import struct
import time

from fastapi import APIRouter, File, Form, HTTPException, UploadFile
from fastapi.responses import Response

import audio
import embeddings
import extraction
import profiles
import verification
from config import settings
from models import ErrorResponse

router = APIRouter(prefix="/v1")


def _numpy_to_wav_bytes(waveform, sample_rate: int = 16000) -> bytes:
    """Encode a float32 numpy array as 16-bit PCM WAV bytes."""
    import numpy as np

    # Clip and convert to int16
    pcm = np.clip(waveform, -1.0, 1.0)
    pcm = (pcm * 32767).astype(np.int16)
    raw = pcm.tobytes()

    # Build WAV header (44 bytes)
    num_channels = 1
    bits_per_sample = 16
    byte_rate = sample_rate * num_channels * bits_per_sample // 8
    block_align = num_channels * bits_per_sample // 8
    data_size = len(raw)

    buf = io.BytesIO()
    buf.write(b"RIFF")
    buf.write(struct.pack("<I", 36 + data_size))
    buf.write(b"WAVE")
    buf.write(b"fmt ")
    buf.write(struct.pack("<I", 16))  # fmt chunk size
    buf.write(struct.pack("<H", 1))  # PCM format
    buf.write(struct.pack("<H", num_channels))
    buf.write(struct.pack("<I", sample_rate))
    buf.write(struct.pack("<I", byte_rate))
    buf.write(struct.pack("<H", block_align))
    buf.write(struct.pack("<H", bits_per_sample))
    buf.write(b"data")
    buf.write(struct.pack("<I", data_size))
    buf.write(raw)
    return buf.getvalue()


@router.post(
    "/extract",
    responses={
        200: {"content": {"audio/wav": {}}, "description": "Extracted speaker audio"},
        400: {"model": ErrorResponse},
        404: {"model": ErrorResponse},
        503: {"model": ErrorResponse},
    },
)
async def extract_speaker(
    audio_file: UploadFile = File(..., alias="audio"),
    profile_id: str = Form(...),
    verify: bool = Form(True),
):
    """Extract target speaker audio from a mixture using an enrolled profile.

    Returns the isolated speaker waveform as 16kHz mono 16-bit PCM WAV.
    Verification score and match status are provided in response headers.
    """
    overall_start = time.monotonic()

    if not extraction.is_model_loaded():
        raise HTTPException(
            status_code=503,
            detail="TSE model not loaded or not available",
        )

    # Load stored profile embedding
    profile_embedding = profiles.load_embedding(profile_id)
    if profile_embedding is None:
        raise HTTPException(
            status_code=404,
            detail=f"Profile '{profile_id}' not found",
        )

    # Read and process audio
    audio_bytes = await audio_file.read()
    if not audio_bytes:
        raise HTTPException(status_code=400, detail="Empty audio file")

    try:
        waveform, duration = audio.load_and_normalize(audio_bytes)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    try:
        audio.validate_duration(duration, settings.VERIFY_MIN_SECONDS)
    except audio.AudioProcessingError as e:
        raise HTTPException(status_code=400, detail=str(e))

    # Run extraction: separate sources and select the one matching the profile
    waveform_np = waveform.squeeze(0).numpy()
    extracted = extraction.extract_target_speaker(
        waveform_np, profile_embedding, sample_rate=settings.SAMPLE_RATE,
    )

    overall_ms = (time.monotonic() - overall_start) * 1000
    headers = {"X-Duration-Ms": f"{overall_ms:.0f}"}

    if extracted is None:
        # Target speaker not detected — return near-silent WAV with low score
        import numpy as np

        silent = np.zeros(settings.SAMPLE_RATE, dtype=np.float32)  # 1s silence
        wav_bytes = _numpy_to_wav_bytes(silent, settings.SAMPLE_RATE)
        headers.update({
            "X-Speaker-Score": "0.0",
            "X-Speaker-Matched": "false",
            "X-Audio-Seconds": "1.0",
        })
        return Response(
            content=wav_bytes,
            media_type="audio/wav",
            headers=headers,
        )

    # Optionally verify the extracted audio for confidence
    score = 0.0
    matched = False
    if verify:
        import torch

        extracted_tensor = torch.from_numpy(extracted).float().unsqueeze(0)
        extracted_embedding = embeddings.extract_embedding(extracted_tensor)
        result = verification.verify(profile_embedding, extracted_embedding)
        score = result["score"]
        matched = result["matched"]

    audio_seconds = len(extracted) / settings.SAMPLE_RATE
    wav_bytes = _numpy_to_wav_bytes(extracted, settings.SAMPLE_RATE)

    headers.update({
        "X-Speaker-Score": f"{score:.6f}",
        "X-Speaker-Matched": str(matched).lower(),
        "X-Audio-Seconds": f"{audio_seconds:.2f}",
    })

    return Response(
        content=wav_bytes,
        media_type="audio/wav",
        headers=headers,
    )
