"""Repeatable, bounded export of the pinned Big-LaMa generator to fixed 1024 ONNX.

This module intentionally does not import the upstream notebook or Lightning
training module. It is invoked only by the closed model BuildProfile in the
Go installer, after both source and weight archives have been hash-verified.
"""
from __future__ import annotations

import argparse
import gc
import hashlib
import json
import os
from pathlib import Path, PurePosixPath
import resource
import shutil
import sys
import tempfile
import time
import traceback
import zipfile


SOURCE_ROOT = "lama-source"
MAX_SOURCE_UNPACKED = 96 * 1024 * 1024
RESOLUTION = 1024
OPSET = 17
CPU_CLAIM_MILLICORES = 2000
TORCH_THREAD_LIMIT = 2
SOURCE_COMMIT = "f5fb39a18022c34a71bf9a47a6ec393c804b49ca"
CHECKPOINT_REVISION = "353e64100236fd934e1dcbaa1cbff13ea8362151"
REFERENCE_ONNX_SHA256 = "1faef5301d78db7dda502fe59966957ec4b79dd64e16f03ed96913c7a4eb68d6"
REFERENCE_PROBE_TOLERANCE = 0.5
CHECKPOINT_KEY_PREFIX = "model.generator."
UPSTREAM_UTILS_SHA256 = "0a399137ac692a8c0eac6b8f1befacdd9d8f3443397d8a0b000845010c2b0cfc"
TRAINING_SEED_IMPORT = "from pytorch_lightning import seed_everything\n"
TRAINING_SEED_REPLACEMENT = """def seed_everything(seed):
    import random
    import numpy as np
    random.seed(seed)
    np.random.seed(seed)
    torch.manual_seed(seed)
    return seed
"""


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _process_peak_rss_bytes() -> int:
    # Linux reports ru_maxrss in KiB; this is a process high-water mark,
    # not a sum of stage usage or a sample of the full cgroup.
    return int(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss * 1024)


def _safe_extract(archive: Path, destination: Path, max_unpacked: int) -> list[str]:
    """Extract regular files only, rejecting traversal, links, and zip bombs."""
    destination.mkdir(parents=True, exist_ok=True)
    names: list[str] = []
    total = 0
    with zipfile.ZipFile(archive) as bundle:
        for member in bundle.infolist():
            raw = member.filename.replace("\\", "/")
            rel = PurePosixPath(raw)
            mode = member.external_attr >> 16
            if rel.is_absolute() or ".." in rel.parts or not rel.parts:
                raise ValueError(f"unsafe archive path: {member.filename!r}")
            if mode and (mode & 0o170000) not in (0, 0o100000, 0o040000):
                raise ValueError(f"archive links/special files are not allowed: {member.filename!r}")
            total += member.file_size
            if total > max_unpacked:
                raise ValueError("archive exceeds the declared unpacked-size bound")
            target = destination.joinpath(*rel.parts)
            if not target.resolve().is_relative_to(destination.resolve()):
                raise ValueError(f"archive path escapes destination: {member.filename!r}")
            if member.is_dir():
                target.mkdir(parents=True, exist_ok=True)
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            with bundle.open(member) as src, target.open("xb") as dst:
                shutil.copyfileobj(src, dst)
            os.chmod(target, 0o644)
            names.append(raw)
    return names


