"""smoke — install-time load-smoke probe.

Proves a freshly-installed model can be CONSTRUCTED / LOADED by the provisioned
runtime BEFORE a user op depends on it, so "installed but broken" is caught at
install time (and surfaced in doctor/health/ready_state) instead of as a late job
crash. It never runs a full forward pass by default — cost control for multi-GB
models. The Go SmokeRunner invokes this via the scenario's private venv
interpreter; a non-zero exit (with an actionable stderr message) is a smoke
failure that fails the install.

Probe kinds:
  * diffusers — resolve the family's pipeline class against the installed
    diffusers and validate the install shape (delegates to _diffusers.smoke).
  * diffusers-inpaint — resolve the architecture's inpaint pipeline class (the
    derived base-checkpoint inpaint path) and validate the install shape
    (delegates to _diffusers.inpaint_smoke).
  * onnx — construct an onnxruntime InferenceSession from the installed .onnx
    weight (a real, cheap load that catches corruption / opset mismatch).
  * sam2 — load the local Transformers SAM 2 checkpoint; --deep also runs a
    synthetic point-prompt forward pass through the operation adapter.
  * segformer — load a SegFormer semantic parser and its label set; --deep also
    runs a synthetic forward pass.
  * florence — load the Florence-2 checkpoint and tokenizer; --deep also describes
    a synthetic image end to end.
"""

from __future__ import annotations

import argparse
import os
import sys

from . import _common


def _smoke_diffusers(model_dir: str, family: str, deep: bool) -> str:
    if not family:
        _common.fail("diffusers smoke requires --family", code=2)
    from . import _diffusers  # local import: avoids torch import for onnx smoke

    return _diffusers.smoke(family=family, model_dir=model_dir, deep=deep)


def _smoke_diffusers_inpaint(model_dir: str, architecture: str, deep: bool, variant: str) -> str:
    if not architecture:
        _common.fail("diffusers-inpaint smoke requires --architecture", code=2)
    from . import _diffusers  # local import: avoids torch import for onnx smoke

    return _diffusers.inpaint_smoke(architecture=architecture, model_dir=model_dir, deep=deep, variant=variant)


def _smoke_onnx(model_dir: str) -> str:
    """Load every installed ONNX graph: multi-graph models (a face restorer plus
    its detector) must not pass on whichever file sorts first."""
    if os.path.isdir(model_dir):
        paths = [os.path.join(model_dir, f) for f in sorted(os.listdir(model_dir)) if f.endswith(".onnx")]
    else:
        paths = []
    if not paths:
        paths = [_common.resolve_onnx_path(model_dir)]
    loaded = []
    for onnx_path in paths:
        session = _common.make_session(onnx_path)  # fails loud on a corrupt/opset-bad model
        inputs = ", ".join(i.name for i in session.get_inputs())
        loaded.append(f"{onnx_path} (inputs: {inputs})")
    return "onnx smoke OK: loaded InferenceSession from " + "; ".join(loaded)


def _smoke_sam2(model_dir: str, deep: bool) -> str:
    import os

    from . import grounded_segment, sam2

    # A grounded install (detector/ + segmenter/) proves both models; a plain
    # SAM 2.1 install proves the one checkpoint.
    if os.path.isdir(os.path.join(model_dir, "detector")):
        return grounded_segment.smoke(model_dir, deep=deep)
    return sam2.smoke(model_dir, deep=deep)


def _smoke_segformer(model_dir: str, deep: bool) -> str:
    from . import semantic_segment

    return semantic_segment.smoke(model_dir, deep=deep)


def _smoke_florence(model_dir: str, deep: bool) -> str:
    from . import florence

    return florence.smoke(model_dir, deep=deep)


def main() -> None:
    p = argparse.ArgumentParser(description="image-tools install-time load-smoke probe")
    p.add_argument("--kind", required=True, choices=["diffusers", "diffusers-inpaint", "onnx", "sam2", "segformer", "florence"])
    p.add_argument("--model-dir", required=True, help="installed model directory")
    p.add_argument("--family", default="", help="diffusers family (required for --kind diffusers)")
    p.add_argument("--architecture", default="", help="model architecture (required for --kind diffusers-inpaint)")
    p.add_argument("--variant", default="", help="Diffusers weight variant for repository models")
    p.add_argument("--deep", action="store_true", help="also run an opt-in model-specific forward smoke (expensive)")
    args = p.parse_args()

    if args.kind == "diffusers":
        summary = _smoke_diffusers(args.model_dir, args.family, args.deep)
    elif args.kind == "diffusers-inpaint":
        summary = _smoke_diffusers_inpaint(args.model_dir, args.architecture, args.deep, args.variant)
    elif args.kind == "onnx":
        summary = _smoke_onnx(args.model_dir)
    elif args.kind == "segformer":
        summary = _smoke_segformer(args.model_dir, args.deep)
    elif args.kind == "florence":
        summary = _smoke_florence(args.model_dir, args.deep)
    else:
        summary = _smoke_sam2(args.model_dir, args.deep)

    sys.stdout.write(summary + "\n")
    sys.exit(0)


if __name__ == "__main__":
    main()
