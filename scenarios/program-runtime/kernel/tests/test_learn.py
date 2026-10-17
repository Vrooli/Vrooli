"""Focused contract tests for the in-program learning verb surface."""
import sys
import contextvars
from types import SimpleNamespace
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).parents[1]))

from host.engine import Handle, SessionKernel  # noqa: E402
from host.learn import Learn, StepFailed  # noqa: E402


def test_learn_allocates_identity_lazily_and_emits_a_receipt():
    kernel = SessionKernel()
    result = kernel.execute(
        "learn.task(scope='demo', operation='demo.program', key={'device': 'tv'})\n"
        "learn.note('preference', {'option_id': 'quiet'})\n"
        "learn.outcome('verified_success', ['observed:ok'])",
        program_id="demo.program",
        provenance="agent",
    )

    assert result["ok"]
    receipt = result["learning"]
    assert receipt["task_id"].startswith("task-")
    assert receipt["attempt_id"].startswith("attempt-")
    assert receipt["resume_token"].startswith("prt_resume_v1_")
    assert receipt["outcome"] == "verified_success"


def test_invalid_learning_calls_do_not_start_a_checkpoint():
    import contextvars

    # Constructing a real ContextVar keeps this test independent of SessionKernel.
    learn = Learn(object(), contextvars.ContextVar("ctx", default={}), lambda: {})

    with pytest.raises(ValueError, match="invalid learning outcome"):
        learn.outcome("not-a-status")
    assert not learn.started

    with pytest.raises(ValueError, match="unknown learning note kind"):
        learn.note("not-a-kind", {})
    assert not learn.started


def test_step_records_unknown_when_no_outcome_is_set():
    kernel = SessionKernel()
    result = kernel.execute("with learn.step('later'):\n    pass")
    assert result["ok"]
    assert result["learning"]["steps"][0]["name"] == "later"
    assert result["learning"]["steps"][0]["outcome"] == "unknown"


def test_foreign_scope_requires_a_declared_memory_write():
    import contextvars

    learn = Learn(object(), contextvars.ContextVar("ctx", default={}), lambda: {}, "demo.program")
    with pytest.raises(ValueError, match="foreign learning scope"):
        learn.task(scope="other-usage")

    allowed = Learn(object(), contextvars.ContextVar("ctx2", default={}), lambda: {}, "demo.program",
                    bindings=({"id": "other/learning/record", "effect": "write"},))
    assert allowed.task(scope="other-usage")["scope"] == "other-usage"


def test_browser_automation_scope_alias_is_first_class():
    context = contextvars.ContextVar("bas_scope_ctx", default={})
    learn = Learn(object(), context, lambda: {}, "browser-automation-studio.do-task")
    assert learn.task(scope="bas-usage")["scope"] == "bas-usage"


def test_analyzer_reports_learning_calls():
    from host import analyze

    result = analyze.analyze("learn.task()\nlearn.note('trace', {})")
    assert result["learn_calls"] == [
        {"verb": "task", "line": 1},
        {"verb": "note", "line": 2},
    ]


def test_gather_registers_children_under_the_active_step():
    kernel = SessionKernel()
    source = """
def child(name):
    with learn.step(name) as current:
        current.outcome('verified_success', ['step:' + name])
with learn.step('parent'):
    gathered = gather(lambda: child('one'), lambda: child('two'), lambda: child('three'))
learn.outcome('verified_success', ['run:ok'])
"""
    result = kernel.execute(source, program_id="demo.program", provenance="agent")
    assert result["ok"], result
    steps = result["learning"]["steps"]
    assert {step["name"] for step in steps} == {"parent", "one", "two", "three"}


def test_step_depth_and_count_bounds_are_recorded():
    kernel = SessionKernel()
    # Keep this assertion focused on the public receipt rather than the
    # generated source indentation: a fifth nested step is bounded.
    result = kernel.execute("with learn.step('one'):\n    with learn.step('two'):\n        with learn.step('three'):\n            with learn.step('four'):\n                with learn.step('five'):\n                    pass\n")
    assert result["ok"], result
    bounded = [step for step in result["learning"]["steps"] if step["name"] == "five"]
    assert bounded and bounded[0]["bound"] and bounded[0]["error_class"] == "learning_bound"


