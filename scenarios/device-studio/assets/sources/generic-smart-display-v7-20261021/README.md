# Smart-display v7 coplanar-cap repair

The native v6 review showed radial stripes where the display rear-shell front cap and continuous bezel occupy the same plane. Geometry inspection measured the shell-cap plane at 0.0267138936 m and the bezel ring at 0.0267138918–0.0267138937 m along the display front normal; the active display is only 0.05 mm ahead. This is coplanar overlap (z-fighting), not a normal-smoothing defect.

`derive_smart_display_v7.py` derives the new candidate from the immutable v6 GLB. It removes only the 192 shell triangles whose geometric normals face the display front and whose vertices lie on that measured cap plane. It retains the rear shell's side/back triangles and every original vertex attribute; display mesh, screen positions/UVs, materials, nodes, transforms, and source asset bytes remain unchanged. The new GLB and its lineage manifest are at `api/internal/physicalruntime/assets/smart-display-v7/`.

Run the focused regression with:

```sh
python3 -m unittest discover -s scenarios/device-studio/assets/sources/generic-smart-display-v7-20261021 -p 'test_*.py' -v
```

The regression pins v3–v6 hashes, checks deterministic derivation, confirms the display plane/hierarchy/materials are unchanged, and verifies that no shell front-cap triangle remains at the overlapping plane. This candidate has not received a native appearance review and remains unqualified; v6 is preserved.
