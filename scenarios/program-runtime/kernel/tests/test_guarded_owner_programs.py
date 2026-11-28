"""Owner-program fixtures for the shared per-read exception guard."""
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

from host.program_helper import ProgramHelper  # noqa: E402


class Rows:
    def __init__(self, values=(), metadata=None):
        self.values = list(values)
        self.metadata = metadata or {}

    def count(self):
        return len(self.values)

    def filter(self, predicate):
        return Rows([value for value in self.values if predicate(value)], self.metadata)

    def map(self, mapper):
        return Rows([mapper(value) for value in self.values], self.metadata)

    def group_by(self, key):
        groups = {}
        for value in self.values:
            group = value.get(key, "")
            groups[group] = groups.get(group, 0) + 1
        return groups

    def sort(self, key, reverse=False):
        return Rows(sorted(self.values, key=lambda value: value.get(key, ""), reverse=reverse), self.metadata)

    def head(self, limit):
        return self.values[:limit]

    def meta(self):
        return self.metadata


def execute_owner_program(scenario, name, inputs, bindings):
    path = REPO / "scenarios" / scenario / ".vrooli/program-runtime" / f"{name}.py"
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    guarded_calls = []
    original_guarded = helper.guarded

    def guarded(call):
        guarded_calls.append(call)
        return original_guarded(call)

    helper.guarded = guarded
    namespace = {"inputs": inputs, **bindings}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    raw = output.getvalue()
    try:
        result = json.loads(raw)
    except json.JSONDecodeError:
        result = ast.literal_eval(raw)
    return result, guarded_calls


def test_pipeline_inspect_uses_shared_guard_for_two_read_bindings():
    pipeline = Rows(metadata={"scenarioName": "fixture", "pipelineId": "pipe-1", "status": "STAGE_STATUS_COMPLETED", "stages": {}})
    bindings = {"scenario_to_desktop": SimpleNamespace(
        pipeline=SimpleNamespace(status=lambda **_kwargs: pipeline),
        tasks=SimpleNamespace(list=lambda **_kwargs: Rows()),
    ), "gather": lambda *calls: [call() for call in calls]}
    result, guarded = execute_owner_program("scenario-to-desktop", "pipeline-inspect", {"scenario": "fixture", "pipeline_id": "pipe-1"}, bindings)
    assert len(guarded) == 2
    assert result["status"] == "ok"
    assert result["signals"]["pipeline"]["id"] == "pipe-1"
    assert result["signals"]["task_count"] == 0


def test_evidence_inventory_uses_shared_guard_and_keeps_empty_inventory():
    bindings = {"scenario_to_desktop": SimpleNamespace(
        evidence=SimpleNamespace(list=lambda **_kwargs: Rows(), summary=lambda **_kwargs: Rows(metadata={"count": 0, "totalBytes": 0})),
    ), "gather": lambda *calls: [call() for call in calls]}
    result, guarded = execute_owner_program("scenario-to-desktop", "evidence-inventory", {"scenario": "fixture"}, bindings)
    assert len(guarded) == 2
    assert result["status"] == "ok"
    assert result["signals"]["capture_count"] == 0
    assert result["signals"]["total_bytes"] == 0


def test_setpoint_read_uses_shared_guard_and_preserves_readable_condition():
    binding_ids = (
        "scenario-to-desktop/pipeline/list", "scenario-to-desktop/pipeline/status",
        "scenario-to-desktop/tasks/list", "scenario-to-desktop/evidence/list",
        "scenario-to-desktop/evidence/summary",
    )
    condition = Rows([{"bindingId": binding_id, "status": "CONDITION_STATUS_HEALTHY"} for binding_id in binding_ids])
    learning = Rows(metadata={"reliable": False, "reason": "unreliable:insufficient_data"})
    bindings = {
        "program_runtime": SimpleNamespace(bindings=SimpleNamespace(condition=lambda **_kwargs: condition)),
        "vrooli_memory": SimpleNamespace(learning=SimpleNamespace(measure=lambda **_kwargs: learning)),
        "gather": lambda *calls: [call() for call in calls],
    }
    result, guarded = execute_owner_program("scenario-to-desktop", "setpoint-read", {}, bindings)
    assert len(guarded) == 2
    assert result["status"] == "ok"
    condition_row = next(row for row in result["signals"]["rows"] if row["row"] == "binding-condition")
    assert condition_row["reading"]["healthy"] == 5
    assert condition_row["in_band"] is True


@pytest.mark.parametrize("name", ["pipeline-inspect", "setpoint-read", "evidence-inventory"])
def test_owner_sources_reference_shared_guard_without_local_copy(name):
    source = (REPO / "scenarios/scenario-to-desktop/.vrooli/program-runtime" / f"{name}.py").read_text()
    assert "guarded = program.guarded" in source
    assert "def guarded(" not in source