def test_recall_keeps_summaries_filters_since_and_supports_zoom():
    zoom_calls = []
    rows = Handle([
        {"entry_id": "summary-1", "text": "summary", "summary": "summary", "depth": "summary", "node_id": "node-1", "created_at": "2026-09-09T00:00:00Z"},
        {"entry_id": "leaf-1", "text": "leaf", "kind": "preference", "body": {"option_id": "safe"}, "created_at": "2026-09-08T00:00:00Z"},
    ], "recall")
    memory = SimpleNamespace(recall=SimpleNamespace(
        recall=lambda **kwargs: rows,
        zoom=lambda **kwargs: zoom_calls.append(kwargs) or Handle([{"entry_id": "leaf-1"}], "zoom"),
    ))
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=memory)
    context = contextvars.ContextVar("recall_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {"device": "tv"}, "demo.program")
    result = learn.recall(query="tv", depth="tree", since="2026-09-08T00:00:00Z")
    assert [hit["entry_id"] for hit in result["hits"]] == ["summary-1", "leaf-1"]
    assert result["notes"]["preference"][0]["option_id"] == "safe"
    result["hits"][0].zoom()
    assert zoom_calls == [{"node_id": "node-1", "limit": 20}]


def test_infer_routes_by_schema_and_validates_output():
    calls = []

    class AI:
        def classify(self, **kwargs):
            calls.append("classify")
            return Handle([{"label": "safe"}], "classify")

        def extract(self, **kwargs):
            calls.append("extract")
            return Handle([{"label": "safe"}], "extract")

        def judge(self, **kwargs):
            calls.append("judge")
            return Handle([True], "judge")

    owner = SimpleNamespace(_bridge_url="", ai=AI(), vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle([]))))
    context = contextvars.ContextVar("infer_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "demo.program")
    assert learn.infer("classify", "choose", {}, {"type": "object", "properties": {"label": {"enum": ["safe"]}}, "required": ["label"]}) == {"label": "safe"}
    assert learn.infer("judge", "is safe", {}, {"type": "boolean"}) is True
    assert calls == ["classify", "judge"]


def test_choose_uses_evidence_weight_and_derives_applied_verdict():
    rows = Handle([
        {"entry_id": "a-1", "kind": "preference", "body": {"option_id": "safe"}, "supported": 2, "created_at": "2026-09-09T00:00:00Z"},
        {"entry_id": "a-2", "kind": "preference", "body": {"option_id": "fast"}, "contradicted": 1, "created_at": "2026-09-09T00:00:00Z"},
    ])
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: rows)))
    context = contextvars.ContextVar("choose_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "demo.program")
    chosen = learn.choose(["safe", "fast"], "fast")
    assert chosen["selected_id"] == "safe"
    assert {why["entry_id"] for why in chosen["why"]} == {"a-1", "a-2"}
    learn.outcome("verified_success", ["choice:applied"])
    assert learn._decisions[0]["verdict"] == "supported"
    assert learn._decisions[0]["derived"] is True


def test_infer_replays_examples_and_retains_only_verified_examples():
    instructions = []

    class AI:
        def classify(self, **kwargs):
            instructions.append(kwargs.get("instruction", ""))
            return Handle([{"label": "safe"}])

    memory = SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle([
        {"entry_id": "example-1", "kind": "example", "text": "prior example", "body": {"output": {"label": "safe"}}}
    ])))
    owner = SimpleNamespace(_bridge_url="", ai=AI(), vrooli_memory=memory)
    context = contextvars.ContextVar("example_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {"prompt": "free"}, "demo.program", free_text_inputs=("prompt",))
    learn.infer("classify", "choose", {"prompt": "free"}, {"type": "object", "properties": {"label": {"enum": ["safe"]}}, "required": ["label"]}, verify=lambda value: ("verified_success", ["label:safe"]) if value == {"label": "safe"} else ("failed", []))
    learn.outcome("verified_success", ["inference:ok"])
    receipt = learn.finalize({"status": "ok"})["learning"]
    assert receipt["steps"][0]["demos_used"] == 1
    assert "prior example" in instructions[0]
    assert receipt["notes"][0]["kind"] == "example"

    failed = Learn(owner, context, lambda: {}, "demo.program")
    failed.infer("classify", "choose", {}, {"type": "object", "properties": {"label": {"enum": ["safe"]}}, "required": ["label"]})
    failed.outcome("failed")
    assert not failed.finalize({"status": "failed"})["learning"]["notes"]


def test_act_defers_cache_until_verified_and_stops_after_write_effect():
    model_calls = []

    class AI:
        def write(self, **kwargs):
            model_calls.append(1)
            fragment = "def step(inputs, bindings):\n    return {'ok': True}"
            if "write then fail" in kwargs.get("source", ""):
                fragment = "def step(inputs, bindings):\n    bindings.demo.work.run()\n    raise StepFailed('boom')"
            return Handle([{"fragment": fragment}])

    owner = SimpleNamespace(_bridge_url="", _invocations=[], ai=AI())
    context = contextvars.ContextVar("act_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    Learn._fragment_cache.clear()
    learn = Learn(owner, context, lambda: {"key": "same"}, "demo.program")
    learn.act("hole", "return ok", {}, {"type": "object", "required": ["ok"]}, ["demo/work/run"], verify=lambda value: ("verified_success", ["ok:true"]) if value.get("ok") is True else ("failed", []), verifier_revision="ok-v1", min_verified=1, min_contexts=1)
    assert not Learn._fragment_cache
    learn.outcome("verified_success", ["act:ok"])
    learn.finalize({"status": "ok"})
    again = Learn(owner, context, lambda: {"key": "same"}, "demo.program")
    cached = again.act("hole", "return ok", {}, {"type": "object", "required": ["ok"]}, ["demo/work/run"], verify=lambda value: ("verified_success", ["ok:true"]) if value.get("ok") is True else ("failed", []), verifier_revision="ok-v1", min_verified=1, min_contexts=1)
    assert cached["fragment_source"] == "cache" and cached["model_calls"] == 0
    assert len(model_calls) == 1

    class WritingBinding:
        effect = "write"

        def __call__(self):
            owner._invocations.append({"binding_id": "demo/work/run", "effect": "write"})
            return Handle([])

    owner.demo = SimpleNamespace(work=SimpleNamespace(run=WritingBinding()))
    failing = Learn(owner, context, lambda: {}, "demo.program")
    with pytest.raises(StepFailed, match="uncertain_effects"):
        failing.act("write-hole", "write then fail", {}, {"type": "object"}, ["demo/work/run"], verify=lambda value: ("verified_success", ["ok:true"]) if value.get("ok") is True else ("failed", []), verifier_revision="ok-v1", min_verified=1, min_contexts=1)
    assert failing._steps[next(iter(failing._steps))]["error_class"] == "uncertain_effects"


def test_section_key_pools_fragment_evidence_across_differing_task_keys():
    """A task-independent section must not restart its cache every run.

    Without a section key the fragment inherits the task key, so a program keyed per session or
    per prompt digest never reaches min_verified. With one, evidence pools across tasks; the
    step attempt record stays keyed by the task.
    """
    written = []

    class AI:
        def write(self, **kwargs):
            written.append(1)
            return Handle([{"fragment": "def step(inputs, bindings):\n    return {'ok': True}"}])

    owner = SimpleNamespace(_bridge_url="", _invocations=[], ai=AI())
    context = contextvars.ContextVar("section_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    verify = lambda value: ("verified_success", ["ok:true"]) if value.get("ok") is True else ("failed", [])
    section = dict(intent="return ok", schema={"type": "object", "required": ["ok"]},
                   verify=verify, verifier_revision="ok-v1", min_verified=1, min_contexts=1)

    def run(task_key, **extra):
        learn = Learn(owner, context, lambda: {"task": task_key}, "demo.program")
        acted = learn.act("hole", section["intent"], {}, section["schema"], [],
                          verify=section["verify"], verifier_revision=section["verifier_revision"],
                          min_verified=1, min_contexts=1, **extra)
        learn.outcome("verified_success", ["act:ok"])
        learn.finalize({"status": "ok"})
        return acted

    Learn._fragment_cache.clear()
    run("task-one")
    assert run("task-two")["fragment_source"] == "model"  # a new task key starts over
    assert len(written) == 2

    written.clear()
    Learn._fragment_cache.clear()
    run("task-one", key="demo/shape/v1")
    reused = run("task-two", key="demo/shape/v1")
    assert reused["fragment_source"] == "cache" and reused["model_calls"] == 0
    assert len(written) == 1


def test_cached_fragment_is_verified_on_every_run_without_ai_review():
    """[REQ:LV-10] Reuse skips AI, but retains the independent domain postcondition."""
    calls, checks = [], []
    def write(**kwargs):
        calls.append(kwargs)
        return Handle([{"fragment": "def step(inputs, bindings):\n    return {'ok': True}"}])
    def verify(value):
        checks.append(value)
        return ("verified_success", ["ok:true"]) if value == {"ok": True} else ("failed", [])
    owner = SimpleNamespace(_bridge_url="", ai=SimpleNamespace(write=write))
    ctx = contextvars.ContextVar("always_verify", default={})
    Learn._fragment_cache.clear()
    for _ in range(3):
        current = Learn(owner, ctx, lambda: {}, "demo.verify")
        result = current.act("hole", "return ok", {}, {"type": "object"}, [], verify=verify,
                             verifier_revision="ok-v1", min_verified=1, min_contexts=1)
        current.outcome("verified_success", ["domain:ok"])
        current.finalize({"status": "ok"})
    assert len(checks) == 3 and len(calls) == 1
    assert result["fragment_source"] == "cache"


def test_delegate_uses_agent_surface_and_caller_verification():
    calls = []

    class Agent:
        def start(self, **request):
            calls.append(("start", request))
            return Handle([{"execution_id": "exec-1"}])

        def collect(self, handle, wait_seconds=0):
            calls.append(("collect", wait_seconds))
            return Handle([{"answer": "verified", "claimed_outcome": "failed"}])

    owner = SimpleNamespace(_bridge_url="", agent=Agent(), vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle([]))))
    context = contextvars.ContextVar("delegate_ctx", default={"program_id": "demo.program", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "demo.program")
    output = learn.delegate("research", "inspect the target", {"target": "x"}, {"type": "object", "required": ["answer"]}, ["demo/read/run"], {"web": False}, lambda result: ("verified_success", ["agent:exec-1"]))
    assert output["answer"] == "verified"
    step = next(iter(learn._steps.values()))
    assert step["outcome"] == "verified_success"
    assert any(note["body"].get("attribution") == "unavailable" for note in step["notes"] if note["kind"] == "trace")
    assert calls[0][1]["brief"].endswith("inspect the target")
    assert calls[1] == ("collect", 30)


def _memory_rows(*bodies):
    """Rows shaped the way vrooli-memory really returns learning notes: text-encoded observations."""
    import json as _json
    rows = []
    for index, (kind, body) in enumerate(bodies):
        text = "learning-observation/v1 " + _json.dumps({
            "observationId": f"observation-{index}", "attemptId": "attempt-x", "disposition": "unknown",
            "methodRevision": "learn." + kind, "correction": kind + "/v1 " + _json.dumps(body, sort_keys=True),
            "provenance": "agent", "observedAt": "2026-09-09T10:09:34.339Z"})
        rows.append({"entry_id": f"e-{index}", "text": text, "created_at": "2026-09-09T10:09:34.339Z"})
    return rows


def test_recall_with_kinds_leads_with_kind_markers_and_parses_note_bodies():
    calls = []
    rows = Handle(_memory_rows(("preference", {"option_id": "wf@1"}), ("avoid", {"option_id": "wf@2"})))
    memory = SimpleNamespace(recall=SimpleNamespace(recall=lambda **kwargs: calls.append(kwargs) or rows))
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=memory)
    context = contextvars.ContextVar("kinds_ctx", default={"program_id": "bas.do-task", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {"site": "example"}, "bas.do-task")
    result = learn.recall(kinds=["preference", "avoid"])
    assert calls[0]["query"].startswith("preference/v1 avoid/v1 bas.do-task ")
    assert calls[0]["limit"] == 20
    assert result["notes"]["preference"] == [{"option_id": "wf@1"}]
    assert result["notes"]["avoid"] == [{"option_id": "wf@2"}]
    assert learn.recall_status == "matched"


def test_choose_recalls_by_kind_even_after_an_unrelated_recall_and_honours_avoid():
    def recall(**kwargs):
        if kwargs["query"].startswith("preference/v1 avoid/v1 option_id wf@1 wf@2"):
            return Handle(_memory_rows(("preference", {"option_id": "wf@1"}), ("avoid", {"option_id": "wf@2"})))
        return Handle([{"entry_id": "attempt-row", "text": "Learning: bas.do-task | verified_success", "outcome": "verified_success"}])
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=recall)))
    context = contextvars.ContextVar("choose_kind_ctx", default={"program_id": "bas.do-task", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {"site": "example"}, "bas.do-task")
    prior = learn.recall()
    assert prior["attempts"] and not prior["notes"]
    chosen = learn.choose(["wf@1", "wf@2", "wf@3"], "wf@3")
    assert chosen["selected_id"] == "wf@1"
    assert chosen["source"] == "advice"
    avoided = learn.choose(["wf@2", "wf@3"], "wf@3")
    assert avoided["selected_id"] == "wf@3"
    assert avoided["source"] == "default"


def test_recall_status_is_not_requested_until_memory_is_asked():
    kernel = SessionKernel()
    result = kernel.execute(
        "learn.task(scope='demo', operation='demo.program', key={'device': 'tv'})\n"
        "learn.outcome('unknown', [], measurements={'reused_workflow': True, 'tool_round_trips': 2})",
        program_id="demo.program", provenance="agent")
    assert result["ok"], result
    receipt = result["learning"]
    assert receipt["recall_status"] == "not_requested"
    assert receipt["attempts"][0]["recall_status"] == "not_requested"
    assert receipt["attempts"][0]["reused_workflow"] is True
    assert receipt["attempts"][0]["tool_round_trips"] == 2
    assert receipt["measurements"] == {"reused_workflow": True, "tool_round_trips": 2}


def test_outcome_rejects_unknown_measurements():
    learn = Learn(object(), contextvars.ContextVar("m_ctx", default={}), lambda: {})
    with pytest.raises(ValueError, match="measurements accepts only"):
        learn.outcome("unknown", [], measurements={"latency": 3})
    with pytest.raises(ValueError, match="reused_workflow must be a bool"):
        learn.outcome("unknown", [], measurements={"reused_workflow": "yes"})


def test_feedback_grades_an_earlier_attempt_through_memory_observe():
    observed = []
    memory = SimpleNamespace(
        recall=SimpleNamespace(recall=lambda **_: Handle([])),
        learning=SimpleNamespace(observe=lambda **kwargs: observed.append(kwargs) or Handle([{"entryId": "entry-9"}])))
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=memory)
    context = contextvars.ContextVar("fb_ctx", default={"program_id": "browser-automation-studio.do-task", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "browser-automation-studio.do-task")
    learn.task(scope="bas-usage", operation="browser-automation-studio.do-task", key={"site": "example"})
    receipt = learn.feedback("attempt-earlier", "supported", ["execution:1"])
    assert receipt["delivery"] == "delivered" and receipt["entry_id"] == "entry-9"
    observation = observed[0]["observation"]
    assert observed[0]["scope"] == "bas-usage"
    assert observation["attempt_id"] == "attempt-earlier"
    assert observation["disposition"] == "supported"
    assert observation["evidence_refs"] == ["execution:1"]
    assert observation["method_revision"] == "learn.feedback"
    assert observation["observation_id"].startswith("observation-")
    with pytest.raises(ValueError, match="feedback requires non-empty evidence"):
        learn.feedback("attempt-earlier", "contradicted", [])
    with pytest.raises(ValueError, match="disposition"):
        learn.feedback("attempt-earlier", "unknown", ["x"])
    learn.outcome("unknown", [])
    finished = learn.finalize({"status": "ok", "evidence": []})
    assert finished["learning"]["feedback"][0]["observation_id"] == observation["observation_id"]


def test_recall_decodes_text_encoded_attempt_records():
    text = ("Learning: bas.do-task | failed | ctx-v1:abc\nlearning-attempt/v1 " + __import__("json").dumps(
        {"attemptId": "attempt-prior", "outcome": "failed", "failureFingerprint": "bas.do-task|verify|step_failed", "operation": "bas.do-task"}))
    rows = Handle([{"entry_id": "e-1", "text": text, "created_at": "2026-09-09T10:09:34.339Z"}])
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: rows)))
    context = contextvars.ContextVar("attempt_ctx", default={"program_id": "bas.do-task", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "bas.do-task")
    result = learn.recall(query="prior")
    assert result.repeats(outcome="failed") == 1
    assert result.last_outcome == "failed"
    assert result.last_fingerprint == "bas.do-task|verify|step_failed"
    assert result["attempts"][0]["attempt_id"] == "attempt-prior"


def test_feedback_failure_is_preserved_in_finish_intent():
    """[REQ:LV-08] Failed later-run feedback is retained without repeating domain work."""
    calls = []
    def observe(**request):
        calls.append(request)
        raise RuntimeError("unavailable")
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(learning=SimpleNamespace(observe=observe)))
    learn = Learn(owner, contextvars.ContextVar("feedback_durable", default={}), lambda: {}, "demo.program")
    learn.feedback("earlier", "contradicted", ["assertion:failed"])
    learn.outcome("verified_success", ["domain:done"])
    result = learn.finalize({"status": "ok"})
    assert len(calls) == 1
    assert any(item["attempt_id"] == "earlier" for item in result["learning"]["observations"])
    assert result["learning"]["feedback"][0]["delivery"] == "failed"


