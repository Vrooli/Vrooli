"""Iteration selection and admission contract; no real suites started."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace
import unittest

ROOT = Path(__file__).parents[1]

class IterateTests(unittest.TestCase):
    def invoke(self, receipt_extra=None, **changes):
        calls = []
        def create(**kwargs):
            calls.append(kwargs)
            return SimpleNamespace(head=lambda n: [], meta=lambda: {"receipt": {"receiptId": "receipt-1", "state": "RECEIPT_STATE_QUEUED", **(receipt_extra or {})}})
        values = {"scenario": "demo", "request_id": "change-1", **changes}
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "iterate.py").read_text(), "iterate", "exec"), {
                "inputs": values, "test_genie": SimpleNamespace(validation=SimpleNamespace(create=create))})
        return json.loads(stream.getvalue()), calls

    def test_required_phases_survive_small_budget_and_single_admission(self):
        result, calls = self.invoke(phases=["programs", "unit", "unit"], budget_seconds=30)
        self.assertEqual(len(calls), 1)
        intent = calls[0]["intent"]
        self.assertEqual(intent["phases"], ["programs", "unit"])
        self.assertEqual(intent["deadline_policy"]["execution_budget"], "30s")
        self.assertEqual(result["signals"]["outcome"], "unknown")
        self.assertIn("validation wait receipt-1", result["signals"]["wait_command"])

    def test_no_scope_selects_owner_quick_planner_not_comprehensive(self):
        result, calls = self.invoke()
        self.assertEqual(calls[0]["intent"]["required_strength"], "VALIDATION_STRENGTH_TARGETED")
        self.assertEqual(calls[0]["intent"]["phases"], [])
        self.assertEqual(result["signals"]["selection"]["mode"], "owner_budget_profile")

    def test_plan_only_does_not_start_work_and_replay_keeps_exact_intent(self):
        result, calls = self.invoke(plan_only=True, phases=["unit"])
        self.assertEqual(calls, [])
        again, _ = self.invoke(plan_only=True, phases=["unit"])
        self.assertEqual(result["signals"]["intent"], again["signals"]["intent"])

    def test_bad_input_never_admits(self):
        for values in ({"scenario": "../outside"}, {"phases": ["unit;echo"]}, {"budget_seconds": True}, {"request_id": ""}, {"dependency_inputs": [{"root": "packages"}]}):
            result, calls = self.invoke(**values)
            self.assertEqual(result["status"], "failed")
            self.assertEqual(calls, [])

    def test_contract(self):
        import jsonschema
        schema = json.loads((ROOT.parents[2] / "program-runtime/schemas/program-contract.schema.json").read_text())
        jsonschema.validate(json.loads((ROOT / "iterate.json").read_text()), schema)

    def test_large_owner_inventory_does_not_hide_receipt_or_wait_command(self):
        result, _ = self.invoke(receipt_extra={"admittedIdentity": {
            "identity": "ci:v1:fixture", "roots": [{"files": ["x" * 100000]}]}})
        self.assertLess(len(json.dumps(result)), 4096)
        self.assertEqual(result["signals"]["receipt"]["identity"], "ci:v1:fixture")
        self.assertIn("receipt-1", result["signals"]["wait_command"])

if __name__ == "__main__":
    unittest.main()
