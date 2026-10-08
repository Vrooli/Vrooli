import copy
import sys
from pathlib import Path
from types import SimpleNamespace

sys.path.insert(0, str(Path(__file__).parents[1]))
from host.engine import Handle, SessionKernel
from tasks import Tasks, ACTIVE_TASK


def complete(result, candidates=None, return_record=False):
    spec = {"name": "operation", "scenario": "fixture", "digest": "digest",
            "declaration": {"learning_task": {"outcome": {
                "status_path": "signals.outcome",
                "mapping": {"failed": "failed", "verified_success": "verified_success"},
                "evidence_paths": ["evidence"]}}}}
    owner = SimpleNamespace(lib=SimpleNamespace(_contracts={"fixture": {"operation": spec}}))
    task = Tasks(owner, None, Handle, 1)
    record = {"attempt_id": "attempt", "operation": "fixture.operation", "digest": "digest",
              "context_key": "stable-context", "scope": "fixture", "state": "running",
              "prepare": {"signals": {"attempt": {"recall_status": "no_match", "advice": []},
                                      "advice_candidates": candidates or []}}}

    def bridge(action, **kwargs):
        if action == "get":
            return copy.deepcopy(record)
        assert action == "complete"
        record.update(state="completed", finish_inputs=kwargs["finish_inputs"], outcome=kwargs["outcome"])
        return copy.deepcopy(record)

    task._bridge = bridge
    task.finish(attempt_id="attempt", result=result)
    return record if return_record else record["finish_inputs"]["attempt"]


def test_unhandled_candidates_are_exposure_not_rejection():
    candidates = [{"entry_id": "used"}, {"entry_id": "unread"}]
    result = {"signals": {"learning": {"advice": [{"entry_id": "used", "decision": "applied",
              "decision_change": "selected verified route", "verdict": "unknown"}]}}}
    attempt = complete(result, candidates)
    assert attempt["recall_status"] == "matched"
    assert attempt["advice"][0]["decision"] == "applied"
    assert attempt["advice"][1] == {"entry_id": "unread", "decision": "unassessed",
                                    "verdict": "unknown", "evidence_refs": []}


def test_observed_effort_and_feedback_survive_automatic_capture():
    feedback = {"observation_id": "feedback", "disposition": "supported", "evidence_refs": ["run:1"],
                "observed_at": "2026-09-01T00:00:00Z", "provenance": "test"}
    result = {"signals": {"outcome": "verified_success", "learning": {
        "measurements": {"tool_round_trips": 0, "reused_workflow": False},
        "observations": [feedback]}}, "evidence": ["run:1"]}
    record = complete(result, return_record=True)
    attempt = record["finish_inputs"]["attempt"]
    assert attempt["tool_round_trips"] == 0
    assert attempt["reused_workflow"] is False
    assert "first_action_at" not in attempt
    assert "visual_reasoning_calls" not in attempt
    assert record["finish_inputs"]["observations"] == [feedback]
    assert "attempt_id" not in feedback


def test_failure_fingerprint_tracks_stable_cause_not_error_detail():
    def attempt(detail, klass="timeout", where="execute"):
        return complete({"signals": {"outcome": "failed"}, "errors": [
            {"class": klass, "where": where, "detail": detail}], "evidence": []})

    first = attempt("request 100 failed at 10:00")
    retry = attempt("request 299 failed at 12:43")
    assert first["failure_fingerprint"] == retry["failure_fingerprint"]
    assert first["failure_fingerprint"] != attempt("request failed", klass="refused")["failure_fingerprint"]
    assert first["failure_fingerprint"] != attempt("request failed", where="verify")["failure_fingerprint"]


def test_verified_success_requires_explicit_signal_and_evidence():
    for result in ({"status": "ok", "evidence": ["artifact:1"]},
                   {"status": "ok", "signals": {"outcome": "verified_success"}, "evidence": []}):
        assert complete(result)["outcome"] == "unknown"
    result = {"signals": {"outcome": "verified_success"}, "evidence": ["artifact:1", "artifact:1"]}
    attempt = complete(result)
    assert attempt["outcome"] == "verified_success"
    assert attempt["evidence_refs"] == ["artifact:1"]


def test_tasks_namespace_is_protected_and_current_does_not_expose_resume_authority():
    result = SessionKernel().execute("tasks = {}")
    assert not result["ok"]
    assert "protected runtime name" in result["error"]
    token = ACTIVE_TASK.set({"attempt_id": "attempt", "resume_token": "secret"})
    try:
        assert Tasks(None, None, Handle, 1).current() == {"attempt_id": "attempt"}
    finally:
        ACTIVE_TASK.reset(token)


def test_learning_result_exposes_bounded_advice_without_implying_adoption():
    candidates = [{"entry_id": str(i), "text": "🙂" * 200} for i in range(5)]
    decisions = [{"entry_id": str(i), "decision": "rejected", "verdict": "unknown"} for i in range(5)]
    record = {"result": {"status": "ok", "signals": {"outcome": "unknown"}},
              "prepare": {"signals": {"recall_status": "matched", "advice_candidates": candidates}},
              "finish_inputs": {"attempt": {"advice": decisions}}}
    result = Tasks(None, None, Handle, 1)._result(record).head(1)[0]
    assert result["status"] == "ok"
    assert "learning" not in record["result"]
    learning = result["learning"]
    assert learning["recall_status"] == "matched"
    assert len(learning["advice_candidates"]) == 3
    assert learning["advice_candidates_omitted"] == 2
    for candidate in learning["advice_candidates"]:
        assert len(candidate["text"].encode("utf-8")) == 240
        assert candidate["text_truncated"] is True
    assert learning["advice_decisions"] == decisions[:3]
    assert "do not imply adoption" in learning["guidance"]
