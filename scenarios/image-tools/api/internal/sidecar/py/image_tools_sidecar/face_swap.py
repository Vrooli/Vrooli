"""face_swap — put the face of a reference (source) image onto a target image with
an identity-conditioned ONNX face swapper (HyperSwap, InsightFace inswapper,
AlphaFace or any export with the same ``source``/``target`` contract).

The installed model directory holds four ONNX graphs, told apart by signature
rather than by filename:

  * the face detector — OpenCV Zoo YuNet (see face_restore);
  * the identity embedder — ArcFace w600k_r50, ``(1,3,112,112)`` RGB in [-1, 1]
    to a 512-d embedding;
  * the occluder — XSeg, ``(1,256,256,3)`` BGR in [0, 1] to a face-visible mask
    (optional: without it hands, hair and glasses are not protected);
  * the swapper — inputs ``source`` (512-d identity) and ``target`` (aligned face).

The swapper family decides the crop size, the pixel normalisation and how the
embedding is prepared (see ``PROFILES``). Supporting a new family is one profile
entry plus a registry row.

Each selected target face is warped onto the ArcFace-128 template, swapped,
colour-matched to the face it replaces, and pasted back through a feathered box
mask intersected with the occlusion mask, so pixels outside the face keep their
original values and the output keeps the target's size. A target or reference
with no detectable face fails loud (exit 9) instead of returning the input.
"""

from __future__ import annotations

import argparse
import json
import os

from . import _common
from .face_restore import detect_faces, similarity_transform, _is_detector

MAX_INPUT_PIXELS = 64_000_000
DEFAULT_SCORE_THRESHOLD = 0.6
# 5-point templates, normalised to the crop (eye centres, nose tip, mouth
# corners; image-left first). Values are FaceFusion's warp templates.
ARCFACE_112_V2 = (
    (0.34191607, 0.46157411),
    (0.65653393, 0.45983393),
    (0.50022500, 0.64050536),
    (0.37097589, 0.82469196),
    (0.63151696, 0.82325089),
)
ARCFACE_128 = (
    (0.36167656, 0.40387734),
    (0.63696719, 0.40235469),
    (0.50019687, 0.56044219),
    (0.38710391, 0.72160547),
    (0.61507734, 0.72034453),
)
EMBED_SIZE = 112
OCCLUDER_SIZE = 256
# Box-mask feather as a fraction of the crop (FaceFusion's face_mask_blur 0.3).
BOX_BLUR = 0.3

# Swapper families. ``embedding`` is how the ArcFace vector reaches the model:
# "emap" multiplies the unit vector by the model's last initializer and
# renormalises (InsightFace inswapper), "unit" feeds the unit vector, "raw" feeds
# the unnormalised vector. ``denormalize`` undoes (x - mean) / std on the output.
PROFILES = {
    "inswapper": {"size": 128, "mean": 0.0, "std": 1.0, "embedding": "emap", "denormalize": False},
    "hyperswap": {"size": 256, "mean": 0.5, "std": 0.5, "embedding": "unit", "denormalize": True},
    "alphaface": {"size": 256, "mean": 0.0, "std": 1.0, "embedding": "raw", "denormalize": False},
}


class Graphs:
    """The four sessions of an installed face-swap model directory."""

    def __init__(self, detector, embedder, swapper, occluder, profile, swapper_path):
        self.detector, self.embedder, self.swapper, self.occluder = detector, embedder, swapper, occluder
        self.profile, self.swapper_path = profile, swapper_path


def _input_names(session):
    return {i.name for i in session.get_inputs()}


def _is_swapper(session) -> bool:
    return {"source", "target"} <= _input_names(session)


def _is_embedder(session) -> bool:
    shape = session.get_inputs()[0].shape
    return len(shape) == 4 and shape[2:] == [EMBED_SIZE, EMBED_SIZE]


def _is_occluder(session) -> bool:
    shape = session.get_inputs()[0].shape
    return len(shape) == 4 and shape[1:3] == [OCCLUDER_SIZE, OCCLUDER_SIZE] and shape[3] == 3


def profile_for(filename: str):
    stem = os.path.basename(filename).lower()
    for family, profile in PROFILES.items():
        if stem.startswith(family):
            return family, profile
    return None, None


