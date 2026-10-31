"""Closed Blender derivative: neutral display and matte island on authored phone geometry."""
import hashlib
import json
import struct
from pathlib import Path

import bpy


ROOT = Path(__file__).resolve().parent
MANIFEST = ROOT / "phone-starter-manifest.json"
INPUTS = ROOT / "inputs"
OUTPUTS = ROOT / "outputs"
SOURCE_NAME = "authored-phone-source.glb"
EXPECTED_ISLAND_NODE = "Dynamic Island black sensor cutout"
EXPECTED_SOURCE_ISLAND_MATERIAL = "black glass perimeter"
EXPECTED_DERIVED_ISLAND_MATERIAL = "Matte black sensor cutout"
EXPECTED_DISPLAY_IMAGE = "portable-display"


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def read_manifest():
    require(MANIFEST.is_file() and not MANIFEST.is_symlink(), "manifest must be a regular staged file")
    require(MANIFEST.stat().st_size <= 64 * 1024, "manifest exceeds fixed bound")
    raw = MANIFEST.read_bytes()
    manifest = json.loads(raw.decode("utf-8"))
    require(manifest["profile_id"] == "device-studio-authored-phone-starter-v1", "wrong fixed profile")
    require(list(bpy.app.version[:3]) == manifest["blender_version"], "Blender version mismatch")
    source = manifest["source"]
    path = INPUTS / source["staged_name"]
    require(source["staged_name"] == SOURCE_NAME and path.is_file() and not path.is_symlink(), "source is not fixed staged input")
    require(path.stat().st_size == source["size_bytes"] <= 8 * 1024 * 1024, "source size mismatch or exceeds bound")
    data = path.read_bytes()
    require(len(data) == source["size_bytes"] and sha256(data) == source["sha256"], "source digest mismatch")
    return manifest, sha256(raw), path


def image_digest(image):
    packed = image.packed_file
    require(packed is not None, "source display image must be embedded in the GLB")
    return sha256(bytes(packed.data))


def parse_export(data, manifest):
    require(len(data) >= 20 and data[:4] == b"glTF", "output is not GLB")
    magic, version, declared = struct.unpack_from("<4sII", data, 0)
    require(version == 2 and declared == len(data), "GLB header mismatch")
    offset, chunks = 12, []
    while offset < len(data):
        require(offset + 8 <= len(data), "truncated GLB chunk header")
        size, kind = struct.unpack_from("<II", data, offset)
        offset += 8
        require(size <= len(data) - offset, "GLB chunk exceeds file")
        chunks.append((kind, data[offset:offset + size]))
        offset += size
    require(len(chunks) == 2 and chunks[0][0] == 0x4E4F534A and chunks[1][0] == 0x004E4942,
            "GLB must contain JSON and BIN chunks only")
    doc = json.loads(chunks[0][1].decode("utf-8").rstrip(" \t\r\n\0"))
    require(not any("uri" in image for image in doc.get("images", [])), "output may not externalize image bytes")
    require(not doc.get("cameras") and not doc.get("animations") and not doc.get("skins"), "unexpected camera/animation/skin")
    nodes = doc.get("nodes", [])
    mesh_nodes = [node for node in nodes if "mesh" in node]
    require(len(mesh_nodes) == manifest["output"]["expected_mesh_node_count"], "mesh node count changed")
    triangle_count = 0
    for mesh in doc.get("meshes", []):
        for primitive in mesh.get("primitives", []):
            require(primitive.get("mode", 4) == 4, "non-triangle primitive found")
            indices = primitive.get("indices")
            if indices is not None:
                triangle_count += doc["accessors"][indices]["count"] // 3
            else:
                position = primitive["attributes"]["POSITION"]
                triangle_count += doc["accessors"][position]["count"] // 3
    require(triangle_count == manifest["output"]["expected_triangle_count"], "triangle count changed")
    island = [node for node in nodes if node.get("name") == EXPECTED_ISLAND_NODE]
    require(len(island) == 1, "island semantic node binding differs")
    material_indices = {
        primitive.get("material")
        for primitive in doc["meshes"][island[0]["mesh"]].get("primitives", [])
    }
    require(len(material_indices) == 1, "island mesh must bind one explicit material")
    material_index = next(iter(material_indices))
    materials = doc.get("materials", [])
    require(0 <= material_index < len(materials), "island material index is invalid")
    material = materials[material_index]
    require(material.get("name") == EXPECTED_DERIVED_ISLAND_MATERIAL, "derived island material semantic binding differs")
    pbr = material.get("pbrMetallicRoughness", {})
    require(abs(pbr.get("metallicFactor", 0) - manifest["material_derivation"]["target_metallic"]) < 1e-6,
            "island metallic factor differs")
    require(abs(pbr.get("roughnessFactor", 1) - manifest["material_derivation"]["target_roughness"]) < 1e-6,
            "island roughness differs")
    ext = material.get("extensions", {}).get("KHR_materials_specular", {})
    require(abs(ext.get("specularFactor", 1) - manifest["material_derivation"]["target_specular_factor"]) < 1e-6,
            "island specular factor differs or extension absent")
    images = doc.get("images", [])
    display = [item for item in images if item.get("name") == EXPECTED_DISPLAY_IMAGE]
    require(len(images) == 1 and len(display) == 1 and "bufferView" in display[0],
            "only the one neutralized embedded display image is permitted")
    view = doc["bufferViews"][display[0]["bufferView"]]
    start, size = view.get("byteOffset", 0), view.get("byteLength", 0)
    require(size > 0 and start <= len(chunks[1][1]) and size <= len(chunks[1][1]) - start,
            "embedded neutral display buffer bounds are invalid")
    embedded_display = chunks[1][1][start:start + size]
    require(sha256(embedded_display) != manifest["source"]["embedded_display"]["sha256"],
            "source display image bytes remain in derivative")
    return doc, embedded_display, {"mesh_node_count": len(mesh_nodes), "triangle_count": triangle_count,
                 "island_node_index": nodes.index(island[0]), "island_mesh_index": island[0]["mesh"],
                 "island_material_index": material_index, "island_material": material}


