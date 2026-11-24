import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace


BASE = Path(__file__).parents[1]


class Handle:
    def __init__(self, rows, meta=None):
        self.rows = rows
        self._meta = meta or {}

    def materialize(self, limit):
        return self.rows[:limit]

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return self._meta

    def count(self):
        return len(self.rows)


class Program:
    def __init__(self, inputs):
        self._inputs = inputs

    def inputs(self):
        return self._inputs

    def report(self, envelope):
        print(json.dumps(envelope, allow_nan=False))


def run_program(filename, inputs, bindings):
    namespace = {"program": Program(inputs), "vrooli": SimpleNamespace(scenario=SimpleNamespace(**bindings))}
    source = BASE / filename
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(source.read_text(), str(source), "exec"), namespace)
    return json.loads(output.getvalue())


def test_wait_routes_status_then_one_bounded_wait_and_keeps_timeout_typed():
    calls = []

    def status(**kwargs):
        calls.append(("status", kwargs))
        return Handle([], {"scenario": {"status": "running", "health": "healthy"}})

    def wait(**kwargs):
        calls.append(("wait", kwargs))
        return Handle([], {"success": False, "scenario": "fixture", "verdict": "timeout", "exitCode": 124,
                           "source": "attached", "waitedSeconds": 17, "operation": {"private": "must not retain"}})

    result = run_program("scenario-wait.py", {"scenario": "fixture", "timeout_seconds": 17}, {"status": status, "wait": wait})
    assert calls == [("status", {"name": "fixture"}), ("wait", {"name": "fixture", "timeout": 17})]
    assert result["status"] == "partial"
    assert result["signals"] == {"scenario": "fixture", "lifecycle_status": "running", "health": "healthy",
                                 "verdict": "timeout", "exit_code": 124, "source": "attached", "waited_seconds": 17}
    assert "must not retain" not in json.dumps(result)


def test_wait_rejects_timeout_outside_positive_ceiling_before_owner_calls():
    calls = []
    def unused(**kwargs):
        calls.append(kwargs)
        return Handle([])
    result = run_program("scenario-wait.py", {"scenario": "fixture", "timeout_seconds": 601}, {"status": unused, "wait": unused})
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
    assert calls == []


def test_deploy_report_keeps_only_bounded_safe_projection():
    calls = []
    items = [{"component": "api", "checkType": "go_module", "baseline": "recorded", "running": "replaced",
              "absolutePath": "/private", "fileContent": "private body"}]
    meta = {"scenario": "fixture", "changed": True, "sourceRole": "working_tree",
            "changedPaths": ["api/main.go", "../secret", "/private/abs"],
            "buildKeyNames": ["go_version"], "environmentSwitchNames": ["FEATURE_GATE"],
            "recorded": "SECRET_BUILD_VALUE", "environmentValues": ["SECRET_ENV_VALUE"],
            "captureId": "PRIVATE_CAPTURE"}

    def deploy_report(**kwargs):
        calls.append(kwargs)
        return Handle(items, meta)

    result = run_program("scenario-deploy-report.py", {"scenario": "fixture"}, {"deploy_report": deploy_report})
    assert calls == [{"name": "fixture"}]
    assert result["status"] == "ok"
    assert result["signals"]["artifacts"] == [{"component": "api", "checkType": "go_module", "check_type": "", "baseline": "recorded", "running": "replaced"}]
    assert result["signals"]["changed_paths"] == ["api/main.go"]
    assert result["signals"]["build_key_names"] == ["go_version"]
    assert result["signals"]["environment_switch_names"] == ["FEATURE_GATE"]
    payload = json.dumps(result)
    for secret in ("SECRET_BUILD_VALUE", "SECRET_ENV_VALUE", "PRIVATE_CAPTURE", "/private", "private body"):
        assert secret not in payload
