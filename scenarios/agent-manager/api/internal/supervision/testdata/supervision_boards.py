import contextlib, io, json, pathlib, sys
from types import SimpleNamespace as N
sys.path.insert(0, str(pathlib.Path(sys.argv[1]).resolve().parents[2] / "program-runtime" / "kernel"))
from host.program_helper import ProgramHelper

class Handle:
    def __init__(self, meta=None, rows=None, raw=None):
        self.m, self.r, self.value = meta or {}, rows or [], raw
    def meta(self): return self.m
    def head(self, n): return self.r[:n]
    def count(self): return len(self.r)
    def raw(self): return self.value

def run(name, inputs, agent_manager):
    path = pathlib.Path(sys.argv[1]) / f"{name}.py"
    out = io.StringIO()
    environment = {"inputs": inputs, "agent_manager": agent_manager, "gather": lambda *fs: tuple(f() for f in fs)}
    helper = ProgramHelper()
    helper._bind(environment)
    environment["program"] = helper
    with contextlib.redirect_stdout(out): exec(compile(path.read_text(), str(path), "exec"), environment)
    assert len(out.getvalue().encode()) <= 65536
    return json.loads(out.getvalue())

inspection = {"watch": {"watchId": "watch", "revision": "4", "spec": {"policyVersion": "candidate"}, "lastDecision": {"decisionId": "decision", "classification": "stalled"}}, "subjectStates": [{"runId": "child", "status": "complete", "terminal": True}], "cursorResetRequired": True}
coverage = {"outcomes": 19, "assessedOutcomes": 0, "families": 1, "population": "retained nonsuperseded outcomes"}
watch = N(inspect=lambda **kw: Handle(inspection), actions=lambda **kw: Handle(rows=[{"actionId": "action", "decisionId": "decision", "state": "WATCH_ACTION_STATE_APPLIED", "targetRunId": "child"}]), policy_outcomes=lambda **kw: Handle({"coverage": coverage}, [{"observedClass": "", "familyExecutionId": "family"}]), policy_get=lambda **kw: Handle({"policy": {"version": "candidate"}, "digest": "pinned"}))
case = run("supervision-case-read", {"watch_id": "watch"}, N(watch=watch))
assert case["status"] == "ok" and case["signals"]["current_children"][0]["terminal"]
assert case["signals"]["unassessed_sample"] == 1 and case["signals"]["next_action"] == "reconcile_cursor"
assert case["signals"]["classification"] == "stalled"  # historical decision is not overwritten by later completion
experiment = run("supervision-experiment-read", {"policy_version": "candidate"}, N(watch=watch))
assert experiment["status"] == "ok" and experiment["signals"]["evaluation"] is None
assert experiment["signals"]["next_action"] == "collect_assessed_evidence" and experiment["signals"]["coverage"]["families"] == 1

