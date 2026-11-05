# Generic round smartwatch concept v2

This is a deterministic, Vrooli-authored GLB for a generic round-watch candidate. It replaces the prior wrong-shape source only through a new versioned starter; it does not modify or re-label the earlier square/rectangular candidate bytes.

The circular app surface is node `One clean circular replaceable display`, mesh 2, primitive 0, material 2. It is a single fan-triangulated disk with unit UVs and top-left image origin (V=0 at the top). The circular active surface is 36.7 mm authored diameter. It faces local +Z; the v2 preset uses a face-on review camera. The case shell is authored at 47.4 x 47.4 x 12.1 mm for visual comparison, and the complete model includes protruding glass/bezel, side crown/buttons, upper/lower charcoal elastomer strap sections, buckle details, optical-back rings and contact pads. Strap/contact and tabletop stability remain hypotheses, not measurements or qualification.

Official comparison references:

- Samsung Galaxy Watch Ultra specifications: https://www.samsung.com/uk/watches/galaxy-watch/galaxy-watch-ultra-titanium-grey-lte-sm-l705fdaaeua/ . Samsung lists body dimensions 47.4 x 47.1 x 12.1 mm; that value is comparison-only.
- Official product image/gallery: https://www.samsung.com/ae/watches/galaxy-watch/galaxy-watch-ultra-titanium-gray-lte-sm-l705fdaaxsg/ . Photographs were consulted as held-out shape/material references. No photo, logo, product geometry, brand, or identity is embedded or claimed.

`generate.py` uses only the Python standard library. It produces deterministic GLB bytes; the embedded candidate hash and source manifest live at `../../../api/internal/physicalruntime/assets/roundwatch-v2/asset-manifest-v2.json`. The candidate is unqualified pending root-owned native review.
