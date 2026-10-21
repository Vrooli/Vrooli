# Generic standing picture-frame source

This directory retains Innerscene's exact original CC0 GLB, acquired anonymously from its public part endpoint. The source describes a standing photo frame with a deep opening, blank backing, glazing, and rear easel. It is a physical picture frame, not an electronic smart display. Its separately named glazing node and unprinted insert make it a credible digital-photo-frame adaptation source, pending owner screen-binding and visual review.

The measured source bounds agree with Innerscene's declared 18.9 × 12.2 × 24.6 cm W × D × H. The GLB uses meters and embeds its materials/images. No source edits or derivatives are included. See `source-manifest.json` and the retained source page, terms, and preview in `docs/internal/goal/evidence/E4/family-candidates-20261021/`.

## Screen-visible v4 descendants

`derive_visible_frame.py` creates new tabletop and wall descendants from the frozen runtime v2/v3 candidates. It removes only the `Clear protective picture glazing` node and remaps the remaining node indices so the GLB scene remains valid. The glass mesh/material/image records and complete input BIN chunk remain unchanged and unreferenced; the screen insert, screen UVs, frame, and tabletop/wall support stay unchanged. This omits the source's 0.96-transmission, 0.06-roughness pane from the active digital-frame render because the actual Device Studio stills showed it reflecting over and obscuring the app. The derivatives are candidates only and need a native re-render before acceptance. Manifest: `scenarios/device-studio/api/internal/physicalruntime/assets/digital-frame/asset-manifest-v4.json`.
