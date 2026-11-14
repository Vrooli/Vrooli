import importlib.util
import pathlib
import unittest


SCRIPT = pathlib.Path(__file__).with_name("derive_candidates.py")
spec = importlib.util.spec_from_file_location("derive_candidates", SCRIPT)
derive = importlib.util.module_from_spec(spec)
spec.loader.exec_module(derive)


class CandidateDerivationTests(unittest.TestCase):
    def test_frame_variants_preserve_source_binary_and_only_replace_screen_material(self):
        source = derive.FRAME_SOURCE.read_bytes()
        source_doc, source_bin = derive.glb_parts(source)
        table = derive.digital_frame(source, False)
        wall = derive.digital_frame(source, True)
        table_doc, table_bin = derive.glb_parts(table)
        wall_doc, wall_bin = derive.glb_parts(wall)

        self.assertTrue(table_bin.startswith(source_bin))
        self.assertTrue(wall_bin.startswith(source_bin))
        self.assertEqual(table_doc["nodes"][8]["children"], source_doc["nodes"][8]["children"])
        for index, mesh in enumerate(source_doc["meshes"]):
            if index == 7:
                candidate_primitive = table_doc["meshes"][index]["primitives"][0]
                source_primitive = mesh["primitives"][0]
                self.assertEqual(candidate_primitive["attributes"]["POSITION"], source_primitive["attributes"]["POSITION"])
                self.assertEqual(candidate_primitive["attributes"]["NORMAL"], source_primitive["attributes"]["NORMAL"])
                self.assertEqual(candidate_primitive["indices"], source_primitive["indices"])
            else:
                self.assertEqual(table_doc["meshes"][index], mesh)
        self.assertEqual(table_doc["meshes"][7]["primitives"][0]["material"], len(source_doc["materials"]))
        self.assertEqual(table_doc["materials"][-1]["name"], "Neutral project display surface")
        self.assertNotEqual(table_doc["materials"][-1], source_doc["materials"][4])
        source_uv_index = source_doc["meshes"][7]["primitives"][0]["attributes"]["TEXCOORD_0"]
        table_primitive = table_doc["meshes"][7]["primitives"][0]
        self.assertNotEqual(table_primitive["attributes"]["TEXCOORD_0"], source_uv_index)
        unit_uv = table_doc["accessors"][table_primitive["attributes"]["TEXCOORD_0"]]
        self.assertEqual((unit_uv["min"], unit_uv["max"]), ([0, 0], [1, 1]))
        wall_nodes = {node["name"]: node for node in wall_doc["nodes"]}
        self.assertNotIn("Easel limiting cord", wall_nodes)
        self.assertNotIn("Hinged rear easel strut", wall_nodes)
        self.assertIn("WallMountCleat", wall_nodes)
        self.assertEqual(wall_nodes["WallMountCleat"]["mesh"], len(source_doc["meshes"]))
        self.assertEqual(wall_nodes["Unprinted frame insert"]["mesh"], 7)
        reachable = set()
        pending = list(wall_doc["scenes"][0]["nodes"])
        while pending:
            index = pending.pop()
            self.assertNotIn(index, reachable)
            reachable.add(index)
            pending.extend(wall_doc["nodes"][index].get("children", []))
        self.assertEqual(reachable, set(range(len(wall_doc["nodes"]))))
        for index, mesh in enumerate(source_doc["meshes"]):
            if index != 7:
                self.assertEqual(wall_doc["meshes"][index], mesh)

    def test_authored_screens_have_exact_unit_uv_and_declared_envelopes(self):
        for data, expected in ((derive.smart_display(), (0.180, 0.125, 0.090)),
                               (derive.vehicle_console(), (0.380, 0.250, 0.120))):
            doc, _ = derive.glb_parts(data)
            root_mesh = doc["meshes"][0]["primitives"][0]
            position = doc["accessors"][root_mesh["attributes"]["POSITION"]]
            self.assertEqual(position["min"], [-expected[0] / 2, -expected[1] / 2, -expected[2] / 2])
            self.assertEqual(position["max"], [expected[0] / 2, expected[1] / 2, expected[2] / 2])
            active = next(node for node in doc["nodes"] if node["name"] in ("MainDisplay", "CenterDisplay"))
            primitive = doc["meshes"][active["mesh"]]["primitives"][0]
            uv = doc["accessors"][primitive["attributes"]["TEXCOORD_0"]]
            self.assertEqual((uv["min"], uv["max"]), ([0, 0], [1, 1]))


if __name__ == "__main__":
    unittest.main()
