"""YOLOX head decoding: pure numpy, no weights. Run with pytest."""
import numpy as np

from . import detect

SIZE = (416, 416)


def raw_head(entries):
    """A raw head with the given (cell index, dx, dy, log w, log h, objectness, class) rows."""
    n = detect._grid_cells(SIZE)
    raw = np.zeros((1, n, 85), dtype=np.float32)
    raw[:, :, 4] = 0.0
    for cell, dx, dy, lw, lh, obj, cls in entries:
        raw[0, cell, :4] = (dx, dy, lw, lh)
        raw[0, cell, 4] = obj
        raw[0, cell, 5 + cls] = 0.9
    return raw


def test_grid_cell_count_matches_the_official_tiny_head():
    assert detect._grid_cells(SIZE) == 3549


def test_a_raw_head_is_recognised_and_a_decoded_one_is_not():
    assert detect.is_raw_yolox_head(np, raw_head([]), SIZE)
    assert not detect.is_raw_yolox_head(np, np.zeros((1, 100, 6), dtype=np.float32), SIZE)


def test_decode_places_a_stride_8_cell_at_its_grid_position():
    # cell (row 2, col 3) at stride 8 is index 2*52+3; offset (0.5, 0.5), size e^1 * 8 px.
    cell = 2 * 52 + 3
    decoded = detect.decode_grid(np, raw_head([(cell, 0.5, 0.5, 1.0, 1.0, 0.9, 0)])[0], SIZE)
    cx, cy, w, h = decoded[cell, :4]
    assert (cx, cy) == ((3 + 0.5) * 8, (2 + 0.5) * 8)
    assert abs(w - np.e * 8) < 1e-3 and abs(h - np.e * 8) < 1e-3


def test_boxes_are_mapped_back_through_the_letterbox_with_labels_and_nms():
    # A 832x416 source letterboxed into 416x416: scale 0.5, padding below.
    cell = 10 * 52 + 20
    neighbour = cell + 1  # an overlapping duplicate of the same class must be suppressed
    other = 5 * 26 + 5 + 52 * 52  # a stride-16 cell with another class
    raw = raw_head([
        (cell, 0.5, 0.5, 2.0, 2.0, 0.95, 16),  # dog
        (neighbour, 0.5, 0.5, 2.0, 2.0, 0.60, 16),
        (other, 0.5, 0.5, 1.0, 1.0, 0.90, 0),  # person
    ])
    found = detect.yolox_detections(np, raw, SIZE, 0.5, 832, 416, threshold=0.3)
    labels = sorted(d["label"] for d in found)
    assert labels == ["dog", "person"], found
    dog = next(d for d in found if d["label"] == "dog")
    # centre (20.5*8, 10.5*8) = (164, 84) letterboxed -> (328, 168) source px.
    cx = (dog["box"][0] + dog["box"][2]) / 2 * 832
    cy = (dog["box"][1] + dog["box"][3]) / 2 * 416
    assert abs(cx - 328) < 1 and abs(cy - 168) < 1
    assert all(0.0 <= v <= 1.0 for d in found for v in d["box"])


def test_low_scores_are_dropped():
    raw = raw_head([(100, 0.5, 0.5, 1.0, 1.0, 0.1, 3)])
    assert detect.yolox_detections(np, raw, SIZE, 1.0, 416, 416, threshold=0.3) == []
