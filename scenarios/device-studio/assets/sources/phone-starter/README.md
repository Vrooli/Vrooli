# Neutral-display authored phone derivative

This fixed Blender operation consumes one hash-pinned research GLB and emits a
separate starter derivative. It keeps the authored mesh and UVs, replaces the
embedded `portable-display` pixels with a neutral uniform gray, and copies the
shared `black glass perimeter` material only onto the Dynamic Island semantic
mesh before setting metallic 0, roughness 0.82 and glTF
`KHR_materials_specular.specularFactor` 0. The other three users of the shared
source material remain unchanged. The source GLB and its embedded screen image
are not copied to the starter package.

The emitted GLB and receipt are fixed outputs. The operation verifies source
image digest/dimensions, mesh and triangle census, the material mapping, the
neutralized output image, and the output material factors. A parent-owned
operation must run it and review its emitted bytes before populating
`api/internal/physicalruntime/assets/phone/`. No renderer logic branches on phone/node names; the
derived asset carries ordinary GLB geometry/material data.
