import copy
import hashlib
import importlib.util
import json
import math
import shutil
import subprocess
import sys
import tempfile
import unittest
import struct
import os
from pathlib import Path


HERE = Path(__file__).resolve().parent
REPO = HERE.parents[4]
RESEARCH = REPO / "scenarios/device-studio/docs/internal/goal/evidence/E4/free-models/sceneview-research"
SPEC = importlib.util.spec_from_file_location("derive_ipad_starter", HERE / "derive_ipad_starter.py")
DERIVER = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(DERIVER)


class IPadStarterDerivationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.manifest = json.loads((HERE / "ipad-starter-manifest.json").read_text(encoding="utf-8"))
        cls.source, cls.source_doc, cls.source_bin = synthetic_glb()
        cls.manifest = copy.deepcopy(cls.manifest)
        cls.manifest["source"]["sha256"] = hashlib.sha256(cls.source).hexdigest()
        cls.manifest["source"]["size_bytes"] = len(cls.source)
        payloads = [b"IMG0DATA", b"PRIVATE_SCREEN_ART_00000", b"IMG2DATA"]
        for row, payload in zip(cls.manifest["source"]["embedded_images"], payloads):
            row["size_bytes"] = len(payload)
            row["sha256"] = hashlib.sha256(payload).hexdigest()
        cls.manifest["source"]["screen_binding"]["uv_min"] = [0.0, 0.0]
        cls.manifest["source"]["screen_binding"]["uv_max"] = [1.0, 1.0]
        cls.source_doc, cls.source_bin = DERIVER.parse_glb(cls.source)

    def test_real_derivative_preserves_every_other_buffer_view(self):
        manifest = self.manifest
        out, screen_png, uv, hashes = DERIVER.replace_screen_image(self.source, manifest)
        doc, binary = DERIVER.parse_glb(out)
        source_doc, source_bin = self.source_doc, self.source_bin
        target = manifest["source"]["screen_binding"]["image_index"]
        target_view = source_doc["images"][target]["bufferView"]
        expected_uv = ([0.0, 0.0], [1.0, 1.0])
        for actual, expected in zip(uv["uv_min"] + uv["uv_max"], expected_uv[0] + expected_uv[1]):
            self.assertTrue(math.isclose(actual, expected, abs_tol=1e-9))
        self.assertEqual(doc["images"][target]["mimeType"], "image/png")
        self.assertEqual(hashlib.sha256(screen_png).hexdigest(), DERIVER.sha256(
            DERIVER.buffer_view_bytes(doc, binary, target_view)))
        self.assertEqual(len(hashes), len(source_doc["bufferViews"]) - 1)
        for index in range(len(source_doc["bufferViews"])):
            if index == target_view:
                continue
            before = DERIVER.buffer_view_bytes(source_doc, source_bin, index)
            after = DERIVER.buffer_view_bytes(doc, binary, index)
            self.assertEqual(after, before, f"bufferView {index} payload changed")
            self.assertEqual(hashes[str(index)], hashlib.sha256(before).hexdigest())
        self.assertNotIn(DERIVER.buffer_view_bytes(source_doc, source_bin, target_view), binary)

    def test_pinned_source_and_image_bindings_are_required(self):
        damaged = bytearray(self.source)
        damaged[-1] ^= 1
        with self.assertRaisesRegex(ValueError, "source GLB pin mismatch"):
            DERIVER.replace_screen_image(bytes(damaged), self.manifest)
        doc = copy.deepcopy(self.source_doc)
        doc["nodes"][14]["mesh"] = 9
        with self.assertRaisesRegex(ValueError, "screen node/mesh binding differs"):
            DERIVER.validate_source(doc, self.source_bin, self.manifest)

    def test_overlapping_buffer_view_is_rejected(self):
        doc = copy.deepcopy(self.source_doc)
        doc["bufferViews"].append(copy.deepcopy(doc["bufferViews"][0]))
        with self.assertRaisesRegex(ValueError, "bufferViews overlap"):
            DERIVER.validate_source(doc, self.source_bin, self.manifest)

    def test_screen_buffer_view_cannot_be_shared_with_geometry(self):
        doc = copy.deepcopy(self.source_doc)
        doc["accessors"][0]["bufferView"] = doc["images"][1]["bufferView"]
        with self.assertRaisesRegex(ValueError, "screen bufferView is shared with an accessor"):
            DERIVER.validate_source(doc, self.source_bin, self.manifest)

    def test_explicit_uv_normalization_adds_private_view_and_preserves_existing_payloads(self):
        source, manifest = neutral_synthetic_glb()
        out, facts = DERIVER.normalize_screen_uv(source, manifest)
        source_doc, source_bin = DERIVER.parse_glb(source)
        out_doc, out_bin = DERIVER.parse_glb(out)
        self.assertEqual(out_bin[:len(source_bin)], source_bin)
        self.assertEqual(len(out_doc["bufferViews"]), len(source_doc["bufferViews"]) + 1)
        self.assertEqual(out_doc["accessors"][0]["bufferView"], len(source_doc["bufferViews"]))
        self.assertEqual(out_doc["accessors"][0].get("byteOffset", 0), 0)
        self.assertEqual(facts["source_uv_min"], [0.0, 0.0])
        self.assertEqual(facts["source_uv_max"], [1.0, 1.0])
        self.assertEqual(facts["normalized_uv_min"], [0.0, 0.0])
        self.assertEqual(facts["normalized_uv_max"], [1.0, 1.0])
        for index, before in enumerate(source_doc["bufferViews"]):
            self.assertEqual(DERIVER.buffer_view_bytes(source_doc, source_bin, index),
                             DERIVER.buffer_view_bytes(out_doc, out_bin, index))
        expected = copy.deepcopy(source_doc)
        expected["bufferViews"].append(copy.deepcopy(out_doc["bufferViews"][-1]))
        expected["accessors"][0].update({"bufferView": len(source_doc["bufferViews"]),
                                          "min": [0.0, 0.0], "max": [1.0, 1.0]})
        expected["buffers"][0]["byteLength"] = len(out_bin)
        self.assertEqual(out_doc, expected)

    def test_uv_normalization_refuses_shared_uv_accessor(self):
        source, manifest = neutral_synthetic_glb(shared_uv=True)
        with self.assertRaisesRegex(ValueError, "screen UV accessor is shared outside"):
            DERIVER.normalize_screen_uv(source, manifest)

    def test_orientation_derivative_rotates_only_the_private_screen_uv_view(self):
        source, base_manifest = neutral_synthetic_glb()
        normalized, _ = DERIVER.normalize_screen_uv(source, base_manifest)
        source_doc, source_bin = DERIVER.parse_glb(normalized)
        accessor_index = 0
        source_uv = DERIVER.read_float_accessor(source_doc, source_bin, source_doc["accessors"][accessor_index], 2)
        oriented_bytes = struct.pack("<" + "f" * (len(source_uv) * 2),
                                     *(component for row in source_uv for component in (1.0 - row[0], 1.0 - row[1])))
        binding = dict(base_manifest["source"]["screen_binding"],
                       )
        orientation = copy.deepcopy(base_manifest["normalization"])
        orientation.update({"rotation_degrees": 180,
                            "oriented_uv_sha256": hashlib.sha256(oriented_bytes).hexdigest(),
                            "source_bin_sha256": hashlib.sha256(source_bin).hexdigest()})
        manifest = {
            "source": {"sha256": hashlib.sha256(normalized).hexdigest(), "size_bytes": len(normalized),
                       "neutral_screen_sha256": base_manifest["source"]["neutral_screen_sha256"],
                       "screen_binding": binding},
            "orientation": orientation,
            "output": {"max_bytes": 1_000_000},
        }
        output, facts = DERIVER.normalize_screen_uv(normalized, manifest)
        out_doc, out_bin = DERIVER.parse_glb(output)
        self.assertEqual(out_bin[:len(source_bin)], source_bin)
        self.assertEqual(len(out_doc["bufferViews"]), len(source_doc["bufferViews"]) + 1)
        self.assertEqual(out_doc["accessors"][accessor_index]["bufferView"], len(source_doc["bufferViews"]))
        self.assertEqual(DERIVER.read_float_accessor(out_doc, out_bin, out_doc["accessors"][accessor_index], 2),
                         [(1.0, 1.0), (0.0, 0.0)])
        self.assertEqual(facts["rotation"], "(u,v) -> (1-u,1-v)")
        self.assertEqual(facts["source_bin_sha256"], hashlib.sha256(source_bin).hexdigest())

    def test_orientation_derivative_refuses_accessor_retarget_and_source_drift(self):
        source, base_manifest = neutral_synthetic_glb()
        normalized, _ = DERIVER.normalize_screen_uv(source, base_manifest)
        doc, binary = DERIVER.parse_glb(normalized)
        uv = DERIVER.read_float_accessor(doc, binary, doc["accessors"][0], 2)
        oriented_bytes = struct.pack("<" + "f" * (len(uv) * 2),
                                     *(component for row in uv for component in (1.0 - row[0], 1.0 - row[1])))
        binding = dict(base_manifest["source"]["screen_binding"])
        orientation = copy.deepcopy(base_manifest["normalization"])
        orientation.update({"rotation_degrees": 180,
                            "oriented_uv_sha256": hashlib.sha256(oriented_bytes).hexdigest(),
                            "source_bin_sha256": hashlib.sha256(binary).hexdigest()})
        manifest = {"source": {"sha256": hashlib.sha256(normalized).hexdigest(), "size_bytes": len(normalized),
                               "neutral_screen_sha256": base_manifest["source"]["neutral_screen_sha256"],
                               "screen_binding": binding},
                    "orientation": orientation,
                    "output": {"max_bytes": 1_000_000}}
        damaged = copy.deepcopy(doc)
        damaged["meshes"][10]["primitives"][0]["attributes"]["TEXCOORD_0"] = 1
        bad_bytes = encode_glb(damaged, binary)
        bad_manifest = copy.deepcopy(manifest)
        bad_manifest["source"]["sha256"] = hashlib.sha256(bad_bytes).hexdigest()
        bad_manifest["source"]["size_bytes"] = len(bad_bytes)
        with self.assertRaisesRegex(ValueError, "screen UV/position accessor layout differs"):
            DERIVER.normalize_screen_uv(bad_bytes, bad_manifest)
        drift = bytearray(normalized)
        drift[-1] ^= 1
        with self.assertRaisesRegex(ValueError, "source GLB pin mismatch"):
            DERIVER.normalize_screen_uv(bytes(drift), manifest)

    @unittest.skipUnless(os.environ.get("DEVICE_STUDIO_E4_IPAD_SOURCE_TEST") == "1",
                         "set DEVICE_STUDIO_E4_IPAD_SOURCE_TEST=1 for the acquisition-backed qualification")
    def test_fixed_script_writes_only_neutral_derivative_and_receipt(self):
        source_path = RESEARCH / "ipad_mini_6.glb"
        self.assertTrue(source_path.is_file(), f"pinned acquisition input unavailable: {source_path}")
        source_bytes = source_path.read_bytes()
        self.assertEqual(len(source_bytes), 1_986_244)
        self.assertEqual(hashlib.sha256(source_bytes).hexdigest(),
                         "a462088b7bd84b7752802c109beee04110047c3b1a4f355624c863f790e93d5e")
        base_manifest = json.loads((HERE / "ipad-starter-manifest.json").read_text(encoding="utf-8"))
        with tempfile.TemporaryDirectory(prefix="ipad-starter-test-") as temp:
            root = Path(temp)
            shutil.copy2(HERE / "derive_ipad_starter.py", root / "derive_ipad_starter.py")
            shutil.copy2(HERE / "ipad-starter-manifest.json", root / "ipad-starter-manifest.json")
            (root / "inputs").mkdir()
            (root / "inputs/ipad_mini_6.glb").write_bytes(source_bytes)
            subprocess.run([sys.executable, str(root / "derive_ipad_starter.py")], check=True, timeout=15)
            output = root / "outputs/ipad-mini-neutral-screen.glb"
            receipt_path = root / "outputs/ipad-starter-derivation.json"
            self.assertTrue(output.is_file())
            receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
            self.assertEqual(receipt["source_sha256"], base_manifest["source"]["sha256"])
            self.assertEqual(receipt["derived_glb_sha256"], hashlib.sha256(output.read_bytes()).hexdigest())
            self.assertEqual(receipt["derived_glb_sha256"],
                             "8cd65349dcd38cb4f47fd897779db97dbbbf0fc7f6c81cd6c2ab05eaa6b47b5e",
                             "default mode must replay the original neutral-screen derivative unchanged")
            self.assertEqual(receipt["derived_screen_dimensions"], [512, 1024])
            self.assertNotIn(DERIVER.buffer_view_bytes(*DERIVER.parse_glb(source_bytes), 1),
                             DERIVER.parse_glb(output.read_bytes())[1])


