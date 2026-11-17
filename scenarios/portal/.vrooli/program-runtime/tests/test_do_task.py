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

class RuntimeProgram:
    def __init__(self, inputs):
        self._inputs = inputs
        self._envelope = None
    def inputs(self): return self._inputs
    def envelope(self, _name, _version, phase, inputs):
        self._envelope = {"status": "failed", "phase": phase, "inputs": inputs, "signals": {}, "errors": [], "evidence": []}
        return self._envelope
    def fail(self, status, klass, detail, phase, detail_limit=160):
        self._envelope["status"] = status
        self._envelope["phase"] = phase
        self._envelope["errors"].append({"class": klass, "detail": str(detail)[:detail_limit], "where": phase})
        return "report"
    def classify(self, _exc): return "failed", "remote_error"

class ProgramTest(unittest.TestCase):
    def test_contract_declares_the_read_receipt_owner(self):
        contract = json.loads((ROOT / "do-task.json").read_text())
        bindings = {item["id"]: item["effect"] for item in contract["bindings"]}
        self.assertEqual("write", bindings["browser-automation-studio/workflows/execute"])
        self.assertEqual("read", bindings["browser-automation-studio/executions/get"])
        self.assertEqual("read", bindings["browser-automation-studio/executions/wait-for-execution"])
        self.assertEqual("write", bindings["browser-automation-studio/executions/stop"])

    def run_program(self, inputs, browser, desktop):
        if not hasattr(browser, "executions"):
            issued = {}
            execute = browser.workflows.execute
            def capture_execution(**kwargs):
                handle = execute(**kwargs)
                rows = handle.head(1)
                if rows:
                    row = rows[0]
                    execution_id = row.get("executionId", row.get("execution_id"))
                    if execution_id:
                        issued[execution_id] = {
                            "executionId": execution_id,
                            "workflowId": kwargs["workflow_id"],
                            "workflowVersion": kwargs["workflow_version"],
                            "status": "EXECUTION_STATUS_COMPLETED",
                        }
                return handle
            browser.workflows.execute = capture_execution
            browser.executions = NS(
                wait_for_execution=lambda **kwargs: Handle([{"status": "completed"}]),
                get=lambda **kwargs: Handle([issued[kwargs["execution_id"]]] if kwargs["execution_id"] in issued else []),
                stop=lambda **kwargs: Handle([{"status": "stopped"}]),
            )
        elif not hasattr(browser.executions, "wait_for_execution"):
            browser.executions.wait_for_execution = lambda **kwargs: Handle([{"status": "completed"}])
        if not hasattr(browser.executions, "stop"):
            browser.executions.stop = lambda **kwargs: Handle([{"status": "stopped"}])
        scope = {"inputs": inputs, "browser_automation_studio": browser, "device_control": desktop, "program": RuntimeProgram(inputs)}
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

    def test_execution_admission_is_async_and_wait_is_single_exact_id_operation(self):
        calls = []
        browser = NS(
            workflows=NS(execute=lambda **kw: calls.append(("execute", kw)) or Handle([{"executionId": "b-once"}])),
            executions=NS(
                wait_for_execution=lambda **kw: calls.append(("wait", kw)) or Handle([{"status": "completed"}]),
                get=lambda **kw: calls.append(("get", kw)) or Handle([{"executionId": "b-once", "workflowId": "wf", "workflowVersion": 2, "status": "completed"}]),
            ),
        )
        desktop = NS(flow=NS(replay=lambda **kw: calls.append(("desktop", kw)) or Handle([{"status": "passed"}])))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("ok", result["status"])
        self.assertFalse(calls[0][1]["wait_for_completion"])
        self.assertEqual(("wait", {"execution_id": "b-once", "timeout_ms": 50000}), calls[1])
        self.assertEqual("get", calls[2][0])
        self.assertEqual("desktop", calls[3][0])

    def test_wait_failure_stops_exact_execution_and_requires_durable_receipt(self):
        calls = []
        browser = NS(
            workflows=NS(execute=lambda **kw: calls.append(("execute", kw)) or Handle([{"executionId": "b-timeout"}])),
            executions=NS(
                wait_for_execution=lambda **kw: calls.append(("wait", kw)) or (_ for _ in ()).throw(TimeoutError("deadline exceeded")),
                stop=lambda **kw: calls.append(("stop", kw)) or Handle([{"status": "stopped"}]),
                get=lambda **kw: calls.append(("get", kw)) or Handle([{"executionId": "b-timeout", "workflowId": "wf", "workflowVersion": 2, "status": "cancelled"}]),
            ),
        )
        desktop = NS(flow=NS(replay=lambda **kw: calls.append(("desktop", kw))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["execute", "wait", "stop", "get"], [kind for kind, _ in calls])
        self.assertEqual({"execution_id": "b-timeout"}, calls[2][1])

    def test_wait_failure_without_stop_receipt_never_reads_forward_or_runs_desktop(self):
        calls = []
        browser = NS(
            workflows=NS(execute=lambda **kw: Handle([{"executionId": "b-uncertain"}])),
            executions=NS(
                wait_for_execution=lambda **kw: (_ for _ in ()).throw(TimeoutError("deadline exceeded")),
                stop=lambda **kw: calls.append("stop") or Handle([]),
                get=lambda **kw: calls.append("get"),
            ),
        )
        desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["stop"], calls)

    def test_browser_failure_prevents_desktop_side_effect(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([{"status": "failed"}]))))
        desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["browser"], calls)

    def test_browser_receipt_must_match_workflow_revision_before_desktop(self):
        for field, value in (("workflowId", "other"), ("workflowVersion", 1), ("executionId", "other"), ("status", "failed")):
            with self.subTest(field=field):
                calls = []
                browser = NS(
                    workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([{"executionId": "b-identity"}]))),
                    executions=NS(get=lambda **kw: Handle([{"execution": {"executionId": "b-identity", "workflowId": "wf", "workflowVersion": 2, "status": "completed", field: value}}])),
                )
                desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
                result = self.run_program(self.valid(), browser, desktop)
                self.assertEqual("browser_failed", result["errors"][0]["class"])
                self.assertEqual(["browser"], calls)

    def test_browser_execution_id_is_required_before_desktop(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([{"status": "completed"}]))))
        desktop = NS(flow=NS(replay=lambda **kw: calls.append("desktop")))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["browser"], calls)

    def test_browser_durable_receipt_is_required_before_desktop(self):
        calls = []
        browser = NS(
            workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([{"executionId": "b-no-receipt"}]))),
            executions=NS(get=lambda **kw: Handle([])),
        )
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

    def test_browser_scalar_metadata_without_execution_identity_is_rejected(self):
        calls = []
        browser = NS(workflows=NS(execute=lambda **kw: (calls.append("browser") or Handle([], {"status": "EXECUTION_STATUS_COMPLETED", "executionId": "b-meta"}))))
        desktop = NS(flow=NS(replay=lambda **kw: (calls.append("desktop") or Handle([{"status": "passed", "runId": "d-meta"}]))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("browser_failed", result["errors"][0]["class"])
        self.assertEqual(["browser"], calls)

    def test_missing_saved_flow_is_classified_as_desktop_failure(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([{"status": "completed", "executionId": "b-2"}])))
        desktop = NS(flow=NS(replay=lambda **kw: (_ for _ in ()).throw(RuntimeError("saved flow not found"))))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("desktop_failed", result["errors"][0]["class"])
        self.assertIn("saved flow not found", result["errors"][0]["detail"])

    def test_desktop_scalar_metadata_is_accepted(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([{"executionId": "b-3"}])))
        desktop = NS(flow=NS(replay=lambda **kw: Handle([], {"disposition": "passed", "runId": "d-meta"})))
        result = self.run_program(self.valid(), browser, desktop)
        self.assertEqual("ok", result["status"])
        self.assertEqual(["browser:b-3", "desktop:d-meta"], result["evidence"])

    def test_report_is_a_json_envelope(self):
        browser = NS(workflows=NS(execute=lambda **kw: Handle([], {"status": "completed", "executionId": "b-json"})))
        desktop = NS(flow=NS(replay=lambda **kw: Handle([], {"disposition": "passed", "runId": "d-json"})))
        stdout = io.StringIO()
        scope = {"inputs": self.valid(), "browser_automation_studio": browser, "device_control": desktop, "program": RuntimeProgram(self.valid())}
        with contextlib.redirect_stdout(stdout):
            exec(compile((ROOT / "do-task.py").read_text(), "do-task.py", "exec"), scope)
        self.assertEqual(scope["envelope"], json.loads(stdout.getvalue()))
