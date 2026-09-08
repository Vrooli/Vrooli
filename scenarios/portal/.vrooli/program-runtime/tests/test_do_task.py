import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]

class Handle:
    def __init__(self, rows, meta=None): self.rows, self._meta = rows, (meta or {})
    def head(self, n): return self.rows[:n]
    def count(self): return len(self.rows)
    def meta(self): return self._meta

class ProgramTest(unittest.TestCase):
    def run_program(self, inputs, browser, desktop):
        scope = {"inputs": inputs, "browser_automation_studio": browser, "device_control": desktop}
        with contextlib.redirect_stdout(io.StringIO()):
            exec(compile((ROOT / "do-task.py").read_text(), "do-task.py", "exec"), scope)
        return scope["envelope"]

    def valid(self):
        return {"workflow_id": "wf", "workflow_version": 2, "device_id": "office", "context_key": "office:v1", "flow_id": "flow", "flow_version": 3, "actor": "portal"}

    def test_browser_receipt_precedes_desktop_and_each_owner_receipt_is_preserved(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append(("browser", kw)) or Handle([{"status": "completed", "executionId": "b-1"}]))))
        desktop = NS(flow=NS(replay=lambda **kw: (calls.append(("desktop", kw)) or Handle([{"status": "passed", "runId": "d-1"}]))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("ok", result["status"])
        self.assertEqual(["browser", "desktop"], [name for name, _ in calls])
        self.assertEqual(["browser:b-1", "desktop:d-1"], result["evidence"])
        self.assertEqual("wf", calls[0][1]["workflow_id"])
        self.assertEqual("office", calls[1][1]["device_id"])

    def test_browser_failure_prevents_desktop_side_effect(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([{"status": "failed"}]))))
        desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["browser"], calls)

    def test_invalid_input_is_rejected_before_bindings(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: calls.append("browser")))
        desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
        result = self.run_program({}, browser, desktop)
        self.assertEqual("invalid_input", result["errors"][0]["class"])
        self.assertEqual([], calls)

    def test_browser_scalar_metadata_is_accepted_before_desktop(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([], {"status": "EXECUTION_STATUS_COMPLETED", "executionId": "b-meta"}))))
        desktop = NS(flow=NS(replay=lambda **kw: (calls.append("desktop") or Handle([{"status": "passed", "runId": "d-meta"}]))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("ok", result["status"])
        self.assertEqual(["browser", "desktop"], calls)
        self.assertEqual(["browser:b-meta", "desktop:d-meta"], result["evidence"])

    def test_missing_saved_flow_is_classified_as_desktop_failure(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([], {"status": "completed", "executionId": "b-2"})))
        desktop = NS(flow=NS(replay=lambda **kw: (_ for _ in ()).throw(RuntimeError("saved flow not found"))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("desktop_failed", result["errors"][0]["class"])
        self.assertIn("saved flow not found", result["errors"][0]["detail"])

    def test_desktop_scalar_metadata_is_accepted(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([], {"status": "completed", "executionId": "b-3"})))
        desktop = NS(flow=NS(replay=lambda **kw: Handle([], {"disposition": "passed", "runId": "d-meta"})))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("ok", result["status"])
        self.assertEqual(["browser:b-3", "desktop:d-meta"], result["evidence"])

    def test_report_is_a_json_envelope(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([], {"status": "completed", "executionId": "b-json"})))
        desktop = NS(flow=NS(replay=lambda **kw: Handle([], {"disposition": "passed", "runId": "d-json"})))
        stdout = io.StringIO()
        scope = {"inputs": self.valid(), "browser_automation_studio": browser, "device_control": desktop}
        with contextlib.redirect_stdout(stdout):
            exec(compile((ROOT / "do-task.py").read_text(), "do-task.py", "exec"), scope)
        self.assertEqual(scope["envelope"], json.loads(stdout.getvalue()))
