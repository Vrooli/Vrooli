"""YuE2 through the pinned native runtime, inside one Songgen worker lifetime."""
import base64
import ctypes
import hashlib
import json
import math
import os
from pathlib import Path
import re
import signal
import socket
import subprocess
import time
import urllib.error
import urllib.request


def native_inputs(root, profile, request, port, model_id="yue2"):
    relative = Path(profile["model_dir"])
    if relative.is_absolute() or ".." in relative.parts:
        raise RuntimeError("invalid native model path")
    config = {"host": "127.0.0.1", "port": port, "backend": "cuda", "device": 0,
              "threads": profile["cpu_threads"], "lazy_load": True, "max_loaded_models": 1,
              "ui": False, "ui_management": False, "log_request_body": False,
              "max_request_body_bytes": 65536,
              "models": [{"id": model_id, "family": "yue2", "path": str(root / relative),
                          "task": "gen", "mode": "offline",
                          "session_options": {"yue2.model_gguf": "yue2-3b-q8_0.gguf",
                                              "yue2.vae_gguf": "yue2-vae-f16.gguf"}}]}
    options = {"style": request["prompt"], "cot": "full", "export_semantic": True,
               "guidance_scale": request["guidance"], "num_inference_steps": request["steps"],
               "semantic_min_tokens": 0, "semantic_max_tokens": math.floor(request["duration"] * 25)}
    if "score" in request:
        options["abc"] = request["score"]
    return config, {"model": model_id, "request": {"lyrics": request["lyrics"],
                                                   "seed": request["seed"], "options": options}}


def resolved_sampling(log, request):
    witnesses = [line.split("yue2.request ", 1)[1] for line in log.splitlines() if "yue2.request " in line]
    if len(witnesses) != 1:
        raise RuntimeError("native sampling witness missing or ambiguous")
    fields = dict(re.findall(r"(\w+)=([^\s]+)", witnesses[0]))
    try:
        actual = {"seed": int(fields["seed"]), "steps": int(fields["num_inference_steps"]),
                  "guidance": float(fields["guidance_scale"])}
    except (KeyError, ValueError) as exc:
        raise RuntimeError("invalid native sampling witness") from exc
    if any(actual[k] != request[k] for k in actual):
        raise RuntimeError("native actual sampling differs from requested controls")
    expected_score = "provided" if "score" in request else "generated"
    if fields.get("abc") != expected_score:
        raise RuntimeError("native score conditioning witness differs from request")
    actual["score_source"] = fields["abc"]
    return actual


def retain_artifacts(stage, result, provided_score=None):
    retained = {}
    if provided_score is not None:
        data = provided_score.encode("utf-8")
        (stage / "score.abc").write_bytes(data)
        retained["score"] = {"sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data),
                             "truncated": False, "source": "provided", "text": provided_score}
    for artifact in result.get("artifacts", []):
        name = artifact.get("id")
        if name not in {"score", "semantic"} or name in retained:
            raise RuntimeError("unexpected or duplicate native artifact")
        encoded = artifact.get("payload", "")
        if not isinstance(encoded, str) or len(encoded) > 262144:
            raise RuntimeError("native artifact exceeds bounded payload")
        data = base64.b64decode(encoded, validate=True)
        text = data.decode("utf-8")
        if not text.strip():
            raise RuntimeError("empty native artifact")
        meta = artifact.get("meta", {})
        if meta.get("truncated") not in {"true", "false"}:
            raise RuntimeError("native artifact missing truncation witness")
        if name == "semantic":
            tokens = json.loads(text)
            if not isinstance(tokens, list) or not tokens or any(type(t) is not int or not 0 <= t < 32768 for t in tokens):
                raise RuntimeError("invalid native semantic tokens")
        filename = "score.abc" if name == "score" else "semantic.json"
        (stage / filename).write_bytes(data)
        retained[name] = {"sha256": hashlib.sha256(data).hexdigest(), "bytes": len(data),
                          "truncated": meta["truncated"] == "true", "source": "generated", "text": text}
    if "score" not in retained:
        raise RuntimeError("full-planning native result has no generated score")
    if "semantic" not in retained:
        raise RuntimeError("native result has no semantic token witness")
    return retained


def bind_child_to_parent(parent_pid):
    """Called in the single-threaded worker's child before exec, never in a server thread."""
    if ctypes.CDLL(None, use_errno=True).prctl(1, signal.SIGKILL) != 0:
        raise OSError(ctypes.get_errno(), "cannot bind native runtime lifetime")
    if os.getppid() != parent_pid:
        os._exit(1)


def stop_child(child):
    if child.poll() is None:
        child.terminate()
        try:
            child.wait(timeout=5)
        except subprocess.TimeoutExpired:
            child.kill()
            child.wait(timeout=5)


