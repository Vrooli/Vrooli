"""Actual owner fixtures for shared no-retry failure handling across programs."""
import ast
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
sys.path.insert(0, str(KERNEL))

from host.program_helper import DeadlineExceeded, ProgramHelper, ScenarioUnreachable  # noqa: E402


def execute_owner(source_path, contract_path, inputs, namespace):
    contract = json.loads(contract_path.read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    namespace.update({"inputs": inputs, "program": helper})
    helper._bind(namespace)
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(source_path.read_text(), str(source_path), "exec"), namespace)
    value = output.getvalue()
    try:
        return json.loads(value)
    except json.JSONDecodeError:
        return ast.literal_eval(value)


def invoke_prepare_task(error):
    calls = {"device_list": 0}

    def device_list():
        calls["device_list"] += 1
        raise error

    source = REPO / "scenarios/device-control/.vrooli/program-runtime/prepare-task.py"
    contract = source.with_suffix(".json")
    result = execute_owner(source, contract, {"device": "fixture-tv", "context_key": "app:v1"}, {
        "device_control": SimpleNamespace(device=SimpleNamespace(list=device_list)),
    })
    return result, calls["device_list"]


def invoke_plan_investigate(error):
    calls = {"investigate": 0}

    def investigate(**_kwargs):
        calls["investigate"] += 1
        raise error

    source = REPO / "scenarios/plan-manager/.vrooli/program-runtime/investigate.py"
    contract = source.with_suffix(".json")
    result = execute_owner(source, contract, {
        "caller_key": "fixture-key", "execution_id": "execution-1", "phase_id": "phase-1",
        "phase_generation": "1", "question": "Why did this fixture fail?", "run_ids": ["run-1"],
        "wait_seconds": 30,
    }, {"lib": SimpleNamespace(agent_manager=SimpleNamespace(investigate=investigate))})
    return result, calls["investigate"]


@pytest.mark.parametrize("invoke", [invoke_prepare_task, invoke_plan_investigate],
                         ids=["device-prepare", "plan-investigate"])
@pytest.mark.parametrize("error_factory,expected_status,expected_class,detail_marker", [
    (lambda: ScenarioUnreachable("fixture owner endpoint is unavailable", binding_id="fixture/owner/read"),
     "unavailable", "scenario_unreachable", "no retry was attempted"),
    (lambda: DeadlineExceeded("budget ended", binding_id="fixture/owner/read"),
     "failed", "deadline_exceeded", "no retry or budget extension is automatic"),
    (lambda: NameError("declared owner method is absent"),
     "failed", "unresolved_name", "no alias substitution or retry is safe"),
], ids=["unreachable", "deadline", "unresolved-name"])
def test_actual_owners_use_shared_failure_refusal_once(invoke, error_factory, expected_status, expected_class, detail_marker):
    error = error_factory()
    result, calls = invoke(error)

    assert calls == 1
    assert result["status"] == expected_status
    assert len(result["errors"]) == 1
    issue = result["errors"][0]
    assert issue["class"] == expected_class
    contract_path = REPO / ("scenarios/device-control/.vrooli/program-runtime/prepare-task.json"
                            if result["program"] == "device-control.prepare-task"
                            else "scenarios/plan-manager/.vrooli/program-runtime/investigate.json")
    contract = json.loads(contract_path.read_text())
    assert expected_class in contract["outputs"]["errors"]["classes"]
    expected_where = "collect" if result["program"] == "device-control.prepare-task" else "delegate"
    assert issue["where"] == expected_where
    assert str(error) in issue["detail"]
    assert detail_marker in issue["detail"]


@pytest.mark.parametrize("invoke,limit", [
    (invoke_prepare_task, 160),
    (invoke_plan_investigate, 200),
], ids=["device-prepare-160", "plan-investigate-200"])
def test_actual_owner_refusals_keep_the_declared_detail_bound(invoke, limit):
    error = ScenarioUnreachable("owner detail " + ("x" * 400), binding_id="fixture/owner/read")
    result, calls = invoke(error)

    assert calls == 1
    assert result["status"] == "unavailable"
    assert len(result["errors"]) == 1
    issue = result["errors"][0]
    assert issue["class"] == "scenario_unreachable"
    assert len(issue["detail"]) <= limit
    assert "no retry was attempted" in issue["detail"]