def test_advice_decisions_are_bounded_to_memory_limit_with_applied_first():
    rows = Handle(_memory_rows(*[("preference", {"option_id": "wf@%d" % i}) for i in range(20)]))
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: rows)))
    context = contextvars.ContextVar("bound_ctx", default={"program_id": "bas.do-task", "provenance": "agent"})
    learn = Learn(owner, context, lambda: {}, "bas.do-task")
    exposed = learn.recall(kinds=["preference"])
    assert len(exposed["hits"]) == 20
    assert len(learn._decisions) == 10
    chosen = learn.choose(["wf@17", "wf@0"], "wf@0")
    assert chosen["selected_id"] in ("wf@17", "wf@0")
    assert len(learn._decisions) == 10
    assert learn._decisions[0]["decision"] == "applied"


def test_contradicted_advice_and_avoided_defaults_are_ineligible():
    """[REQ:LV-12] Negative evidence survives recall; defaults obey exclusions."""
    rows = [{"entry_id": "bad", "kind": "preference", "body": {"option_id": "bad"}, "contradicted": 9}]
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle(rows))))
    ctx = contextvars.ContextVar("negative", default={})
    assert Learn(owner, ctx, lambda: {}, "demo.program").choose(["good", "bad"], "good")["selected_id"] == "good"
    rows[:] = [{"entry_id": "avoid", "kind": "avoid", "body": {"option_id": "bad"}}]
    assert Learn(owner, ctx, lambda: {}, "demo.program").choose(["good", "bad"], "bad")["selected_id"] == "good"
    result = Learn(owner, ctx, lambda: {}, "demo.program").choose(["bad"], "bad")
    assert result["selected_id"] is None and result["source"] == "unavailable"


