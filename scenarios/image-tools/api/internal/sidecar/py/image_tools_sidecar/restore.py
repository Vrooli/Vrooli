"""restore — RestoreFormer++ class face / old-photo restoration sidecar.

This module is intentionally strict about dependencies. RestoreFormer++ is not
an ONNX CPU-floor model; it needs the heavier PyTorch restoration stack. The Go
backend doctor probes those imports before selection so operators see a clear
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
            f"missing RestoreFormer++ dependency ({exc}). Provision through "
            "Scenario Dependency Analyzer: torch, basicsr, facexlib, pillow, numpy",
            code=3,
        )


def main() -> None:
    args = _common.parse_io_args("RestoreFormer++ face / old-photo restoration")
    _require_restore_deps()

    if not os.path.isdir(args.model):
        _common.fail(f"RestoreFormer++ model directory not found: {args.model!r}", code=4)
    weights = [
        f
        for f in sorted(os.listdir(args.model))
        if f.endswith((".ckpt", ".pth", ".pt", ".safetensors"))
    ]
    if not weights:
        _common.fail(f"no RestoreFormer++ checkpoint found in {args.model!r}", code=4)

    _common.fail(
        "RestoreFormer++ runtime dependencies are present, but the execution "
        "adapter is not yet promoted to an E2E vertical. Keep this model gated "
        "until the Phase 2 restoration golden path lands.",
        code=8,
    )


if __name__ == "__main__":
    main()
