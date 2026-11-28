"""Actual owner sources share ProgramHelper's one-envelope JSON emission."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
sys.path.insert(0, str(KERNEL))
sys.path.insert(0, str(KERNEL / "host"))

from host.program_helper import ProgramHelper  # noqa: E402


class Rows:
    def head(self, _limit):
        return []


CASES = {
    "scenarios/program-runtime/.vrooli/program-runtime/validation-sweep.py": (
        {"scenarios": ""}, {}, "failed", "invalid_input"),
    "scenarios/vrooli-memory/.vrooli/program-runtime/inspect-task.py": (
        {}, {}, "failed", "task_unavailable"),
    "scenarios/vrooli-memory/.vrooli/program-runtime/resume-task.py": (
        {}, {}, "failed", "task_unavailable"),
    "scenarios/vrooli-memory/.vrooli/program-runtime/run-task.py": (
        {}, {}, "failed", "task_unavailable"),
    "scenarios/web-search/.vrooli/program-runtime/research-l3.py": (
        {}, {}, "failed", "invalid_input"),
    "scenarios/web-search/.vrooli/program-runtime/research.py": (
        {}, {}, "failed", "invalid_input"),
    "scenarios/web-search/.vrooli/program-runtime/learning-read.py": (
        {"unexpected": True}, {}, "failed", "invalid_input"),
    "scenarios/test-genie/.vrooli/program-runtime/iterate.py": (
        {}, {}, "failed", "invalid_input"),
    "scenarios/test-genie/.vrooli/program-runtime/validation-digest.py": (
        {}, {"test_genie": SimpleNamespace(validation=SimpleNamespace(list=lambda **_kwargs: Rows()))}, "ok", None),
    "scenarios/ai-gateway/.vrooli/program-runtime/decide-batch.py": (
        {}, {}, "failed", "invalid_input"),
    "scenarios/knowledge-observatory/.vrooli/program-runtime/candidate-proposals.py": (
        {}, {}, "failed", "invalid_input"),
}


def execute(path, inputs, bindings):
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original_report = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original_report(envelope)

    helper.report = report
    namespace = {"inputs": inputs, **bindings}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    return contract, reports, output.getvalue()


def test_clean_sources_report_once_with_original_status_policy():
    for relative, (inputs, bindings, expected_status, expected_error) in CASES.items():
        path = REPO / relative
        contract, reports, raw = execute(path, inputs, bindings)
        result = json.loads(raw)
        assert len(reports) == 1, relative
        assert isinstance(reports[0], dict), relative
        assert raw.endswith("\n"), relative
        assert result["program"] == contract["name"], relative
        expected_phase = "validate" if relative.endswith("candidate-proposals.py") else "report"
        assert result["phase"] == expected_phase, relative
        assert result["status"] == expected_status, relative
        if expected_error:
            assert result["errors"][0]["class"] == expected_error, relative
        else:
            assert result["errors"] == [], relative