def test_act_repairs_with_diagnostics_and_verifies_cached_results():
    """[REQ:LV-10] A valid-shaped wrong answer is repaired and never trusted by shape alone."""
    calls = []
    def write(**request):
        calls.append(request)
        value = 0 if len(calls) == 1 else 7
        return Handle([{"fragment": "def step(inputs, bindings):\n    return {'value': " + str(value) + "}"}])
    owner = SimpleNamespace(_bridge_url="", _invocations=[], ai=SimpleNamespace(write=write))
    learn = Learn(owner, contextvars.ContextVar("repair", default={}), lambda: {}, "demo.repair")
    result = learn.act("hole", "return seven", {}, {"type": "object", "required": ["value"]}, ["demo/read/run"],
                       verify=lambda result: ("verified_success", ["expected:7"]) if result["value"] == 7 else ("failed", ["expected:7"]), verifier_revision="v1")
    assert result["output"] == {"value": 7} and len(calls) == 2
    assert "verification_failed" in calls[1]["source"]
    assert "schema" in calls[1]["source"] and "bindings" in calls[1]["source"]


def test_fragment_qualification_requires_diverse_inputs_and_invalidates_on_contract_change():
    """[REQ:LV-10] Repeated identical examples cannot qualify broad reuse."""
    calls = []
    def write(**request):
        calls.append(request)
        return Handle([{"fragment": "def step(inputs, bindings):\n    return {'value': inputs['value']}"}])
    owner = SimpleNamespace(_bridge_url="", ai=SimpleNamespace(write=write))
    ctx = contextvars.ContextVar("diversity", default={})
    Learn._fragment_cache.clear()
    def run(value, revision="v1", intent="echo"):
        current = Learn(owner, ctx, lambda: {}, "demo.diversity")
        current.task(key="shared")
        result = current.act("echo", intent, {"value": value}, {"type": "object"}, [],
            verify=lambda out: ("verified_success", ["echo:equal"]) if out == {"value": value} else ("failed", []), verifier_revision=revision)
        current.outcome("verified_success", ["echo:equal"])
        current.finalize({"status": "ok"})
        return result
    for _ in range(4):
        assert run("a")["model_calls"] == 1
    assert run("b")["model_calls"] == 1
    assert run("c")["model_calls"] == 0
    assert run("c", revision="v2")["model_calls"] == 1
    assert run("c", intent="echo with a changed objective")["model_calls"] == 1


