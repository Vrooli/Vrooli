"""The composing programs call the shared child helper directly.

These fixtures protect the phase-derived error location that two local
forwarding wrappers used to add at each call site.
"""
import contextlib
import ast
import io
import hashlib
import json
import sys
from pathlib import Path
from types import SimpleNamespace

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
sys.path.insert(0, str(KERNEL))

from host.program_helper import AmbiguousResponse, DeadlineExceeded, ProgramHelper, Refused, ScenarioUnreachable  # noqa: E402


class ChildHandle:
    def __init__(self, child, output_contract, digest="sha256:fixture"):
        self.child = child
        self.output_contract = output_contract
        self.digest = digest

    def head(self, limit):
        return [self.child][:limit]

    def count(self):
        return 1

    def meta(self):
        return {"program": self.child.get("program", "fixture.child"), "version": "1", "digest": self.digest, "output_contract": self.output_contract}


class RowsHandle:
    def __init__(self, rows=()):
        self.rows = list(rows)

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return len(self.rows)

    def meta(self):
        return {"total": len(self.rows), "generatedAt": "2026-11-24T00:00:00Z", "truncated": False}

    def filter(self, predicate):
        return RowsHandle(row for row in self.rows if predicate(row))

    def sort(self, *_args, **_kwargs):
        return self


class EmptyOwner:
    """Make any declared read return an empty synthetic handle."""

    def __getattr__(self, _name):
        return self

    def __call__(self, *_args, **_kwargs):
        return RowsHandle()


class LearnStep:
    def __enter__(self):
        return self

    def __exit__(self, *_exc):
        return False

    def outcome(self, *_args, **_kwargs):
        return None


class LearnFixture:
    def task(self, **_kwargs):
        return {"attempt_id": "attempt-1", "task_id": "task-1", "key": "fixture"}

    def step(self, _name):
        return LearnStep()

    def outcome(self, *_args, **_kwargs):
        return None

    def result(self, *_args, **_kwargs):
        return {}

    def result_status(self, **_kwargs):
        return {"available": True, "eligible": True}

    def note(self, *_args, **_kwargs):
        return None

    def feedback(self, *_args, **_kwargs):
        return {}


def execute(path, inputs, libraries, extra=None):
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    namespace = {"inputs": inputs, "lib": libraries, "learn": LearnFixture()}
    namespace.update(extra or {})
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    try:
        return json.loads(output.getvalue())
    except json.JSONDecodeError:
        return ast.literal_eval(output.getvalue())


