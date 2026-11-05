"""One model generation. Imports are isolated to the selected pinned closure."""
import ctypes
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import signal
import sys


def main():
    stage = Path(sys.argv[1]).resolve(strict=True)
    job = json.loads((stage / "job.json").read_text())
    # Set before importing any CUDA-aware library. Covers hard supervisor death
    # as well as ordinary cancellation of this worker's process group.
    if ctypes.CDLL(None, use_errno=True).prctl(1, signal.SIGKILL) != 0:
        raise OSError(ctypes.get_errno(), "cannot bind worker to supervisor lifetime")
    if os.getppid() != job["parent_pid"]:
        raise RuntimeError("supervisor exited before worker initialization")
    artifact, root = Path(job["artifact_root"]), Path(job["data_root"])
    profile, request = job["profile"], job["request"]
    if profile.get("wheels"):
        sys.path.insert(0, str(artifact / profile["wheels"]))
    sys.path.insert(0, str(artifact / "source"))
    if profile.get("source_dir"):
        sys.path.insert(0, str(artifact / profile["source_dir"]))
    for entry in profile["files"]:
        path = root / entry["path"]
        with path.open("rb") as stream:
            actual = hashlib.file_digest(stream, "sha256").hexdigest()
        if actual != entry["sha256"]:
            raise RuntimeError("model checksum mismatch: " + entry["path"])
    if profile["backend"] == "yue2-native":
        from yue2 import generate_yue2, native_failure
        try:
            result = generate_yue2(artifact, root, stage, profile, request)
        except Exception as exc:
            raise native_failure(stage, exc) from exc
        result.update({"model_id": profile["model_id"], "revision": profile["revision"],
                       "weight_files_verified": len(profile["files"]), "quantization": profile["quantization"]})
        (stage / "result.json").write_text(json.dumps(result, allow_nan=False, sort_keys=True))
        return
    import torch
    if not torch.cuda.is_available():
        raise RuntimeError("qualified CUDA backend unavailable; no fallback")
    torch.set_num_threads(profile["cpu_threads"])
    torch.cuda.set_per_process_memory_fraction(profile["cuda_fraction"])
    free, total = torch.cuda.mem_get_info()
    if free < profile["minimum_free_vram_bytes"]:
        raise RuntimeError("currently free VRAM is below this profile's qualified minimum")
    from backends import generate_heartmula, generate_minimax
    dispatch = {"heartmula": generate_heartmula, "minimax": generate_minimax}
    result = dispatch[profile["backend"]](root, stage, profile, request)
    result.update({"seed": request["seed"], "model_id": profile["model_id"], "revision": profile["revision"],
                   "weight_files_verified": len(profile["files"]), "quantization": profile.get("quantization"),
                   "cuda_allocator_fraction": profile["cuda_fraction"], "entry_free_vram_bytes": free,
                   "device_total_bytes": total, "torch_peak_allocated_bytes": torch.cuda.max_memory_allocated(),
                   "versions": {name: importlib.metadata.version(name) for name in profile["version_packages"]}})
    (stage / "result.json").write_text(json.dumps(result, allow_nan=False, sort_keys=True))


if __name__ == "__main__":
    main()
