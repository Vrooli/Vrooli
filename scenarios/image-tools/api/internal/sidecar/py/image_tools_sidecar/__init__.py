"""image_tools_sidecar — the in-repo Python backend for image-tools AI ops.

This package is the CPU-tractable, provisionable backend the Go engine shells
out to (`python3 -m image_tools_sidecar.<op>`). It runs real ONNX weights via
onnxruntime with no GPU required. The Go binary embeds these sources and
materializes them to a cache dir at boot, then puts that dir on PYTHONPATH — so
the only host provisioning needed is the Python runtime packages (onnxruntime,
Pillow, numpy), documented in docs/reference/backends.md.

Modules:
  bg_removal  — background removal (U^2-Net / IS-Net family ONNX → alpha matte)
  denoise     — classical CPU denoise (no model required)
  colorize    — colorization ONNX → RGB PNG
  depth       — monocular depth ONNX → grayscale PNG
  deblur      — image-to-image restoration ONNX → RGB PNG
  detect      — detector ONNX → structured bounding-box JSON
  segment     — segmentation ONNX/SAM pair → alpha-mask PNG
  tagging     — tagger ONNX → structured tag-score JSON
  nsfw        — classifier ONNX → structured safety JSON
  embedding   — vision encoder ONNX → embedding JSON
  restore     — RestoreFormer++ dependency/weight gate for restoration verticals
"""

__all__ = [
    "bg_removal",
    "denoise",
    "colorize",
    "depth",
    "deblur",
    "detect",
    "segment",
    "tagging",
    "nsfw",
    "embedding",
    "restore",
]
