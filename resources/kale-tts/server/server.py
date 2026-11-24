"""Experimental OpenAI-shaped HTTP adapter for Kale TTS Nano."""
import io
import json
import os
import threading
import wave
from contextlib import asynccontextmanager

import numpy as np
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse, Response
from pydantic import BaseModel, ConfigDict, Field

MODEL_ID = "kale-tts-1-nano"
MODEL_DIR = os.environ.get("KALE_MODEL_DIR", "models/kale-tts-1-nano")
MODEL_FILE = os.environ.get("KALE_MODEL_FILE", "kale-tts-1-nano.onnx")
FRONTEND_DIR = os.environ.get("KALE_FRONTEND_DIR", "frontend")
FRONTEND_BUNDLE = os.environ.get("KALE_FRONTEND_BUNDLE", "kale_frontend.js")
_engine = None
_engine_lock = threading.Lock()
_synthesis_lock = threading.Lock()


def engine():
    global _engine
    if _engine is None:
        with _engine_lock:
            if _engine is None:
                from kale_tts import KaleTTS, QuickJSFrontEnd
                _engine = KaleTTS(
                    MODEL_DIR,
                    threads=1,
                    frontend=QuickJSFrontEnd(frontend_dir=FRONTEND_DIR, bundle=FRONTEND_BUNDLE),
                    model_file=MODEL_FILE,
                )
    return _engine


@asynccontextmanager
async def lifespan(_app):
    # Keep graph/session/voice loading lazy: health probes must remain cheap.
    yield


app = FastAPI(title="Kale TTS Nano (experimental)", lifespan=lifespan)


class SpeechRequest(BaseModel):
    model_config = ConfigDict(extra="forbid")
    model: str = MODEL_ID
    input: str = Field(min_length=1, max_length=5000)
    voice: str = "kenzie"
    response_format: str = "wav"


@app.middleware("http")
async def bound_request_body(request, call_next):
    # The text limit is the semantic bound; cap JSON overhead as well, including
    # requests rejected later by Pydantic, to avoid oversized body buffering.
    content_length = request.headers.get("content-length")
    if content_length:
        try:
            if int(content_length) > 65536:
                return JSONResponse({"detail": "request body exceeds 64 KiB"}, status_code=413)
        except ValueError:
            return JSONResponse({"detail": "invalid content-length"}, status_code=400)
    return await call_next(request)


@app.get("/healthz")
def healthz():
    ready = all(os.path.isfile(os.path.join(MODEL_DIR, name)) for name in (
        MODEL_FILE, "voices.json", "tokenizer.json"
    )) and all(os.path.isfile(os.path.join(FRONTEND_DIR, "words", name)) for name in (
        "lexicon.tsv", "override.public.tsv", "terms.tsv", "g2p.bin"
    )) and os.path.isfile(FRONTEND_BUNDLE)
    return JSONResponse(status_code=200 if ready else 503,
                        content={"status": "ok" if ready else "model_missing", "model": MODEL_ID,
                                 "loaded": _engine is not None, "ready": ready})


@app.get("/v1/audio/voices")
def voices():
    try:
        with open(os.path.join(MODEL_DIR, "voices.json"), encoding="utf-8") as source:
            metadata = json.load(source)
        listed = []
        for item in metadata.get("voices", []):
            name = item.get("name", item["id"])
            description = item.get("description")
            listed.append({"id": item["id"], "name": name, "description": description,
                           "accent": item.get("accent", "en-us"),
                           "label": f"{name} · {description}" if description else name})
        return {"voices": listed, "model": MODEL_ID}
    except FileNotFoundError as exc:
        raise HTTPException(status_code=503, detail="voice catalogue is not installed") from exc
    except Exception as exc:
        raise HTTPException(status_code=503, detail=f"voice catalogue unavailable: {exc}") from exc


@app.post("/v1/audio/speech")
def speech(request: SpeechRequest):
    if request.model != MODEL_ID:
        raise HTTPException(status_code=400, detail=f"model must be {MODEL_ID}")
    fmt = request.response_format.lower()
    if not request.input.strip():
        raise HTTPException(status_code=400, detail="input must contain non-whitespace text")
    if fmt not in ("wav", "pcm"):
        raise HTTPException(status_code=400, detail="response_format must be wav or pcm")
    try:
        # ONNX sessions and the Python frontend are serialized to cap CPU/RAM.
        # Consume the stream while holding the session lock. Converting each
        # chunk avoids retaining the full float waveform and a second
        # concatenated copy while producing the response.
        out = io.BytesIO()
        with _synthesis_lock:
            tts = engine()
            if fmt == "wav":
                with wave.open(out, "wb") as wav:
                    wav.setnchannels(1)
                    wav.setsampwidth(2)
                    wav.setframerate(24000)
                    _write_pcm_chunks(wav, tts.stream(request.input, voice=request.voice))
            else:
                _write_pcm_chunks(out, tts.stream(request.input, voice=request.voice))
    except ValueError as exc:
        raise HTTPException(status_code=400, detail=str(exc)) from exc
    except Exception as exc:
        raise HTTPException(status_code=500, detail=f"synthesis failed: {exc}") from exc
    if fmt == "pcm":
        return Response(out.getvalue(), media_type="audio/pcm", headers={"X-Sample-Rate": "24000"})
    return Response(out.getvalue(), media_type="audio/wav")


def _write_pcm_chunks(target, chunks):
    """Convert float audio chunks to the API's signed 16-bit little-endian PCM."""
    write = target.write if isinstance(target, io.BytesIO) else target.writeframesraw
    wrote_audio = False
    for chunk in chunks:
        # Do not modify arrays owned by the TTS library. Clipping allocates one
        # chunk-sized float array; subsequent conversion is chunk bounded.
        clipped = np.clip(np.asarray(chunk, dtype=np.float32), -1.0, 1.0)
        np.multiply(clipped, 32767.0, out=clipped)
        write(clipped.astype("<i2").tobytes())
        wrote_audio = True
    if not wrote_audio:
        raise ValueError("synthesis returned no audio")