def malformed_child(name, contract_path):
    contract = json.loads(contract_path.read_text())
    return ChildHandle({"program": name, "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}, contract["outputs"])


def test_device_do_task_direct_child_helper_preserves_collect_error_location():
    program = REPO / "scenarios/device-control/.vrooli/program-runtime/do-task.py"
    child_contract = REPO / "scenarios/device-control/.vrooli/program-runtime/prepare-task.json"
    child = malformed_child("device-control.prepare-task", child_contract)
    libraries = SimpleNamespace(device_control=SimpleNamespace(prepare_task=lambda **_kwargs: child))
    result = execute(program, {"device": "device-1", "context_key": "test", "actor": "operator"}, libraries)
    issue = next(error for error in result["errors"] if error["class"] == "invalid_response")
    assert issue["where"] == "collect"
    assert issue["detail"] == "child status is missing or outside its declared enum"
    assert "def child_result" not in program.read_text()


def test_browser_do_task_direct_child_helper_uses_current_act_phase():
    program = REPO / "scenarios/browser-automation-studio/.vrooli/program-runtime/do-task.py"
    child_contract = REPO / "scenarios/browser-automation-studio/.vrooli/program-runtime/smoke-flow.json"
    child = malformed_child("browser-automation-studio.smoke-flow", child_contract)
    libraries = SimpleNamespace(browser_automation_studio=SimpleNamespace(smoke_flow=lambda **_kwargs: child))
    result = execute(program, {"task": "open a page", "workflow_id": "workflow-1", "version": 1}, libraries)
    issue = next(error for error in result["errors"] if error["class"] == "invalid_response")
    assert issue["where"] == "act"
    assert issue["detail"] == "child status is missing or outside its declared enum"
    assert "def child_result" not in program.read_text()


def test_inference_programs_adopt_partial_child_outcome_and_append_artifact_evidence():
    child_contract = json.loads((REPO / "scenarios/ai-gateway/.vrooli/program-runtime/classify-batch.json").read_text())["outputs"]
    child_data = {
        "program": "ai-gateway.classify-batch", "version": "3", "status": "partial", "phase": "report",
        "inputs": {"documents": 2}, "signals": {"labels": ["infra", None], "validated": 1},
        "errors": [{"class": "invalid_result", "detail": "one item was rejected", "where": "classify:1"}],
        "evidence": [{"binding": "ai-gateway/inference/run-batch"}],
    }

    for name in ("batch-inference", "typed-inference"):
        program = REPO / f"scenarios/program-runtime/.vrooli/program-runtime/{name}.py"
        digest = "0e07025e32853027f155dd4216da6f0a28a705b0c887f88fcdcf0fe35b76edd8" if name == "typed-inference" else "sha256:fixture"
        child = ChildHandle(child_data, child_contract, digest=digest)
        libraries = SimpleNamespace(ai_gateway=SimpleNamespace(classify_batch=lambda **_kwargs: child))
        result = execute(program, {}, libraries)
        assert result["status"] == "partial"
        assert result["signals"] == child_data["signals"]
        assert result["errors"] == child_data["errors"]
        assert result["evidence"][0] == child_data["evidence"][0]
        assert result["evidence"][1]["program"] == "ai-gateway.classify-batch"


def test_learning_read_programs_adopt_child_fields_while_preserving_parent_projection():
    child_contract = json.loads((REPO / "scenarios/vrooli-memory/.vrooli/program-runtime/compare-outcomes.json").read_text())["outputs"]
    child_data = {
        "program": "vrooli-memory.compare-outcomes", "version": "2", "status": "partial", "phase": "report",
        "inputs": {"operation": "record_attempt"},
        "signals": {"reliability": {"state": "partial"}, "rows": [
            {"row": "failure-recurrence", "reading": {"cohorts": [{"operation": "record_attempt", "context": "search", "attempts": 4, "failed": 1, "private": "dropped"}]}},
            {"row": "unselected", "reading": {"value": 99}},
        ]},
        "errors": [{"class": "scenario_unreachable", "detail": "one measurement is unavailable", "where": "collect"}],
        "evidence": [{"program": "vrooli-memory.compare-outcomes"}],
    }
    for scenario in ("browser-automation-studio", "knowledge-observatory"):
        program = REPO / f"scenarios/{scenario}/.vrooli/program-runtime/learning-read.py"
        child = ChildHandle(child_data, child_contract)
        libraries = SimpleNamespace(vrooli_memory=SimpleNamespace(compare_outcomes=lambda **_kwargs: child))
        result = execute(program, {}, libraries)
        assert result["status"] == "partial"
        assert result["inputs"] == child_data["inputs"]
        assert result["errors"] == child_data["errors"]
        assert result["evidence"] == child_data["evidence"]
        assert [row["row"] for row in result["signals"]["rows"]] == ["failure-recurrence"]
        cohort = result["signals"]["rows"][0]["reading"]["cohorts"][0]
        assert cohort == {"operation": "record_attempt", "context": "search", "attempts": 4, "failed": 1,
                          "unavailable": None, "unknown": None, "recurringFailureFingerprints": None}
        assert result["signals"]["reliability"] == child_data["signals"]["reliability"]


def test_inference_program_adopts_normalized_invalid_child_response():
    program = REPO / "scenarios/program-runtime/.vrooli/program-runtime/batch-inference.py"
    child_contract = json.loads((REPO / "scenarios/ai-gateway/.vrooli/program-runtime/classify-batch.json").read_text())["outputs"]
    child_data = {"program": "ai-gateway.classify-batch", "version": "3", "status": "mystery", "phase": "report",
                  "inputs": {}, "signals": {}, "errors": [], "evidence": []}
    child = ChildHandle(child_data, child_contract)
    result = execute(program, {}, SimpleNamespace(ai_gateway=SimpleNamespace(classify_batch=lambda **_kwargs: child)))
    assert result["status"] == "failed"
    assert result["errors"][-1]["class"] == "invalid_response"
    assert result["errors"][-1]["where"] == "classify"


def test_inference_programs_use_shared_runtime_catch_for_unreachable_classification():
    for name in ("batch-inference", "typed-inference"):
        program = REPO / f"scenarios/program-runtime/.vrooli/program-runtime/{name}.py"
        calls = []

        def classify_batch(**_kwargs):
            calls.append("called")
            raise ScenarioUnreachable("fixture inference endpoint is unavailable", binding_id="ai-gateway/inference/run-batch")

        result = execute(program, {}, SimpleNamespace(ai_gateway=SimpleNamespace(classify_batch=classify_batch)))
        issue = result["errors"][-1]
        assert calls == ["called"]
        assert result["status"] == "unavailable"
        assert issue["class"] == "scenario_unreachable"
        assert issue["where"] == "classify"
        assert "no retry was attempted" in issue["detail"]


def test_watch_set_shared_runtime_catch_preserves_collected_evidence_on_classifier_failure():
    class AgentFixture:
        def start(self, **_kwargs):
            return {"run_id": "run-1"}

        def collect(self, _handle, **_kwargs):
            return SimpleNamespace(head=lambda _limit: [{"execution_id": "run-1", "output": {"status": "ok"}}])

    calls = []

    def classify_batch(**_kwargs):
        calls.append("called")
        raise ScenarioUnreachable("fixture inference endpoint is unavailable", binding_id="ai-gateway/inference/run-batch")

    result = execute(
        REPO / "scenarios/program-runtime/.vrooli/program-runtime/watch-set.py",
        {"requests": [{"owner": "fixture", "workflow_key": "fixture.workflow", "input": {}}]},
        SimpleNamespace(ai_gateway=SimpleNamespace(classify_batch=classify_batch)),
        extra={"agent": AgentFixture()},
    )
    issue = result["errors"][-1]
    assert calls == ["called"]
    assert result["status"] == "unavailable"
    assert result["signals"]["started"] == 1
    assert result["signals"]["collected"] == 1
    assert result["evidence"] == ["run-1"]
    assert issue["class"] == "scenario_unreachable"
    assert issue["where"] == "classify"


def test_plan_investigate_preserves_partial_child_and_uses_shared_runtime_catch():
    child_contract = json.loads((REPO / "scenarios/agent-manager/.vrooli/program-runtime/investigate.json").read_text())["outputs"]
    child_data = {
        "program": "agent-manager.investigate", "version": "1", "status": "partial", "phase": "report",
        "inputs": {}, "signals": {"pending": True, "investigation_id": "investigation-1"},
        "errors": [{"class": "scenario_unreachable", "detail": "one evidence source is unavailable", "where": "collect"}],
        "evidence": [{"program": "agent-manager.investigate"}],
    }
    inputs = {"caller_key": "task-1", "execution_id": "execution-1", "phase_id": "phase-1",
              "phase_generation": "generation-1", "question": "What failed?", "run_ids": ["run-1"]}
    child = ChildHandle(child_data, child_contract)
    partial = execute(
        REPO / "scenarios/plan-manager/.vrooli/program-runtime/investigate.py", inputs,
        SimpleNamespace(agent_manager=SimpleNamespace(investigate=lambda **_kwargs: child)),
    )
    assert partial["status"] == "partial"
    assert partial["signals"]["pending"] is True
    assert partial["signals"]["nested_status"] == "partial"
    assert partial["errors"] == child_data["errors"]
    assert partial["evidence"][-1] == child_data["evidence"][0]

    calls = []

    def investigate(**_kwargs):
        calls.append("called")
        raise ScenarioUnreachable("fixture investigation service is unavailable", binding_id="agent-manager/investigate")

    unavailable = execute(
        REPO / "scenarios/plan-manager/.vrooli/program-runtime/investigate.py", inputs,
        SimpleNamespace(agent_manager=SimpleNamespace(investigate=investigate)),
    )
    issue = unavailable["errors"][-1]
    assert calls == ["called"]
    assert unavailable["status"] == "unavailable"
    assert issue["class"] == "scenario_unreachable"
    assert issue["where"] == "delegate"
    assert issue["detail"] == "scenario_unreachable"


def test_command_center_portfolio_pulse_uses_shared_deadline_refusal():
    calls = []

    def expired_backlog(**_kwargs):
        calls.append("backlog")
        raise TimeoutError("owner deadline expired; commit state unknown")

    result = execute(
        REPO / "scenarios/command-center/.vrooli/program-runtime/portfolio-pulse.py", {}, SimpleNamespace(),
        extra={
            "gather": lambda *calls: [call() for call in calls],
            "swarm_manager": SimpleNamespace(
                backlog=SimpleNamespace(list=expired_backlog),
                goals=SimpleNamespace(list=lambda **_kwargs: RowsHandle()),
            ),
            "command_center": SimpleNamespace(walk=SimpleNamespace(read=lambda **_kwargs: RowsHandle())),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "deadline_exceeded")
    assert calls == ["backlog"]
    assert result["status"] == "partial"
    assert failure["where"] == "stale_backlog"
    assert "no retry or budget extension is automatic" in failure["detail"]
    assert "commit state unknown" in failure["detail"]


def test_browser_author_flow_uses_shared_no_grant_refusal_without_acting():
    calls = []

    def validate(**_kwargs):
        calls.append("validate")
        raise Refused("owner refused workflow validation", binding_id="browser-automation-studio/workflows/validate")

    def execute_adhoc(**_kwargs):
        calls.append("execute")
        raise AssertionError("an unvalidated flow must never execute")

    result = execute(
        REPO / "scenarios/browser-automation-studio/.vrooli/program-runtime/author-flow.py",
        {"flow": {"nodes": [{"id": "node-1"}]}, "project_id": "project-1"},
        SimpleNamespace(browser_automation_studio=SimpleNamespace(workflows=SimpleNamespace(
            validate=validate, execute_adhoc=execute_adhoc,
        ))),
        extra={"browser_automation_studio": SimpleNamespace(workflows=SimpleNamespace(
            validate=validate, execute_adhoc=execute_adhoc,
        ))},
    )
    issue = result["errors"][-1]
    assert calls == ["validate"]
    assert result["status"] == "refused"
    assert result["phase"] == "report"
    assert issue["class"] == "no_grant"
    assert issue["where"] == "collect"
    assert "permission was not inferred" in issue["detail"]
    assert "owner refused workflow validation" in issue["detail"]
    assert result["signals"]["outcome"] == "unknown"
    assert result["evidence"] == []


def test_browser_find_flows_uses_shared_deadline_refusal_without_retry():
    calls = []

    def list_workflows(**_kwargs):
        calls.append("workflows.list")
        raise DeadlineExceeded("workflow listing reached its deadline", binding_id="browser-automation-studio/workflows/list")

    def search(**_kwargs):
        calls.append("search-hub.query")
        raise AssertionError("a failed primary owner read must stop the search")

    result = execute(
        REPO / "scenarios/browser-automation-studio/.vrooli/program-runtime/find-flows.py",
        {"task": "open settings"},
        SimpleNamespace(
            browser_automation_studio=SimpleNamespace(workflows=SimpleNamespace(list=list_workflows)),
            search_hub=SimpleNamespace(query=SimpleNamespace(query=search)),
        ),
        extra={
            "browser_automation_studio": SimpleNamespace(workflows=SimpleNamespace(list=list_workflows)),
            "search_hub": SimpleNamespace(query=SimpleNamespace(query=search)),
        },
    )
    issue = result["errors"][-1]
    assert calls == ["workflows.list"]
    assert result["status"] == "failed"
    assert result["phase"] == "report"
    assert issue["class"] == "deadline_exceeded"
    assert issue["where"] == "collect"
    assert "no retry or budget extension is automatic" in issue["detail"]
    assert "workflow listing reached its deadline" in issue["detail"]
    assert result["signals"]["candidates"] == []
    assert result["evidence"] == []


def test_browser_find_flows_keeps_optional_no_grant_as_partial_with_shared_refusal():
    calls = []

    def list_workflows(**_kwargs):
        calls.append("workflows.list")
        return RowsHandle()

    def search(**_kwargs):
        calls.append("search-hub.query")
        raise Refused("query permission missing", binding_id="search-hub/query/query")

    result = execute(
        REPO / "scenarios/browser-automation-studio/.vrooli/program-runtime/find-flows.py",
        {"task": "open settings"},
        SimpleNamespace(
            browser_automation_studio=SimpleNamespace(workflows=SimpleNamespace(list=list_workflows)),
            search_hub=SimpleNamespace(query=SimpleNamespace(query=search)),
        ),
        extra={
            "browser_automation_studio": SimpleNamespace(workflows=SimpleNamespace(list=list_workflows)),
            "search_hub": SimpleNamespace(query=SimpleNamespace(query=search)),
        },
    )
    issue = next(error for error in result["errors"] if error["class"] == "no_grant")
    assert calls == ["workflows.list", "search-hub.query"]
    assert result["status"] == "partial"
    assert issue["where"] == "collect"
    assert issue["detail"].startswith("search-hub: No grant was supplied")
    assert "query permission missing" in issue["detail"]
    assert "not retried" in issue["detail"]
    assert result["signals"]["candidates"] == []
    assert result["evidence"] == []


def test_local_work_snapshot_pins_research_workspace_child_digest(tmp_path):
    parent_path = REPO / "scenarios/program-runtime/.vrooli/program-runtime/local-work-snapshot.py"
    child_path = REPO / "scenarios/program-runtime/.vrooli/program-runtime/research-workspace-review.py"
    child_contract_path = child_path.with_suffix(".json")
    child_contract = json.loads(child_contract_path.read_text())
    child_source_digest = hashlib.sha256(child_contract_path.read_bytes() + b"\0" + child_path.read_bytes()).hexdigest()
    parent_source = parent_path.read_text()
    assert child_source_digest in parent_source

    research_root = tmp_path / "research"
    (research_root / "topics").mkdir(parents=True)
    (research_root / "CATALOG.json").write_text(json.dumps({"version": "fixture", "topics": [], "runs": []}))

    for metadata_digest, expected_child_status in ((child_source_digest, "ok"), ("", "failed"), ("f" * 64, "failed")):
        child_calls = []

        def run_workspace_child(*, research_root):
            child_calls.append(research_root)
            child_result = execute(child_path, {"research_root": research_root}, SimpleNamespace())
            return ChildHandle(child_result, child_contract["outputs"], digest=metadata_digest)

        result = execute(
            parent_path,
            {"research_root": str(research_root)},
            SimpleNamespace(program_runtime=SimpleNamespace(research_workspace_review=run_workspace_child)),
            extra={"agent_manager": EmptyOwner(), "prompt_manager": EmptyOwner(),
                   "swarm_manager": EmptyOwner(), "program_runtime": EmptyOwner()},
        )
        workspace = result["signals"]["sources"]["research_workspace"]
        assert child_calls == [str(research_root)]
        assert result["status"] == "partial"
        assert workspace["status"] == expected_child_status
        assert workspace["query_complete"] is (expected_child_status == "ok")
        if expected_child_status == "ok":
            assert workspace["artifact"]["digest"] == child_source_digest
            assert workspace["errors"] == []
        else:
            issue = workspace["errors"][-1]
            assert issue["class"] == "invalid_response"
            assert "child artifact digest is absent or does not match its declared pin" in issue["detail"]


def test_command_center_vision_prep_uses_shared_deadline_refusal():
    calls = []

    def expired_walk(**_kwargs):
        calls.append("outcomes")
        raise TimeoutError("owner deadline expired; observation may be incomplete")

    teams = ["director-swarm", "monetization", "marketing-crew", "meta-optimization", "infra-health", "scenario-qa"]
    source_ledger = SimpleNamespace(journal=SimpleNamespace(list=lambda **_kwargs: RowsHandle()))
    result = execute(
        REPO / "scenarios/command-center/.vrooli/program-runtime/vision-walk-prep.py", {}, SimpleNamespace(),
        extra={
            "gather": lambda *calls: [call() for call in calls],
            "prompt_manager": SimpleNamespace(team=SimpleNamespace(list=lambda **_kwargs: RowsHandle([{"id": team} for team in teams]))),
            "command_center": SimpleNamespace(walk=SimpleNamespace(read=expired_walk)),
            "swarm_manager": SimpleNamespace(
                goals=SimpleNamespace(list=lambda **_kwargs: RowsHandle()),
                backlog=SimpleNamespace(list=lambda **_kwargs: RowsHandle()),
            ),
            "meta_optimization_manager": SimpleNamespace(focus=SimpleNamespace(next=lambda **_kwargs: RowsHandle())),
            "infrastructure_manager": SimpleNamespace(focus=SimpleNamespace(next=lambda **_kwargs: RowsHandle())),
            "source_ledger": source_ledger,
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "deadline_exceeded")
    assert calls == ["outcomes"]
    assert result["status"] == "partial"
    assert failure["where"] == "outcomes"
    assert "no retry or budget extension is automatic" in failure["detail"]
    assert "observation may be incomplete" in failure["detail"]


def test_agent_manager_supervision_read_uses_shared_unresolved_name_refusal():
    calls = []

    def policy_get(**_kwargs):
        calls.append("policy")
        raise NameError("missing_policy_owner_binding")

    def policy_outcomes(**_kwargs):
        calls.append("outcomes")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/supervision-experiment-read.py",
        {"policy_version": "policy-v1"}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(watch=SimpleNamespace(policy_get=policy_get, policy_outcomes=policy_outcomes)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "unresolved_name")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "failed" and result["phase"] == "report"
    assert result["inputs"] == {"policy_version": "policy-v1"}
    assert failure["where"] == "collect"
    assert "missing_policy_owner_binding" in failure["detail"]
    assert "no alias substitution or retry is safe" in failure["detail"]


def test_agent_manager_investigation_read_uses_shared_unresolved_name_refusal():
    calls = []

    def report(**_kwargs):
        calls.append("report")
        raise NameError("missing_run_report_binding")

    def accounting(**_kwargs):
        calls.append("accounting")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/investigation-evidence.py",
        {"run_ids": ["run-1"]}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(run=SimpleNamespace(report=report, accounting=accounting)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "unresolved_name")
    assert calls == ["report", "accounting"]
    assert result["status"] == "partial" and result["phase"] == "report"
    assert result["inputs"] == {"run_ids": ["run-1"]}
    assert result["signals"]["subjects"][0]["availability"] == "partial"
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]
    assert failure["where"] == "collect:run-1:report"
    assert "missing_run_report_binding" in failure["detail"]
    assert "no alias substitution or retry is safe" in failure["detail"]


def test_agent_manager_supervision_read_uses_shared_ambiguous_response_refusal():
    calls = []

    def policy_get(**_kwargs):
        calls.append("policy")
        raise AmbiguousResponse("binding has no determinable primary response field: rows, result")

    def policy_outcomes(**_kwargs):
        calls.append("outcomes")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/supervision-experiment-read.py",
        {"policy_version": "policy-v1"}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(watch=SimpleNamespace(policy_get=policy_get, policy_outcomes=policy_outcomes)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "ambiguous_response")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "failed" and result["inputs"] == {"policy_version": "policy-v1"}
    assert failure["where"] == "collect"
    assert "no row field was guessed or retried" in failure["detail"]
    assert "no determinable primary response field" in failure["detail"]


def test_agent_manager_investigation_read_uses_shared_ambiguous_response_refusal():
    calls = []

    def report(**_kwargs):
        calls.append("report")
        raise AmbiguousResponse("binding has no determinable primary response field: rows, result")

    def accounting(**_kwargs):
        calls.append("accounting")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/investigation-evidence.py",
        {"run_ids": ["run-1"]}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(run=SimpleNamespace(report=report, accounting=accounting)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "ambiguous_response")
    assert calls == ["report", "accounting"]
    assert result["status"] == "partial"
    assert result["signals"]["subjects"][0]["availability"] == "partial"
    assert result["signals"]["subjects"][0]["report_availability"] == "unknown"
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]
    assert failure["where"] == "collect:run-1:report"
    assert "no row field was guessed or retried" in failure["detail"]
    assert "no determinable primary response field" in failure["detail"]


def test_agent_manager_supervision_read_uses_shared_no_grant_refusal():
    calls = []

    def policy_get(**_kwargs):
        calls.append("policy")
        raise Refused("policy owner requires an explicit grant", binding_id="agent-manager/watch/policy-get")

    def policy_outcomes(**_kwargs):
        calls.append("outcomes")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/supervision-experiment-read.py",
        {"policy_version": "policy-v1"}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(watch=SimpleNamespace(policy_get=policy_get, policy_outcomes=policy_outcomes)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "no_grant")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "refused" and result["inputs"] == {"policy_version": "policy-v1"}
    assert failure["where"] == "collect"
    assert "permission was not inferred" in failure["detail"]
    assert "call was not retried" in failure["detail"]
    assert "policy owner requires an explicit grant" in failure["detail"]


def test_agent_manager_supervision_partial_outcomes_keeps_no_grant_refusal():
    calls = []

    def policy_get(**_kwargs):
        calls.append("policy")
        return SimpleNamespace(meta=lambda: {
            "policy": {"version": "policy-v1", "evaluatorDigest": "evaluator-digest"},
            "digest": "policy-digest", "inferenceIdentityDigest": "inference-digest",
            "state": "active", "evaluation": {"eligible": True},
        })

    def policy_outcomes(**_kwargs):
        calls.append("outcomes")
        raise Refused("outcome owner requires an explicit grant", binding_id="agent-manager/watch/policy-outcomes")

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/supervision-experiment-read.py",
        {"policy_version": "policy-v1"}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(watch=SimpleNamespace(policy_get=policy_get, policy_outcomes=policy_outcomes)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "no_grant")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "partial"
    assert result["signals"]["outcomes_available"] is False
    assert result["signals"]["promotion_authority"].startswith("agent-manager;")
    assert result["evidence"] == ["policy-v1"]
    assert failure["where"] == "collect"
    assert "permission was not inferred" in failure["detail"]
    assert "outcome owner requires an explicit grant" in failure["detail"]


def test_agent_manager_investigation_read_keeps_partial_no_grant_cause_and_evidence():
    calls = []

    def report(**_kwargs):
        calls.append("report")
        raise Refused("run report owner requires an explicit grant", binding_id="agent-manager/run/report")

    def accounting(**_kwargs):
        calls.append("accounting")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/investigation-evidence.py",
        {"run_ids": ["run-1"]}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(run=SimpleNamespace(report=report, accounting=accounting)),
        },
    )
    failure = next(error for error in result["errors"] if error["class"] == "no_grant")
    assert calls == ["report", "accounting"]
    assert result["status"] == "partial"
    assert result["signals"]["subjects"][0]["availability"] == "partial"
    assert result["signals"]["subjects"][0]["accounting_availability"] == "available"
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]
    assert failure["where"] == "collect:run-1:report"
    assert "permission was not inferred" in failure["detail"]
    assert "call was not retried" in failure["detail"]
    assert "run report owner requires an explicit grant" in failure["detail"]


