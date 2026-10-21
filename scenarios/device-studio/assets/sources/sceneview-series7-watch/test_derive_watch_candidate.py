import importlib.util
import json
import struct
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("derive_watch_candidate.py")
SPEC = importlib.util.spec_from_file_location("derive_watch_candidate", SCRIPT)
derive_watch_candidate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(derive_watch_candidate)


class WatchDerivativeTests(unittest.TestCase):
    def test_derivative_is_reproducible_and_pins_rights_and_unknown_variant(self):
        first, manifest = derive_watch_candidate.derive()
        second, replay_manifest = derive_watch_candidate.derive()
        self.assertEqual(first, second)
        self.assertEqual(manifest, replay_manifest)
        self.assertEqual(derive_watch_candidate.sha(first), manifest["output"]["sha256"])
        self.assertEqual(manifest["geometry"]["variant"], "unknown")
        self.assertIn("CC BY 4.0", manifest["source"]["license"])
        self.assertIn("SDC PERFORMANCE", manifest["source"]["attribution"])
        self.assertIn("bharathsrk13", manifest["source"]["attribution"])
        self.assertIn("original branded image is excluded", manifest["rights_and_claims"]["rights_limit"])
        self.assertIn("not a round-watch family", manifest["rights_and_claims"]["candidate_scope"])

    def test_only_display_uv_image_and_material_change(self):
        source = derive_watch_candidate.SOURCE.read_bytes()
        output, manifest = derive_watch_candidate.derive()
        source_doc, source_bin = derive_watch_candidate.load_glb(source)
        output_doc, output_bin = derive_watch_candidate.load_glb(output)
        expected = json.loads(json.dumps(source_doc))
        expected["buffers"][0]["byteLength"] = output_doc["buffers"][0]["byteLength"]
        expected["bufferViews"].append(output_doc["bufferViews"][-1])
        expected["accessors"].append(output_doc["accessors"][-1])
        expected["meshes"][14]["primitives"][0]["attributes"]["TEXCOORD_0"] = 46
        expected["images"][0]["mimeType"] = "image/png"
        expected["materials"][11] = output_doc["materials"][11]
        self.assertEqual(output_doc, expected)
        source_start = source_doc["bufferViews"][0]["byteOffset"]
        source_size = source_doc["bufferViews"][0]["byteLength"]
        source_non_image = source_bin[:source_start] + source_bin[source_start + source_size :]
        output_non_image = output_bin[:source_start] + output_bin[source_start + source_size : len(source_bin)]
        self.assertEqual(source_non_image, output_non_image)
        self.assertEqual(len(source_bin), manifest["derivation"]["source_bin_bytes"])
        self.assertEqual(output_doc["accessors"][output_doc["meshes"][14]["primitives"][0]["indices"]]["count"], 90)
        self.assertEqual(output_doc["nodes"], source_doc["nodes"])
        self.assertEqual(output_doc["meshes"][14]["primitives"][0]["indices"], source_doc["meshes"][14]["primitives"][0]["indices"])

    def test_exact_source_support_and_display_center_are_reproducible(self):
        source_doc, source_bin = derive_watch_candidate.load_glb(derive_watch_candidate.SOURCE.read_bytes())
        facts = derive_watch_candidate.exact_position_measurements(source_doc, source_bin)
        self.assertEqual(derive_watch_candidate.sha(derive_watch_candidate.SOURCE.read_bytes()), derive_watch_candidate.SOURCE_SHA)
        self.assertEqual(facts["transformed_position_vertex_count"], 161468)
        self.assertEqual(facts["minimum_y_m"], 0.0)
        self.assertEqual(facts["display_center_vertex_count"], 86)
        for actual, expected in zip(facts["display_center_m"], [6.9653163122769475e-06, 0.03194426510567318, 0.015982393424878862]):
            self.assertAlmostEqual(actual, expected, delta=1e-12)

    def test_referenced_screen_uvs_are_unit_rectangle_and_topology_is_unchanged(self):
        source = derive_watch_candidate.SOURCE.read_bytes()
        output, manifest = derive_watch_candidate.derive()
        source_doc, source_bin = derive_watch_candidate.load_glb(source)
        output_doc, output_bin = derive_watch_candidate.load_glb(output)
        source_primitive = source_doc["meshes"][14]["primitives"][0]
        output_primitive = output_doc["meshes"][14]["primitives"][0]
        self.assertEqual(source_primitive["indices"], output_primitive["indices"])
        self.assertEqual(source_primitive["attributes"]["POSITION"], output_primitive["attributes"]["POSITION"])
        self.assertEqual(source_primitive["attributes"]["NORMAL"], output_primitive["attributes"]["NORMAL"])
        ids = derive_watch_candidate.index_values(source_doc, source_bin, source_primitive["indices"])
        uv_index = output_primitive["attributes"]["TEXCOORD_0"]
        uv = derive_watch_candidate.accessor_rows(output_doc, output_bin, uv_index, "<ff", 8)
        self.assertEqual(len(ids), 90)
        self.assertEqual(len(set(ids)), 86)
        self.assertEqual([min(uv[i][axis] for i in ids) for axis in range(2)], [0.0, 0.0])
        self.assertEqual([max(uv[i][axis] for i in ids) for axis in range(2)], [1.0, 1.0])
        self.assertEqual(manifest["display_binding"]["triangle_count"], 30)

    def test_changed_source_fails_closed(self):
        original = derive_watch_candidate.SOURCE
        try:
            derive_watch_candidate.SOURCE = SCRIPT
            with self.assertRaisesRegex(ValueError, "source GLB or creator record hash changed"):
                derive_watch_candidate.derive()
        finally:
            derive_watch_candidate.SOURCE = original


if __name__ == "__main__":
    unittest.main()