board = {
    "observedAt": "2026-10-16T12:00:00Z", "activeCount": 4, "partial": False, "limitations": [],
    "discovery": {"partial": False, "scannedCount": 4, "scanLimit": 100, "findings": []},
    "rows": [
        {"enrollment": {"effortRef": "effort:review", "displayName": "Review owner", "withdrawn": False, "authorizedBy": ""}, "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"}, "assignments": [{"subject": {"owner": "agent-manager", "kind": "run", "role": "orchestrator", "runId": "run-review"}, "runtimeState": "needs_review", "unavailableReason": ""}], "goalHome": {"path": "goals/review", "state": "available", "activeEpoch": "E2", "activeStatus": "in_progress", "activeWorkers": ["run-review"], "limitations": []}, "nextAction": "reconcile retained run", "blockers": ["manual review"], "limitations": ["runtime unknown"], "evidenceRefs": ["goal:review"]},
        {"enrollment": {"effortRef": "effort:prepared", "displayName": "Prepared only", "withdrawn": False, "authorizedBy": ""}, "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"}, "assignments": [], "goalHome": {"path": "goals/prepared", "state": "available", "activeEpoch": "E7", "activeStatus": "in_progress", "activeWorkers": ["folder-run"], "needsOperator": ["review decision"], "openFeedbackIds": ["FB-1"], "openWorkarounds": 2, "limitations": []}, "nextAction": "observe", "blockers": [], "limitations": [], "evidenceRefs": ["goal:prepared"]},
        {"enrollment": {"effortRef": "effort:retired", "displayName": "Retired owner", "withdrawn": True, "withdrawalReason": "owner retired"}, "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"}, "assignments": [], "goalHome": {"path": "goals/retired", "state": "available", "activeEpoch": "E4", "activeStatus": "in_progress", "activeWorkers": ["old-run"], "limitations": []}, "nextAction": "retired", "blockers": [], "limitations": [], "evidenceRefs": ["goal:retired"]},
        {"enrollment": {"effortRef": "effort:blocked", "displayName": "Blocked owner", "withdrawn": False, "authorizedBy": ""}, "runtimeState": "unknown", "freshness": "unavailable", "outcomeStanding": {"state": "unknown"}, "assignments": [{"subject": {"owner": "agent-manager", "kind": "run", "role": "worker", "runId": "run-blocked"}, "runtimeState": "unknown", "unavailableReason": "run owner unavailable"}], "goalHome": {"path": "", "state": "unavailable", "activeEpoch": "", "activeStatus": "", "activeWorkers": [], "limitations": ["destination ref unavailable"]}, "nextAction": "reconcile metadata", "blockers": ["receiving owner unavailable"], "limitations": ["goal home unreadable"], "evidenceRefs": ["goal:blocked"]},
    ],
}
calls = []
def read_board(**kw):
    calls.append(kw)
    return Handle(raw=board)
owner = N(board=read_board)
compact = run("supervision-observation-read", {"page_size": 8, "page_token": ""}, N(effort=owner))
assert calls == [{"page_size": 8, "page_token": "", "rows": "rows"}]
assert compact["status"] == "ok" and compact["signals"]["selected_count"] == 2
assert compact["signals"]["coverage"] == "complete" and compact["signals"]["authorized_by_used_for_selection"] is False
by_id = {row["id"]: row for row in compact["signals"]["efforts"]}
assert by_id["effort:review"]["eligible"] is True and by_id["effort:review"]["runtime_state"] == "unknown"
assert by_id["effort:review"]["owner_refs"][0]["runtime_state"] == "needs_review"
assert by_id["effort:review"]["owner_refs_total"] == 1 and by_id["effort:review"]["owner_refs_truncated"] is False
assert by_id["effort:blocked"]["eligible"] is True and by_id["effort:blocked"]["freshness"] == "unavailable"
assert by_id["effort:prepared"]["eligible"] is False and by_id["effort:prepared"]["disposition"] == "admission_or_completion_unknown"
assert by_id["effort:prepared"]["goal_home"]["active_epoch"] == "E7"
assert "does not establish admission" in by_id["effort:prepared"]["reason"]
assert by_id["effort:retired"]["eligible"] is False and by_id["effort:retired"]["disposition"] == "retired"
assert by_id["effort:review"]["detail_ref"] == "agent-manager:GetEffortBoard:effort:review"
assert compact["signals"]["discovery"]["scanned_count"] == 4