def test_reviewed_baseline_runs_without_ai_or_memory_and_detects_tampering():
    """[REQ:LV-11] An installed reviewed baseline is useful with optional services absent."""
    import copy
    from host.fragments import baseline_artifact
    contract = {"version": 2, "intent": "echo", "schema": {"type": "object"}, "bindings": [],
                "binding_contracts": [], "verifier_revision": "v1", "environment": {}}
    seed = baseline_artifact("def step(inputs, bindings):\n    return {'value': inputs['value']}", contract,
        "test-review", ["fixture:echo"], [{"inputs": {"value": "portable"}, "expected": {"value": "portable"}}])
    owner = SimpleNamespace(_bridge_url="")
    current = Learn(owner, contextvars.ContextVar("offline", default={}), lambda: {}, "demo.offline", baselines={"echo": seed})
    result = current.act("echo", "echo", {"value": "portable"}, {"type": "object"}, [], allow_ai=False,
        verify=lambda out: ("verified_success", ["echo:equal"]) if out == {"value": "portable"} else ("failed", []), verifier_revision="v1")
    assert result["fragment_source"] == "baseline" and result["model_calls"] == 0
    broken = copy.deepcopy(seed); broken["fragment"] = "def step(inputs, bindings):\n    return {'value': 'wrong'}"
    with pytest.raises(StepFailed, match="adaptation_unavailable"):
        current.act("echo-broken", "echo", {"value": "portable"}, {"type": "object"}, [], allow_ai=False,
            baseline=broken, verify=lambda _: ("verified_success", ["unused"]), verifier_revision="v1")


def test_fragment_failure_after_write_never_uses_baseline_or_retries():
    """[REQ:LV-08] Uncertain effects stop adaptation even when a fallback exists."""
    calls = []
    owner = SimpleNamespace(_bridge_url="", _invocations=[])
    def write_domain():
        calls.append("domain")
        owner._invocations.append({"effect": "write"})
        raise ConnectionError("response lost")
    owner.demo = SimpleNamespace(work=SimpleNamespace(run=write_domain))
    current = Learn(owner, contextvars.ContextVar("uncertain", default={}), lambda: {}, "demo.uncertain")
    with pytest.raises(StepFailed, match="uncertain_effects"):
        current.act("write", "write", {}, {"type": "object"}, ["demo/work/run"],
            fallback_fragment="def step(inputs, bindings):\n    return bindings.demo.work.run()", allow_ai=False,
            verify=lambda _: ("verified_success", ["unused"]), verifier_revision="v1")
    assert calls == ["domain"]


def test_schema_checks_all_items_and_numeric_bounds():
    from host.learn import _matches_schema
    assert not _matches_schema([1] * 100 + ["bad"], {"type": "array", "items": {"type": "integer"}})
    assert not _matches_schema({"value": -1}, {"type": "object", "properties": {"value": {"type": "integer", "minimum": 0}}})
    with pytest.raises(ValueError, match="unsupported schema"):
        _matches_schema({}, {"$ref": "unresolved"})


def test_fragments_reject_private_access_and_bound_loops():
    from host.fragments import wrap
    for fragment in ("def step(inputs, bindings):\n    return bindings._owner", "import os\ndef step(inputs, bindings):\n    return {}"):
        with pytest.raises(Exception):
            wrap(fragment)
    with pytest.raises(Exception, match="fragment_execution_budget"):
        wrap("def step(inputs, bindings):\n    while True:\n        pass")({}, None)


def test_fragments_expose_isinstance_and_set_but_not_type():
    """Normalization fragments are defensive by nature; `type` stays forbidden."""
    from host.fragments import wrap
    narrow = wrap("def step(inputs, bindings):\n"
                  "    rows = [r for r in inputs['rows'] if isinstance(r, dict)]\n"
                  "    return {'kinds': sorted(set(str(r.get('kind') or '') for r in rows))}")
    assert narrow({"rows": [{"kind": "click"}, "not-a-row", {"kind": "click"}, {"kind": "type"}]}, None) == {"kinds": ["click", "type"]}
    with pytest.raises(Exception, match="fragment_unsafe_call"):
        wrap("def step(inputs, bindings):\n    return {'t': type(inputs)}")


def test_choose_dictionary_options_preserves_the_declared_default():
    owner = SimpleNamespace(_bridge_url="")
    learn = Learn(owner, contextvars.ContextVar("dictionary-choice", default={}), lambda: {}, "demo.program")
    options = [{"id": "first"}, {"id": "second"}]
    assert learn.choose(options, options[1])["selected_id"] == "second"


def test_encoded_attempts_from_other_contexts_are_not_recalled():
    import json
    rows = [{"entry_id": "one", "text": "Learning: learning-attempt/v1 " + json.dumps({
        "attemptId": "one", "operation": "demo.program", "contextKey": "other", "outcome": "failed", "provenance": "agent"})}]
    owner = SimpleNamespace(_bridge_url="", vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle(rows))))
    learn = Learn(owner, contextvars.ContextVar("context-filter", default={"provenance": "agent"}), lambda: {}, "demo.program")
    assert not learn.recall()["attempts"]


def test_fragment_bindings_cannot_expose_or_mutate_the_transport_and_audit_log():
    from host.learn import _AllowedBindings
    from fragments import wrap
    calls = []
    def target(**arguments):
        calls.append(arguments)
        return Handle([{"value": "ok"}])
    target.invocations = calls
    owner = SimpleNamespace(demo=SimpleNamespace(read=SimpleNamespace(run=target)))
    allowed = _AllowedBindings(owner, {"demo/read/run"})
    assert wrap("def step(inputs, bindings):\n    return bindings.demo.read.run().head(1)[0]")({}, allowed) == {"value": "ok"}
    with pytest.raises(AttributeError):
        wrap("def step(inputs, bindings):\n    bindings.demo.read.run.invocations.clear()\n    return {}")({}, allowed)
    assert calls == [{}]
    with pytest.raises(StepFailed):
        wrap("def step(inputs, bindings):\n    bindings.demo.read.run.bridge_url = 'elsewhere'\n    return {}")


def test_transport_enum_provenance_keeps_test_fragments_out_of_live_reuse():
    from fragments import baseline_artifact
    schema = {"type": "object"}
    contract = {"version": 2, "intent": "echo", "schema": schema, "bindings": [], "binding_contracts": [], "verifier_revision": "v1", "environment": {}}
    baseline = baseline_artifact("def step(inputs, bindings):\n    return inputs", contract, "reviewer", ["fixture:echo"], [{"inputs": {}, "expected": {}}])
    keys = []
    for provenance in ("PROVENANCE_TEST", "PROVENANCE_OPERATOR", "PROVENANCE_AGENT", "PROVENANCE_REPLAY"):
        context = contextvars.ContextVar(provenance)
        context.set({"provenance": provenance})
        learn = Learn(SimpleNamespace(_bridge_url=""), context, lambda: {}, "demo.program")
        keys.append(learn.act("echo", "echo", {}, schema, [], verify=lambda _: ("verified_success", ["checked:echo"]), verifier_revision="v1", baseline=baseline, allow_ai=False)["step_key"])
    assert keys[0] != keys[1] == keys[2]
    assert keys[3] not in keys[:3]


