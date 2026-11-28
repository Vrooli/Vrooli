"""Offer Desk readers share ProgramHelper reporting but retain their own read policies."""
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
    def __init__(self, rows=(), *, total=None, metadata=None):
        self.rows = list(rows)
        self.total = len(self.rows) if total is None else total
        self.metadata = metadata or {}

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return self.total

    def meta(self):
        return self.metadata


def execute(name, inputs, bindings):
    path = REPO / "scenarios/offer-desk/.vrooli/program-runtime" / f"{name}.py"
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original_report = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original_report(envelope)

    helper.report = report
    namespace = {"inputs": inputs, "gather": lambda *calls: [call() for call in calls], **bindings}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    result = json.loads(output.getvalue())
    assert len(reports) == 1
    assert result["program"] == contract["name"]
    assert result["phase"] == "report"
    return result


def test_offer_board_keeps_snapshot_diff_truncation_and_unavailable_states():
    calls = []

    def board_show(**kwargs):
        calls.append(kwargs)
        return Rows([{"nodeId": "n1", "status": "ACTIVE", "title": "Current"}])

    bindings = {"offer_desk": SimpleNamespace(offers=SimpleNamespace(board_show=board_show))}
    result = execute("board-read", {"baseline": {"entries": {"n1": "IDEA", "n2": "RETIRED"}}}, bindings)
    assert calls == [{"rows": "entries"}]
    assert result["status"] == "ok"
    assert result["evidence"] == ["offer-desk/offers/board-show"]
    assert result["signals"]["changes"]["status_changed"] == [
        {"node_id": "n1", "title": "Current", "from": "IDEA", "to": "ACTIVE"}]
    assert result["signals"]["changes"]["removed"] == [{"node_id": "n2", "was": "RETIRED"}]

    many = [{"nodeId": f"n{i}", "status": "ACTIVE"} for i in range(81)]
    bindings = {"offer_desk": SimpleNamespace(offers=SimpleNamespace(
        board_show=lambda **kwargs: Rows(many[:80], total=81)))}
    result = execute("board-read", {}, bindings)
    assert result["status"] == "partial"
    assert result["signals"]["truncated"] is True
    assert result["signals"]["materialized"] == 80
    assert result["errors"][0]["class"] == "row_cap_reached"

    def unavailable(**_kwargs):
        raise RuntimeError("connection refused")

    bindings = {"offer_desk": SimpleNamespace(offers=SimpleNamespace(board_show=unavailable))}
    result = execute("board-read", {}, bindings)
    assert result["status"] == "unavailable"
    assert result["errors"][0]["class"] == "scenario_unreachable"
    assert result["errors"][0]["where"] == "collect"
    assert result["evidence"] == []


def test_learning_read_keeps_scope_reliability_and_empty_cohort_semantics():
    calls = []

    def measure(**kwargs):
        calls.append(kwargs)
        return Rows(metadata={"eligibleAttempts": 0, "reliable": False, "reason": "no eligible attempts"})

    bindings = {"vrooli_memory": SimpleNamespace(learning=SimpleNamespace(measure=measure))}
    result = execute("learning-read", {"operation": "publish"}, bindings)
    assert calls == [{"scope": "offer-desk-usage", "operation": "publish", "rows": "cohorts"}]
    assert result["status"] == "ok"
    assert result["evidence"] == ["vrooli-memory/learning/measure"]
    assert result["signals"]["eligible_attempts"] == 0
    assert result["signals"]["comparable"] is False
    assert result["signals"]["comparability_reason"] == "no eligible attempts"

    def unavailable(**_kwargs):
        raise RuntimeError("connection refused")

    bindings = {"vrooli_memory": SimpleNamespace(learning=SimpleNamespace(measure=unavailable))}
    result = execute("learning-read", {}, bindings)
    assert result["status"] == "unavailable"
    assert result["errors"][0]["class"] == "scenario_unreachable"
    assert result["errors"][0]["where"] == "collect"
    assert result["evidence"] == []