# The compact owner-reference cap must not hide a current resumable obligation
# behind older completed runs, and it must disclose omitted history/obligations.
terminal_then_current = {
    "enrollment": {"effortRef": "effort:terminal-then-current", "displayName": "Recent owner", "withdrawn": False},
    "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"},
    "assignments": [
        {"subject": {"role": "orchestrator", "runId": f"old-{i}"}, "runtimeState": "complete"}
        for i in range(8)
    ] + [{"subject": {"role": "orchestrator", "runId": "current-parked"}, "runtimeState": "parked"}],
}
unfinished_overflow = {
    "enrollment": {"effortRef": "effort:unfinished-overflow", "displayName": "Many unfinished", "withdrawn": False},
    "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"},
    "assignments": [
        {"subject": {"role": "worker", "runId": f"unknown-{i}"}, "runtimeState": "unknown"}
        for i in range(10)
    ],
}
all_terminal_history = {
    "enrollment": {"effortRef": "effort:terminal-history", "displayName": "Terminal history", "withdrawn": False},
    "runtimeState": "complete", "freshness": "fresh", "outcomeStanding": {"state": "unknown"},
    "assignments": [
        {"subject": {"role": "orchestrator", "runId": f"history-{i}"}, "runtimeState": "failed"}
        for i in range(3)
    ],
}
long_reference = "owner-reference-" + ("x" * 160)
long_reference_owner = {
    "enrollment": {"effortRef": "effort:long-owner-reference", "displayName": "Long reference", "withdrawn": False},
    "runtimeState": "unknown", "freshness": "fresh", "outcomeStanding": {"state": "unknown"},
    "assignments": [
        {"subject": {"role": "orchestrator", "reference": long_reference}, "runtimeState": "needs_review"},
    ],
}
priority_board = dict(board)
priority_board["rows"] = [terminal_then_current, unfinished_overflow, all_terminal_history, long_reference_owner]
priority = run("supervision-observation-read", {"page_size": 8}, N(effort=N(board=lambda **kw: Handle(raw=priority_board))))
priority_by_id = {row["id"]: row for row in priority["signals"]["efforts"]}
current_refs = priority_by_id["effort:terminal-then-current"]["owner_refs"]
assert current_refs[0]["reference"] == "current-parked" and current_refs[0]["runtime_state"] == "parked"
assert len(current_refs) == 8 and current_refs[1]["reference"] == "old-0"
assert priority_by_id["effort:terminal-then-current"]["owner_refs_total"] == 9
assert priority_by_id["effort:terminal-then-current"]["owner_refs_truncated"] is True
unfinished_refs = priority_by_id["effort:unfinished-overflow"]["owner_refs"]
assert [ref["reference"] for ref in unfinished_refs] == [f"unknown-{i}" for i in range(8)]
assert priority_by_id["effort:unfinished-overflow"]["owner_refs_total"] == 10
assert priority_by_id["effort:unfinished-overflow"]["owner_refs_truncated"] is True
history = priority_by_id["effort:terminal-history"]
assert [ref["reference"] for ref in history["owner_refs"]] == [f"history-{i}" for i in range(3)]
assert history["owner_refs_total"] == 3 and history["owner_refs_truncated"] is False
assert priority_by_id["effort:long-owner-reference"]["owner_refs"][0]["reference"] == long_reference

partial_board = dict(board)
partial_board["partial"], partial_board["nextPageToken"] = True, "next-cut"
compact = run("supervision-observation-read", {"page_size": 4, "page_token": "prior-cut"}, N(effort=N(board=lambda **kw: Handle(raw=partial_board))))
assert compact["status"] == "partial" and compact["signals"]["page_has_more"] is True
assert compact["signals"]["next_page_token"] == "next-cut" and compact["signals"]["coverage"] == "partial"
def unavailable_board(**kw):
    raise RuntimeError("owner unavailable")
unavailable = run("supervision-observation-read", {"page_size": 8}, N(effort=N(board=unavailable_board)))
assert unavailable["status"] == "unavailable" and unavailable["signals"]["coverage"] == "unavailable"
invalid = run("supervision-observation-read", {"page_size": 0}, N(effort=owner))
assert invalid["status"] == "failed" and invalid["errors"][0]["class"] == "invalid_input"
print("supervision boards: bounded owner selection, unknown/review inclusion, prioritized capped owner refs, populated-unadmitted/retired exclusion, and partial paging passed")
