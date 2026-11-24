"""Clean read-only readers use ProgramHelper for their final JSON envelope."""
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
    def __init__(self, rows=(), meta=None):
        self.rows = list(rows)
        self.metadata = meta or {}

    def head(self, limit):
        return self.rows[:limit]

    def filter(self, predicate):
        return Rows([row for row in self.rows if predicate(row)], self.metadata)

    def meta(self):
        return self.metadata


def conversation_bindings(*, failed_mode=None):
    hit = {"stableHitId": "hit-1", "runId": "run-1", "eventId": "event-1",
           "title": "Fixture memory", "role": "assistant", "snippet": "synthetic fixture content"}

    def search(**kwargs):
        if kwargs["mode"].endswith("TEXT") and failed_mode == "text":
            raise RuntimeError("connection refused")
        return Rows([hit], {"degradations": []})

    def gather(*calls):
        results = []
        for call in calls:
            try:
                results.append(call())
            except Exception as exc:
                results.append(exc)
        return results

    return {"agent_manager": SimpleNamespace(conversation=SimpleNamespace(search=search)),
            "gather": gather}


def web_search_bindings(*, broken=()):
    broken = set(broken)

    def read(name, result):
        if name in broken:
            raise RuntimeError("connection refused")
        return result

    return {
        "web_search": type("WebSearch", (), {"findings": type("Findings", (), {
            "effectiveness": staticmethod(lambda **_kw: read("effectiveness", Rows())),
            "count": staticmethod(lambda **_kw: read("count", Rows(meta={"count": 0}))),
        })})(),
        "search_hub": type("SearchHub", (), {
            "insights": type("Insights", (), {"insights": staticmethod(lambda **_kw: read("insights", Rows()))})(),
            "providers": type("Providers", (), {"list": staticmethod(lambda **_kw: read("providers", Rows()))})(),
        })(),
        "gather": lambda *calls: [call() for call in calls],
    }


def run(relative, inputs, bindings=None):
    path = REPO / relative
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original(envelope)

    helper.report = report
    namespace = {"inputs": inputs, **(bindings or {})}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    result = json.loads(output.getvalue())
    assert len(reports) == 1
    assert output.getvalue().endswith("\n")
    assert result["program"] == contract["name"]
    assert result["phase"] == "report"
    return result


def test_git_control_tower_readers_keep_output_and_status_semantics():
    advisory = "scenarios/git-control-tower/.vrooli/program-runtime/advisory.py"
    result = run(advisory, {"mode": "summary", "subject_digest": "sha256:abc",
                            "claims": [{"text": "grounded", "evidence_refs": ["change:1"]}],
                            "unknowns": ["missing owner evidence"]})
    assert result["status"] == "partial"
    assert result["signals"]["claim_count"] == 1
    assert result["evidence"] == ["change:1"]
    invalid = run(advisory, {"mode": "activate", "subject_digest": "sha256:abc", "claims": []})
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "unsupported_mode"

    evidence = "scenarios/git-control-tower/.vrooli/program-runtime/change-evidence.py"
    result = run(evidence, {"operation_id": "op-1", "subject": {
        "repository_id": "repo-1", "kind": "change", "snapshot_digest": "sha256:s",
        "scope": {"paths": ["a.py"], "selection_digest": "sha256:p"}},
        "coverage": {"included_files": 2, "omitted_files": 1, "omissions": ["budget"]}})
    assert result["status"] == "partial"
    assert result["signals"]["coverage"]["omissions"] == ["budget"]
    invalid = run(evidence, {})
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"


def test_agent_manager_conversation_reader_keeps_bounded_metadata_and_partial_status():
    path = "scenarios/agent-manager/.vrooli/program-runtime/conversation-recall.py"
    result = run(path, {"clues": ["synthetic memory clue"], "policy": "direct"}, conversation_bindings())
    assert result["status"] == "ok"
    assert result["signals"]["candidate_count"] == 1
    assert result["signals"]["candidates"][0]["stable_hit_id"] == "hit-1"
    assert "synthetic memory clue" not in json.dumps(result)

    partial = run(path, {"clues": ["synthetic memory clue"], "policy": "direct"},
                   conversation_bindings(failed_mode="text"))
    assert partial["status"] == "partial", partial
    assert partial["signals"]["legs_succeeded"] == 1
    assert partial["signals"]["legs_failed"] == 1
    assert partial["errors"][0]["class"] == "scenario_unreachable"

    invalid = run(path, {"clues": []}, conversation_bindings())
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"

    setpoint = "scenarios/git-control-tower/.vrooli/program-runtime/setpoint-read.py"
    result = run(setpoint, {"observations": [{"row": "owner-review", "standing": "unavailable", "evidence": "ref-1"}]})
    assert result["status"] == "ok"
    assert result["signals"]["rows"][0]["row"] == "owner-review"
    invalid = run(setpoint, {"observations": "not-a-list"})
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"


def test_web_search_setpoint_keeps_success_partial_unavailable_and_validation_status():
    path = "scenarios/web-search/.vrooli/program-runtime/setpoint-read.py"
    result = run(path, {}, web_search_bindings())
    assert result["status"] == "ok"
    assert [row["row"] for row in result["signals"]["rows"]] == [
        "surfaced-rate", "used-rate", "never-surfaced-share", "live-vs-local-ratio",
        "cache-hit-rate", "budget-exhaustion", "provider-lifecycle", "capture-volume"]

    partial = run(path, {}, web_search_bindings(broken=("effectiveness",)))
    assert partial["status"] == "partial"
    assert partial["errors"][0]["class"] == "scenario_unreachable"
    unavailable = run(path, {}, web_search_bindings(broken=("effectiveness", "insights")))
    assert unavailable["status"] == "unavailable"
    invalid = run(path, {"effectiveness_limit": 0}, web_search_bindings())
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"
