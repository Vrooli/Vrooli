#!/usr/bin/env python3
"""Keep both target runners' per-cell assertion contracts tied to the matrix."""
import ast
import json
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[2]
MATRIX = json.loads((ROOT / "api/certification/matrix.json").read_text())
PROGRAMS = (
    ("cloud-qemu-qualification.py", "qemu"),
    ("cloud-real-vps-qualification.py", "real_vps"),
)


def assignments(source):
    tree = ast.parse(source)
    result = {}
    for node in tree.body:
        if not (isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name)):
            continue
        try:
            result[node.targets[0].id] = ast.literal_eval(node.value)
        except (ValueError, TypeError):
            continue
    return result


def load_driver(source, globals_for_driver):
    tree = ast.parse(source)
    function = next(
        node for node in tree.body
        if isinstance(node, ast.FunctionDef) and node.name == "assertion_driver"
    )
    namespace = dict(globals_for_driver)
    exec(compile(ast.Module(body=[function], type_ignores=[]), "driver", "exec"), namespace)
    return namespace["assertion_driver"]


class CertificationDriverContractTests(unittest.TestCase):
    def test_case_assertions_match_frozen_matrix(self):
        expected = {
            row["id"]: {"trigger": row["trigger"], "assertion": row["assertion"]}
            for row in MATRIX["cases"]
            if row["required"]
        }
        for filename, _ground in PROGRAMS:
            with self.subTest(program=filename):
                source = (ROOT / ".vrooli/program-runtime" / filename).read_text()
                self.assertEqual(assignments(source)["CASE_ASSERTIONS"], expected)

    def test_every_ground_lane_case_has_explicit_driver_identity(self):
        owners = {
            "package": "scenario-to-cloud",
            "api": "scenario-to-cloud",
            "workflow": "scenario-to-cloud",
            "relay": "Vrooli Bridge",
            "qemu": "scenario-to-cloud QEMU lane",
            "real_vps": "scenario-to-cloud VPS lane",
            "bas_ui": "Browser Automation Studio",
            "independent_review": "independent reviewer",
            "soak": "scenario-to-cloud soak owner",
        }
        for filename, ground in PROGRAMS:
            with self.subTest(program=filename):
                source = (ROOT / ".vrooli/program-runtime" / filename).read_text()
                values = assignments(source)
                self.assertEqual(values["LANE_EVIDENCE_OWNERS"], owners)
                lane_map_name = "QEMU_LANE_CASES" if ground == "qemu" else "REAL_VPS_LANE_CASES"
                lanes = values[lane_map_name]
                cells = {
                    (case_id, lane, ground)
                    for lane, case_ids in lanes.items()
                    for case_id in case_ids
                }
                expected_cells = {
                    (pair["case_id"], pair["lane"], ground)
                    for pair in MATRIX["ground_applicability"]
                    if ground in pair["grounds"]
                }
                self.assertEqual(cells, expected_cells)
                self.assertEqual(len(cells), 266)
                self.assertIn('"status":"owner_assertion_required"', source)
                self.assertIn('"target_receipt_refs"', source)
                self.assertIn('"?fence="', source)
                driver = load_driver(source, {
                    "CASE_ASSERTIONS": values["CASE_ASSERTIONS"],
                    "LANE_EVIDENCE_OWNERS": owners,
                    "TARGET_GROUND": ground,
                })
                for case_id, lane, target_ground in cells:
                    result = driver(case_id, lane)
                    self.assertEqual(result["driver_ref"], f"certification-assertion:v1:{case_id}:{lane}:{target_ground}")
                    self.assertEqual(result["owner"], owners[lane])
                    self.assertEqual(result["expected_assertion"], values["CASE_ASSERTIONS"][case_id]["assertion"])
                    self.assertEqual(result["status"], "owner_assertion_required")


if __name__ == "__main__":
    unittest.main()