def resolve_graphs(model_dir: str) -> Graphs:
    """Open every ONNX graph in the model directory and assign each its role."""
    if not os.path.isdir(model_dir):
        _common.fail(f"face_swap needs the installed model directory, got {model_dir!r}", code=4)
    detector = embedder = swapper = occluder = profile = swapper_path = None
    for name in sorted(os.listdir(model_dir)):
        if not name.endswith(".onnx"):
            continue
        path = os.path.join(model_dir, name)
        session = _common.make_session(path)
        if _is_detector(session):
            detector = detector or session
        elif _is_swapper(session):
            family, found = profile_for(name)
            if found is None:
                _common.fail(f"unknown face swapper {name!r}; known families: {', '.join(PROFILES)}", code=4)
            swapper, profile, swapper_path = session, dict(found, family=family), path
        elif _is_embedder(session):
            embedder = embedder or session
        elif _is_occluder(session):
            occluder = occluder or session
    for role, found in (("YuNet face detector", detector), ("ArcFace embedder", embedder), ("face swapper", swapper)):
        if found is None:
            _common.fail(f"no {role} ONNX found in {model_dir!r}", code=4)
    return Graphs(detector, embedder, swapper, occluder, profile, swapper_path)


def template_points(np, template, size):
    return np.array(template, dtype=np.float64) * size


def warp(np, Image, image, to_crop, size):
    """Warp ``image`` into a ``size``-square crop through the 3x3 ``to_crop``.

    PIL's affine transform does not low-pass, so a strong downscale (a large face
    squeezed into a small crop) is first reduced with an antialiased resize and
    the matrix is composed with that reduction.
    """
    scale = float(np.sqrt(abs(np.linalg.det(to_crop[:2, :2]))))
    matrix = to_crop
    if scale < 0.5:
        reduction = min(1.0, scale * 2.0)
        resized = image.resize(
            (max(1, round(image.width * reduction)), max(1, round(image.height * reduction))), Image.LANCZOS
        )
        matrix = to_crop @ np.diag([1.0 / reduction, 1.0 / reduction, 1.0])
        image = resized
    inverse = np.linalg.inv(matrix)
    data = (inverse[0, 0], inverse[0, 1], inverse[0, 2], inverse[1, 0], inverse[1, 1], inverse[1, 2])
    return image.transform((size, size), Image.AFFINE, data, resample=Image.BICUBIC)


def embed_face(np, Image, graphs, image, landmarks):
    """ArcFace embedding of one face: (raw, unit) 512-d float32 vectors."""
    to_crop = similarity_transform(np, landmarks, template_points(np, ARCFACE_112_V2, EMBED_SIZE))
    crop = np.asarray(warp(np, Image, image, to_crop, EMBED_SIZE), dtype=np.float32)
    x = (crop / 127.5 - 1.0).transpose(2, 0, 1)[np.newaxis]
    raw = graphs.embedder.run(None, {graphs.embedder.get_inputs()[0].name: x})[0].reshape(-1).astype(np.float32)
    return raw, raw / max(float(np.linalg.norm(raw)), 1e-9)


_EMAP_CACHE = {}


def _emap(path):
    """inswapper stores its identity projection as the graph's last initializer."""
    if path not in _EMAP_CACHE:
        import onnx  # noqa: WPS433
        from onnx import numpy_helper  # noqa: WPS433

        _EMAP_CACHE.clear()
        _EMAP_CACHE[path] = numpy_helper.to_array(onnx.load(path, load_external_data=False).graph.initializer[-1])
    return _EMAP_CACHE[path]


def source_embedding(np, graphs, raw, unit):
    """The identity vector the swapper's ``source`` input expects."""
    mode = graphs.profile["embedding"]
    if mode == "raw":
        return raw.reshape(1, -1)
    if mode == "emap":
        latent = unit.reshape(1, -1) @ _emap(graphs.swapper_path)
        return (latent / max(float(np.linalg.norm(latent)), 1e-9)).astype(np.float32)
    return unit.reshape(1, -1)


