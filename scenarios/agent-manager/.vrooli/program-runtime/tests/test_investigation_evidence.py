"""Research evidence must keep source failures and healthy siblings distinct."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS

SOURCE = Path(__file__).parents[1] / "investigation-evidence.py"


class Handle:
    def __init__(self, rows=(), metadata=None):
        self.rows, self.metadata = list(rows), metadata or {}
    def meta(self):
        return self.metadata
    def head(self, limit):
        return self.rows[:limit]


def execute(report, episodes, run_ids=None):
    output = io.StringIO()
    def unexpected_failure(*args):
        raise AssertionError(f"unexpected failure: {args}")
    program = NS(inputs=lambda: {"run_ids": run_ids or ["run-1"]}, classify=lambda exc: ("unavailable", "scenario_unreachable"), fail=unexpected_failure)
    lib = NS(run=NS(report=report, episodes=episodes))
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {"program": program, "agent_manager": lib, "gather": lambda *calls: [call() for call in calls]})
    return json.loads(output.getvalue())


def test_missing_receipt_coverage_is_partial_and_projection_is_explicit():
    def report(**kwargs):
        assert kwargs == {"run_id": "run-1", "rows": "tools"}
        return Handle(metadata={"status": "complete", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "unobserved"}})
    result = execute(report, lambda **_: Handle())
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "partial"
    assert subject["report"]["status"] == "complete"
    assert subject["report"]["receipts_availability"]["state"] == "unobserved"
    assert subject["report_availability"] == "partial"
    assert subject["episodes_availability"] == "available"


def test_episode_failure_preserves_report():
    def failed(**kwargs):
        raise RuntimeError("episodes unavailable")
    result = execute(lambda **_: Handle(metadata={"status": "active", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "available"}}), failed)
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "partial"
    assert subject["report"]["status"] == "active"
    assert subject["report_availability"] == "available"
    assert subject["episodes_availability"] == "unknown"


def test_maximum_subjects_keep_bounded_evidence():
    ids = ["run-" + str(i) for i in range(8)]
    report = lambda **_: Handle(metadata={"status": "active", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "available"}})
    episodes = lambda **_: Handle(rows=[{"episodeId": str(i), "pattern": "repeated-work", "ownerConfidence": "high"} for i in range(4)])
    result = execute(report, episodes, ids)
    assert result["status"] == "ok"
    assert len(result["signals"]["subjects"]) == 8
    assert result["signals"]["coverage"] == "complete"
    assert all(subject["source_completeness"] == "unknown" and subject["source_freshness"] == "unknown" for subject in result["signals"]["subjects"])
