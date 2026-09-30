"""Contract and behavior tests for the real mutation-pilot program helper."""
import contextlib
import io
import json
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace as NS

ROOT = Path(__file__).resolve().parents[1]
REPO = Path(__file__).resolve().parents[5]
sys.dont_write_bytecode = True
sys.path.insert(0, str(REPO / "scenarios/program-runtime/kernel"))
from host.program_helper import ProgramHelper  # noqa: E402

SOURCE = (ROOT / "mutation-pilot.py").read_text()


class MutationPilotTests(unittest.TestCase):
    def invoke(self, values, payload=None, error=None):
        calls = []

        def pilot(**kwargs):
            calls.append(kwargs)
            if error is not None:
                raise error
            return NS(raw=lambda: payload or {
                "runId": "mutation-1", "workspacePath": "/cache/mutation/mutation-1",
                "summary": {"generated": 2, "killed": 1, "survived": 1, "killRate": 0.5},
                "receipts": [{"id": "mutant-1", "disposition": "killed"}], "limitations": [],
            })

        helper = ProgramHelper()
        env = {"inputs": values, "program": helper, "unit_health": NS(mutation=NS(pilot=pilot))}
        helper._bind(env)
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile(SOURCE, "mutation-pilot", "exec"), env)
        lines = [line for line in stream.getvalue().splitlines() if line.strip()]
        self.assertEqual(len(lines), 1)
        return json.loads(lines[0]), calls

    def test_invalid_operator_fails_before_owner_call(self):
        result, calls = self.invoke({"scenario": "unit-health", "seed": "s", "operators": ["invalid"]})
        self.assertEqual(result["status"], "failed")
        self.assertEqual(result["errors"][0]["class"], "invalid_input")
        self.assertEqual(calls, [])

    def test_success_uses_one_governed_binding_and_bounds_receipts(self):
        result, calls = self.invoke({"scenario": "unit-health", "seed": "s", "max_mutants": 2})
        self.assertEqual(result["status"], "ok")
        self.assertEqual(calls, [{"scenario": "unit-health", "workspace": "api", "package": "./internal/testquality/...", "operators": ["boundary", "negate-condition", "return-constant"], "max_mutants": 2, "seed": "s", "rows": "receipts"}])
        self.assertEqual(result["signals"]["summary"]["killed"], 1)
        self.assertEqual(result["signals"]["receipts"][0]["disposition"], "killed")
        self.assertEqual(result["evidence"][0]["run_id"], "mutation-1")

    def test_owner_unreachable_is_unavailable(self):
        class Unreachable(Exception):
            pass

        result, _ = self.invoke({"scenario": "unit-health", "seed": "s"}, error=Unreachable("connection refused"))
        self.assertIn(result["status"], ("partial", "unavailable"))

    def test_contract_declares_budget_fixture_and_binding(self):
        import jsonschema
        schema = json.loads((REPO / "scenarios/program-runtime/schemas/program-contract.schema.json").read_text())
        contract = json.loads((ROOT / "mutation-pilot.json").read_text())
        jsonschema.validate(contract, schema)
        self.assertEqual(contract["budget"]["wall_ms"], 90000)
        self.assertEqual(contract["budget"]["delegated_runs"], 0)
        self.assertEqual(contract["bindings"], [{"id": "unit-health/mutation/pilot", "effect": "read"}])


if __name__ == "__main__":
    unittest.main()