def blend_identity(np, source, target, strength):
    """The (raw, unit) identity a fraction ``strength`` of the way from the target's
    own face to the source's, so 0 keeps the target's identity and 1 is the full
    swap. Mixing the embeddings moves the generated face between the two people;
    cross-fading the pixels instead would only ghost one face over the other."""
    mixed = []
    for src, tgt in zip(source, target):
        mixed.append((1.0 - strength) * tgt + strength * src)
    raw, unit = mixed
    return raw, unit / max(float(np.linalg.norm(unit)), 1e-9)


# The identity a swapper puts on the face does not move in step with the mix of
# the two embeddings: it saturates, differently for every pair of faces (half the
# mix can leave the target's own face almost untouched, or give a face between the
# two). So a requested strength is met by measurement, not by the mixing weight.
IDENTITY_SOLVE_STEPS = 6
# Below this spread between the target-identity and source-identity swaps the
# measurement says nothing (two near-identical faces), and the weight is used as is.
MIN_IDENTITY_SPAN = 0.1


def embed_crop(np, Image, graphs, crop_rgb, size):
    """Unit ArcFace embedding of a swapper-space crop (aligned to the swapper's
    template): the identity the swapped face actually carries."""
    to_embed = similarity_transform(np, template_points(np, ARCFACE_128, size), template_points(np, ARCFACE_112_V2, EMBED_SIZE))
    crop = np.asarray(warp(np, Image, Image.fromarray(crop_rgb), to_embed, EMBED_SIZE), dtype=np.float32)
    x = (crop / 127.5 - 1.0).transpose(2, 0, 1)[np.newaxis]
    raw = graphs.embedder.run(None, {graphs.embedder.get_inputs()[0].name: x})[0].reshape(-1).astype(np.float32)
    return raw / max(float(np.linalg.norm(raw)), 1e-9)


def solve_mix(progress_at, target, steps=IDENTITY_SOLVE_STEPS):
    """The embedding-mix weight whose measured identity progress is nearest
    ``target``. ``progress_at(w)`` is 0 at w=0 (the target's own identity) and 1 at
    w=1 (the full swap) and rises with w, not linearly; bisection finds the weight
    without assuming a curve."""
    best_weight, best_gap = (0.0, abs(target)) if abs(target) <= abs(1.0 - target) else (1.0, abs(1.0 - target))
    low, high = 0.0, 1.0
    for _ in range(steps):
        mid = (low + high) / 2.0
        progress = progress_at(mid)
        if abs(progress - target) < best_gap:
            best_weight, best_gap = mid, abs(progress - target)
        if progress < target:
            low = mid
        else:
            high = mid
    return best_weight


def calibrated_swap(np, Image, graphs, original, source_id, target_id, strength):
    """Swap one aligned crop so the face lands ``strength`` of the way from the
    target's identity to the source's, measured with the same ArcFace embedder.
    Returns (swapped crop, mix weight used, progress reached)."""
    size = graphs.profile["size"]
    measured = {}

    def run(weight):
        if weight not in measured:
            mix = blend_identity(np, source_id, target_id, weight)
            crop = swap_crop(np, graphs, original, source_embedding(np, graphs, *mix))
            unit = embed_crop(np, Image, graphs, crop, size)
            measured[weight] = (crop, float(unit @ source_id[1] - unit @ target_id[1]))
        return measured[weight]

    toward_target, toward_source = run(0.0)[1], run(1.0)[1]
    span = toward_source - toward_target
    if span < MIN_IDENTITY_SPAN:
        weight = strength
        return run(weight)[0], weight, strength
    weight = solve_mix(lambda w: (run(w)[1] - toward_target) / span, strength)
    return run(weight)[0], weight, (run(weight)[1] - toward_target) / span


def gaussian_blur(np, plane, sigma):
    """Separable Gaussian blur of a float32 HxW plane (PIL cannot blur mode F)."""
    if sigma <= 0:
        return plane
    radius = max(int(sigma * 3.0 + 0.5), 1)
    x = np.arange(-radius, radius + 1, dtype=np.float32)
    kernel = np.exp(-(x * x) / (2.0 * sigma * sigma))
    kernel /= kernel.sum()
    padded = np.pad(plane, radius, mode="edge")
    rows = sum(kernel[i] * padded[i:i + plane.shape[0], radius:-radius] for i in range(len(kernel)))
    padded = np.pad(rows, ((0, 0), (radius, radius)), mode="edge")
    return sum(kernel[i] * padded[:, i:i + plane.shape[1]] for i in range(len(kernel))).astype(np.float32)


