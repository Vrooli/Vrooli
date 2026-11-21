# Generic smart display v3 construction reference study

This is a new deterministic GLB study for a generic tabletop smart display. It is sized from Google's published Nest Hub (2nd gen) envelope and informed by official product photographs. It does not reuse product geometry, logos, screen artwork, or branded surface details. It is not a Nest Hub model and has no manufacturer identity.

The authored envelope is exactly 177.4 × 120.4 × 69.5 mm (X width × Y height × Z depth), matching the published Nest Hub 2nd gen overall dimensions for proportion reference. The source describes a 7-inch 1024×600 screen; the model's replaceable active surface is 153.0 × 89.4 mm, an authored estimate with matching near-16:9 proportions, not a manufacturer-published active-corner measurement. The screen panel leans back slightly, meets a continuous rounded bezel, and joins a shallow wedge speaker base. The base includes a woven-fabric material, repeated recessed acoustic ports, two bottom silicone contacts, small sensor apertures, and a status indicator. Materials are restrained graphite, smoked glass, warm charcoal fabric, and silicone.

The exact display target is node 8 `MainDisplayActivePlane`, mesh 6 primitive 0, material index 2 `Replaceable active display plane`. Its local POSITION bounds are `[-0.0765, 0.02445, 0.0149565]` to `[0.0765, 0.11385, 0.0243435]` metres. Its `TEXCOORD_0` spans fully from `[0,0]` to `[1,1]`; V=0 is mapped to the top edge and V=1 to the bottom edge. The panel sits 0.05 mm proud of the bezel face to avoid z-fighting. Replace this primitive's material to bind independent content; it contains no embedded image. The GLB has no external URIs, extensions, animations, or scripts.

## Reference review and limits

The official support specification page lists the Nest Hub 2nd gen dimensions as 177.4 mm wide, 120.4 mm high, and 69.5 mm deep, and a 7-inch 1024×600 touch screen. Official images show a wide rounded screen housing above a fabric speaker section, shallow tabletop support, top-edge sensors, and a rear microphone switch. The front and rear reference captures are retained to compare broad silhouette, screen-to-body ratio, speaker-base height, and control placement only.

The image files and saved Google page HTML are reference evidence only. Google does not state a license for these photos here. Do not put them in the GLB, screen content, published previews, or redistributable product package. The screen material is neutral and blank. This v3 asset has not yet had an actual Device Studio native render reviewed; the stored GLB/inventory facts establish file structure and proportions, not professional visual qualification, physical accuracy, or support stability. The parent reviewer must inspect native renders before admission.

## Build and provenance

Run `python3 derive_smart_display.py` in this directory to build the byte-identical GLB using Python's standard library. The script refuses to overwrite a different existing artifact; remove only the generated GLB when intentionally regenerating after a source change. All dimensions are in metres, Y is up, and the display faces +Z. The bevels and acoustic pattern are deterministic procedural geometry.

Reference sources:

- Google Nest and Home device specifications: https://support.google.com/googlehome/answer/7072284?hl=en
- Google Nest Hub (2nd gen) official overview: https://store.google.com/product/nest_hub_2nd_gen?hl=en-US
- Google Nest Hub (2nd gen) technical specifications: https://store.google.com/product/nest_hub_2nd_gen_specs?hl=en-US
