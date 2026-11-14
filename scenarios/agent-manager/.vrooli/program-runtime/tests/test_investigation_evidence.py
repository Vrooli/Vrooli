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


def available_accounting(**_):
    return Handle(metadata={"terminal": True, "tokensKnown": True, "nonCacheTokens": 0,
                            "cacheReadTokens": 0, "chargeMeasured": True, "chargeMicroUsd": 0})


def execute(report, run_ids=None, accounting=available_accounting):
    output = io.StringIO()
    def unexpected_failure(*args):
        raise AssertionError(f"unexpected failure: {args}")
    program = NS(inputs=lambda: {"run_ids": run_ids or ["run-1"]}, classify=lambda exc: ("unavailable", "scenario_unreachable"), fail=unexpected_failure)
    lib = NS(run=NS(report=report, accounting=accounting))
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
    def accounting(run_id):
        if run_id == "run-2":
            raise RuntimeError("accounting unavailable")
        return available_accounting()
    result = execute(report, ["run-1", "run-2"], accounting)
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


def complete_report(**_):
    return Handle(metadata={"status": "complete", "eventsAvailability": {"state": "available"},
                            "receiptsAvailability": {"state": "available"}})


def test_complete_projection_is_not_terminal_revision_or_work_lineage_proof():
    result = execute(complete_report)
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "ok" and result["signals"]["coverage"] == "complete"
    assert subject["availability"] == "complete"
    assert subject["source_completeness"] == "unknown"
    assert subject["source_freshness"] == "unknown"
    # This exact allowlist pins the owner projection. Adding evidence to the
    # governed program requires a new qualified source, not a guessed value.
    assert set(subject) == {"run_id", "availability", "source_completeness", "source_freshness",
                            "report", "report_availability", "accounting", "accounting_availability"}
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]


def test_report_extensions_cannot_smuggle_bridge_evidence():
    def report(**_):
        metadata = complete_report().meta().copy()
        metadata.update({"revision": "fabricated-immutable-revision", "updatedAt": "2026-11-07T00:00:00Z",
                         "sourceFreshness": "current", "terminalEventIds": ["made-up-event"],
                         "workReferences": [{"id": "made-up-work", "verified": True}]})
        return Handle(metadata=metadata)
    result = execute(report)
    encoded = json.dumps(result)
    assert "made-up" not in encoded and "fabricated" not in encoded
    subject = result["signals"]["subjects"][0]
    assert subject["source_freshness"] == "unknown"
    assert subject["source_completeness"] == "unknown"


def test_accounting_failure_keeps_report_and_explicit_unknown():
    def unavailable(**_):
        raise RuntimeError("accounting unavailable")
    result = execute(complete_report, accounting=unavailable)
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "partial" and result["signals"]["coverage"] == "partial"
    assert subject["report_availability"] == "available"
    assert subject["report"]["status"] == "complete"
    assert subject["accounting_availability"] == "unknown" and subject["accounting"] is None
    assert result["errors"][0]["where"] == "collect:run-1:accounting"


def test_report_failure_retains_available_accounting():
    def unavailable(**_):
        raise RuntimeError("report unavailable")
    result = execute(unavailable)
    subject = result["signals"]["subjects"][0]
    assert result["status"] == "partial"
    assert subject["report_availability"] == "unknown" and subject["report"] == {}
    assert subject["accounting_availability"] == "available"
    assert subject["accounting"]["terminal"] is True
    assert subject["source_freshness"] == "unknown"
    assert result["signals"]["available"] == 1 and result["signals"]["unknown"] == 1


def test_missing_accounting_values_do_not_become_zero_or_known():
    result = execute(complete_report, accounting=lambda **_: Handle())
    accounting = result["signals"]["subjects"][0]["accounting"]
    assert accounting and all(value is None for value in accounting.values())


def test_known_zero_accounting_survives_without_claiming_work_completion():
    result = execute(complete_report)
    accounting = result["signals"]["subjects"][0]["accounting"]
    assert accounting["tokensKnown"] is True and accounting["nonCacheTokens"] == 0
    assert accounting["chargeMeasured"] is True and accounting["chargeMicroUsd"] == 0
    assert "work_status" not in result["signals"]["subjects"][0]