def _supervision_error_result(error, *, outcome_error=False):
    calls = []

    def policy_get(**_kwargs):
        calls.append("policy")
        if not outcome_error:
            raise error
        return SimpleNamespace(meta=lambda: {
            "policy": {"version": "policy-v1", "evaluatorDigest": "evaluator-digest"},
            "digest": "policy-digest", "inferenceIdentityDigest": "inference-digest",
            "state": "active", "evaluation": {"eligible": True},
        })

    def policy_outcomes(**_kwargs):
        calls.append("outcomes")
        if outcome_error:
            raise error
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/supervision-experiment-read.py",
        {"policy_version": "policy-v1"}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(watch=SimpleNamespace(policy_get=policy_get, policy_outcomes=policy_outcomes)),
        },
    )
    return result, calls


def _investigation_error_result(error):
    calls = []

    def report(**_kwargs):
        calls.append("report")
        raise error

    def accounting(**_kwargs):
        calls.append("accounting")
        return RowsHandle()

    result = execute(
        REPO / "scenarios/agent-manager/.vrooli/program-runtime/investigation-evidence.py",
        {"run_ids": ["run-1"]}, SimpleNamespace(),
        extra={
            "gather": lambda *steps: [step() for step in steps],
            "agent_manager": SimpleNamespace(run=SimpleNamespace(report=report, accounting=accounting)),
        },
    )
    return result, calls