def test_lost_write_response_stops_adaptation_before_any_replay():
    calls = []
    def change(**arguments):
        calls.append(arguments)
        raise TimeoutError("response lost after remote commit")
    change.effect = "write"
    owner = SimpleNamespace(_bridge_url="", _invocations=[], demo=SimpleNamespace(work=SimpleNamespace(change=change)),
        ai=SimpleNamespace(write=lambda **_: Handle([{"fragment": "def step(inputs, bindings):\n    bindings.demo.work.change(value=1)\n    return {}"}])))
    learn = Learn(owner, contextvars.ContextVar("lost-write"), lambda: {}, "demo.program")
    with pytest.raises(StepFailed, match="uncertain_effects"):
        learn.act("change", "change", {}, {"type": "object"}, ["demo/work/change"],
                  verify=lambda _: ("verified_success", ["checked:change"]), verifier_revision="v1")
    assert calls == [{"value": 1}]


def test_generated_code_cannot_change_the_verifiers_expected_input():
    inputs = {"expected": "correct"}
    owner = SimpleNamespace(_bridge_url="", ai=SimpleNamespace(write=lambda **_: Handle([{"fragment":
        "def step(inputs, bindings):\n    inputs['expected'] = 'wrong'\n    return {'value': 'wrong'}"}])))
    learn = Learn(owner, contextvars.ContextVar("verifier-isolation"), lambda: inputs, "demo.program")
    with pytest.raises(StepFailed, match="fragment_failed"):
        learn.act("check", "return expected", inputs, {"type": "object"}, [], attempts=1,
                  verify=lambda output: ("verified_success", ["value:equal"]) if output["value"] == inputs["expected"] else ("failed", ["value:unequal"]), verifier_revision="v1")
    assert inputs == {"expected": "correct"}


def test_result_refs_preserve_exact_external_revision_and_later_feedback(monkeypatch):
    learn = Learn(SimpleNamespace(), contextvars.ContextVar('result-ref', default={}), lambda: {}, 'demo.program')
    sent = []
    monkeypatch.setattr(learn, '_learning_write', lambda action, request: sent.append((action, request)) or {'delivery': 'delivered'})
    reference = learn.result('browser', artifact={'kind': 'workflow', 'owner': 'browser-automation-studio', 'id': 'flow-1', 'revision': '7'})
    assert reference.startswith('prt_feedback_v1_')
    assert learn.result_ref('browser') == reference == learn.result_ref()
    assert sent[0][1]['artifact']['revision'] == '7'
    receipt = learn.feedback(reference, 'contradicted', ['user:wrong-account'], dimension='usefulness')
    assert sent[-1][0] == 'learning_feedback'
    assert sent[-1][1]['dimension'] == 'usefulness'
    assert receipt['feedback_ref'] == reference
    assert learn.finalize({'status': 'ok'})['learning']['result_refs']['browser'] == reference


def test_capability_intents_cannot_expand_declared_authority_or_effects():
    owner = SimpleNamespace(discover=lambda **kwargs: {'binding_id': 'demo/records/update'},
                            _binding_contract_specs=[{'id': 'demo/records/update', 'effect': 'write'}])
    learn = Learn(owner, contextvars.ContextVar('resolve', default={}), lambda: {}, bindings=[{'id': 'demo/records/update', 'effect': 'write'}])
    with pytest.raises(StepFailed, match='capability_not_authorized'):
        learn._resolve_capabilities(['update record'], None)
    assert learn._resolve_capabilities(['update record'], ['write']) == ['demo/records/update']
    owner._binding_contract_specs = []
    with pytest.raises(StepFailed, match='capability_not_authorized'):
        learn._resolve_capabilities(['update record'], ['write'])


def test_learning_journal_survives_recreation_and_replays_in_order(tmp_path, monkeypatch):
    from host.fragments import durable_learning_write
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    owner = SimpleNamespace(_bridge_url='http://fixture/bindings/invoke')
    context = contextvars.ContextVar('journal', default={'program_id': 'demo.program'})
    def unavailable(*args, **kwargs):
        raise OSError('offline')
    first = durable_learning_write(owner, context, 'test', 'learning_result_put', {'feedback_ref': 'first'}, unavailable)
    assert first['delivery'] == 'pending'
    replayed = []
    receipt = durable_learning_write(owner, context, 'test', 'learning_feedback', {'feedback_ref': 'first', 'observation_id': 'second'},
                                    lambda action, **request: replayed.append((action, request)) or {})
    assert receipt['delivery'] == 'delivered'
    assert [item[0] for item in replayed] == ['learning_result_put', 'learning_feedback']
    assert durable_learning_write(owner, context, 'test', None, {}, unavailable)['delivery'] == 'delivered'


def test_learning_journal_never_replays_live_evidence_into_test_cohort(tmp_path, monkeypatch):
    from host.fragments import durable_learning_write
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    owner = SimpleNamespace(_bridge_url='http://fixture/bindings/invoke')
    context = contextvars.ContextVar('journal-cohort', default={'program_id': 'demo.program'})
    def unavailable(*args, **kwargs):
        raise OSError('offline')
    durable_learning_write(owner, context, 'agent', 'learning_feedback', {'observation_id': 'live'}, unavailable)
    replayed = []
    durable_learning_write(owner, context, 'test', None, {}, lambda *args, **kwargs: replayed.append(args))
    assert replayed == []


def test_invalid_feedback_dimension_does_not_start_task():
    learn = Learn(object(), contextvars.ContextVar('bad-dimension', default={}), lambda: {})
    with pytest.raises(ValueError, match='feedback dimension'):
        learn.feedback('prt_feedback_v1_example', 'contradicted', ['observed:bad'], dimension='guessed')
    assert not learn.started


def test_result_delivery_failure_never_repeats_successful_domain_effect(tmp_path, monkeypatch):
    blocked = tmp_path / 'not-a-directory'
    blocked.write_text('occupied')
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(blocked))
    owner = SimpleNamespace(_bridge_url='')
    learn = Learn(owner, contextvars.ContextVar('delivery-failure', default={}), lambda: {})
    learn.task()
    # The domain ran before the bridge/spool became unavailable.
    owner._bridge_url = 'http://unreachable/bindings/invoke'
    reference = learn.result('completed-work')
    assert reference.startswith('prt_feedback_v1_')
    assert learn._result_delivery[-1]['delivery'] == 'unavailable'


