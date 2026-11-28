import json
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).parents[1]))
from host.engine import SessionKernel
from tasks import ACTIVE_TASK, Tasks


@pytest.mark.parametrize("count, grandchild_status, expected_status, schema_state", [
    ("wrong-type", "ok", "failed", "mismatch"),
    (4, "partial", "partial", "validated"),
])
def test_task_grandchild_schema_and_partial_status_propagate_without_retry(
        monkeypatch, count, grandchild_status, expected_status, schema_state):
    outputs = {
        "status": {"enum": ["ok", "partial", "unavailable", "refused", "failed"]},
        "phase": {"enum": ["collect", "report"]},
        "errors": {"classes": ["invalid_response", "domain_interrupted"]},
    }
    grandchild_schema = {
        "type": "object",
        "required": ["program", "version", "status", "phase", "inputs", "signals", "errors", "evidence"],
        "properties": {
            "program": {"const": "fixture.grandchild"},
            "signals": {"type": "object", "required": ["count"],
                        "properties": {"count": {"type": "integer", "minimum": 0}}},
        },
        "additionalProperties": True,
    }
    grandchild = {
        "name": "grandchild", "scenario": "fixture", "contract": True, "current": True, "version": 1,
        "digest": "a" * 64,
        "declaration": {
            "name": "fixture.grandchild", "version": "1", "helpers": {"program": "2"}, "inputs": {},
            "outputs": outputs, "budget": {"output_bytes": 4096},
            "output_schema": "grandchild.output.schema.json", "_runtime_output_schema": grandchild_schema,
            "learning_task": {"operation": "fixture.grandchild", "outcome": {
                "status_path": "status", "mapping": {"failed": "unknown"}, "evidence_paths": ["evidence"]}},
        },
        "source": ("print({'program':'fixture.grandchild','version':'1','status':" + json.dumps(grandchild_status)
                   + ",'phase':'report','inputs':{},'signals':{'count':" + json.dumps(count)
                   + "},'errors':[],'evidence':[]})"),
    }
    child = {
        "name": "child", "scenario": "fixture", "contract": True, "current": True, "version": 1,
        "digest": "b" * 64,
        "declaration": {"name": "fixture.child", "version": "1", "helpers": {"program": "2"}, "inputs": {},
                        "outputs": outputs, "budget": {"output_bytes": 4096}},
        "source": """envelope = program.envelope('fixture.child', '1', phase='collect')
task = lib.fixture.grandchild()
result = program.child_envelope(task, invalid_class='invalid_response', where='grandchild')
envelope['status'] = result['status']
envelope['signals']['grandchild_schema_validation'] = task.meta()['output_schema_validation']
envelope['errors'] = result['errors']
envelope['evidence'] = result['evidence']
envelope['phase'] = 'report'
program.report()""",
    }
    parent_source = """envelope = program.envelope('fixture.parent', '1', phase='collect')
result = program.child_envelope(lib.fixture.child(), invalid_class='invalid_response', where='child')
envelope['status'] = result['status']
envelope['signals'] = result['signals']
envelope['errors'] = result['errors']
envelope['evidence'] = result['evidence']
envelope['phase'] = 'report'
program.report()"""

    calls = []

    def prepare(self, *, operation, inputs, **_kwargs):
        record = {"state": "running", "operation": operation, "attempt_id": "attempt-1", "task_id": "task-1",
                  "attempt_number": 1, "digest": grandchild["digest"], "resume_token": "receipt-1",
                  "prepare": {"signals": {}}}
        ACTIVE_TASK.set(record)
        return self.handle([record], "tasks.prepare")

    def finish(self, *, attempt_id, result, **_kwargs):
        calls.append(result)
        record = {"state": "completed", "operation": grandchild["name"], "attempt_id": attempt_id,
                  "task_id": "task-1", "attempt_number": 1, "digest": grandchild["digest"],
                  "resume_token": "receipt-1", "result": result, "prepare": {"signals": {}},
                  "finish_inputs": {"attempt": {"advice": []}}}
        return self.handle([record], "tasks.finish")

    monkeypatch.setattr(Tasks, "prepare", prepare)
    monkeypatch.setattr(Tasks, "finish", finish)
    result = SessionKernel(libraries=[child, grandchild]).execute(
        parent_source, declared_helper_contract={"version": "2", "outputs": outputs})

    assert result["ok"], result
    envelope = json.loads(result["stdout"])
    assert len(calls) == 1
    assert envelope["status"] == expected_status
    assert envelope["signals"]["grandchild_schema_validation"]["state"] == schema_state
    if schema_state == "mismatch":
        assert envelope["errors"][0]["class"] == "invalid_response"
        assert envelope["errors"][0]["where"] == "fixture.grandchild"
        assert "output schema mismatch for fixture.grandchild" in envelope["errors"][0]["detail"]
        assert "$.signals.count does not match declared type" in envelope["errors"][0]["detail"]
    else:
        assert envelope["errors"] == []
