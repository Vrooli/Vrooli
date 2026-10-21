"""Small deterministic coordinate helpers shared by the fixed iPad exporter."""
import math


def finite3(point):
    return len(point) == 3 and all(math.isfinite(float(value)) for value in point)


def blender_to_canonical(point):
    if not finite3(point):
        raise ValueError("Blender point must have three finite coordinates")
    x, y, z = (float(value) for value in point)
    return (x, z, -y)


def canonical_delta_from_blender_xy(dx, dy):
    if not math.isfinite(dx) or not math.isfinite(dy):
        raise ValueError("Blender placement delta must be finite")
    return (float(dx), 0.0, -float(dy))


def table_bounds_from_dimensions(center_xy, dimensions_xy, edge_margin):
    values = (*center_xy, *dimensions_xy, edge_margin)
    if len(center_xy) != 2 or len(dimensions_xy) != 2 or not all(math.isfinite(float(v)) for v in values):
        raise ValueError("table bounds require finite XY center, dimensions and margin")
    cx, cy = (float(v) for v in center_xy)
    width, depth = (float(v) for v in dimensions_xy)
    margin = float(edge_margin)
    if width <= 0 or depth <= 0 or margin < 0 or margin >= min(width, depth) / 2:
        raise ValueError("table dimensions and edge margin are inconsistent")
    return (cx - width / 2 + margin, cy - depth / 2 + margin,
            cx + width / 2 - margin, cy + depth / 2 - margin)


def validate_search(search):
    required = {"x_min", "x_max", "y_min", "y_max", "step"}
    if set(search) != required or not all(math.isfinite(float(search[key])) for key in required):
        raise ValueError("placement search must contain five finite bounds")
    if search["step"] <= 0 or search["x_min"] > search["x_max"] or search["y_min"] > search["y_max"]:
        raise ValueError("placement search bounds or step are invalid")
    columns = math.floor((search["x_max"] - search["x_min"]) / search["step"] + 1.000001)
    rows = math.floor((search["y_max"] - search["y_min"]) / search["step"] + 1.000001)
    if columns * rows > 100000:
        raise ValueError("placement search exceeds its fixed point bound")
    return columns, rows


def project_camera_local(point, focal_mm, sensor_width_mm, aspect):
    if not finite3(point) or not all(math.isfinite(value) and value > 0
                                     for value in (focal_mm, sensor_width_mm, aspect)):
        raise ValueError("camera point and optical values must be finite and positive")
    x, y, z = (float(value) for value in point)
    depth = -z
    if depth <= 0.05:
        raise ValueError("projected geometry crosses or approaches the camera plane")
    return (0.5 + x * focal_mm / (depth * sensor_width_mm),
            0.5 - y * focal_mm * aspect / (depth * sensor_width_mm))


def bounds_within_frame(bounds, margin):
    if len(bounds) != 4 or not math.isfinite(margin) or margin < 0 or margin >= 0.5:
        return False
    if not all(math.isfinite(float(value)) for value in bounds):
        return False
    return bounds[0] >= margin and bounds[1] >= margin and bounds[2] <= 1 - margin and bounds[3] <= 1 - margin
