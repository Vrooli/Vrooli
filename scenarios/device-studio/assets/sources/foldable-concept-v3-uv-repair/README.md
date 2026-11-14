# Generic book-fold concept v3 UV repair

This deterministic derivative uses the existing `flipScreenUVV` GLB utility to append a private TEXCOORD_0 accessor for `LeftInnerDisplay` mesh 1 primitive 0. It maps V to `1 - V` so top-origin app imagery is upright. The source BIN prefix and all other accessors, geometry, materials, transforms, and right/outer display mappings remain unchanged. The source v2 candidate remains byte-for-byte unchanged.

Source SHA-256: `90e1e5195ee88018e37b98d8d08af4d1407d8161e6216cf3fb699921d77ce4ac`. The candidate asset and manifest hashes are in `api/internal/physicalruntime/assets/foldable-v3/asset-manifest-v3.json`. This is a screen-mapping correction only; native appearance review remains required. No product geometry, flexible display, or hinge qualification is claimed.