def box_mask(np, Image, ImageFilter, size, blur=BOX_BLUR):
    """A crop-space mask: opaque interior with a Gaussian-feathered border."""
    amount = int(size * 0.5 * blur)
    border = max(amount // 2, 1)
    mask = np.zeros((size, size), np.float32)
    mask[border:size - border, border:size - border] = 1.0
    return np.clip(gaussian_blur(np, mask, amount * 0.25), 0.0, 1.0)


def occlusion_mask(np, Image, ImageFilter, graphs, crop_rgb):
    """1 where the face is visible, 0 where something (a hand, hair, glasses)
    covers it; softened so the pasted edge follows the occluder smoothly."""
    size = crop_rgb.shape[0]
    resized = Image.fromarray(crop_rgb).resize((OCCLUDER_SIZE, OCCLUDER_SIZE), Image.BILINEAR)
    x = (np.asarray(resized, dtype=np.float32) / 255.0)[:, :, ::-1][np.newaxis]  # RGB -> BGR, NHWC
    raw = graphs.occluder.run(None, {graphs.occluder.get_inputs()[0].name: np.ascontiguousarray(x)})[0]
    visible = np.clip(np.squeeze(raw).astype(np.float32), 0.0, 1.0)
    visible = np.asarray(Image.fromarray(visible).resize((size, size), Image.BILINEAR), dtype=np.float32)
    blurred = gaussian_blur(np, visible, 5.0)
    return (np.clip(blurred, 0.5, 1.0) - 0.5) * 2.0


_D65 = (0.95047, 1.0, 1.08883)


def rgb_to_lab(np, rgb):
    """sRGB uint8 HxWx3 to CIELAB float32 (D65)."""
    c = rgb.astype(np.float32) / 255.0
    linear = np.where(c > 0.04045, ((c + 0.055) / 1.055) ** 2.4, c / 12.92)
    m = np.array([[0.4124564, 0.3575761, 0.1804375], [0.2126729, 0.7151522, 0.0721750], [0.0193339, 0.1191920, 0.9503041]], np.float32)
    xyz = (linear @ m.T) / np.array(_D65, np.float32)
    f = np.where(xyz > 0.008856, np.cbrt(xyz), 7.787 * xyz + 16.0 / 116.0)
    return np.stack([116.0 * f[..., 1] - 16.0, 500.0 * (f[..., 0] - f[..., 1]), 200.0 * (f[..., 1] - f[..., 2])], axis=-1)


def lab_to_rgb(np, lab):
    """CIELAB (D65) to sRGB uint8."""
    fy = (lab[..., 0] + 16.0) / 116.0
    f = np.stack([fy + lab[..., 1] / 500.0, fy, fy - lab[..., 2] / 200.0], axis=-1)
    xyz = np.where(f ** 3 > 0.008856, f ** 3, (f - 16.0 / 116.0) / 7.787) * np.array(_D65, np.float32)
    m = np.array([[3.2404542, -1.5371385, -0.4985314], [-0.9692660, 1.8760108, 0.0415560], [0.0556434, -0.2040259, 1.0572252]], np.float32)
    linear = np.clip(xyz @ m.T, 0.0, 1.0)
    c = np.where(linear > 0.0031308, 1.055 * np.power(linear, 1.0 / 2.4) - 0.055, 12.92 * linear)
    return np.clip(c * 255.0 + 0.5, 0, 255).astype(np.uint8)


def match_colour(np, swapped, original, mask, strength):
    """Shift the swapped face's CIELAB statistics to the face it replaces.

    The swapper reproduces the source photo's skin tone and light; matching mean
    and spread inside the mask makes the pasted face agree with its new context.
    ``strength`` 0 leaves the swap untouched, 1 matches fully.
    """
    if strength <= 0:
        return swapped
    inside = mask > 0.5
    if int(inside.sum()) < 64:
        return swapped
    a, b = rgb_to_lab(np, swapped), rgb_to_lab(np, original)
    out = a.copy()
    for c in range(3):
        mean_a, std_a = float(a[..., c][inside].mean()), float(a[..., c][inside].std()) + 1e-6
        mean_b, std_b = float(b[..., c][inside].mean()), float(b[..., c][inside].std()) + 1e-6
        matched = (a[..., c] - mean_a) * (std_b / std_a) + mean_b
        out[..., c] = a[..., c] * (1.0 - strength) + matched * strength
    return lab_to_rgb(np, out)


def swap_crop(np, graphs, crop_rgb, identity):
    """Run the swapper on one aligned RGB uint8 crop; returns the swapped RGB uint8."""
    profile = graphs.profile
    x = (crop_rgb.astype(np.float32) / 255.0 - profile["mean"]) / profile["std"]
    feed = {"source": identity.astype(np.float32), "target": np.ascontiguousarray(x.transpose(2, 0, 1)[np.newaxis])}
    out = graphs.swapper.run([graphs.swapper.get_outputs()[0].name], feed)[0]
    out = np.squeeze(out, 0).transpose(1, 2, 0)
    if profile["denormalize"]:
        out = out * profile["std"] + profile["mean"]
    return np.clip(out * 255.0 + 0.5, 0, 255).astype(np.uint8)


def paste_back(np, Image, base, crop_rgb, crop_mask, to_crop):
    """Warp a swapped crop and its mask back through ``to_crop`` and blend them
    into ``base`` (float32 HxWx3, updated in place)."""
    height, width = base.shape[:2]
    size = crop_rgb.shape[0]
    corners = np.array([[0, 0, 1], [size, 0, 1], [0, size, 1], [size, size, 1]], dtype=np.float64)
    quad = (np.linalg.inv(to_crop) @ corners.T).T[:, :2]
    x0, y0 = (max(0, int(np.floor(v))) for v in quad.min(axis=0))
    x1, y1 = min(width, int(np.ceil(quad[:, 0].max()))), min(height, int(np.ceil(quad[:, 1].max())))
    if x1 <= x0 or y1 <= y0:
        return
    shift = np.array([[1, 0, x0], [0, 1, y0], [0, 0, 1]], dtype=np.float64)
    inverse = to_crop @ shift  # region -> crop
    data = (inverse[0, 0], inverse[0, 1], inverse[0, 2], inverse[1, 0], inverse[1, 1], inverse[1, 2])
    region = (x1 - x0, y1 - y0)
    face = Image.fromarray(crop_rgb).transform(region, Image.AFFINE, data, resample=Image.BICUBIC)
    alpha = Image.fromarray(crop_mask.astype(np.float32)).transform(region, Image.AFFINE, data, resample=Image.BILINEAR)
    a = np.clip(np.asarray(alpha, dtype=np.float32), 0.0, 1.0)[:, :, None]
    patch = base[y0:y1, x0:x1]
    patch[:] = a * np.asarray(face, dtype=np.float32) + (1.0 - a) * patch


def face_area(face):
    x1, y1, x2, y2 = face["box"]
    return (x2 - x1) * (y2 - y1)


def sorted_faces(faces):
    """Largest face first: the stable, human-meaningful index order."""
    return sorted(faces, key=face_area, reverse=True)


def select_targets(faces, index, every):
    if every:
        return faces
    if index >= len(faces):
        _common.fail(f"target_face {index} is out of range: the target image has {len(faces)} detected face(s)", code=2)
    return [faces[index]]


def swap_image(np, Image, ImageFilter, graphs, target, reference, options):
    """Swap the chosen reference face onto the selected target face(s).

    Returns (RGB image, summary). ``options`` carries source_face, target_face,
    all_faces, occlusion, color_match, face_strength and score_threshold.
    """
    threshold = options.get("score_threshold", DEFAULT_SCORE_THRESHOLD)
    ref_faces = sorted_faces(detect_faces(np, Image, graphs.detector, reference, threshold))
    if not ref_faces:
        _common.fail("no face detected in the reference image; nothing to take the identity from", code=9)
    source_index = int(options.get("source_face", 0))
    if source_index >= len(ref_faces):
        _common.fail(f"source_face {source_index} is out of range: the reference image has {len(ref_faces)} detected face(s)", code=2)
    tgt_faces = sorted_faces(detect_faces(np, Image, graphs.detector, target, threshold))
    if not tgt_faces:
        _common.fail("no face detected in the target image; nothing to swap", code=9)
    chosen = select_targets(tgt_faces, int(options.get("target_face", 0)), bool(options.get("all_faces", False)))

    source_id = embed_face(np, Image, graphs, reference, ref_faces[source_index]["landmarks"])
    face_strength = min(1.0, max(0.0, float(options.get("face_strength", 1.0))))
    size = graphs.profile["size"]
    template = template_points(np, ARCFACE_128, size)
    use_occlusion = bool(options.get("occlusion", True)) and graphs.occluder is not None
    strength = float(options.get("color_match", 1.0))
    base = np.asarray(target, dtype=np.float32).copy()
    feather = box_mask(np, Image, ImageFilter, size)
    mixes, reached = [], []
    # Smaller faces first so a large foreground face is blended last and wins any overlap.
    for face in sorted(chosen, key=face_area):
        to_crop = similarity_transform(np, face["landmarks"], template)
        original = np.asarray(warp(np, Image, target, to_crop, size), dtype=np.uint8)
        if face_strength >= 1.0:
            swapped = swap_crop(np, graphs, original, source_embedding(np, graphs, *source_id))
        else:
            target_id = embed_face(np, Image, graphs, target, face["landmarks"])
            swapped, weight, progress = calibrated_swap(np, Image, graphs, original, source_id, target_id, face_strength)
            mixes.append(weight)
            reached.append(progress)
        mask = feather
        if use_occlusion:
            mask = np.minimum(mask, occlusion_mask(np, Image, ImageFilter, graphs, original))
        swapped = match_colour(np, swapped, original, mask, strength)
        paste_back(np, Image, base, swapped, mask, to_crop)
    out = Image.fromarray(np.clip(base + 0.5, 0, 255).astype(np.uint8))
    return out, {
        "swapper": graphs.profile["family"],
        "reference_faces": len(ref_faces),
        "target_faces": len(tgt_faces),
        "faces_swapped": len(chosen),
        "occlusion_mask": use_occlusion,
        "face_strength": face_strength,
        # What the strength cost: the embedding-mix weight that met it and the
        # identity progress the swap measured (0 = the target's face, 1 = the full swap).
        **({"identity_mix": round(sum(mixes) / len(mixes), 3), "identity_progress": round(sum(reached) / len(reached), 3)} if mixes else {}),
    }


def main():
    p = argparse.ArgumentParser(description="ONNX face swap (detect, embed, swap, blend back)")
    p.add_argument("--model", required=True, help="installed model directory (swapper, ArcFace, YuNet, optional XSeg)")
    p.add_argument("--image", required=True, help="target image path (the face to replace lives here)")
    p.add_argument("--reference", required=True, help="reference image path (the face to put on the target)")
    p.add_argument("--out", required=True, help="output image path")
    p.add_argument("--device", choices=["cpu", "cuda"], default="cpu")
    p.add_argument("--params", default="", help="JSON operation parameters (face_swap params_schema)")
    args = p.parse_args()
    os.environ["IMAGE_TOOLS_DEVICE"] = args.device
    try:
        options = json.loads(args.params) if args.params else {}
    except json.JSONDecodeError as exc:
        _common.fail(f"--params is not valid JSON: {exc}", code=2)
    if not isinstance(options, dict):
        _common.fail("--params must be a JSON object", code=2)

    np, _ort, Image = _common.require_deps()
    from PIL import ImageFilter  # noqa: WPS433

    graphs = resolve_graphs(args.model)
    try:
        target = _common.load_oriented_image(args.image, MAX_INPUT_PIXELS)[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open target image {args.image!r}: {exc}", code=6)
    try:
        reference = _common.load_oriented_image(args.reference, MAX_INPUT_PIXELS)[0]
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to open reference image {args.reference!r}: {exc}", code=6)
    out, summary = swap_image(np, Image, ImageFilter, graphs, target, reference, options)
    try:
        out.save(args.out, format="PNG")
    except Exception as exc:  # noqa: BLE001
        _common.fail(f"failed to write output {args.out!r}: {exc}", code=7)
    return summary


if __name__ == "__main__":
    main()
