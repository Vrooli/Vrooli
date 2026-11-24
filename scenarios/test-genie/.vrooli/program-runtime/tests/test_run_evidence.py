"""Run-evidence projection validation; no Test Genie service is contacted."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace
import unittest

ROOT = Path(__file__).parents[1]
sys.path.insert(0, str(ROOT.parents[2] / "program-runtime/kernel/host"))
from program_helper import ProgramHelper


class RunEvidenceTests(unittest.TestCase):
    def invoke(self, **changes):
        calls = []
        def binding(name):
            def call(**kwargs):
                calls.append((name, kwargs))
                if name.endswith("/artifact"):
                    rows = [{"id": kwargs.get("artifact_id", ""), "kind": "findings", "sizeBytes": 12,
                             "accessCapability": "ARTIFACT_ACCESS_CAPABILITY_STREAM"}]
                elif name.endswith("/artifacts"):
                    rows = [{"id": "artifact-fixture", "kind": "findings", "sizeBytes": 12,
                             "accessCapability": "ARTIFACT_ACCESS_CAPABILITY_STREAM"}]
                else:
                    rows = [{"owner": name, "limit": 100, **kwargs}]
                return SimpleNamespace(head=lambda limit: rows[:limit], meta=lambda: {"digest": "sha256:fixture", "legacyDiscovered": False}, count=lambda: len(rows))
            return call
        runs = SimpleNamespace(**{name: binding("test-genie/runs/" + name)
                                  for name in ("show", "artifacts", "findings", "artifact")})
        inputs = {"view": "artifacts", "scenario": "demo", "run_id": "run-1", **changes}
        helper = ProgramHelper()
        scope = {"inputs": inputs, "program": helper, "test_genie": SimpleNamespace(runs=runs)}
        helper._bind(scope)
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile((ROOT / "run-evidence.py").read_text(), "run-evidence", "exec"), scope)
        return json.loads(stream.getvalue()), calls

    def test_catalog_forwards_exact_scenario_and_run(self):
        result, calls = self.invoke(kinds="findings,logs", phase="unit")
        self.assertEqual(calls, [("test-genie/runs/artifacts", {
            "rows": "artifacts", "scenario": "demo", "run_id": "run-1", "kinds": "findings,logs", "phase": "unit"})])
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["signals"]["result"][0]["id"], "artifact-fixture")
        self.assertEqual(result["signals"]["metadata"]["count"], 1)

    def test_views_call_only_their_owner_binding(self):
        for view, expected in (("show", "show"), ("findings", "findings"), ("artifact", "artifact")):
            changes = {"view": view}
            if view == "artifact":
                changes["artifact_id"] = "opaque-id"
            result, calls = self.invoke(**changes)
            self.assertEqual(result["status"], "ok")
            self.assertEqual(calls[0][0], "test-genie/runs/" + expected)
            self.assertEqual(len(calls), 1)

    def test_invalid_or_missing_artifact_id_never_calls_owner(self):
        for changes in ({"view": "artifact"}, {"scenario": "../outside"}, {"run_id": "bad id"}, {"view": "unknown"}):
            result, calls = self.invoke(**changes)
            self.assertEqual(result["status"], "failed")
            self.assertEqual(calls, [])

    def test_contract(self):
        import jsonschema
        schema = json.loads((ROOT.parents[2] / "program-runtime/schemas/program-contract.schema.json").read_text())
        jsonschema.validate(json.loads((ROOT / "run-evidence.json").read_text()), schema)


if __name__ == "__main__":
    unittest.main()
