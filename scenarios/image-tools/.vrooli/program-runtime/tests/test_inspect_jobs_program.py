import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace


SOURCE = Path(__file__).parents[1] / "scene-preparation-stage.py"


class Response:
    def __init__(self, payload):
        self.payload = payload

    def raw(self):
        return self.payload


class Program:
    def __init__(self, inputs):
        self._inputs = inputs
        self._envelope = None

    def inputs(self):
        return self._inputs

    def envelope(self, name, version):
        self._envelope = {
            "program": name, "version": version, "status": "failed",
            "phase": "validate", "inputs": {}, "signals": {},
            "errors": [], "evidence": [],
        }
        return self._envelope

    def fail(self, status, klass, detail, phase):
        self._envelope["status"] = status
        self._envelope["phase"] = phase
        self._envelope["errors"].append({"class": klass, "detail": detail})
        self.report()
        return None

    def classify(self, _exc):
        return "failed", "binding_error"

    def report(self):
        print(json.dumps(self._envelope, separators=(",", ":")))

    def run(self, states, initial):
        state = initial
        while state:
            state = states[state]()


def execute(inputs, current_job, recent_jobs):
    calls = []

    def get(**kwargs):
        calls.append(("get", kwargs))
        return Response({"job": current_job})

    def list_jobs(**kwargs):
        calls.append(("list", kwargs))
        return Response({"jobs": recent_jobs})

    program = Program(inputs)
    namespace = {
        "program": program,
        "image_tools": SimpleNamespace(jobs=SimpleNamespace(get=get, list=list_jobs)),
    }
    stdout = io.StringIO()
    with contextlib.redirect_stdout(stdout):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), namespace)
    return json.loads(stdout.getvalue()), calls


def test_inspect_jobs_gets_exact_target_then_bounded_safe_list():
    sensitive = "PRIVATE_JOB_CONTENT_DO_NOT_RETAIN"
    result, calls = execute(
        {"action": "inspect-jobs", "job_id": "job-123", "job_limit": 2},
        {"id": "job-123", "operation": "segment", "state": "RUNNING",
         "progress": 0.4, "createdAt": "2026-11-24T00:00:00Z",
         "prompt": sensitive, "error": sensitive, "resultRef": sensitive},
        [
            {"id": "job-123", "operation": "segment", "state": "RUNNING",
             "progress": 0.4, "createdAt": "2026-11-24T00:00:00Z", "prompt": sensitive},
            {"id": "job-456", "operation": "inpaint", "state": "SUCCEEDED",
             "progress": 1, "createdAt": "2026-11-23T00:00:00Z", "error": sensitive},
            {"id": "job-789", "operation": "object_removal", "state": "FAILED",
             "progress": 1, "createdAt": "2026-11-22T00:00:00Z"},
        ],
    )

    assert calls == [("get", {"id": "job-123"}), ("list", {"limit": 2})]
    assert result["status"] == "ok"
    assert result["signals"]["job"] == {
        "id": "job-123", "operation": "segment", "state": "RUNNING",
        "progress": 0.4, "created_at": "2026-11-24T00:00:00Z",
    }
    assert len(result["signals"]["recent_jobs"]) == 2
    assert result["signals"]["target_in_recent"] is True
    assert sensitive not in json.dumps(result)


def test_inspect_jobs_stops_on_identity_mismatch_before_list():
    result, calls = execute(
        {"action": "inspect-jobs", "job_id": "requested", "job_limit": 10},
        {"id": "different"}, [],
    )

    assert calls == [("get", {"id": "requested"})]
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "owner_identity_mismatch"


def test_inspect_jobs_refuses_unbounded_limit_before_owner_reads():
    result, calls = execute(
        {"action": "inspect-jobs", "job_id": "requested", "job_limit": 21},
        {"id": "requested"}, [],
    )

    assert calls == []
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
