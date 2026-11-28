import ast
import contextlib
import hashlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

import pytest


KERNEL = Path(__file__).resolve().parents[1]
PROGRAMS = KERNEL.parent / ".vrooli/program-runtime"
AI_GATEWAY_PROGRAM = Path(__file__).resolve().parents[3] / "ai-gateway/.vrooli/program-runtime/classify-batch.json"
sys.path.insert(0, str(KERNEL))

from host.program_helper import ProgramHelper, Refused, RemoteError, ScenarioUnreachable  # noqa: E402
from host.engine import SessionKernel  # noqa: E402


PROGRAM_NAMES = ("batch-inference", "typed-inference")
AI_ARTIFACT_DIGEST = "0e07025e32853027f155dd4216da6f0a28a705b0c887f88fcdcf0fe35b76edd8"
WATCH_SET = PROGRAMS / "watch-set.py"
AI_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "ai-gateway/.vrooli/program-runtime/classify-batch.json").read_text())["outputs"]
VISITED_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "visited-tracker/.vrooli/program-runtime/attention-select.json").read_text())["outputs"]
AGENT_MANAGER_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "agent-manager/.vrooli/program-runtime/investigation-evidence.json").read_text())["outputs"]
PORTFOLIO_OUTPUTS = json.loads((PROGRAMS / "portfolio-audit.json").read_text())["outputs"]
INVESTIGATE_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "agent-manager/.vrooli/program-runtime/investigate.json").read_text())["outputs"]
UNIT_HEALTH_PROGRAMS = Path(__file__).resolve().parents[3] / "unit-health/.vrooli/program-runtime"
PLAN_MANAGER_PROGRAMS = Path(__file__).resolve().parents[3] / "plan-manager/.vrooli/program-runtime"
TECH_TREE_PROGRAMS = Path(__file__).resolve().parents[3] / "tech-tree-designer/.vrooli/program-runtime"
DEVICE_CONTROL_PROGRAMS = Path(__file__).resolve().parents[3] / "device-control/.vrooli/program-runtime"
SKILL_SET_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "prompt-manager/.vrooli/program-runtime/skill-set-read.json").read_text())["outputs"]
VALIDATION_DIGEST_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "test-genie/.vrooli/program-runtime/validation-digest.json").read_text())["outputs"]
FRICTION_DIGEST_OUTPUTS = json.loads((Path(__file__).resolve().parents[3] / "agent-manager/.vrooli/program-runtime/friction-digest.json").read_text())["outputs"]
UNIT_HEALTH_SETPOINT_PROGRAMS = Path(__file__).resolve().parents[3] / "unit-health/.vrooli/program-runtime"
MEMORY_CONSUMERS = {
    "browser-automation-studio": Path(__file__).resolve().parents[3] / "browser-automation-studio/.vrooli/program-runtime/learning-read.py",
    "knowledge-observatory": Path(__file__).resolve().parents[3] / "knowledge-observatory/.vrooli/program-runtime/learning-read.py",
}
WEB_SEARCH_PROGRAMS = Path(__file__).resolve().parents[3] / "web-search/.vrooli/program-runtime"
MEMORY_OUTPUTS = {
    "status": {"enum": ["ok", "partial", "unavailable", "refused", "failed"]},
    "phase": {"enum": ["validate", "collect", "report"]},
    "errors": {"classes": ["invalid_input", "scenario_unreachable", "binding_error", "invalid_response"]},
}
MEMORY_FINISH_OUTPUTS = {
    "status": {"enum": ["ok", "partial", "unavailable", "refused", "failed"]},
    "phase": {"enum": ["validate", "collect", "report"]},
    "errors": {"classes": ["invalid_input", "capture_failed", "remote_error", "no_governed_binding"]},
}


class Handle:
    def __init__(self, rows, metadata=None, raw_payload=None):
        self.rows = rows
        self.metadata = metadata
        self.raw_payload = raw_payload

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return len(self.rows)

    def filter(self, predicate):
        return Handle([row for row in self.rows if predicate(row)], metadata=self.metadata)

    def meta(self):
        return self.metadata or {"program": "ai-gateway.classify-batch", "digest": "fake-digest", "output_contract": AI_OUTPUTS}

    def raw(self):
        return self.raw_payload


class FakeGateway:
    def __init__(self, result=None, error=None, digest=AI_ARTIFACT_DIGEST):
        self.result = result
        self.error = error
        self.digest = digest
        self.calls = 0

    def classify_batch(self, **_kwargs):
        self.calls += 1
        if self.error:
            raise self.error
        return Handle(self.result, metadata={"program": "ai-gateway.classify-batch", "digest": self.digest,
                                             "output_contract": AI_OUTPUTS})