def test_agent_manager_supervision_read_shares_unreachable_refusal():
    error = ScenarioUnreachable("policy owner is unreachable; last observation is stale")
    result, calls = _supervision_error_result(error)
    failure = next(error for error in result["errors"] if error["class"] == "scenario_unreachable")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "unavailable" and failure["where"] == "collect"
    assert "last observation is stale" in failure["detail"]
    assert "no retry was attempted" in failure["detail"]


def test_agent_manager_investigation_read_keeps_partial_unreachable_cause_and_evidence():
    error = ScenarioUnreachable("run report owner is unreachable; source freshness is unknown")
    result, calls = _investigation_error_result(error)
    failure = next(error for error in result["errors"] if error["class"] == "scenario_unreachable")
    assert calls == ["report", "accounting"]
    assert result["status"] == "partial"
    assert result["signals"]["subjects"][0]["availability"] == "partial"
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]
    assert failure["where"] == "collect:run-1:report"
    assert "source freshness is unknown" in failure["detail"]
    assert "no retry was attempted" in failure["detail"]


def test_agent_manager_supervision_read_shares_deadline_refusal():
    error = DeadlineExceeded("policy read deadline expired; owner commit state is unknown")
    result, calls = _supervision_error_result(error)
    failure = next(error for error in result["errors"] if error["class"] == "deadline_exceeded")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "failed" and failure["where"] == "collect"
    assert "no retry or budget extension is automatic" in failure["detail"]
    assert "owner commit state is unknown" in failure["detail"]


