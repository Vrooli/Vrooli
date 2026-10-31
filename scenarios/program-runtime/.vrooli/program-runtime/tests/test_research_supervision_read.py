"""A supervisor packet preserves partial evidence and unknown source coverage."""
import contextlib
import io
import json
from pathlib import Path
import sys
from types import SimpleNamespace as NS

SOURCE = Path(__file__).parents[1] / "research-supervision-read.py"
sys.path.insert(0, str(Path(__file__).parents[3] / "kernel"))
from host.program_helper import ProgramHelper


class Handle:
    def __init__(self, row):
        self.row = row
    def head(self, n):
        return [self.row]
    def count(self):
        return 1
    def meta(self):
        return {"program": "fixture", "version": "1", "digest": "sha256:fixture", "output_contract": {"status": {"enum": ["ok", "partial", "failed", "unavailable", "refused"]}, "phase": {"enum": ["report"]}, "errors": {"classes": ["binding_error"]}}}


def execute(inputs, native, evidence, event_page=None):
    output = io.StringIO()
    program = ProgramHelper("2", {"status": {"enum": ["ok", "partial", "failed"]}, "errors": {"classes": ["invalid_response"]}})
    program._bind({"inputs": inputs})
    lib = NS(web_console=NS(native_thread_state=native), agent_manager=NS(investigation_evidence=evidence))
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {"program": program, "lib": lib, "agent_manager": NS(run=NS(event_page=event_page))})
    return json.loads(output.getvalue())


def child(status, **signals):
    return Handle({"status": status, "phase": "report", "signals": signals, "errors": [], "evidence": []})


def test_partial_child_does_not_hide_healthy_sibling():
    result = execute({"threads": ["t"], "run_ids": ["r"], "session_id": "s"},
                     lambda **_: child("ok", threads=[{"status": "active"}]),
                     lambda **_: child("partial", subjects=[{"availability": "unknown"}]))
    assert result["status"] == "partial"
    assert result["inputs"]["session_id"] == "s"
    assert result["signals"]["sources"]["native_state"]["signals"]["threads"][0]["status"] == "active"
    assert result["signals"]["sources"]["run_evidence"]["status"] == "partial"
    assert result["signals"]["decision"] == "review_required"


def test_attention_identifies_explicit_blockers_without_claiming_health():
    result = execute({"threads": ["t"], "run_ids": ["r"], "session_id": "s"},
                     lambda **_: child("ok", threads=[{"threadId": "t", "goalStatus": "blocked"}]),
                     lambda **_: child("partial", subjects=[{"run_id": "r", "source_freshness": "unknown", "report": {"status": "failed"}}]))
    assert result["signals"]["attention"] == [
        {"source": "native_state", "subject": "t", "signal": "goal_blocked"},
        {"source": "run_evidence", "subject": "r", "signal": "run_failed", "freshness": "unknown"},
    ]
    assert result["signals"]["decision"] == "review_required"


def test_oversized_source_preserves_healthy_sibling_with_explicit_omission():
    result = execute({"threads": ["t"], "run_ids": ["r"], "session_id": "s"},
                     lambda **_: child("ok", threads=[{"detail": "x" * 70000}]),
                     lambda **_: child("ok", subjects=[{"run_id": "r", "state": "active"}]))
    assert result["status"] == "partial"
    assert result["signals"]["output_truncated"] is True
    assert result["signals"]["sources"]["native_state"] == {
        "status": "unknown", "omitted_due_to_output_limit": True,
    }
    assert result["signals"]["sources"]["run_evidence"]["signals"]["subjects"][0]["state"] == "active"
    assert result["signals"]["decision"] == "review_required"
    assert result["errors"][-1]["class"] == "output_limit"
    assert len(json.dumps(result, separators=(",", ":")).encode()) <= 65536


def test_oversized_artifact_metadata_falls_back_to_bounded_failure():
    class LargeMetadata(Handle):
        def meta(self):
            return {**super().meta(), "digest": "d" * 70000}
    result = execute({"threads": ["t"], "session_id": "s"},
                     lambda **_: LargeMetadata({"status": "ok", "phase": "report", "signals": {}, "errors": [], "evidence": []}),
                     lambda **_: None)
    assert result["status"] == "failed"
    assert result["signals"]["output_truncated"] is True
    assert result["evidence"] == []
    assert result["errors"][-1]["class"] == "output_limit"
    assert len(json.dumps(result, separators=(",", ":")).encode()) + 1 <= 65536


def test_unavailable_source_and_not_requested_are_not_success():
    def unavailable(**_):
        raise RuntimeError("owner unavailable")
    result = execute({"threads": ["t"], "session_id": "s"}, unavailable, lambda **_: None)
    assert result["status"] == "partial"
    assert result["signals"]["sources"]["native_state"]["status"] == "unknown"
    assert result["signals"]["sources"]["run_evidence"]["status"] == "not_requested"


def test_malformed_child_is_not_accepted_as_complete():
    result = execute({"threads": ["t"], "session_id": "s"},
                     lambda **_: Handle({"status": "ok", "signals": {}, "errors": []}), lambda **_: None)
    source = result["signals"]["sources"]["native_state"]
    assert source["status"] == "failed"
    assert source["errors"][-1]["class"] == "invalid_response"