def native_environment(artifact, environ):
    env = dict(environ)
    # The published server also links NCCL, already pinned in this artifact's
    # HeartMuLa dependency closure. Loading its library does not import Torch.
    env["LD_LIBRARY_PATH"] = os.pathsep.join(str(artifact / path) for path in
        ("upstream/cudart", "wheels/heartmula/nvidia/nccl/lib"))
    return env


def native_failure(stage, error):
    # The supervisor deletes failed staging directories, but retains the worker
    # traceback. Carry bounded native diagnostics across that cleanup boundary.
    tails = []
    for name in ("native-launch.log", "native.log"):
        path = stage / name
        if path.exists():
            with path.open("rb") as stream:
                stream.seek(max(0, path.stat().st_size - 3072))
                tails.append(name + ": " + stream.read(3072).decode(errors="replace"))
    return RuntimeError(str(error) + "\n" + "\n".join(tails))


def read_json(url, *, body=None, timeout=5, limit=128 << 20):
    request = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})
    # This transaction-local native child is loopback-only; never use an HTTP proxy.
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    with opener.open(request, timeout=timeout) as response:
        raw = response.read(limit + 1)
    if len(raw) > limit:
        raise RuntimeError("native response exceeds declared bound")
    return json.loads(raw)


def generate_yue2(artifact, root, stage, profile, request):
    visible = os.environ.get("CUDA_VISIBLE_DEVICES", "0").split(",")[0]
    probe = subprocess.run(["nvidia-smi", "--id=" + visible, "--query-gpu=memory.free,memory.total",
                            "--format=csv,noheader,nounits"], capture_output=True, text=True, timeout=10, check=True)
    free_mb, total_mb = [int(value.strip()) for value in probe.stdout.strip().split(",")]
    if free_mb * (1 << 20) < profile["minimum_free_vram_bytes"]:
        raise RuntimeError("currently free VRAM is below the native profile minimum")
    executable = artifact / "upstream/audiocpp/audiocpp_server"
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]
    native_id = "yue2-" + stage.name.removeprefix(".stage-")
    config, payload = native_inputs(root, profile, request, port, native_id)
    (stage / "native-config.json").write_text(json.dumps(config, sort_keys=True))
    raw_request = json.dumps(payload, ensure_ascii=False, allow_nan=False).encode()
    (stage / "native-request.json").write_bytes(raw_request)
    parent = os.getpid()
    env = native_environment(artifact, os.environ)
    log_path = stage / "native.log"
    with (stage / "native-launch.log").open("wb") as launch_log:
        child = subprocess.Popen([str(executable), "--config", str(stage / "native-config.json"),
                                  "--log-file", str(log_path)], cwd=stage, env=env,
                                 stdin=subprocess.DEVNULL, stdout=launch_log, stderr=subprocess.STDOUT,
                                 preexec_fn=lambda: bind_child_to_parent(parent))
        try:
            deadline = time.monotonic() + 30
            endpoint = f"http://127.0.0.1:{port}"
            while True:
                if child.poll() is not None:
                    raise RuntimeError("native runtime exited before readiness")
                try:
                    models = read_json(endpoint + "/v1/models", limit=65536)
                    if not any(m.get("id") == native_id for m in models.get("data", [])):
                        raise RuntimeError("native child model identity mismatch")
                    break
                except (urllib.error.URLError, TimeoutError):
                    if time.monotonic() >= deadline:
                        raise RuntimeError("native child readiness timed out")
                    time.sleep(0.1)
            health = read_json(endpoint + "/health", limit=65536)
            if health.get("version") != profile["native_version"] or health.get("backend") != "cuda":
                raise RuntimeError("native runtime build version mismatch")
            result = read_json(endpoint + "/v1/tasks/run", body=raw_request,
                               timeout=request["timeout_seconds"])
        finally:
            stop_child(child)
    with log_path.open() as log:
        witness_lines = [line for line in log if "yue2.request " in line]
    actual = resolved_sampling("".join(witness_lines), request)
    artifacts = retain_artifacts(stage, result, request.get("score"))
    if len(json.loads(artifacts["semantic"]["text"])) > math.floor(request["duration"] * 25):
        raise RuntimeError("native semantic output exceeded the requested duration bound")
    audio = base64.b64decode(result.get("audio", ""), validate=True)
    if not audio:
        raise RuntimeError("native runtime returned no audio")
    (stage / "audio.wav").write_bytes(audio)
    return {"seed": actual.pop("seed"), "sampling": actual, "voice_reference_used": False,
            "duration_control": "upper_bound", "composition_artifacts": artifacts,
            "native_runtime": health, "native_child_pid": child.pid, "native_child_reaped": child.poll() is not None,
            "entry_free_vram_bytes": free_mb * (1 << 20), "device_total_bytes": total_mb * (1 << 20),
            "versions": {"audio.cpp": health["version"]},
            "limitations": ["Q8 main and F16 VAE differ from upstream BF16/F32", "section and singer directions are soft",
                            "a truncated score or semantic stream does not establish complete-song form",
                            "native HTTP output is PCM16; no mastering or waveform-preserving edit is implied"]}