def test_cached_selection_passes_eligibility_and_honors_baseline_quarantine(monkeypatch):
    import hashlib
    from host.learn import Tasks
    from host.fragments import baseline_artifact
    fragment = 'def step(inputs, bindings):\n    return {}'
    contract = {'version': 2, 'intent': 'echo', 'schema': {'type': 'object'}, 'bindings': [],
                'binding_contracts': [], 'verifier_revision': 'v1', 'environment': {}}
    seed = baseline_artifact(fragment, contract, 'reviewer', ['fixture:echo'], [{'inputs': {}, 'expected': {}}])
    owner = SimpleNamespace(_bridge_url='')
    learn = Learn(owner, contextvars.ContextVar('quarantine', default={}), lambda: {})
    learn.task()
    owner._bridge_url = 'http://fixture/bindings/invoke'
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'delivered'})
    calls = []
    def bridge(self, action, **request):
        calls.append((action, request))
        return {'found': False, 'rejected_hashes': [hashlib.sha256(fragment.encode()).hexdigest()]}
    monkeypatch.setattr(Tasks, '_bridge', bridge)
    with pytest.raises(StepFailed, match='adaptation_unavailable'):
        learn.act('echo', 'echo', {}, {'type': 'object'}, [], baseline=seed, allow_ai=False,
                  verify=lambda _: ('verified_success', ['fixture:echo']), verifier_revision='v1')
    read = next(request for action, request in calls if action == 'fragment_get')
    assert (read['min_verified'], read['min_contexts'], read['max_age_days']) == (3, 2, 30)


def test_result_registration_sends_verb_field_and_leaves_projection_to_server(tmp_path, monkeypatch):
    from host.learn import Tasks
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    owner = SimpleNamespace(_bridge_url='')
    context = contextvars.ContextVar('bridge-verb', default={'program_id': 'instance-one', 'provenance': 'test'})
    learn = Learn(owner, context, lambda: {}, 'demo.program')
    learn.task()
    owner._bridge_url = 'http://fixture/bindings/invoke'
    requests = []
    def bridge(self, action, **payload):
        requests.append((action, payload))
        if action == 'learning_feedback':
            return {'target': {'attempt_id': 'prior', 'scope': 'demo-usage'}}
        return {'stored': True}
    monkeypatch.setattr(Tasks, '_bridge', bridge)
    ref = learn.result('step', verb='act')
    assert requests[-1][1]['verb'] == 'act'
    assert learn._result_delivery[-1]['delivery'] == 'delivered'
    learn.feedback(ref, 'contradicted', ['downstream:failed'])
    assert learn._feedback == []
    assert requests[-1][0] == 'learning_feedback'


def test_terminal_adaptation_failure_automatically_captures_a_finding(monkeypatch):
    owner = SimpleNamespace(_bridge_url='')
    learn = Learn(owner, contextvars.ContextVar('failure-capture', default={}), lambda: {})
    writes = []
    monkeypatch.setattr(learn, '_learning_write', lambda action, request: writes.append((action, request)) or {'delivery': 'delivered'})
    with pytest.raises(StepFailed, match='adaptation_unavailable'):
        learn.act('unimplemented', 'return value', {}, {'type': 'object'}, [], allow_ai=False,
                  verify=lambda _: ('verified_success', ['expected:value']), verifier_revision='v1')
    registration = next(payload for action, payload in writes if action == 'learning_result_put')
    feedback = next(payload for action, payload in writes if action == 'learning_feedback')
    assert registration['artifact']['kind'] == 'adaptation-failure'
    assert feedback['feedback_ref'] == registration['feedback_ref']
    assert feedback['disposition'] == 'contradicted'


def test_later_choice_correction_overrides_positive_memory_weight(monkeypatch):
    from host.learn import Tasks
    rows = [{'entry_id': 'old-preference', 'kind': 'preference', 'body': {'option_id': 'wrong'}, 'supported': 100}]
    owner = SimpleNamespace(_bridge_url='', vrooli_memory=SimpleNamespace(recall=SimpleNamespace(recall=lambda **_: Handle(rows))))
    learn = Learn(owner, contextvars.ContextVar('correct-choice', default={}), lambda: {}, 'demo.program')
    learn.task(key='same-task')
    owner._bridge_url = 'http://fixture/bindings/invoke'
    def bridge(self, action, **payload):
        return {'eligible': payload.get('artifact', {}).get('id') != 'wrong', 'contradicted': payload.get('artifact', {}).get('id') == 'wrong'}
    monkeypatch.setattr(Tasks, '_bridge', bridge)
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'delivered'})
    selected = learn.choose(['wrong', 'good'], 'good')
    assert selected['selected_id'] == 'good'
    assert selected['source'] == 'default'


def test_public_result_status_distinguishes_outage_from_eligible(monkeypatch):
    from host.learn import Tasks
    owner = SimpleNamespace(_bridge_url='http://fixture/bindings/invoke')
    learn = Learn(owner, contextvars.ContextVar('public-status', default={}), lambda: {})
    artifact = {'kind': 'workflow', 'owner': 'browser-automation-studio', 'id': 'flow', 'revision': '2'}
    monkeypatch.setattr(Tasks, '_bridge', lambda *args, **kwargs: {'found': False, 'eligible': True, 'contradicted': False})
    assert learn.result_status(artifact)['eligible'] is True
    def unavailable(*args, **kwargs):
        raise OSError('unavailable')
    monkeypatch.setattr(Tasks, '_bridge', unavailable)
    status = learn.result_status(artifact)
    assert status['available'] is False and status['eligible'] is False


def test_learning_journal_replays_after_runtime_port_and_invocation_change(tmp_path, monkeypatch):
    from host.fragments import durable_learning_write
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    context = contextvars.ContextVar('port-change', default={'program_id': 'old-instance'})
    def unavailable(*args, **kwargs):
        raise OSError('unavailable')
    durable_learning_write(SimpleNamespace(_bridge_url='http://localhost:1001/bindings/invoke'), context,
                           'test', 'learning_result_put', {'feedback_ref': 'pending'}, unavailable, program_name='demo.program')
    context.set({'program_id': 'new-instance'})
    delivered = []
    result = durable_learning_write(SimpleNamespace(_bridge_url='http://localhost:2002/bindings/invoke'), context,
                           'test', None, {}, lambda *args, **kwargs: delivered.append(kwargs), program_name='demo.program')
    assert result['delivery'] == 'delivered'
    assert delivered == [{'feedback_ref': 'pending'}]


def test_capability_policy_resolves_live_contract_without_method_ids():
    owner = SimpleNamespace(discover=lambda **kwargs: {'binding_id': 'demo/records/read'},
                            _binding_contract_specs=[{'id': 'demo/records/read', 'scenario': 'demo', 'effect': 'read'}])
    learn = Learn(owner, contextvars.ContextVar('scoped-policy', default={}), lambda: {})
    with pytest.raises(StepFailed, match='capability_not_authorized'):
        learn._resolve_capabilities(['read a record'], None)
    learn.capability_policy = [{'scenario': 'demo', 'effects': ['read']}]
    assert learn._resolve_capabilities(['read a record'], None) == ['demo/records/read']
    owner._binding_contract_specs[0]['effect'] = 'write'
    with pytest.raises(StepFailed, match='capability_not_authorized'):
        learn._resolve_capabilities(['read a record'], ['write'])


