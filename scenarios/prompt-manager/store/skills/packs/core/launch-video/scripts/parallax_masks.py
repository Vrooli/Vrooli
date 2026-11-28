#!/usr/bin/env python3
"""Create source-bound grayscale masks for an authored layered-parallax cutout set.

Inputs are deterministic polygon paths in source-plate pixel coordinates. This
is a mask authoring aid, not segmentation or hidden-pixel reconstruction.
"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw


def sha256(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def build(spec_path, out_dir):
    spec_path = Path(spec_path).resolve()
    spec = json.loads(spec_path.read_text())
    plate = Path(spec['plate']).expanduser().resolve()
    active = Path(spec['active_screen_mask']).expanduser().resolve()
    if not plate.is_file() or not active.is_file():
        raise ValueError('plate and active screen mask must be regular local files')
    with Image.open(plate) as im:
        im.verify()
    with Image.open(active) as im:
        im.verify()
    with Image.open(plate) as src:
        width, height = src.size
    with Image.open(active) as src_mask:
        if src_mask.size != (width, height):
            raise ValueError('active screen mask dimensions do not match the plate')
        active_mask = src_mask.convert('L')
    layers = spec.get('layers')
    if not isinstance(layers, list) or len(layers) not in (2, 3):
        raise ValueError('exactly two or three authored layers are required')
    out_dir = Path(out_dir).resolve()
    results = []
    masks = []
    with Image.open(plate) as src:
        plate_rgba = src.convert('RGBA')
    for layer in layers:
        name = layer['id']
        points = layer.get('polygon_xy', [])
        if not isinstance(name, str) or not name or '/' in name or '\\' in name:
            raise ValueError('layer id must be a simple filename stem')
        if len(points) < 3:
            raise ValueError(f'{name}: polygon requires at least three points')
        depth = layer.get('depth_basis')
        if isinstance(depth, bool) or not isinstance(depth, int) or not 0 <= depth <= 1000:
            raise ValueError(f'{name}: depth_basis must be an integer in [0,1000]')
        checked = []
        for point in points:
            if (not isinstance(point, list) or len(point) != 2 or
                    any(isinstance(v, bool) or not isinstance(v, int) for v in point)):
                raise ValueError(f'{name}: points must be integer [x,y] pairs')
            x, y = point
            if not 0 <= x < width or not 0 <= y < height:
                raise ValueError(f'{name}: polygon point is outside source bounds')
            checked.append((x, y))
        mask = Image.new('L', (width, height), 0)
        ImageDraw.Draw(mask).polygon(checked, fill=255)
        mask_bytes, active_bytes = mask.tobytes(), active_mask.tobytes()
        overlap = Image.frombytes('L', mask.size, bytes(255 if a and b else 0 for a, b in zip(mask_bytes, active_bytes)))
        if overlap.getbbox():
            raise ValueError(f'{name}: polygon intersects active screen mask at {overlap.getbbox()}')
        masks.append((name, mask, checked, depth))
    for i, (name, mask, points, depth) in enumerate(masks):
        current = mask.tobytes()
        for other_name, other_mask, _, _ in masks[:i]:
            overlap = Image.frombytes('L', mask.size, bytes(255 if a and b else 0 for a, b in zip(current, other_mask.tobytes())))
            if overlap.getbbox():
                raise ValueError(f'{name}: mask intersects layer {other_name} at {overlap.getbbox()}')
    out_dir.mkdir(parents=True, exist_ok=False)
    for name, mask, points, depth in masks:
        target = out_dir / f'{name}.png'
        mask.save(target, format='PNG', optimize=True)
        cutout = plate_rgba.copy()
        cutout.putalpha(mask)
        image_target = out_dir / f'{name}-cutout.png'
        cutout.save(image_target, format='PNG', optimize=True)
        results.append({'id': name, 'image_path': str(image_target), 'image_sha256': sha256(image_target),
                        'mask_path': str(target), 'mask_sha256': sha256(target),
                        'dimensions': [width, height], 'polygon_xy': [list(p) for p in points],
                        'active_screen_overlap_pixels': 0, 'depth_basis': depth})
    receipt = {'schema': 1, 'spec': str(spec_path), 'spec_sha256': sha256(spec_path),
               'plate': {'path': str(plate), 'sha256': sha256(plate), 'dimensions': [width, height]},
               'active_screen_mask': {'path': str(active), 'sha256': sha256(active)},
               'method': 'manual polygon masks and alpha cutouts derived from the exact source plate with Pillow; no segmentation or hidden pixels',
               'layers': results}
    (out_dir / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n')
    return receipt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('spec', type=Path)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    result = build(args.spec, args.out)
    print(json.dumps({'status': 'pass', 'receipt': str((args.out.resolve() / 'receipt.json')),
                      'layers': result['layers']}, indent=2))


if __name__ == '__main__':
    main()
