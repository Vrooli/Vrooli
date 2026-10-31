# Neutral dashboard studio backdrop v1

`derive_vehicle_studio_context.py` starts from the pinned Vrooli-authored
cup-free study-room GLB and changes only the active glTF scene root list to
retain `RoomBackWall`. Table legs/top, window rails, and the light panel become
inactive; node/mesh/material/accessor records and the complete binary chunk
remain in the file. The starter binds the dashboard host to a candidate-only
generic wall fixture and uses this flat neutral backdrop for review framing.

This is a studio presentation context. It does not represent an automobile
interior, vehicle fit, installation, or compatibility. The support and camera
require independent native appearance review.

Run from `scenarios/device-studio`:

```sh
python3 assets/sources/vehicle-neutral-studio-context-v1/derive_vehicle_studio_context.py
```