def _generator_from_upstream(source_dir: Path, config_path: Path, checkpoint_path: Path):
    """Load only the pinned FFC generator implementation and tensor state."""
    import torch
    torch.set_num_threads(TORCH_THREAD_LIMIT)
    torch.set_num_interop_threads(TORCH_THREAD_LIMIT)
    import yaml

    prepared_utils_sha256 = _prepare_upstream_utils(source_dir / "saicinpainting" / "utils.py")
    sys.path.insert(0, str(source_dir))
    from saicinpainting.training.modules import make_generator

    with config_path.open("r", encoding="utf-8") as stream:
        config = yaml.safe_load(stream)
    if not isinstance(config, dict) or not isinstance(config.get("generator"), dict):
        raise ValueError("pinned LaMa config has no generator object")
    generator_config = dict(config["generator"])
    kind = generator_config.pop("kind", None)
    if kind != "ffc_resnet":
        raise ValueError(f"unsupported pinned generator kind: {kind!r}")

    # Resolve only the interpolation values in the pinned Big-LaMa config.
    init = dict(generator_config.get("init_conv_kwargs", {}))
    down = dict(generator_config.get("downsample_conv_kwargs", {}))
    resnet = dict(generator_config.get("resnet_conv_kwargs", {}))
    if init.get("ratio_gout") != 0 or down.get("ratio_gin") != "${generator.init_conv_kwargs.ratio_gout}" or down.get("ratio_gout") != "${generator.downsample_conv_kwargs.ratio_gin}" or resnet.get("ratio_gout") != "${generator.resnet_conv_kwargs.ratio_gin}":
        raise ValueError("pinned Big-LaMa Fourier channel ratios changed; exporter profile must be reviewed")
    down["ratio_gin"] = init["ratio_gout"]
    down["ratio_gout"] = down["ratio_gin"]
    resnet["ratio_gout"] = resnet["ratio_gin"]
    resnet["use_jit"] = True  # upstream notebook's export-specific FourierUnit variant
    generator_config["downsample_conv_kwargs"] = down
    generator_config["resnet_conv_kwargs"] = resnet
    generator = make_generator(None, kind, **generator_config)

    _load_generator_weights(generator, checkpoint_path)
    generator.eval().cpu()

    class Inpaint(torch.nn.Module):
        def __init__(self, model):
            super().__init__()
            self.generator = model

        def forward(self, image, mask):
            masked = image * (1 - mask)
            masked = torch.cat((masked, mask), dim=1)
            predicted = self.generator(masked)
            inpainted = mask * predicted + (1 - mask) * image
            return torch.clamp(inpainted * 255, min=0, max=255)

    return Inpaint(generator).eval().cpu(), torch, prepared_utils_sha256


def _load_generator_weights(generator, checkpoint_path: Path) -> None:
    """Load the generator state from its pinned wrapper after exact parity checks."""
    from safetensors.torch import load_file

    checkpoint = load_file(str(checkpoint_path), device="cpu")
    if not checkpoint or any(not key.startswith(CHECKPOINT_KEY_PREFIX) for key in checkpoint):
        raise ValueError(f"safetensors keys must all use the pinned {CHECKPOINT_KEY_PREFIX!r} wrapper")
    weights = {key[len(CHECKPOINT_KEY_PREFIX):]: value for key, value in checkpoint.items()}
    if len(weights) != len(checkpoint):
        raise ValueError("safetensors key normalization produced collisions")
    expected = generator.state_dict()
    if set(weights) != set(expected):
        missing = sorted(set(expected) - set(weights))
        unexpected = sorted(set(weights) - set(expected))
        raise ValueError(f"safetensors generator keys differ from closed FFC model: missing={missing[:8]} unexpected={unexpected[:8]}")
    shape_mismatches = [key for key in expected if weights[key].shape != expected[key].shape]
    if shape_mismatches:
        raise ValueError(f"safetensors tensor shapes differ from closed FFC model: {shape_mismatches[:8]}")
    generator.load_state_dict(weights, strict=True)


def _prepare_upstream_utils(path: Path, expected_sha256: str = UPSTREAM_UTILS_SHA256) -> str:
    """Apply the one reviewed source patch needed for inference-only import.

    The source digest and exact import string fence this transformation to the
    pinned upstream file. A source revision or construct change fails closed.
    Returns the transformed file digest for the build receipt.
    """
    original = path.read_bytes()
    original_sha256 = hashlib.sha256(original).hexdigest()
    if original_sha256 != expected_sha256:
        raise ValueError(f"pinned LaMa utils source digest changed: {original_sha256}")
    text = original.decode("utf-8")
    if text.count(TRAINING_SEED_IMPORT) != 1:
        raise ValueError("pinned LaMa utils training seed import changed")
    transformed = text.replace(TRAINING_SEED_IMPORT, TRAINING_SEED_REPLACEMENT, 1).encode("utf-8")
    path.write_bytes(transformed)
    return hashlib.sha256(transformed).hexdigest()