def test_agent_manager_supervision_partial_outcomes_keeps_deadline_cause():
    error = DeadlineExceeded("outcomes deadline expired; owner may have more rows")
    result, calls = _supervision_error_result(error, outcome_error=True)
    failure = next(error for error in result["errors"] if error["class"] == "deadline_exceeded")
    assert calls == ["policy", "outcomes"]
    assert result["status"] == "partial" and result["signals"]["outcomes_available"] is False
    assert result["evidence"] == ["policy-v1"]
    assert failure["where"] == "collect"
    assert "no retry or budget extension is automatic" in failure["detail"]
    assert "owner may have more rows" in failure["detail"]


def test_agent_manager_investigation_read_keeps_partial_deadline_cause_and_evidence():
    error = DeadlineExceeded("report deadline expired; transcript completeness is unknown")
    result, calls = _investigation_error_result(error)
    failure = next(error for error in result["errors"] if error["class"] == "deadline_exceeded")
    assert calls == ["report", "accounting"]
    assert result["status"] == "partial"
    assert result["signals"]["subjects"][0]["availability"] == "partial"
    assert result["evidence"] == ["run-1", "agent-manager/run/report", "agent-manager/run/accounting"]
    assert failure["where"] == "collect:run-1:report"
    assert "no retry or budget extension is automatic" in failure["detail"]
    assert "transcript completeness is unknown" in failure["detail"]