def execute(program_name, *, child=None, error=None, corpus=None, child_digest=AI_ARTIFACT_DIGEST):
    gateway = FakeGateway(result=child, error=error, digest=child_digest)
    namespace = {
        "inputs": {"corpus": ["sample"] if corpus is None else corpus, "labels": ["infra"]},
        "lib": SimpleNamespace(ai_gateway=gateway),
    }
    contract = json.loads((PROGRAMS / f"{program_name}.json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (PROGRAMS / f"{program_name}.py").read_text()
        exec(compile(source, f"{program_name}.py", "exec"), namespace)
    return json.loads(output.getvalue()), gateway.calls


def test_typed_inference_pin_matches_current_ai_gateway_artifact():
    contract_bytes = AI_GATEWAY_PROGRAM.read_bytes()
    source_bytes = AI_GATEWAY_PROGRAM.with_suffix(".py").read_bytes()
    contract = json.loads(contract_bytes)
    schema_bytes = b""
    if contract.get("output_schema"):
        schema_bytes = (AI_GATEWAY_PROGRAM.parent / contract["output_schema"]).read_bytes()
    digest = hashlib.sha256(contract_bytes + b"\0" + source_bytes + (b"\0" + schema_bytes if schema_bytes else b"")).hexdigest()
    parent_contract = json.loads((PROGRAMS / "typed-inference.json").read_text())
    parent_source = (PROGRAMS / "typed-inference.py").read_text()
    assert digest == AI_ARTIFACT_DIGEST
    assert AI_ARTIFACT_DIGEST in parent_source
    assert any(AI_ARTIFACT_DIGEST in invariant for invariant in parent_contract["invariants"])


def test_schema_invalid_grandchild_error_propagates_through_parent_library():
    outputs = {
        "status": {"enum": ["ok", "partial", "unavailable", "refused", "failed"]},
        "phase": {"enum": ["collect", "report"]},
        "errors": {"classes": ["invalid_response", "no_grant"]},
    }
    grandchild_schema = {
        "type": "object",
        "required": ["program", "version", "status", "phase", "signals", "errors", "evidence"],
        "properties": {
            "program": {"const": "fixture.grandchild"},
            "signals": {"type": "object", "required": ["count"],
                        "properties": {"count": {"type": "integer", "minimum": 0}}},
        },
        "additionalProperties": True,
    }
    grandchild = {
        "name": "grandchild", "scenario": "fixture", "contract": True, "current": True, "version": 1,
        "digest": "a" * 64,
        "declaration": {
            "name": "fixture.grandchild", "version": "1", "helpers": {"program": "2"}, "inputs": {},
            "outputs": outputs, "budget": {"output_bytes": 4096},
            "output_schema": "grandchild.output.schema.json", "_runtime_output_schema": grandchild_schema,
        },
        "source": "print({'program':'fixture.grandchild','version':'1','status':'ok','phase':'report','inputs':{},'signals':{'count':'wrong-type'},'errors':[],'evidence':[]})",
    }
    child = {
        "name": "child", "scenario": "fixture", "contract": True, "current": True, "version": 1,
        "digest": "b" * 64,
        "declaration": {"name": "fixture.child", "version": "1", "helpers": {"program": "2"}, "inputs": {},
                        "outputs": outputs, "budget": {"output_bytes": 4096}},
        "source": """envelope = program.envelope('fixture.child', '1', phase='collect')
try:
    result = program.child_envelope(lib.fixture.grandchild(), invalid_class='invalid_response', where='grandchild')
    envelope['status'] = result['status']
    envelope['errors'] = result['errors']
    envelope['evidence'] = result['evidence']
except Exception as exc:
    status, klass = program.classify(exc)
    program.fail(status, klass, str(exc), 'grandchild')
envelope['phase'] = 'report'
program.report()""",
    }
    parent_source = """envelope = program.envelope('fixture.parent', '1', phase='collect')
result = program.child_envelope(lib.fixture.child(), invalid_class='invalid_response', where='child')
envelope['status'] = result['status']
envelope['errors'] = result['errors']
envelope['evidence'] = result['evidence']
envelope['phase'] = 'report'
program.report()"""
    kernel = SessionKernel(libraries=[child, grandchild])
    result = kernel.execute(parent_source, declared_helper_contract={"version": "2", "outputs": outputs})

    assert result["ok"], result
    parent_envelope = json.loads(result["stdout"])
    assert parent_envelope["status"] == "failed"
    assert parent_envelope["errors"][0]["class"] == "invalid_response"
    assert "fixture.grandchild" in parent_envelope["errors"][0]["detail"]
    assert "$.signals.count does not match declared type" in parent_envelope["errors"][0]["detail"]


def execute_watch_set(*, child=None):
    gateway = FakeGateway(result=child)

    class FakeAgent:
        def start(self, **request):
            return request

        def collect(self, _handle, wait_seconds):
            assert wait_seconds == 120
            return Handle([{"execution_id": "run-1", "result": "done"}])

    namespace = {
        "inputs": {"requests": [{"owner": "demo", "workflow_key": "sample", "input": {}}], "labels": ["succeeded"]},
        "lib": SimpleNamespace(ai_gateway=gateway),
        "agent": FakeAgent(),
    }
    contract = json.loads((PROGRAMS / "watch-set.json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = WATCH_SET.read_text()
        exec(compile(source, "watch-set.py", "exec"), namespace)
    return json.loads(output.getvalue()), gateway.calls


def execute_improvement_evidence(*, children):
    class FakeLibrary:
        def __init__(self, child, output_contract, name):
            self.child = child
            self.output_contract = output_contract
            self.name = name

        def read(self, **_kwargs):
            return Handle([self.child], metadata={"name": self.name, "version": "1", "digest": "fake-digest", "output_contract": self.output_contract})

    contract = json.loads((PROGRAMS / "improvement-evidence.json").read_text())
    namespace = {
        "inputs": {"campaign_id": "11111111-1111-4111-8111-111111111111", "run_ids": ["run-1"], "limit": 1},
        "lib": SimpleNamespace(
            visited_tracker=SimpleNamespace(attention_select=FakeLibrary(children[0], VISITED_OUTPUTS, "visited-tracker.attention-select").read),
            agent_manager=SimpleNamespace(investigation_evidence=FakeLibrary(children[1], AGENT_MANAGER_OUTPUTS, "agent-manager.investigation-evidence").read),
        ),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (PROGRAMS / "improvement-evidence.py").read_text()
        exec(compile(source, "improvement-evidence.py", "exec"), namespace)
    raw = output.getvalue()
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return ast.literal_eval(raw)


def execute_setpoint_read(*, child):
    def unavailable(*_args, **_kwargs):
        raise RuntimeError("fixture dependency unavailable")

    contract = json.loads((PROGRAMS / "setpoint-read.json").read_text())
    portfolio_handle = Handle([child], metadata={"program": "program-runtime.portfolio-audit", "version": "1", "digest": "fake-digest", "output_contract": PORTFOLIO_OUTPUTS})
    runtime = SimpleNamespace(
        programs=SimpleNamespace(governance_share=unavailable, mine=unavailable, list=unavailable, portfolio=unavailable),
        bindings=SimpleNamespace(act=unavailable, condition=unavailable),
        sessions=SimpleNamespace(delegations=unavailable),
        library=SimpleNamespace(list=unavailable),
        shapes=SimpleNamespace(list=unavailable),
    )
    namespace = {
        "inputs": {},
        "program_runtime": runtime,
        "lib": SimpleNamespace(program_runtime=SimpleNamespace(portfolio_audit=lambda **_kwargs: portfolio_handle)),
        "tasks": SimpleNamespace(learning_findings=unavailable, delivery_metrics=unavailable, fragment_list=unavailable),
        "Handle": Handle,
        "gather": lambda *workers: [worker() for worker in workers],
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    guarded_calls = []
    original_guarded = helper.guarded
    def guarded(call):
        guarded_calls.append(call)
        return original_guarded(call)
    helper.guarded = guarded
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (PROGRAMS / "setpoint-read.py").read_text()
        exec(compile(source, "setpoint-read.py", "exec"), namespace)
    return json.loads(output.getvalue()), len(guarded_calls)


def execute_browser_setpoint_read(*, child):
    contract_path = Path(__file__).resolve().parents[3] / "browser-automation-studio/.vrooli/program-runtime/setpoint-read.json"
    contract = json.loads(contract_path.read_text())
    friction = Handle([child], metadata={"program": "agent-manager.friction-digest", "version": "1", "digest": "fixture-digest", "output_contract": FRICTION_DIGEST_OUTPUTS})
    measures = SimpleNamespace(
        pass_rate=lambda **_kwargs: Handle([{"rate": 0.95}]),
        p95_duration=lambda **_kwargs: Handle([{"durationMs": 1000}]),
        step_failure_rate=lambda **_kwargs: Handle([{"rate": 0.05}]),
        selector_failure_rate=lambda **_kwargs: Handle([{"rate": 0.05}]),
    )
    browser = SimpleNamespace(executions=SimpleNamespace(list=lambda **_kwargs: Handle([]), screenshots=lambda **_kwargs: Handle([])), measures=measures)
    namespace = {
        "inputs": {"window": 10, "evidence_sample": 0},
        "browser_automation_studio": browser,
        "lib": SimpleNamespace(agent_manager=SimpleNamespace(friction_digest=lambda **_kwargs: friction)),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (contract_path.with_suffix(".py")).read_text()
        exec(compile(source, "browser-automation-studio/setpoint-read.py", "exec"), namespace)
    return json.loads(output.getvalue())


def execute_test_quality_sample(*, child):
    contract = json.loads((UNIT_HEALTH_PROGRAMS / "test-quality-sample.json").read_text())
    gateway = FakeGateway(result=child)

    class BodyHandle:
        def raw(self):
            return {"bodyExcerpt": "assert response.status == 200", "bodyBytes": 30, "redactions": 0}

    owner = SimpleNamespace(validate=SimpleNamespace(test_body=lambda **_kwargs: BodyHandle()))
    namespace = {
        "inputs": {
            "scenario": "unit-health", "seed": "seed-1", "source_identity": "sha256:fixture",
            "sample_size": 1, "candidates": [{"workspace": "api", "file": "api_test.go", "testId": "TestResponse", "framework": "go", "testKind": "unit", "staticStatus": "violation"}],
            "review_ai": True, "include_controls": False,
        },
        "unit_health": owner,
        "lib": SimpleNamespace(ai_gateway=gateway),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (UNIT_HEALTH_PROGRAMS / "test-quality-sample.py").read_text()
        exec(compile(source, "unit-health/test-quality-sample.py", "exec"), namespace)
    return json.loads(output.getvalue()), gateway.calls


def execute_plan_manager_investigate(*, child_rows):
    contract = json.loads((PLAN_MANAGER_PROGRAMS / "investigate.json").read_text())
    handle = Handle(child_rows, metadata={"program": "agent-manager.investigate", "version": "1", "digest": "fake-digest", "output_contract": INVESTIGATE_OUTPUTS})
    namespace = {
        "inputs": {
            "caller_key": "fixture-key", "execution_id": "execution-1", "phase_id": "phase-1", "phase_generation": "1",
            "question": "Why did the phase fail?", "run_ids": ["run-1"], "wait_seconds": 30,
        },
        "lib": SimpleNamespace(agent_manager=SimpleNamespace(investigate=lambda **_kwargs: handle)),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (PLAN_MANAGER_PROGRAMS / "investigate.py").read_text()
        exec(compile(source, "plan-manager/investigate.py", "exec"), namespace)
    return json.loads(output.getvalue())


def execute_tech_tree_setpoint(*, child):
    contract = json.loads((TECH_TREE_PROGRAMS / "setpoint-read.json").read_text())
    skill_handle = Handle([child], metadata={"program": "prompt-manager.skill-set-read", "version": "1", "digest": "fake-digest", "output_contract": SKILL_SET_OUTPUTS})
    namespace = {
        "inputs": {"collect_diagnostics": True},
        "tech_tree_designer": SimpleNamespace(
            plan=SimpleNamespace(list=lambda: Handle([])),
            ontology=SimpleNamespace(coverage=lambda **_kwargs: Handle([], metadata={"totalCapabilities": 12, "totalScenarios": 4})),
        ),
        "lib": SimpleNamespace(prompt_manager=SimpleNamespace(skill_set_read=lambda **_kwargs: skill_handle)),
        "gather": lambda *workers: [worker() for worker in workers],
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (TECH_TREE_PROGRAMS / "setpoint-read.py").read_text()
        exec(compile(source, "tech-tree-designer/setpoint-read.py", "exec"), namespace)
    return json.loads(output.getvalue())


def execute_unit_health_setpoint(*, validation_child, friction_child):
    contract = json.loads((UNIT_HEALTH_SETPOINT_PROGRAMS / "setpoint-read.json").read_text())

    def unavailable(*_args, **_kwargs):
        raise RuntimeError("fixture dependency unavailable")

    validation = Handle([], raw_payload={"status": "ok", "workspaces": [{"workspace": "api"}], "runId": "validation-1"})
    validation_digest = Handle([validation_child], metadata={"program": "test-genie.validation-digest", "version": "1", "digest": "fixture-digest", "output_contract": VALIDATION_DIGEST_OUTPUTS})
    friction = Handle([friction_child], metadata={"program": "agent-manager.friction-digest", "version": "1", "digest": "fixture-digest", "output_contract": FRICTION_DIGEST_OUTPUTS})
    namespace = {
        "inputs": {"roster": ["unit-health"]},
        "unit_health": SimpleNamespace(validate=SimpleNamespace(scenario=lambda **_kwargs: validation), calibrate=SimpleNamespace(run=unavailable)),
        "lib": SimpleNamespace(
            test_genie=SimpleNamespace(validation_digest=lambda **_kwargs: validation_digest),
            agent_manager=SimpleNamespace(friction_digest=lambda **_kwargs: friction),
        ),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (UNIT_HEALTH_SETPOINT_PROGRAMS / "setpoint-read.py").read_text()
        exec(compile(source, "unit-health/setpoint-read.py", "exec"), namespace)
    return json.loads(output.getvalue())


def execute_memory_learning_read(scenario, *, child):
    program_path = MEMORY_CONSUMERS[scenario]
    contract_path = program_path.with_suffix(".json")
    contract = json.loads(contract_path.read_text())
    handle = Handle([child], metadata={"program": "vrooli-memory.compare-outcomes", "version": "1", "digest": "fake-digest", "output_contract": MEMORY_OUTPUTS})
    namespace = {"inputs": {}, "lib": SimpleNamespace(vrooli_memory=SimpleNamespace(compare_outcomes=lambda **_kwargs: handle))}
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = program_path.read_text()
        exec(compile(source, f"{scenario}/learning-read.py", "exec"), namespace)
    raw = output.getvalue()
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return ast.literal_eval(raw)


def execute_device_control_setpoint(*, child):
    contract = json.loads((DEVICE_CONTROL_PROGRAMS / "setpoint-read.json").read_text())
    memory = Handle([child], metadata={"program": "vrooli-memory.compare-outcomes", "version": "1", "digest": "fake-digest", "output_contract": MEMORY_OUTPUTS})
    condition = Handle([{"status": "CONDITION_STATUS_HEALTHY"}])
    namespace = {
        "inputs": {},
        "lib": SimpleNamespace(vrooli_memory=SimpleNamespace(compare_outcomes=lambda **_kwargs: memory)),
        "program_runtime": SimpleNamespace(bindings=SimpleNamespace(condition=lambda **_kwargs: condition)),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (DEVICE_CONTROL_PROGRAMS / "setpoint-read.py").read_text()
        exec(compile(source, "device-control/setpoint-read.py", "exec"), namespace)
    return json.loads(output.getvalue())


def execute_web_search_record_attempt(*, child):
    contract = json.loads((WEB_SEARCH_PROGRAMS / "record-attempt.json").read_text())
    handle = Handle([child], metadata={"program": "vrooli-memory.finish-attempt", "version": "1", "digest": "fake-digest", "output_contract": MEMORY_FINISH_OUTPUTS})
    calls = {"count": 0}

    def finish_attempt(**_kwargs):
        calls["count"] += 1
        return handle

    namespace = {
        "inputs": {"attempt": {"attempt_id": "attempt-1", "outcome": "failure", "approach": "bounded search"}, "quality": {}, "used_finding_ids": []},
        "lib": SimpleNamespace(vrooli_memory=SimpleNamespace(finish_attempt=finish_attempt)),
    }
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        source = (WEB_SEARCH_PROGRAMS / "record-attempt.py").read_text()
        exec(compile(source, "web-search/record-attempt.py", "exec"), namespace)
    return json.loads(output.getvalue()), calls["count"]


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("status", ["ok", "partial", "failed", "unavailable", "refused"])
def test_child_envelope_status_errors_signals_and_evidence_are_preserved(program_name, status):
    child = {
        "status": status,
        "phase": "report",
        "signals": {"results": [{"index": 0, "label": None}], "usage": None},
        "errors": [{"class": "inference_failed", "detail": "fixture", "where": "item:0"}],
        "evidence": ["ai-gateway:batch:fixture"],
    }

    result, calls = execute(program_name, child=[child])

    assert calls == 1
    assert result["status"] == status
    assert result["signals"] == child["signals"]
    assert result["errors"] == child["errors"]
    assert result["evidence"] == child["evidence"] + [{
        "program": "ai-gateway.classify-batch",
        "artifact": {"program": "ai-gateway.classify-batch", "digest": AI_ARTIFACT_DIGEST, "output_contract": AI_OUTPUTS},
    }]


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("error, status, klass", [
    (RemoteError("remote status 400: fixture", http_status=400), "failed", "remote_error"),
    (ScenarioUnreachable("scenario_not_running"), "unavailable", "scenario_unreachable"),
    (Refused("grant required", klass="no_grant"), "refused", "no_grant"),
])
def test_typed_child_call_errors_keep_runtime_classification(program_name, error, status, klass):
    result, calls = execute(program_name, error=error)

    assert calls == 1
    assert result["status"] == status
    assert result["errors"][0]["class"] == klass
    assert result["errors"][0]["where"] == "classify"


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
def test_empty_child_envelope_is_missing_result(program_name):
    result, calls = execute(program_name, child=[])

    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "missing_result"


@pytest.mark.parametrize("program_name", PROGRAM_NAMES)
@pytest.mark.parametrize("child", [None, {"status": "ok"}, {"status": "unknown", "signals": {}, "errors": [], "evidence": []}])
def test_malformed_child_envelope_is_invalid_response(program_name, child):
    result, calls = execute(program_name, child=[child])

    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"


def test_typed_inference_refuses_drifted_child_artifact():
    child = {"status": "ok", "phase": "report", "signals": {"results": [], "usage": None}, "errors": [], "evidence": []}
    result, calls = execute("typed-inference", child=[child], child_digest="f" * 64)
    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"
    assert "does not match its declared pin" in result["errors"][0]["detail"]


@pytest.mark.parametrize("program_name, invalid_corpus", [
    ("batch-inference", ["sample"] * 33),
    ("typed-inference", ["sample"] * 9),
])
def test_invalid_corpus_stops_before_child_call(program_name, invalid_corpus):
    result, calls = execute(program_name, child=[], corpus=invalid_corpus)

    assert calls == 0
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_input"


@pytest.mark.parametrize("status", ["ok", "partial", "failed", "unavailable", "refused"])
def test_watch_set_preserves_child_envelope_status_and_errors(status):
    child = [{
        "status": status,
        "phase": "report",
        "signals": {"results": [{"index": 0, "label": "succeeded", "validated": True, "error": None}], "by_label": {"succeeded": 1}, "usage": None},
        "errors": [{"class": "inference_failed", "detail": "fixture", "where": "item:0"}],
        "evidence": ["ai-gateway:batch:fixture"],
    }]
    result, calls = execute_watch_set(child=child)
    assert calls == 1
    assert result["status"] == status
    assert result["signals"]["runs"] == [{"index": 0, "label": "succeeded", "validated": True, "error": None}]
    assert result["errors"] == [{"class": "inference_failed", "detail": "fixture", "where": "item:0"}]


@pytest.mark.parametrize("child, error_class", [([], "missing_result"), ([{"status": "ok", "phase": "report", "signals": {}, "errors": [], "evidence": []}], "invalid_response")])
def test_watch_set_rejects_empty_or_malformed_child_payload(child, error_class):
    result, calls = execute_watch_set(child=child)
    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == error_class


def test_improvement_evidence_preserves_valid_child_statuses_and_metadata():
    children = [
        {"program": "visited-tracker.attention-select", "version": "1", "status": "partial", "phase": "report", "signals": {"candidates": []}, "errors": [], "evidence": []},
        {"program": "agent-manager.investigation-evidence", "version": "1", "status": "unavailable", "phase": "report", "signals": {"runs": []}, "errors": [], "evidence": []},
    ]
    result = execute_improvement_evidence(children=children)
    assert result["status"] == "partial"
    assert [child["status"] for child in result["signals"]["children"]] == ["partial", "unavailable"]
    assert result["evidence"] == [
        {"program": "visited-tracker.attention-select", "artifact": {"name": "visited-tracker.attention-select", "version": "1", "digest": "fake-digest"}},
        {"program": "agent-manager.investigation-evidence", "artifact": {"name": "agent-manager.investigation-evidence", "version": "1", "digest": "fake-digest"}},
    ]


def test_improvement_evidence_exposes_invalid_child_contract_as_parent_error():
    children = [
        {"program": "visited-tracker.attention-select", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []},
        {"program": "agent-manager.investigation-evidence", "version": "1", "status": "ok", "phase": "report", "signals": {"runs": []}, "errors": [], "evidence": []},
    ]
    result = execute_improvement_evidence(children=children)
    assert result["status"] == "partial"
    child_error = result["signals"]["children"][0]["errors"][-1]
    assert child_error == {"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "collect:visited-tracker.attention-select"}
    assert result["errors"] == [child_error]


def test_setpoint_read_uses_declared_portfolio_child_status():
    child = {"program": "program-runtime.portfolio-audit", "version": "1", "status": "ok", "phase": "report", "signals": {"score": 0.9, "scored": 8}, "errors": [], "evidence": []}
    result, guarded_count = execute_setpoint_read(child=child)
    assert guarded_count == 14
    row = next(row for row in result["signals"]["rows"] if row["row"] == "portfolio-maturity")
    assert result["status"] == "partial"
    assert row["reading"] == {"score": 0.9, "scored": 8}
    assert not any(issue["class"] == "invalid_response" for issue in result["errors"])


def test_setpoint_read_preserves_malformed_portfolio_child_error():
    child = {"program": "program-runtime.portfolio-audit", "version": "1", "status": "unknown", "phase": "report", "signals": {"score": 0.9}, "errors": [], "evidence": []}
    result, guarded_count = execute_setpoint_read(child=child)
    assert guarded_count == 14
    row = next(row for row in result["signals"]["rows"] if row["row"] == "portfolio-maturity")
    assert row["unavailable"]
    child_error = next(issue for issue in result["errors"] if issue["class"] == "invalid_response")
    assert child_error == {"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "classify:portfolio-audit"}


def test_test_quality_sample_preserves_ai_gateway_child_classification():
    child = [{"program": "ai-gateway.classify-batch", "version": "1", "status": "ok", "phase": "report", "signals": {"results": [{"index": 0, "label": "behavioral"}]}, "errors": [], "evidence": []}]
    result, calls = execute_test_quality_sample(child=child)
    assert calls == 1
    assert result["status"] == "ok"
    assert result["signals"]["observations"][0]["label"] == "behavioral"
    assert not result["errors"]


def test_test_quality_sample_reports_malformed_ai_child_envelope():
    child = [{"program": "ai-gateway.classify-batch", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}]
    result, calls = execute_test_quality_sample(child=child)
    assert calls == 1
    assert result["status"] == "partial"
    assert result["signals"]["observations"][0]["status"] == "uncertain"
    assert result["errors"] == [{"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "classify:review"}]


def test_plan_manager_investigation_preserves_nested_status_and_diagnostic():
    child = {"program": "agent-manager.investigate", "version": "1", "status": "partial", "phase": "report", "signals": {"pending": True, "investigation_id": "investigation-1"}, "errors": [{"class": "nested_failed", "detail": "one run incomplete", "where": "collect"}], "evidence": ["investigation-1"]}
    result = execute_plan_manager_investigate(child_rows=[child])
    assert result["status"] == "partial"
    assert result["signals"]["pending"] is True
    assert result["signals"]["nested_status"] == "partial"
    assert result["errors"] == child["errors"]


def test_plan_manager_investigation_preserves_shared_missing_child_class():
    result = execute_plan_manager_investigate(child_rows=[])
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "no_output"


def test_plan_manager_investigation_normalizes_malformed_child_contract():
    child = {"program": "agent-manager.investigate", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_plan_manager_investigate(child_rows=[child])
    assert result["status"] == "failed"
    assert result["signals"]["nested_status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"


def test_tech_tree_setpoint_preserves_child_contract_and_complete_readings():
    row_names = ["registered-under-scenario-pack", "usage-id-present", "improve-id-present", "set-token-size", "read-counts", "programs-declared", "programs-named-in-usage-skill"]
    child = {"program": "prompt-manager.skill-set-read", "version": "1", "status": "ok", "phase": "report", "signals": {"usage_present": True, "improve_present": True, "rows": [{"row": name, "unavailable": False, "reason": None, "reading": {"value": True}} for name in row_names]}, "errors": [], "evidence": []}
    result = execute_tech_tree_setpoint(child=child)
    assert result["status"] == "ok"
    assert result["signals"]["child"]["status"] == "ok"
    assert result["signals"]["diagnostics"][-1]["reading"] == {"usage_present": True, "improve_present": True}


def test_tech_tree_setpoint_reports_malformed_child_contract():
    child = {"program": "prompt-manager.skill-set-read", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_tech_tree_setpoint(child=child)
    assert result["status"] == "partial"
    assert result["signals"]["child"]["status"] == "failed"
    child_error = next(issue for issue in result["errors"] if issue["class"] == "invalid_response")
    assert child_error == {"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "classify:skill-registry"}


def test_unit_health_setpoint_uses_shared_envelopes_for_validation_and_friction_children():
    validation_child = {"program": "test-genie.validation-digest", "version": "1", "status": "ok", "phase": "report", "signals": {"terminal": 1, "with_evidence": 1}, "errors": [], "evidence": []}
    friction_child = {"program": "agent-manager.friction-digest", "version": "1", "status": "ok", "phase": "report", "signals": {"recurring_count": 0}, "errors": [], "evidence": []}
    result = execute_unit_health_setpoint(validation_child=validation_child, friction_child=friction_child)
    assert result["signals"]["rows"]
    assert result["signals"]["readable"] > 0
    assert not any(issue["class"] == "invalid_response" for issue in result["errors"])


def test_unit_health_setpoint_surfaces_malformed_child_envelope():
    validation_child = {"program": "test-genie.validation-digest", "version": "1", "status": "ok", "phase": "report", "signals": {"terminal": 1, "with_evidence": 1}, "errors": [], "evidence": []}
    friction_child = {"program": "agent-manager.friction-digest", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_unit_health_setpoint(validation_child=validation_child, friction_child=friction_child)
    child_error = next(issue for issue in result["errors"] if issue["class"] == "invalid_response")
    assert child_error == {"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "collect:agent-manager-friction-digest"}


def test_browser_setpoint_read_preserves_friction_child_error_detail_and_where():
    child = {"program": "agent-manager.friction-digest", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_browser_setpoint_read(child=child)
    assert result["errors"] == [{"class": "invalid_response", "detail": "child status is missing or outside its declared enum", "where": "collect:external-friction"}]


@pytest.mark.parametrize("scenario", ["browser-automation-studio", "knowledge-observatory"])
def test_memory_learning_read_preserves_child_status_and_reliability(scenario):
    child = {"program": "vrooli-memory.compare-outcomes", "version": "1", "status": "partial", "phase": "report", "signals": {"rows": [], "reliability": {"reliable": False, "reason": "fixture"}}, "errors": [], "evidence": ["memory:fixture"]}
    result = execute_memory_learning_read(scenario, child=child)
    assert result["status"] == "partial"
    assert result["signals"]["reliability"] == child["signals"]["reliability"]
    assert result["evidence"] == child["evidence"]


@pytest.mark.parametrize("scenario", ["browser-automation-studio", "knowledge-observatory"])
def test_memory_learning_read_preserves_malformed_child_contract_error(scenario):
    child = {"program": "vrooli-memory.compare-outcomes", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_memory_learning_read(scenario, child=child)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"


def test_device_control_setpoint_uses_shared_memory_child_envelope():
    child = {"program": "vrooli-memory.compare-outcomes", "version": "1", "status": "partial", "phase": "report", "signals": {"rows": [{"row": "advice-outcomes", "reading": {"applied": 1, "rejected": 1}}]}, "errors": [], "evidence": ["memory:fixture"]}
    result = execute_device_control_setpoint(child=child)
    assert result["status"] == "partial"
    assert result["signals"]["rows"][0]["row"] == "advice-outcomes"
    assert result["evidence"] == child["evidence"]


def test_device_control_setpoint_keeps_malformed_child_invalid_response():
    child = {"program": "vrooli-memory.compare-outcomes", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result = execute_device_control_setpoint(child=child)
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"


def test_web_search_attempt_preserves_write_child_uncertainty_without_retry():
    child = {"program": "vrooli-memory.finish-attempt", "version": "1", "status": "partial", "phase": "report", "signals": {"entry_id": "attempt-memory-1", "capture_status": "capture_failed"}, "errors": [{"class": "capture_failed", "detail": "fixture", "where": "finish"}], "evidence": ["memory:attempt-1"]}
    result, calls = execute_web_search_record_attempt(child=child)
    assert calls == 1
    assert result["status"] == "partial"
    assert result["errors"] == child["errors"]


def test_web_search_attempt_preserves_malformed_write_child_error_without_retry():
    child = {"program": "vrooli-memory.finish-attempt", "version": "1", "status": "unknown", "phase": "report", "signals": {}, "errors": [], "evidence": []}
    result, calls = execute_web_search_record_attempt(child=child)
    assert calls == 1
    assert result["status"] == "failed"
    assert result["errors"][0]["class"] == "invalid_response"
