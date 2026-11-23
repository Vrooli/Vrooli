"""restore — old-photo restoration sidecar (not yet promoted).

old_photo_restore needs a scratch-repair + colour + face pipeline (Bringing Old
Photos Back to Life class) on the heavier PyTorch restoration stack. Face
restoration alone is served by the ONNX ``face_restore`` module. The Go backend
doctor probes this stack's imports before selection so operators see a clear
provisioning row instead of a late job failure.
"""

from __future__ import annotations

import os

from . import _common


def _require_restore_deps() -> None:
    try:
        import basicsr  # noqa: F401,WPS433
        import facexlib  # noqa: F401,WPS433
        import numpy  # noqa: F401,WPS433
        from PIL import Image  # noqa: F401,WPS433
        import torch  # noqa: F401,WPS433
    except ImportError as exc:  # pragma: no cover - exercised on un-provisioned hosts
        _common.fail(
            f"missing old-photo restoration dependency ({exc}). This path is NOT yet "
            "promoted: its extra stack (torch, basicsr, facexlib) is "
            "deliberately excluded from internal/pydeps/requirements.lock, so the "
            "isolated venv does not carry it. The op stays gated until the Phase 2 "
            "restoration golden path lands (see docs/internal/DECISIONS.md).",
            code=3,
        )


def main() -> None:
    args = _common.parse_io_args("old-photo restoration")
    _require_restore_deps()

    if not os.path.isdir(args.model):
        _common.fail(f"old-photo restoration model directory not found: {args.model!r}", code=4)
    weights = [
        f
        for f in sorted(os.listdir(args.model))
        if f.endswith((".ckpt", ".pth", ".pt", ".safetensors"))
    ]
    if not weights:
        _common.fail(f"no old-photo restoration checkpoint found in {args.model!r}", code=4)

    _common.fail(
        "old-photo restoration dependencies are present, but the execution "
        "adapter is not yet promoted to an E2E vertical. Keep this model gated "
        "until the restoration pipeline lands.",
        code=8,
    )


if __name__ == "__main__":
    main()