def _probe_512_reference(model, torch, reference_onnx: Path) -> float:
    """Require generator-only weights to match the installed verified 512 ONNX."""
    import numpy as np
    import onnxruntime as ort

    session = ort.InferenceSession(str(reference_onnx), providers=["CPUExecutionProvider"])
    inputs = {value.name: value for value in session.get_inputs()}
    expected_shapes = {"image": ["batch", 3, 512, 512], "mask": ["batch", 1, 512, 512]}
    if set(inputs) != set(expected_shapes) or any(inputs[name].shape != shape or inputs[name].type != "tensor(float)" for name, shape in expected_shapes.items()):
        raise ValueError("verified 512 reference ONNX has an unexpected image/mask signature")

    image = np.linspace(0.0, 1.0, 3 * 512 * 512, dtype=np.float32).reshape(1, 3, 512, 512)
    mask = np.zeros((1, 1, 512, 512), dtype=np.float32)
    mask[:, :, 96:416, 112:400] = 1.0
    reference = session.run(None, {"image": image, "mask": mask})[0]
    with torch.no_grad():
        candidate = model(torch.from_numpy(image), torch.from_numpy(mask)).cpu().numpy()
    if reference.shape != candidate.shape or not np.isfinite(reference).all() or not np.isfinite(candidate).all():
        raise ValueError("512 reference probe produced invalid shape or non-finite output")
    max_abs = float(np.max(np.abs(reference - candidate)))
    if max_abs > REFERENCE_PROBE_TOLERANCE:
        raise ValueError(f"safetensors weights do not match verified 512 LaMa ONNX: max abs {max_abs:.6f} > {REFERENCE_PROBE_TOLERANCE}")
    return max_abs


