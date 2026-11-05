# Generic round-watch v3 strap shading repair

`derive_roundwatch_v3.py` reads the immutable Vrooli-authored round-watch v2
GLB and writes the versioned candidate at
`api/internal/physicalruntime/assets/roundwatch-v3/`. It changes only the
first and fourth 36-vertex rings of the two silicone strap meshes, assigning
flat ±Z normals to the broad planar strap faces. Case/display geometry,
bevel and side-ring normals, indices, material factors, and UVs remain as in
v2. The output manifest records source/output SHA-256 and candidate-only
limits. This is a normal-smoothing hypothesis pending a native comparison;
it does not establish product identity, material accuracy, or qualification.

Run from `scenarios/device-studio`:

```sh
python3 assets/sources/generic-roundwatch-v3-20261021/derive_roundwatch_v3.py
```
