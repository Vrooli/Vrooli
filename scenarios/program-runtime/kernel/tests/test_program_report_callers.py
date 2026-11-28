"""Program Runtime callers delegate their final JSON emission to ProgramHelper."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
PROGRAMS = REPO / "scenarios/program-runtime/.vrooli/program-runtime"
sys.path.insert(0, str(KERNEL))
sys.path.insert(0, str(KERNEL / "host"))

from host.program_helper import ProgramHelper  # noqa: E402


class LearnStep:
    def __enter__(self):
        return self

    def __exit__(self, *_exc):
        return False

    def note(self, *_args, **_kwargs):
        return None


class LearnFixture:
    def task(self, **_kwargs):
        return {"attempt_id": "fixture-attempt"}

    def step(self, *_args):
        return LearnStep()

    def recall(self, **_kwargs):
        return {"attempts": [], "hits": [], "recall_status": "empty"}

    def choose(self, _options, default):
        return {"selected_id": default, "source": "default", "learning": {}}

    def act(self, *_args, **_kwargs):
        return {"output": {"value": "fixture-value"}}

    def note(self, *_args, **_kwargs):
        return None

    def outcome(self, *_args, **_kwargs):
        return None

    def result(self, *_args, **_kwargs):
        return {}


def execute_report_caller(name):
    path = PROGRAMS / f"{name}.py"
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original_report = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original_report(envelope)

    helper.report = report
    namespace = {
        "inputs": {},
        "program_runtime": SimpleNamespace(library=SimpleNamespace(
            list=lambda **_kwargs: (_ for _ in ()).throw(RuntimeError("fixture unavailable")),
        )),
        "tasks": SimpleNamespace(learning_findings=lambda **_kwargs: (_ for _ in ()).throw(RuntimeError("fixture unavailable"))),
        "learn": LearnFixture(),
    }
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    raw = output.getvalue()
    return reports, raw, json.loads(raw)


def test_actual_program_sources_report_once_through_shared_helper():
    expected_status = {
        "fragment-promote": "failed",
        "learn-verbs-example": "ok",
        "learning-feedback": "failed",
        "learning-maintain": "failed",
        "portfolio-audit": "failed",
        "prepare-operation": "unavailable",
        "research-attention-delivery": "failed",
        "research-attention-index": "failed",
        "research-tools-review": "failed",
        "research-workspace-review": "failed",
    }
    for name, status in expected_status.items():
        reports, raw, result = execute_report_caller(name)
        assert len(reports) == 1, name
        assert isinstance(reports[0], dict), name
        assert raw.endswith("\n"), name
        assert result["program"] == "program-runtime." + name, name
        assert result["phase"] == "report", name
        assert result["status"] == status, name
