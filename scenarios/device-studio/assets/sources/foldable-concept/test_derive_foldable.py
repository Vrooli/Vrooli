import json
import hashlib
import pathlib
import subprocess
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[3]
SCRIPT = ROOT / "assets/sources/foldable-concept/derive_foldable.py"
GLB = ROOT / "api/internal/physicalruntime/assets/foldable/foldable-concept.glb"
INDEX = ROOT / "api/internal/physicalruntime/assets/asset-index.json"
PACKAGE_MANIFEST = ROOT / "api/internal/physicalruntime/assets/foldable/asset-manifest.json"


class FoldableDerivationTest(unittest.TestCase):
    def test_replay_has_two_rigid_halves_and_three_separate_neutral_screens(self):
        subprocess.run(["python3", str(SCRIPT)], check=True)
        raw = GLB.read_bytes()
        self.assertEqual(hashlib.sha256(raw).hexdigest(), "90e1e5195ee88018e37b98d8d08af4d1407d8161e6216cf3fb699921d77ce4ac")
        package_manifest = PACKAGE_MANIFEST.read_bytes()
        index = json.loads(INDEX.read_text())
        catalog_entry = next(entry for entry in index["entries"] if entry["catalog_key"] == "vrooli-generic-book-fold-concept-glb-v2")
        package_entry = next(entry for entry in index["package_manifests"] if entry["path"] == "foldable/asset-manifest.json")
        self.assertEqual((catalog_entry["sha256"], catalog_entry["size_bytes"]), (hashlib.sha256(raw).hexdigest(), len(raw)))
        self.assertEqual(package_entry["sha256"], hashlib.sha256(package_manifest).hexdigest())
        self.assertEqual(raw[:4], b"glTF")
        json_length = int.from_bytes(raw[12:16], "little")
        doc = json.loads(raw[20:20 + json_length])
        nodes = {node["name"]: node for node in doc["nodes"]}
        self.assertEqual(nodes["MainHingePivot"]["children"], [4])
        self.assertEqual(nodes["RightHousing"]["children"], [5, 6])
        self.assertEqual(nodes["LeftHousing"]["children"], [2])
        self.assertEqual(doc["nodes"][0]["children"], [1, 3, 7])
        housing = doc["meshes"][nodes["LeftHousing"]["mesh"]]["primitives"][0]
        housing_positions = doc["accessors"][housing["attributes"]["POSITION"]]
        housing_indices = doc["accessors"][housing["indices"]]
        self.assertEqual((housing_positions["count"], housing_indices["count"]), (114, 672))
        hinge = doc["meshes"][nodes["HingeBarrel"]["mesh"]]["primitives"][0]
        hinge_positions = doc["accessors"][hinge["attributes"]["POSITION"]]
        self.assertEqual(hinge_positions["count"], 7 * 12 * 2)
        self.assertEqual(sum(doc["accessors"][mesh["primitives"][0]["indices"]]["count"] for mesh in doc["meshes"]) // 3, 622)
        for name, material in (("LeftInnerDisplay", 1), ("RightInnerDisplay", 2), ("OuterDisplay", 3)):
            mesh = doc["meshes"][nodes[name]["mesh"]]
            self.assertEqual(len(mesh["primitives"]), 1)
            self.assertEqual(mesh["primitives"][0]["material"], material)
        self.assertEqual(len(doc["meshes"]), 6)
        for material_index in (1, 2, 3):
            material = doc["materials"][material_index]
            self.assertEqual(material["name"].split()[-1], "glass")
            self.assertLess(material["pbrMetallicRoughness"]["roughnessFactor"], 0.3)
            self.assertGreater(max(material["emissiveFactor"]), 0)


if __name__ == "__main__":
    unittest.main()