def export(source_archive: Path, checkpoint_path: Path, reference_onnx: Path, output: Path, receipt_path: Path, capacity_claim_bytes: int, cpu_claim_millicores: int) -> dict:
    started = time.perf_counter()
    with tempfile.TemporaryDirectory(prefix="lama-export-") as temporary:
        work = Path(temporary)
        source_dir = work / SOURCE_ROOT
        source_files = _safe_extract(source_archive, source_dir, MAX_SOURCE_UNPACKED)
        # The archive's known layout is part of this exporter contract.
        config_path = next(source_dir.rglob("configs/training/big-lama.yaml"), None)
        if config_path is None:
            raise ValueError("pinned source archive is missing the Big-LaMa config")
        if not checkpoint_path.is_file():
            raise ValueError("pinned generator-only safetensors file is missing")
        if sha256_file(reference_onnx) != REFERENCE_ONNX_SHA256:
            raise ValueError("existing verified 512 LaMa ONNX reference hash changed")
        repository_root = config_path.parents[2]
        if not (repository_root / "saicinpainting" / "training" / "modules" / "__init__.py").is_file():
            raise ValueError("pinned source archive has an unexpected repository layout")

        model, torch, prepared_utils_sha256 = _generator_from_upstream(repository_root, config_path, checkpoint_path)
        peak_after_load = _process_peak_rss_bytes()
        reference_max_abs = _probe_512_reference(model, torch, reference_onnx)
        gc.collect()
        peak_after_512 = _process_peak_rss_bytes()
        example_image = torch.rand(1, 3, RESOLUTION, RESOLUTION, dtype=torch.float32)
        example_mask = torch.zeros(1, 1, RESOLUTION, RESOLUTION, dtype=torch.float32)
        example_mask[:, :, RESOLUTION // 4:3 * RESOLUTION // 4,
                     RESOLUTION // 4:3 * RESOLUTION // 4] = 1
        output.parent.mkdir(parents=True, exist_ok=True)
        torch.onnx.export(
            model, (example_image, example_mask), str(output),
            input_names=["image", "mask"], output_names=["output"],
            dynamic_axes={"image": {0: "batch"}, "mask": {0: "batch"}, "output": {0: "batch"}},
            export_params=True, do_constant_folding=True, opset_version=OPSET,
            verbose=False, dynamo=False,
        )
        peak_after_export = _process_peak_rss_bytes()

    import onnx
    import onnxruntime as ort
    graph = onnx.load(str(output))
    onnx.checker.check_model(graph)
    session = ort.InferenceSession(str(output), providers=["CPUExecutionProvider"])
    inputs = {value.name: value for value in session.get_inputs()}
    if set(inputs) != {"image", "mask"}:
        raise ValueError(f"export input names differ from fixed profile: {sorted(inputs)}")
    import numpy as np
    image = np.zeros((1, 3, RESOLUTION, RESOLUTION), dtype=np.float32)
    image[:, 0, :, :] = .4
    mask = np.zeros((1, 1, RESOLUTION, RESOLUTION), dtype=np.float32)
    mask[:, :, 200:824, 200:824] = 1
    result = session.run(None, {"image": image, "mask": mask})[0]
    if result.shape != (1, 3, RESOLUTION, RESOLUTION) or not np.isfinite(result).all():
        raise ValueError(f"export runtime smoke has invalid output shape/data: {result.shape}")
    with torch.no_grad():
        reference = model(torch.from_numpy(image), torch.from_numpy(mask)).cpu().numpy()
    parity_max_abs = float(np.max(np.abs(reference - result)))
    peak_after_parity = _process_peak_rss_bytes()
    if not np.isfinite(parity_max_abs) or parity_max_abs > 0.05:
        raise ValueError(f"ONNX parity failed: maximum absolute error {parity_max_abs:.6f} exceeds 0.05")
    receipt = {
        "schema": "vrooli.image-tools.lama-onnx-build/v2",
        "source_repository": "https://github.com/Carve-Photos/lama",
        "source_commit": SOURCE_COMMIT,
        "checkpoint_repository": "https://huggingface.co/AEmotionStudio/lama-inpainting",
        "checkpoint_revision": CHECKPOINT_REVISION,
        "checkpoint_lineage": "AEmotionStudio generator-only safetensors; model card traces to enesmsahin/simple-lama-inpainting and the original Apache-2.0 Big-LaMa weights",
        "checkpoint_license": "Apache-2.0",
        "source_archive_sha256": sha256_file(source_archive),
        "checkpoint_sha256": sha256_file(checkpoint_path),
        "checkpoint_bytes": checkpoint_path.stat().st_size,
        "checkpoint_key_prefix": CHECKPOINT_KEY_PREFIX,
        "source_files": sorted(source_files),
        "upstream_utils_source_sha256": UPSTREAM_UTILS_SHA256,
        "upstream_utils_prepared_sha256": prepared_utils_sha256,
        "source_preparation": "replace pinned training-only seed_everything import with equivalent local seed helper",
        "reference_onnx_sha256": REFERENCE_ONNX_SHA256,
        "reference_probe_resolution": [512, 512],
        "reference_probe_max_abs": reference_max_abs,
        "reference_probe_tolerance": REFERENCE_PROBE_TOLERANCE,
        "capacity_claim_bytes": capacity_claim_bytes,
        "cpu_claim_millicores": cpu_claim_millicores,
        "torch_thread_limit": TORCH_THREAD_LIMIT,
        "operation_cpu_weight": 100,
        "cpu_quota_millicores": 0,
        "output_sha256": sha256_file(output), "output_bytes": output.stat().st_size,
        "resolution": [RESOLUTION, RESOLUTION], "opset": OPSET,
        "onnx_version": onnx.__version__, "onnxruntime_version": ort.__version__,
        "output_shape": list(result.shape), "output_finite": True,
        "parity_max_abs": parity_max_abs,
        "build_elapsed_ms": round((time.perf_counter() - started) * 1000, 3),
        "process_peak_rss_bytes": _process_peak_rss_bytes(),
        "process_peak_rss_scope": "single exporter process kernel high-water mark; excludes other cgroup processes",
        "process_peak_rss_after_model_load_bytes": peak_after_load,
        "process_peak_rss_after_512_probe_bytes": peak_after_512,
        "process_peak_rss_after_onnx_export_bytes": peak_after_export,
        "process_peak_rss_after_runtime_parity_bytes": peak_after_parity,
        "license": "Apache-2.0",
    }
    receipt_path.write_text(json.dumps(receipt, sort_keys=True, separators=(",", ":")), encoding="utf-8")
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-archive", type=Path, required=True)
    parser.add_argument("--checkpoint", type=Path, required=True)
    parser.add_argument("--reference-onnx", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--receipt", type=Path, required=True)
    parser.add_argument("--capacity-claim-bytes", type=int, required=True)
    parser.add_argument("--cpu-claim-millicores", type=int, required=True)
    parser.add_argument("--failure-report", type=Path)
    args = parser.parse_args()
    try:
        if args.capacity_claim_bytes != 16 << 30:
            raise ValueError("exporter build profile requires the pinned 16 GiB native memory ceiling")
        if args.cpu_claim_millicores != CPU_CLAIM_MILLICORES:
            raise ValueError("exporter build profile requires the pinned 2000-millicore CPU reservation")
        result = export(args.source_archive, args.checkpoint, args.reference_onnx, args.output, args.receipt, args.capacity_claim_bytes, args.cpu_claim_millicores)
        print(json.dumps(result, sort_keys=True))
    except Exception as exc:
        if args.failure_report:
            args.failure_report.write_text(json.dumps({"error": type(exc).__name__, "message": str(exc), "traceback": traceback.format_exc()[-24000:]}, sort_keys=True), encoding="utf-8")
        raise


if __name__ == "__main__":
    main()
