# Authored generic phone starter

This package contains a neutral-display derivative of the Vrooli-authored phone model. The private research display image is excluded from the distributed bytes; the embedded display is a uniform neutral placeholder. The source mesh census and material binding are retained in `phone-starter-inventory.json`.

The Dynamic Island sensor cutout uses a derived matte material (`metallic=0`, `roughness=0.82`, `KHR_materials_specular.specularFactor=0`). Parent independently compared semantic mesh-node world-space triangle multiplicities against the research source; the maximum nearest-position delta was 1.405e-9 m. The check does not establish winding or material equivalence.

This is still an unreviewed starter candidate. It is a Vrooli-authored procedural interpretation based on Apple reference material, not manufacturer CAD, a measured physical device, or an Apple-endorsed asset. See `asset-manifest.json` for exact hashes, operation and review provenance.
