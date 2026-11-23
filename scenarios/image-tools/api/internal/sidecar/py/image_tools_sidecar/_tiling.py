"""_tiling — run a fully convolutional image-to-image ONNX model over any size.

Dynamic-shape restoration networks (SCUNet, NAFNet) take ``[1, 3, H, W]`` for any
H and W that satisfy their downsampling, and fail (or lose detail) when the image
is resized to a made-up fixed size instead. This runs them at the image's own
resolution: reflect-pad to the model's stride, cut overlapping tiles with a margin
of context that is discarded afterwards so seams do not show, and crop the output
back to exactly the input's size.
"""

from __future__ import annotations


def pad_to_multiple(np, arr, multiple, minimum=0):
    """Reflect-pad an HxWx3 array so each side is a multiple of ``multiple`` and at
    least ``minimum`` pixels (a network that downsamples several times fails on a
    side smaller than its deepest stage)."""
    h, w = arr.shape[:2]
    target_h = max(h + (-h) % multiple, minimum + (-minimum) % multiple)
    target_w = max(w + (-w) % multiple, minimum + (-minimum) % multiple)
    if (target_h, target_w) == (h, w):
        return arr
    # np.pad's reflect needs a pad no larger than the side; symmetric repeats the
    # border by mirroring and has no such limit once applied repeatedly.
    out = arr
    while out.shape[0] < target_h or out.shape[1] < target_w:
        ph = min(target_h - out.shape[0], out.shape[0])
        pw = min(target_w - out.shape[1], out.shape[1])
        out = np.pad(out, ((0, ph), (0, pw), (0, 0)), mode="reflect" if min(out.shape[:2]) > 1 else "edge")
    return out


def tiled_apply(np, session, rgb, *, tile, margin, multiple, pass_multiple=None, minimum=0):
    """Apply the network to an HxWx3 float32 array in [0, 1]; returns the same shape.

    ``multiple`` pads the whole image, ``pass_multiple`` (default ``multiple``) and
    ``minimum`` pad every pass so the network always sees a valid size.
    """
    pass_multiple = pass_multiple or multiple
    height, width = rgb.shape[:2]
    padded = pad_to_multiple(np, rgb, multiple)
    # Reflect context around the whole image so edge tiles also have a margin.
    framed = np.pad(padded, ((margin, margin), (margin, margin), (0, 0)), mode="reflect")
    ph, pw = padded.shape[:2]
    out = np.empty_like(padded)
    input_name = session.get_inputs()[0].name
    for y in range(0, ph, tile):
        for x in range(0, pw, tile):
            y1, x1 = min(y + tile, ph), min(x + tile, pw)
            window = framed[y : y1 + 2 * margin, x : x1 + 2 * margin]
            chunk = pad_to_multiple(np, window, pass_multiple, minimum)
            nchw = chunk.transpose(2, 0, 1)[np.newaxis].astype(np.float32)
            result = session.run(None, {input_name: nchw})[0]
            tile_out = np.squeeze(result).transpose(1, 2, 0)
            out[y:y1, x:x1] = tile_out[margin : margin + (y1 - y), margin : margin + (x1 - x)]
    return out[:height, :width]