def synthetic_glb():
    image_bytes = [b"IMG0DATA", b"PRIVATE_SCREEN_ART_00000", b"IMG2DATA"]
    uv_bytes = struct.pack("<4f", 0.0, 0.0, 1.0, 1.0)
    payloads = image_bytes + [uv_bytes]
    binary = bytearray()
    views = []
    for payload in payloads:
        while len(binary) % 4:
            binary.append(0)
        views.append({"buffer": 0, "byteOffset": len(binary), "byteLength": len(payload)})
        binary.extend(payload)
    nodes = [{} for _ in range(15)]
    nodes[14] = {"mesh": 10}
    meshes = [{} for _ in range(11)]
    meshes[10] = {"primitives": [{"attributes": {"TEXCOORD_0": 0}, "mode": 4, "material": 10}]}
    materials = [{} for _ in range(11)]
    materials[10] = {"pbrMetallicRoughness": {"baseColorTexture": {"index": 1}}}
    doc = {
        "asset": {"version": "2.0"}, "buffers": [{"byteLength": len(binary)}],
        "bufferViews": views, "accessors": [{"bufferView": 3, "componentType": 5126, "count": 2, "type": "VEC2"}],
        "images": [{"mimeType": "image/jpeg", "bufferView": i} for i in range(3)],
        "textures": [{"source": 0, "sampler": 0}, {"source": 1, "sampler": 1}],
        "samplers": [{}, {}], "materials": materials, "meshes": meshes, "nodes": nodes,
        "scenes": [{"nodes": [14]}], "scene": 0,
    }
    json_bytes = json.dumps(doc, separators=(",", ":")).encode("utf-8")
    json_bytes += b" " * ((-len(json_bytes)) % 4)
    bin_bytes = bytes(binary)
    data = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(bin_bytes))
    data += struct.pack("<II", len(json_bytes), DERIVER.JSON_CHUNK) + json_bytes
    data += struct.pack("<II", len(bin_bytes), DERIVER.BIN_CHUNK) + bin_bytes
    return data, doc, bin_bytes


