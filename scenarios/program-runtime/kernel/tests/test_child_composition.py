import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest


KERNEL = Path(__file__).resolve().parents[1]
PROGRAMS = KERNEL.parent / ".vrooli/program-runtime"
sys.path.insert(0, str(KERNEL))

from host.program_helper import ProgramHelper, Refused, RemoteError, ScenarioUnreachable  # noqa: E402


PROGRAM_NAMES = ("batch-inference", "typed-inference")


class Handle:
    def __init__(self, rows):
        self.rows = rows

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return {"program": "ai-gateway.classify-batch", "digest": "fake-digest"}


class FakeGateway:
    def __init__(self, result=None, error=None):
        self.result = result
        self.error = error
        self.calls = 0

    def classify_batch(self, **_kwargs):
        self.calls += 1
        if self.error:
            raise self.error
        return Handle(self.result)


def execute(program_name, *, child=None, error=None, corpus=None):
    gateway = FakeGateway(result=child, error=error)
    namespace = {
        "inputs": {"corpus": ["sample"] if corpus is None else corpus, "labels": ["infra"]},
        "lib": SimpleNamespace(ai_gateway=gateway),
    }
    helper = ProgramHelper()
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (PROGRAMS / f"{program_name}.py").read_text()
        exec(compile(source, f"{program_name}.py", "exec"), namespace)
    return json.loads(output.getvalue()), gateway.calls


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("status", ["ok", "partial", "failed", "unavailable", "refused"])
def test_child_envelope_status_errors_signals_and_evidence_are_preserved(program_name, status):
    child = {
        "status": status,
        "signals": {"results": [{"index": 0, "label": None}], "usage": None},
        "errors": [{"class": "inference_failed", "detail": "fixture", "where": "item:0"}],
        "evidence": ["ai-gateway:batch:fixture"],
    }

    result, calls = execute(program_name, child=[child])

    assert calls == 1
    assert result["status"] == status
    assert result["signals"] == child["signals"]
    assert result["errors"] == child["errors"]
    assert result["evidence"] == child["evidence"] + [{
        "program": "ai-gateway.classify-batch",
        "artifact": {"program": "ai-gateway.classify-batch", "digest": "fake-digest"},
    }]


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("error, status, klass", [
    (RemoteError("remote status 400: fixture", http_status=400), "failed", "remote_error"),
    (ScenarioUnreachable("scenario_not_running"), "unavailable", "scenario_unreachable"),
    (Refused("grant required", klass="no_grant"), "refused", "no_grant"),
])
def test_typed_child_call_errors_keep_runtime_classification(program_name, error, status, klass):
    result, calls = execute(program_name, error=error)

    assert calls == 1
    assert result["status"] == status
    assert result["errors"][0]["class"] == klass
    assert result["errors"][0]["where"] == "classify"


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
def test_empty_child_envelope_is_missing_result(program_name):
    result, calls = execute(program_name, child=[])

    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "missing_result"


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("child", [None, {"status": "ok"}, {"status": "unknown", "signals": {}, "errors": [], "evidence": []}])
def test_malformed_child_envelope_is_invalid_response(program_name, child):
    result, calls = execute(program_name, child=[child])

    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"


@pytest.mark.parametrize("program_name, invalid_corpus", [
    ("batch-inference", ["sample"] * 33),
    ("typed-inference", ["sample"] * 9),
])
def test_invalid_corpus_stops_before_child_call(program_name, invalid_corpus):
    result, calls = execute(program_name, child=[], corpus=invalid_corpus)

    assert calls == 0
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"
