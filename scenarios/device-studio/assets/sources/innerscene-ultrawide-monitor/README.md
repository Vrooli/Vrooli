# Generic ultrawide monitor source

This source package contains one anonymous CC0 model by Innerscene. It is a generic desktop monitor with its supplied two-foot stand. The source page identifies no manufacturer or product model, so this asset must not be presented as a particular monitor. Rights and dimensions are pinned in `monitor-source-manifest.json`; the source page, terms, preview and independent measurements are retained under `docs/internal/goal/evidence/E4/free-models/anonymous-candidates-20261020/`.

The exact acquired original remains `ultrawide-computer-monitor.glb` and is unmodified. Use `ultrawide-computer-monitor-screen-uv-v1.glb` for a screen-bound candidate. The fixed standard-library script `normalize_monitor_screen_uv.py` changes only the unshared display-glass UV0 accessor. It affinely maps that accessor's actual source bounds to [0,1] so it meets Device Studio's current full-rectangle UV screen binding. It does not rewrite the GLB JSON, geometry, node frames, materials, textures, or embedded images. The existing screen image is a plain dark inactive field; no neutral replacement was needed and no private artwork is included.

Reproduce or verify the derivative from this directory with:

```sh
python3 normalize_monitor_screen_uv.py
```

The script checks the exact source SHA and expected node/material/texture/accessor chain before writing. If an output already exists, it accepts it only when its bytes equal the deterministic result; otherwise it refuses to overwrite. The derivative is 109,512 bytes with SHA-256 `eff6e9a20a4fd26003046e7f0c7b5d0ea737d83d4ce9adb66b1c118561155266`.

The source GLB's full vertex hierarchy measures approximately 700.000 × 451.833 × 219.125 mm (X width, Y height, Z depth), closely matching the creator's rounded 700 × 452 × 219 mm claim. These are model bounds, not independently verified physical dimensions. The measured glass front plane is 679.680 × 279.680 mm; it is not a separately identified active-pixel area. The supplied stand's lowest vertices lie at Y≈0 on both split feet, but this does not prove stable contact or load behavior. No wall-mount interface or moving-part animation is provided.

The public page provides one angled front-three-quarter preview and an anonymous interactive 3D viewer. Separate front, side and rear reference images were not found. For that reason the source and UV derivative are ready for owner integration, while visual and physical qualification remain open. A render of this model cannot serve as an independent reference view.
