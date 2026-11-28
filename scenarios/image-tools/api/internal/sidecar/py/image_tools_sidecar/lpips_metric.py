"""lpips_metric — LPIPS (AlexNet, v0.1) perceptual distance between two images.

Emits one JSON object on stdout:

  {"lpips": 0.183, "net": "alex", "version": "0.1", "tiles": 12, "device": "cpu"}

LPIPS is a learned full-reference distance: 0 means perceptually identical and
larger is worse. Unlike PSNR/SSIM it does not reward a blurry average, so it is
the metric that ranks an upscaler by how the result looks.

The linear heads ship inside the `lpips` wheel. The AlexNet ImageNet backbone is
NOT downloaded here (torchvision would fetch it at first use, outside any
governed install): the caller passes the pinned backbone file that the model
catalog provisioned. Both images must share one size; large images are scored in
non-overlapping tiles and the tile distances are averaged weighted by area, which
bounds peak memory without resizing either image.
"""

from __future__ import annotations

import argparse
import json
import sys

from . import _common

_TILE = 1024
_BACKBONE_SLICES = ("slice1", "slice2", "slice3", "slice4", "slice5")


def _parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="LPIPS perceptual distance")
    p.add_argument("--reference", required=True, help="reference image path")
    p.add_argument("--candidate", required=True, help="candidate image path")
    p.add_argument("--backbone", required=True, help="AlexNet ImageNet weights (torchvision state dict)")
    p.add_argument("--max-pixels", type=int, default=64_000_000)
    p.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    return p.parse_args()


def load_backbone(model, state_dict) -> int:
    """Copy torchvision AlexNet `features.N.*` tensors into LPIPS's sliced copy.

    LPIPS keeps the AlexNet feature stack split into five slices whose children
    keep the original `features` index as their name, so each tensor maps by that
    index. Returns the number of tensors loaded; a backbone that loads none is
    refused instead of silently scoring with random weights.
    """
    loaded = 0
    backbone = model.net
    for slice_name in _BACKBONE_SLICES:
        for name, module in getattr(backbone, slice_name).named_children():
            for param in ("weight", "bias"):
                key = f"features.{name}.{param}"
                if key in state_dict and hasattr(module, param):
                    getattr(module, param).data.copy_(state_dict[key])
                    loaded += 1
    return loaded


def tiles(width: int, height: int, tile: int = _TILE):
    """Non-overlapping tile boxes (left, top, right, bottom) covering the image.

    A trailing sliver narrower than 64px merges into its neighbour so no tile is
    too small for AlexNet's five stride-2/4 stages.
    """

    def spans(length: int):
        edges = list(range(0, length, tile)) + [length]
        if len(edges) > 2 and edges[-1] - edges[-2] < 64:
            del edges[-2]
        return list(zip(edges[:-1], edges[1:]))

    return [(x0, y0, x1, y1) for (y0, y1) in spans(height) for (x0, x1) in spans(width)]


def main() -> None:
    args = _parse_args()
    import os

    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    np, _ort, _Image = _common.require_deps()
    try:
        import lpips
        import torch
    except ImportError as exc:
        _common.fail(f"lpips metric needs the lpips and torch packages ({exc})")

    ref_img, _ = _common.load_oriented_image(args.reference, args.max_pixels)
    cand_img, _ = _common.load_oriented_image(args.candidate, args.max_pixels)
    if ref_img.size != cand_img.size:
        _common.fail(f"LPIPS needs equal sizes, got reference {ref_img.size} and candidate {cand_img.size}")

    device = "cuda" if _common.use_cuda(torch) else "cpu"
    # pnet_rand skips torchvision's own weight download; the pinned backbone is
    # loaded explicitly below.
    model = lpips.LPIPS(net="alex", version="0.1", pretrained=True, pnet_rand=True, verbose=False)
    state = torch.load(args.backbone, map_location="cpu", weights_only=True)
    if load_backbone(model, state) < 10:
        _common.fail(f"{args.backbone} is not an AlexNet torchvision state dict")
    model = model.to(device).eval()

    def tensor(image, box):
        arr = np.asarray(image.crop(box), dtype=np.float32) / 255.0
        return (torch.from_numpy(arr).permute(2, 0, 1).unsqueeze(0) * 2.0 - 1.0).to(device)

    total, weight = 0.0, 0
    boxes = tiles(*ref_img.size)
    with torch.no_grad():
        for box in boxes:
            area = (box[2] - box[0]) * (box[3] - box[1])
            total += float(model(tensor(ref_img, box), tensor(cand_img, box)).mean().item()) * area
            weight += area
    json.dump({"lpips": total / weight, "net": "alex", "version": "0.1", "tiles": len(boxes), "device": device}, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