def main():
    manifest, manifest_sha, source_path = read_manifest()
    require(not OUTPUTS.exists() and not OUTPUTS.is_symlink(), "fixed output directory already exists")
    OUTPUTS.mkdir()
    bpy.ops.wm.read_factory_settings(use_empty=True)
    bpy.ops.import_scene.gltf(filepath=str(source_path))

    image = bpy.data.images.get(EXPECTED_DISPLAY_IMAGE)
    source_image = manifest["source"]["embedded_display"]
    require(image is not None and tuple(image.size) == tuple(source_image["dimensions_pixels"]), "embedded display image missing or dimensions differ")
    require(image_digest(image) == source_image["sha256"], "embedded display image baseline differs")
    require(sum(1 for item in bpy.data.images if item.name == EXPECTED_DISPLAY_IMAGE) == 1,
            "display image name is ambiguous")
    width, height = int(image.size[0]), int(image.size[1])
    require(width > 0 and height > 0 and width * height <= 8_000_000, "display image dimensions exceed bound")
    neutral = manifest["neutral_display_linear_rgba"]
    pixel = tuple(float(channel) for channel in neutral)
    require(len(pixel) == 4 and 0 <= min(pixel) <= max(pixel) <= 1 and pixel[3] == 1.0,
            "neutral display color is invalid")
    image.pixels.foreach_set(pixel * (width * height))
    image.update()
    image.pack()

    node = bpy.data.objects.get(EXPECTED_ISLAND_NODE)
    require(node is not None and node.type == "MESH" and len(node.material_slots) == 1,
            "island node/material binding differs")
    material = node.material_slots[0].material
    require(material is not None and material.name == EXPECTED_SOURCE_ISLAND_MATERIAL,
            "island material mapping differs")
    source_uses = sum(
        1 for obj in bpy.context.scene.objects if obj.type == "MESH"
        for slot in obj.material_slots if slot.material is material
    )
    require(source_uses == manifest["material_derivation"]["source_material_use_count"],
            "source material sharing differs from pinned GLB inventory")
    derived_material = material.copy()
    derived_material.name = EXPECTED_DERIVED_ISLAND_MATERIAL
    node.material_slots[0].material = derived_material
    material = derived_material
    shader = material.node_tree.nodes.get("Principled BSDF") if material.use_nodes else None
    require(shader is not None, "island material is not a Principled material")
    shader.inputs["Metallic"].default_value = manifest["material_derivation"]["target_metallic"]
    shader.inputs["Roughness"].default_value = manifest["material_derivation"]["target_roughness"]
    specular = shader.inputs.get("Specular IOR Level")
    require(specular is not None, "Blender Principled Specular IOR Level input unavailable")
    specular.default_value = manifest["material_derivation"]["target_specular_ior_level"]

    path = OUTPUTS / manifest["output"]["glb_name"]
    bpy.ops.object.select_all(action="SELECT")
    bpy.ops.export_scene.gltf(filepath=str(path), export_format="GLB", use_selection=True,
                              export_yup=True, export_apply=False, export_cameras=False,
                              export_lights=False, export_materials="EXPORT", export_extras=True,
                              export_texcoords=True, export_normals=True, export_animations=False)
    require(path.is_file() and not path.is_symlink(), "output must be a regular GLB")
    require(path.stat().st_size <= manifest["output"]["max_bytes"], "output exceeds fixed bound")
    output = path.read_bytes()
    require(len(output) == path.stat().st_size, "output changed while reading")
    doc, display_png, island_binding = parse_export(output, manifest)
    check_path = OUTPUTS / ".neutral-display-check.png"
    check_path.write_bytes(display_png)
    decoded = bpy.data.images.load(str(check_path), check_existing=False)
    expected = manifest["neutral_display_linear_rgba"]
    pixels = [0.0] * (int(decoded.size[0]) * int(decoded.size[1]) * 4)
    decoded.pixels.foreach_get(pixels)
    require(list(decoded.size) == source_image["dimensions_pixels"], "exported neutral display dimensions changed")
    require(max(pixels[index] for index in range(0, len(pixels), 4)) - min(pixels[index] for index in range(0, len(pixels), 4)) < 0.01 and
            max(pixels[index] for index in range(1, len(pixels), 4)) - min(pixels[index] for index in range(1, len(pixels), 4)) < 0.01 and
            max(pixels[index] for index in range(2, len(pixels), 4)) - min(pixels[index] for index in range(2, len(pixels), 4)) < 0.01,
            "exported screen contains non-neutral source variation")
    require(max(abs(pixels[index] - expected[index % 4]) for index in range(4)) < 0.02,
            "exported neutral display color differs from manifest")
    bpy.data.images.remove(decoded)
    check_path.unlink()
    inventory = {
        "profile_id": manifest["profile_id"], "status": "neutralized_authored_model_unreviewed",
        "source_glb_sha256": manifest["source"]["sha256"], "source_size_bytes": manifest["source"]["size_bytes"],
        "source_embedded_display_sha256": source_image["sha256"],
        "display_pixels_removed": True, "neutral_display_linear_rgba": neutral,
        "display_dimensions_pixels": [width, height], "derived_display_png_sha256": sha256(display_png),
        "derived_glb_sha256": sha256(output), "derived_size_bytes": len(output),
        "gltf_asset": doc.get("asset"), "gltf_extensions": {"used": doc.get("extensionsUsed", []), "required": doc.get("extensionsRequired", [])},
        "mesh_node_count": island_binding["mesh_node_count"], "triangle_count": island_binding["triangle_count"],
        "island_binding": {key: island_binding[key] for key in ("island_node_index", "island_mesh_index", "island_material_index")},
        "island_material": island_binding["island_material"], "manifest_sha256": manifest_sha,
        "export_script_sha256": sha256(Path(__file__).read_bytes()),
        "claims": manifest["rights_and_claims"],
    }
    receipt = OUTPUTS / manifest["output"]["receipt_name"]
    receipt.write_text(json.dumps(inventory, indent=2, sort_keys=True) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
