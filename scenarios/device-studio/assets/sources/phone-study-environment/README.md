# Full phone-study environment exporter

This is a second closed profile, distinct from the historical table/cup
exporter. It imports the exact portable 8-node table GLB pinned in
`full-scene-manifest.json`, adds only the study's back wall, luminous window
panel and four frame bars, and emits a 14-mesh GLB plus exact node/material
inventory. It has no phone, screen, camera, light, animation, external URI, or
embedded texture. Cameras and lights remain renderer-owned typed settings.

The window's accepted study recipe used linear RGB `[0.78, 0.82, 0.88]` at
strength `0.8`. This exporter represents that product as
`EMISSION_V1` with strength one and factor `[0.624, 0.656, 0.704]`; the
manifest records the conversion. The plaster/window-frame recipe is fixed from
the reviewed study script. The table GLB's four Travertine maps remain separate
CC0 texture bindings and are not embedded or copied by this exporter.

The output is an unreviewed scene-study candidate. It does not reproduce the
room lighting setup by itself and must not be promoted to the Library without
owner review of the emitted inventory and renderer parse.
