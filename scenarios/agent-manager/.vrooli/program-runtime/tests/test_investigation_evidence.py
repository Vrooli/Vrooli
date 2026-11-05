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


def execute(report, run_ids=None):
    output = io.StringIO()
    def unexpected_failure(*args):
        raise AssertionError(f"unexpected failure: {args}")
    program = NS(inputs=lambda: {"run_ids": run_ids or ["run-1"]}, classify=lambda exc: ("unavailable", "scenario_unreachable"), fail=unexpected_failure)
    lib = NS(run=NS(report=report))
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {"program": program, "agent_manager": lib, "gather": lambda *calls: [call() for call in calls]})
    return json.loads(output.getvalue())


def test_missing_receipt_coverage_is_partial_and_availability_is_explicit():
    def report(**kwargs):
        assert kwargs == {"run_id": "run-1", "rows": "tools"}
        return Handle(metadata={"status": "complete", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "unobserved"}})
    result = execute(report)
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "partial"
    assert subject["report"]["status"] == "complete"
    assert subject["report"]["receipts_availability"]["state"] == "unobserved"
    assert subject["report_availability"] == "partial"


def test_report_failure_is_an_explicit_unknown_and_a_healthy_sibling_stays_visible():
    healthy = Handle(metadata={"status": "active", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "available"}})
    def report(run_id, **_):
        if run_id == "run-2":
            raise RuntimeError("report unavailable")
        return healthy
    result = execute(report, ["run-1", "run-2"])
    first, second = result["signals"]["subjects"]
    assert result["status"] == "partial"
    assert first["report"]["status"] == "active" and first["report_availability"] == "available"
    assert second["report_availability"] == "unknown" and second["availability"] == "unknown"
    assert result["errors"][0]["where"] == "collect:run-2:report"


def test_maximum_subjects_keep_bounded_evidence():
    ids = ["run-" + str(i) for i in range(8)]
    report = lambda **_: Handle(metadata={"status": "active", "eventsAvailability": {"state": "available"}, "receiptsAvailability": {"state": "available"}})
    result = execute(report, ids)
    assert result["status"] == "ok"
    assert len(result["signals"]["subjects"]) == 8
    assert result["signals"]["coverage"] == "complete"
    assert all(subject["source_completeness"] == "unknown" and subject["source_freshness"] == "unknown" for subject in result["signals"]["subjects"])