def _deployment_fault(program_file, inputs, error, endpoint, where):
    calls = []

    def owner_call(name):
        def invoke(**_kwargs):
            calls.append(name)
            raise error
        return invoke

    reviews = SimpleNamespace(
        get=owner_call("get"), policy_check=owner_call("policy_check"), prepare=owner_call("prepare"),
    )
    result = execute(
        REPO / "scenarios/deployment-manager/.vrooli/program-runtime/" / program_file,
        inputs, SimpleNamespace(),
        extra={
            "gather": lambda *calls: [call() for call in calls],
            "deployment_manager": SimpleNamespace(readiness_reviews=reviews),
        },
    )
    return result, calls, where


def _assert_deployment_failure(program_file, inputs, exception, klass, expected_detail, endpoint, where):
    result, calls, location = _deployment_fault(program_file, inputs, exception, endpoint, where)
    failure = next(error for error in result["errors"] if error["class"] == klass)
    assert calls == [endpoint]
    assert result["status"] == exception.status
    assert result["phase"] == "report" and failure["where"] == location
    assert expected_detail in failure["detail"]
    assert result["evidence"] == []
    assert "no retry was attempted" in failure["detail"] if klass == "scenario_unreachable" else "no retry or budget extension is automatic" in failure["detail"]


def test_deployment_manager_release_programs_share_unreachable_failure_handling():
    cases = (
        ("release-preflight.py", {"review_key": "review-1"}, "get", "collect"),
        ("release-observe.py", {"review_key": "review-1", "limit": 3}, "get", "collect"),
        ("readiness-review.py", {
            "scenario": "demo", "profile_id": "profile-1", "candidate_commit": "abc123",
            "artifact_digest": "sha256:fixture", "channel": "stable", "target": "demo",
            "authorization_epoch": 2, "facts": {"owner": "deploy"},
        }, "prepare", "act"),
    )
    for program_file, inputs, endpoint, where in cases:
        cause = f"{program_file} owner unreachable; deployment status is unknown"
        _assert_deployment_failure(program_file, inputs, ScenarioUnreachable(cause),
                                   "scenario_unreachable", "deployment status is unknown", endpoint, where)


def test_deployment_manager_release_programs_share_deadline_failure_handling():
    cases = (
        ("release-preflight.py", {"review_key": "review-1"}, "get", "collect"),
        ("release-observe.py", {"review_key": "review-1", "limit": 3}, "get", "collect"),
        ("readiness-review.py", {
            "scenario": "demo", "profile_id": "profile-1", "candidate_commit": "abc123",
            "artifact_digest": "sha256:fixture", "channel": "stable", "target": "demo",
            "authorization_epoch": 2, "facts": {"owner": "deploy"},
        }, "prepare", "act"),
    )
    for program_file, inputs, endpoint, where in cases:
        cause = f"{program_file} deadline expired; owner commit state is unknown"
        _assert_deployment_failure(program_file, inputs, DeadlineExceeded(cause),
                                   "deadline_exceeded", "owner commit state is unknown", endpoint, where)
