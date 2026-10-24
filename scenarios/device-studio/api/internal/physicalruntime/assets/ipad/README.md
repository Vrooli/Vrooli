# DatSketch iPad mini 6 candidate assets

These assets are CC BY 4.0 adaptations of DatSketch's “IPad Mini 6 2021”
Sketchfab model. The original screen image was replaced with a neutral
placeholder; it is not bundled here. The original source GLB and creator
snapshot remain in E4 research evidence. The fixed derivation source and its
manifests remain under `scenarios/device-studio/assets/sources/ipad-mini-starter`;
run receipts are retained in E4 evidence.

`ipad-mini-neutral-screen.glb` is the first derivative and retains the source
screen UV chart. That chart has slight edge overscan and does not satisfy the
renderer full-rectangle UV contract. `ipad-mini-screen-uv-normalized.glb` is a
separate follow-on candidate with a private UV accessor normalized from the
measured chart envelope. The original interleaved BIN bytes, positions,
materials, and images are preserved in that derivative.

`ipad-mini-screen-uv-oriented-180-v2.glb` is a separate candidate revision
derived from the normalized v1 bytes. It rotates only the display UV mapping
180 degrees, following the screen orientation observed in native hero-4. Its
screen pixels remain neutral; the private app screenshot used for the local
hero proof is retained only in evidence and is not part of these starter assets.
The v1 candidate and its scene/camera remain unchanged.

The library candidate identifies the model as a DatSketch model and remains
unqualified. Dimensions and the 117.281 × 177.229 mm screen plane are measured
from the GLB under its metre assumption, not measured from physical hardware.
The whole-model 8.775 mm depth includes protruding geometry; body-only depth is
unknown. The tablet rest pose is a source-geometry candidate based on three
calculated support locations. It is not a tested stability or contact claim.

Attribution: “iPad mini 6 2021” by DatSketch, adapted under CC BY 4.0.
Source: https://sketchfab.com/3d-models/ipad-mini-6-2021-f363c2b1c74a462588db184bd4fb024e
License: https://creativecommons.org/licenses/by/4.0/