def test_missing_session_refused_before_owner_call():
    def unexpected(**_):
        raise AssertionError("owner called")
    result = execute({"threads": ["t"]}, unexpected, unexpected)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"


def test_oversized_invalid_inputs_do_not_escape_in_failure_envelope():
    def unexpected(**_):
        raise AssertionError("owner called")
    for inputs in ({"threads": ["t"], "session_id": "x" * 100000},
                   {"threads": ["x" * 100000], "session_id": "s"},
                   {"run_ids": ["r"], "extra": "x" * 100000, "event_cursors": {"other": 1}}):
        result = execute(inputs, unexpected, unexpected, unexpected)
        assert result["status"] == "failed"
        assert result["errors"][0]["class"] == "invalid_input"
        assert len(json.dumps(result).encode()) < 65536


def test_event_pages_resume_without_materializing_payloads():
    requests = []
    class Page:
        def count(self):
            return 0
        def meta(self):
            return {"summary": {"counts": {"tool_call": "3"}, "latestSequence": "12"}, "hasMore": True}
    def event_page(**kwargs):
        requests.append(kwargs)
        return Page()
    result = execute({"run_ids": ["r"], "event_cursors": {"r": 9}}, lambda **_: None,
                     lambda **_: child("ok"), event_page)
    page = result["signals"]["sources"]["event_pages"]["pages"][0]
    assert requests == [{"run_id": "r", "limit": 200, "summary_only": True, "after_sequence": 9}]
    assert page["next_cursor"] == 12 and page["has_more"] is True
    assert result["signals"]["sources"]["event_pages"]["status"] == "partial"
    assert result["signals"]["attention"] == [{"source": "event_pages", "subject": "r", "signal": "event_backlog"}]


def test_event_cursor_requires_selected_run_and_valid_sequence():
    def unexpected(**_):
        raise AssertionError("owner called")
    for cursors in ({"other": 1}, {"r": -1}, {"r": True}, {"r": 1 << 63}):
        result = execute({"run_ids": ["r"], "event_cursors": cursors}, unexpected, unexpected, unexpected)
        assert result["status"] == "failed"


def test_event_failure_preserves_other_evidence():
    def unavailable(**_):
        raise RuntimeError("offline")
    result = execute({"run_ids": ["r"], "event_cursors": {"r": None}}, lambda **_: None,
                     lambda **_: child("ok", subjects=[{"run_id": "r"}]), unavailable)
    assert result["signals"]["sources"]["run_evidence"]["status"] == "ok"
    assert result["signals"]["sources"]["event_pages"]["pages"][0]["status"] == "unknown"


def test_empty_summary_preserves_cursor_and_rejects_payload_fallback():
    class Page:
        def __init__(self, count):
            self.rows = count
        def count(self):
            return self.rows
        def meta(self):
            return {"summary": {}}
    args = {"run_ids": ["r"], "event_cursors": {"r": 9}}
    empty = execute(args, lambda **_: None, lambda **_: child("ok"), lambda **_: Page(0))
    page = empty["signals"]["sources"]["event_pages"]["pages"][0]
    assert page["next_cursor"] == 9 and page["has_more"] is False
    wrong = execute(args, lambda **_: None, lambda **_: child("ok"), lambda **_: Page(1))
    assert wrong["signals"]["sources"]["event_pages"]["pages"][0]["status"] == "unknown"


def test_malformed_event_values_never_advance_cursor():
    class Page:
        def __init__(self, meta):
            self.value = meta
        def count(self):
            return 0
        def meta(self):
            return self.value
    bad = [
        {"summary": {"counts": {"tool_call": 1}, "latestSequence": value}}
        for value in (True, 12.9, "-1", "9" * 20, 1 << 63)
    ] + [
        {"summary": {"counts": {"tool_call": value}, "latestSequence": "12"}}
        for value in (True, 1.5, "-1", "9" * 20)
    ] + [
        {"summary": {"counts": {"tool_call": "1"}, "latestSequence": "12"}, "hasMore": "false"},
        {"summary": {"counts": {"tool_call": "201"}, "latestSequence": "12"}},
        {"summary": {"counts": {"tool_call": "0"}, "latestSequence": "12"}},
        {"summary": {"counts": {"x" * 129: "1"}, "latestSequence": "12"}},
        {"summary": {"counts": {"é" * 65: "1"}, "latestSequence": "12"}},
        {"summary": {"counts": {"tool_call": "4"}, "latestSequence": "12"}},
    ]
    for metadata in bad:
        result = execute({"run_ids": ["r"], "event_cursors": {"r": 9}}, lambda **_: None,
                         lambda **_: child("ok", subjects=[{"run_id": "r"}]), lambda **_: Page(metadata))
        page = result["signals"]["sources"]["event_pages"]["pages"][0]
        assert page["status"] == "unknown"
        assert "next_cursor" not in page
        assert result["signals"]["sources"]["run_evidence"]["signals"]["subjects"][0]["run_id"] == "r"