def encode_glb(doc, binary):
    json_bytes = json.dumps(doc, separators=(",", ":")).encode("utf-8")
    json_bytes += b" " * ((-len(json_bytes)) % 4)
    data = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(binary))
    data += struct.pack("<II", len(json_bytes), DERIVER.JSON_CHUNK) + json_bytes
    data += struct.pack("<II", len(binary), DERIVER.BIN_CHUNK) + binary
    return data


def neutral_synthetic_glb(shared_uv=False):
    image0, image2 = b"IMG0DATA", b"IMG2DATA"
    neutral = DERIVER.neutral_png(512, 1024, [128, 128, 128, 255])
    uv = struct.pack("<4f", 0.0, 0.0, 1.0, 1.0)
    position = struct.pack("<6f", 0.0, 0.0, 0.0, 1.0, 1.0, 0.0)
    payloads = [image0, neutral, image2, uv, position]
    binary, views = bytearray(), []
    for index, payload in enumerate(payloads):
        while len(binary) % 4:
            binary.append(0)
        view = {"buffer": 0, "byteOffset": len(binary), "byteLength": len(payload)}
        if index in (3, 4):
            view["byteStride"] = 8 if index == 3 else 12
        views.append(view)
        binary.extend(payload)
    nodes, meshes, materials = [{} for _ in range(18)], [{} for _ in range(13)], [{} for _ in range(11)]
    nodes[14] = {"mesh": 10}
    attrs = {"TEXCOORD_0": 0, "POSITION": 1}
    meshes[10] = {"primitives": [{"attributes": attrs, "mode": 4, "material": 10}]}
    if shared_uv:
        meshes[11] = {"primitives": [{"attributes": {"TEXCOORD_0": 0}, "mode": 4, "material": 10}]}
        nodes[13] = {"mesh": 11}
    materials[10] = {"pbrMetallicRoughness": {"baseColorTexture": {"index": 1}}}
    doc = {
        "asset": {"version": "2.0"}, "buffers": [{"byteLength": len(binary)}], "bufferViews": views,
        "accessors": [
            {"bufferView": 3, "componentType": 5126, "count": 2, "type": "VEC2"},
            {"bufferView": 4, "componentType": 5126, "count": 2, "type": "VEC3"},
        ],
        "images": [{"mimeType": "image/jpeg", "bufferView": 0},
                   {"mimeType": "image/png", "bufferView": 1},
                   {"mimeType": "image/jpeg", "bufferView": 2}],
        "textures": [{"source": 0}, {"source": 1, "sampler": 1}], "materials": materials,
        "meshes": meshes, "nodes": nodes,
    }
    json_bytes = json.dumps(doc, separators=(",", ":")).encode("utf-8")
    json_bytes += b" " * ((-len(json_bytes)) % 4)
    bin_bytes = bytes(binary)
    glb = struct.pack("<4sII", b"glTF", 2, 12 + 8 + len(json_bytes) + 8 + len(bin_bytes))
    glb += struct.pack("<II", len(json_bytes), DERIVER.JSON_CHUNK) + json_bytes
    glb += struct.pack("<II", len(bin_bytes), DERIVER.BIN_CHUNK) + bin_bytes
    manifest = {
        "source": {"sha256": hashlib.sha256(glb).hexdigest(), "size_bytes": len(glb),
                   "neutral_screen_sha256": hashlib.sha256(neutral).hexdigest(),
                   "screen_binding": {"node_index": 14, "mesh_index": 10, "primitive_index": 0,
                                      "material_index": 10, "texture_index": 1, "image_index": 1}},
        "normalization": {"node_count": 18, "mesh_count": 13, "vertex_count": 2,
                          "uv_min": [0.0, 0.0], "uv_max": [1.0, 1.0],
                          "uv_to_position_xy_affine": [1, 0, 0, 0, 1, 0],
                          "max_uv_position_residual_m": 1e-8,
                          "normalized_uv_sha256": hashlib.sha256(uv).hexdigest()},
        "output": {"max_bytes": 1000000}, "rights_and_claims": {},
    }
    return glb, manifest


if __name__ == "__main__":
    unittest.main()