def test_result_status_withholds_eligibility_until_pending_feedback_is_delivered(monkeypatch):
    from host.learn import Tasks
    learn = Learn(SimpleNamespace(_bridge_url='http://fixture/bindings/invoke'),
                  contextvars.ContextVar('pending-eligibility', default={}), lambda: {})
    calls = []
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'pending'})
    monkeypatch.setattr(Tasks, '_bridge', lambda *args, **kwargs: calls.append(args) or {'eligible': True})
    status = learn.result_status({'kind': 'workflow', 'owner': 'bas', 'id': 'flow', 'revision': '1'})
    assert status['available'] is False and status['eligible'] is False
    assert calls == []


def test_permanently_refused_event_is_retained_without_poisoning_unrelated_work(tmp_path, monkeypatch):
    import sqlite3
    import urllib.error
    from host.fragments import durable_learning_write
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    owner = SimpleNamespace(_bridge_url='http://fixture/bindings/invoke')
    context = contextvars.ContextVar('poison-event', default={})
    def refused(*args, **kwargs):
        raise RuntimeError('task checkpoint refused: sql: no rows in result set') from urllib.error.HTTPError('fixture', 409, 'conflict', {}, None)
    rejected = durable_learning_write(owner, context, 'test', 'learning_feedback', {'feedback_ref': 'unknown'}, refused)
    assert rejected['delivery'] == 'rejected'
    delivered = durable_learning_write(owner, context, 'test', 'learning_result_put', {'feedback_ref': 'valid'}, lambda *args, **kwargs: {})
    assert delivered['delivery'] == 'delivered'
    db = sqlite3.connect(next(tmp_path.glob('*.sqlite3')))
    assert db.execute('SELECT count(*) FROM rejected_events').fetchone()[0] == 1
    assert db.execute('SELECT count(*) FROM events').fetchone()[0] == 0
    db.close()


def test_pending_agent_evidence_is_replayed_before_operator_work(tmp_path, monkeypatch):
    from host.fragments import durable_learning_write
    monkeypatch.setenv('PROGRAM_RUNTIME_LEARNING_STATE_DIR', str(tmp_path))
    owner = SimpleNamespace(_bridge_url='http://fixture/bindings/invoke')
    context = contextvars.ContextVar('live-replay', default={})
    def unavailable(*args, **kwargs):
        raise OSError('unavailable')
    durable_learning_write(owner, context, 'agent', 'learning_feedback', {'feedback_ref': 'pending'}, unavailable)
    replayed = []
    result = durable_learning_write(owner, context, 'operator', None, {}, lambda *args, **kwargs: replayed.append(kwargs))
    assert result['delivery'] == 'delivered'
    assert replayed == [{'feedback_ref': 'pending'}]


def test_pending_evidence_withholds_cache_but_still_runs_a_reviewed_baseline(monkeypatch):
    """[REQ:LV-11] A learning-delivery outage degrades reuse; it never disables the section.

    Undelivered evidence can contradict a learned fragment, so the unreviewed cache is
    withheld. A declared baseline is human-reviewed and fixture-replayed at publication,
    so runtime evidence cannot disqualify it and it must still execute.
    """
    from host.learn import Tasks
    from host.fragments import baseline_artifact
    contract = {'version': 2, 'intent': 'echo', 'schema': {'type': 'object'}, 'bindings': [],
                'binding_contracts': [], 'verifier_revision': 'v1', 'environment': {}}
    seed = baseline_artifact("def step(inputs, bindings):\n    return {'value': inputs['value']}", contract,
        'test-review', ['fixture:echo'], [{'inputs': {'value': 'portable'}, 'expected': {'value': 'portable'}}])
    owner = SimpleNamespace(_bridge_url='')
    learn = Learn(owner, contextvars.ContextVar('pending-baseline', default={}), lambda: {}, 'demo.pending')
    learn.task()
    owner._bridge_url = 'http://fixture/bindings/invoke'
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'pending', 'last_error': 'offline'})
    reads = []
    monkeypatch.setattr(Tasks, '_bridge', lambda self, action, **request: reads.append(action) or {'found': False})
    result = learn.act('echo', 'echo', {'value': 'portable'}, {'type': 'object'}, [], baseline=seed, allow_ai=False,
        verify=lambda out: ('verified_success', ['echo:equal']) if out == {'value': 'portable'} else ('failed', []),
        verifier_revision='v1')
    assert result['fragment_source'] == 'baseline' and result['model_calls'] == 0
    assert result['cache_withheld'] is True and result['eligibility'] == 'reviewed'
    # The withheld cache is never read from the store while evidence is undelivered.
    assert 'fragment_get' not in reads
    assert learn._fragment_delivery[-1]['cache_withheld'] is True


def test_pending_evidence_without_any_reviewed_path_names_the_delivery_outage(monkeypatch):
    """[REQ:LV-11] With no baseline and no adaptation, the refusal names delivery, not a missing asset."""
    from host.learn import Tasks
    owner = SimpleNamespace(_bridge_url='')
    learn = Learn(owner, contextvars.ContextVar('pending-bare', default={}), lambda: {}, 'demo.pending')
    learn.task()
    owner._bridge_url = 'http://fixture/bindings/invoke'
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'pending', 'last_error': 'offline'})
    monkeypatch.setattr(Tasks, '_bridge', lambda self, action, **request: {'found': False})
    with pytest.raises(StepFailed, match='learning_evidence_pending'):
        learn.act('echo', 'echo', {}, {'type': 'object'}, [], allow_ai=False,
                  verify=lambda _: ('verified_success', ['unused']), verifier_revision='v1')


def test_pending_evidence_still_permits_generation_because_it_is_not_reuse(monkeypatch):
    """[REQ:LV-11] Generation is not reuse: an outage must not stop a fresh verified candidate."""
    from host.learn import Tasks
    written = "def step(inputs, bindings):\n    return {'value': inputs['value']}"
    owner = SimpleNamespace(_bridge_url='', ai=SimpleNamespace(write=lambda **kwargs: {'fragment': written}))
    learn = Learn(owner, contextvars.ContextVar('pending-generate', default={}), lambda: {}, 'demo.pending')
    learn.task()
    owner._bridge_url = 'http://fixture/bindings/invoke'
    monkeypatch.setattr(learn, '_learning_write', lambda *args, **kwargs: {'delivery': 'pending', 'last_error': 'offline'})
    monkeypatch.setattr(Tasks, '_bridge', lambda self, action, **request: {'found': False})
    result = learn.act('echo', 'echo', {'value': 'fresh'}, {'type': 'object'}, [],
        verify=lambda out: ('verified_success', ['echo:equal']) if out == {'value': 'fresh'} else ('failed', []),
        verifier_revision='v1')
    assert result['fragment_source'] == 'model' and result['model_calls'] == 1
    assert result['cache_withheld'] is True
