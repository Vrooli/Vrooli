import contextlib
import io
import json
from types import SimpleNamespace
from pathlib import Path


SOURCE = Path(__file__).parents[1] / "scenario-triage.py"


class Handle:
    def __init__(self, rows):
        self.rows = rows

    def materialize(self, limit):
        return self.rows[:limit]


class Program:
    class BindingError(Exception):
        pass

    def __init__(self, inputs):
        self._inputs = inputs

    def inputs(self):
        return self._inputs

    def fail(self, status, klass, detail, phase):
        return None

    def classify(self, _exc):
        return "unavailable", "binding_error"

    def run(self, states, initial):
        state = initial
        while state:
            state = states[state]()


def execute(inputs, *, status_rows, freshness_rows, log_rows, calls):
    def record(name, rows):
        def invoke(**kwargs):
            calls.append((name, kwargs))
            return Handle(rows)
        return invoke

    namespace = {
        "program": Program(inputs),
        "vrooli": SimpleNamespace(scenario=SimpleNamespace(
            status=record("status", status_rows),
            freshness=record("freshness", freshness_rows),
            logs=record("logs", log_rows),
        )),
    }
    stdout = io.StringIO()
    with contextlib.redirect_stdout(stdout):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), namespace)
    return json.loads(stdout.getvalue())


def test_log_body_is_classified_but_never_retained():
    calls = []
    secret = "PRIVATE_TOKEN_DO_NOT_RETAIN /home/operator/private.txt connection refused"
    result = execute(
        {"scenario": "fixture-scenario", "tail": 999},
        status_rows=[{"status": "stopped", "health": "unhealthy"}],
        freshness_rows=[{"stale": True}],
        log_rows=[{"line": secret}],
        calls=calls,
    )

    assert result["status"] == "ok"
    assert result["signals"]["log_lines_read"] == 1
    assert result["signals"]["hypotheses"] == [
        {"cause": "dependency", "evidence": "matched_declared_keyword:dependency"}
    ]
    assert secret not in json.dumps(result)
    assert calls == [
        ("status", {"name": "fixture-scenario"}),
        ("freshness", {"name": "fixture-scenario"}),
        ("logs", {"name": "fixture-scenario", "tail": "200"}),
    ]


def test_log_failure_stays_partial_and_never_invents_zero_evidence():
    calls = []

    def unavailable_logs(**kwargs):
        calls.append(("logs", kwargs))
        raise RuntimeError("owner detail must not enter the envelope")

    program = Program({"scenario": "fixture-scenario", "tail": 4})
    namespace = {
        "program": program,
        "vrooli": SimpleNamespace(scenario=SimpleNamespace(
            status=lambda **kwargs: Handle([{"status": "running", "health": "healthy"}]),
            freshness=lambda **kwargs: Handle([{"stale": False}]),
            logs=unavailable_logs,
        )),
    }
    stdout = io.StringIO()
    with contextlib.redirect_stdout(stdout):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), namespace)
    result = json.loads(stdout.getvalue())

    assert result["status"] == "partial"
    assert result["signals"]["log_lines_read"] == 0
    assert result["signals"]["hypotheses"] == []
    assert "owner detail must not enter the envelope" not in json.dumps(result)

