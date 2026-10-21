import json
import pathlib
import struct
import sys
import unittest

sys.path.insert(0, str(pathlib.Path(__file__).parent))
import derive_screen_orientation as correction


class LaptopScreenOrientationTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.source = correction.SOURCE.read_bytes()
        cls.output, cls.source_bin = correction.derive(cls.source)
        cls.source_doc = json.loads(correction.chunks(cls.source)[0][1])
        cls.output_doc = json.loads(correction.chunks(cls.output)[0][1])
        cls.output_bin = correction.chunks(cls.output)[1][1]

    def test_only_screen_uv_view_changes_and_bin_is_preserved_as_prefix(self):
        self.assertTrue(self.output_bin.startswith(self.source_bin))
        self.assertEqual(len(self.output_bin) - len(self.source_bin), 760)
        self.assertEqual(self.source_doc["nodes"], self.output_doc["nodes"])
        self.assertEqual(self.source_doc["materials"], self.output_doc["materials"])
        self.assertEqual(self.source_doc["images"], self.output_doc["images"])
        self.assertEqual(self.source_doc["textures"], self.output_doc["textures"])
        self.assertEqual(self.source_doc["meshes"][:3], self.output_doc["meshes"][:3])
        source_primitive = self.source_doc["meshes"][3]["primitives"][0]
        output_primitive = self.output_doc["meshes"][3]["primitives"][0]
        self.assertEqual(source_primitive["indices"], output_primitive["indices"])
        for attribute in ("POSITION", "NORMAL"):
            self.assertEqual(source_primitive["attributes"][attribute], output_primitive["attributes"][attribute])
        for index in range(len(self.source_doc["meshes"])):
            if index != 3:
                self.assertEqual(self.source_doc["meshes"][index], self.output_doc["meshes"][index])

    def test_screen_uv_has_exact_180_degree_mapping_and_private_accessor(self):
        source_accessor_index = self.source_doc["meshes"][3]["primitives"][0]["attributes"]["TEXCOORD_0"]
        output_accessor_index = self.output_doc["meshes"][3]["primitives"][0]["attributes"]["TEXCOORD_0"]
        self.assertEqual((source_accessor_index, output_accessor_index), (96, 97))
        source_accessor = self.source_doc["accessors"][source_accessor_index]
        output_accessor = self.output_doc["accessors"][output_accessor_index]
        source_view = self.source_doc["bufferViews"][source_accessor["bufferView"]]
        output_view = self.output_doc["bufferViews"][output_accessor["bufferView"]]
        self.assertEqual(output_view["byteOffset"], len(self.source_bin))
        source_values = struct.unpack_from("<190f", self.source_bin, source_view.get("byteOffset", 0))
        output_values = struct.unpack_from("<190f", self.output_bin, output_view["byteOffset"])
        expected = tuple(struct.unpack("<f", struct.pack("<f", 1.0 - value))[0] for value in source_values)
        self.assertEqual(output_values, expected)
        self.assertEqual(output_accessor["min"], [0.0, 0.0])
        self.assertEqual(output_accessor["max"], [1.0, 1.0])

    def test_derivation_refuses_unpinned_base_bytes(self):
        with self.assertRaisesRegex(ValueError, "candidate hash changed"):
            correction.derive(self.source + b"x")


if __name__ == "__main__":
    unittest.main()
