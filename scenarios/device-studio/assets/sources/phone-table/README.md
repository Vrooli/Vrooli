# Phone-study table scene export source

This is the tracked, fixed-purpose Blender exporter for the Vrooli-authored
tabletop-and-cup scene candidate. It is staged only by the reviewed E3 owner
operation; do not run Blender directly. The matching manifest binds its exact
profile, source study and material-map identities. A staged copy uses the fixed
role basenames under `inputs/` and writes only `outputs/phone-table-scene.glb`
and `outputs/scene-inventory.json`.

The six inputs are hash-checked because they establish source/material
provenance, but only generated geometry is exported. The phone model and screen
raster are not imported. The four CC0 Travertine001 maps remain separate
texture artifacts and are not embedded. The GLB contains no camera, light,
animation, skin, room, or URI. Blender's Y-up export is the sole basis
conversion into canonical meters; the scene candidate must not apply another
axis or scale conversion.

The source adapts study-only procedural materials to the bounded portable
material contract. The tabletop top is a simple slot whose external maps, grade,
texture repeat and bump semantics are declared separately in the evidence
candidate template. The procedural mineral sides are represented by the
midpoint of their documented color and roughness ranges. Timber legs retain the
study color. The missing window/wall/frame geometry means this package is not a
complete reproduction of the approved photograph.

The output inventory and scene-candidate template are evidence artifacts under
`docs/internal/goal/evidence/E3/scene-studies/phone/scene-candidate/`. The
template has no reviewed-library refs and is not a valid or submit-ready
SceneCandidate until those refs and emitted node/primitive indices are filled
from owner-reviewed outputs.
