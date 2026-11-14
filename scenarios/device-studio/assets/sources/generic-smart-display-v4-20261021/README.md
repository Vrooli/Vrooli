# Generic smart display v4 shell-normal repair

V4 is a source-preserving asset derivative of `generic-smart-display-v3-20261021`. It addresses one failure seen in the actual v3 Device Studio still: visible triangular glossy shading on the rounded display rear shell. The derivative smooths only the area-weighted vertex normals of that shell's existing triangles and changes its graphite polymer from metallic/roughness `0.10/0.38` to `0.04/0.56`. This reduces faceted highlights while retaining the authored silhouette, topology, bounds, display plane, node hierarchy, and all other materials and geometry. The original v3 GLB and IDs are unchanged.

The screen target remains node 8 `MainDisplayActivePlane`, mesh 6, primitive 0, material 2. Its geometry and UVs are unchanged; V=0 remains at the top. V4 retains the v3's generic reference-inspired proportions and qualification limits. It does not use Google product geometry, logos, UI, or reference images. It is not a Google/Nest Hub product and is not qualified hardware.

The v3 actual output that motivated this repair is retained at `../../../../docs/internal/goal/evidence/E4/family-native-review/native-runs/lease25-smart-display-v3-1/family-still.png`. The v4 asset itself has not been native-rendered or accepted. Root must review a later public Device Studio native pilot before any family-quality or release claim.

## Rebuild and provenance

Run `python3 derive_smart_display_v4.py`. The standard-library generator reads the retained v3 GLB, validates the `Display rear shell` node/mesh/material contract, updates only that mesh's normal accessor, reduces shell specularity, and writes the sibling v4 GLB and manifest. `manifest.json` records exact source and output hashes and unchanged display binding. The generator refuses unexpected source structure.
