# Generic vehicle dashboard display host v1

This deterministic derivative keeps the source dashboard body, steering wheel, vents, and instrument cluster, replaces the source's four overlapping map/routing screen overlays with one neutral rounded active screen, and recolors the mint/emissive styling to restrained dark dashboard materials. The source GLB remains byte-for-byte unchanged under `source/`.

The replaceable display target is node 3 `vehicle-display-assembly`, mesh 3, primitive 0, material 10 `vehicle-display-active-glass`. It is one rounded screen plane, with `TEXCOORD_0` from `[0,0]` to `[1,1]`; V=0 is at the top, U=0 at the left. The display faces the cabin along local normal approximately `(0, 0.208, 0.978)` in this Y-up source coordinate system. Node translation is retained. Primitive 1 is the separate satin bezel. No screen image, logo, or route graphic is embedded in either primitive.

Dimensions use the source's nominal real-scale metre units. The new active panel measures 246 × 128 mm before accounting for its slight tilt; this is an authored fit inside the source screen envelope, not a manufacturer specification. The body (node 0) is a host/support reference only. The source dashboard is a stylized procedural model, and the inferred Y=0 contact bound is not evidence of physical stability or manufacturing accuracy. The asset is generic and makes no OEM or CarPlay integration claim.

## Provenance and license

The retained source is Grab3D's Car Dashboard model, downloaded anonymously as a $0 CC0 package. The source package LICENSE and manifest state CC0 1.0 Universal / CC0-1.0; its exact source URL, download URL, source hash, byte count, and attribution of uncertainty are recorded in `manifest.json`. Keep `source/` with the derivative to preserve licensing/provenance evidence. CC0 reference: https://creativecommons.org/publicdomain/zero/1.0/ .

Source page: https://grab3d.com/models/vehicles/car-dashboard/ . The saved source preview and inventory evidence are retained in the Device Studio E4 free-model acquisition evidence. This derivative has not received native Device Studio render acceptance; geometry inspection and file checks do not qualify it for release.

## Regeneration

Run `python3 derive_vehicle_host.py`. The script uses Python standard-library modules only. It patches the retained GLB deterministically, preserves original non-screen meshes and embedded texture bytes, and appends the active-plane and bezel geometry. The source body and steering-wheel base-color maps are intentionally unused in the derivative to remove their mint styling; normal and metallic/roughness maps remain referenced. The original remains available for exact source fidelity.
