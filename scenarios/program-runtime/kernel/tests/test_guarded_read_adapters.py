"""Readers adapt ProgramHelper.guarded to their owner-specific read policies."""
import ast
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

    def count(self):
        return len(self.rows)

    def meta(self):
        return self.metadata

    def map(self, mapper):
        return Rows([mapper(row) for row in self.rows], self.metadata)

    def filter(self, predicate):
        return Rows([row for row in self.rows if predicate(row)], self.metadata)


def gather(*calls):
    results = []
    for call in calls:
        try:
            results.append(call())
        except Exception as exc:
            results.append(exc)
    return results


def execute(relative, inputs, bindings, *, python_repr=False):
    path = REPO / relative
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    namespace = {"inputs": inputs, **bindings, "gather": gather}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    raw = output.getvalue()
    result = ast.literal_eval(raw) if python_repr else json.loads(raw)
    assert result["program"] == contract["name"]
    assert result["phase"] == "report"
    return result


def ai_gateway_bindings(*, broken=()):
    broken = set(broken)
    values = {
        "total": {"count": 12}, "success_rate": {"rate": 0.99}, "fallback_rate": {"rate": 0.01},
        "failure_rate": {"rate": 0.01}, "breaker_open": {"count": 0}, "capacity_rejections": {"count": 0},
        "latency_p95": {"latencyMs": 25}, "cost": {"costUsd": 0.25}, "tokens": {"totalTokens": 100},
        "local_share": {"share": 0.9},
    }

    def read(name, **_kwargs):
        if name in broken:
            raise RuntimeError("connection refused")
        return Rows([values[name]])

    measures = SimpleNamespace(**{name: (lambda _name=name, **kw: read(_name, **kw)) for name in values})
    def evidence_list(**_kwargs):
        if "evidence_list" in broken:
            raise RuntimeError("connection refused")
        return Rows()

    return {"ai_gateway": SimpleNamespace(measures=measures,
                                           routing=SimpleNamespace(evidence_list=evidence_list))}


def search_hub_setpoint_bindings(*, broken=()):
    broken = set(broken)

    def read(name, value):
        if name in broken:
            raise RuntimeError("connection refused")
        return value

    evals = SimpleNamespace(
        runs=lambda **kw: read("routing", Rows()),
        list=lambda **kw: read("suites", Rows()),
    )
    metrics = SimpleNamespace(
        provider_degradation_rate=lambda **kw: read("pdr", Rows(meta={"rate": 0.01})),
        degraded_query_rate=lambda **kw: read("dqr", Rows(meta={"rate": 0.01})),
        federated_latency=lambda **kw: read("lat", Rows(meta={"p95Ms": 20, "p50Ms": 10})),
    )
    insights = SimpleNamespace(insights=lambda **kw: read("ins", Rows(meta={"sampleSufficient": True, "zeroResultRate": 0.01})))
    federation = SimpleNamespace(status=lambda **kw: read("fed", Rows()))
    return {"search_hub": SimpleNamespace(evals=evals, metrics=metrics, insights=insights, federation=federation)}


def provider_quality_bindings(*, run_error=None):
    def runs(**_kwargs):
        if run_error:
            raise run_error
        return Rows()

    return {"search_hub": SimpleNamespace(
        insights=SimpleNamespace(insights=lambda **kw: Rows(meta={})),
        providers=SimpleNamespace(list=lambda **kw: Rows()),
        federation=SimpleNamespace(status=lambda **kw: Rows()),
        evals=SimpleNamespace(list=lambda **kw: Rows([{"suiteId": "suite-1", "providerId": "search-hub.alpha"}]),
                             runs=runs),
    )}


def meta_optimization_bindings(*, broken=()):
    broken = set(broken)
    values = {
        "governance-share": Rows([{"attemptedName": "safe-name"}], {"governedShare": 0.95, "observedCalls": 20, "governedCalls": 19}),
        "unresolved-names": Rows([{"attemptedName": "fixture-name", "count": 2}]),
        "discovery-gaps": Rows([{"query": "fixture query", "count": 1}]),
        "skill-usage": Rows([{"skillId": "fixture-skill"}]),
        "focus-next": Rows([{"gap": {"id": "fixture-gap"}, "priorityScore": 0.8, "impact": "synthetic"}]),
    }

    def read(name):
        if name in broken:
            raise RuntimeError("connection refused")
        return values[name]

    return {"program_runtime": SimpleNamespace(programs=SimpleNamespace(
        governance_share=lambda **kw: read("governance-share"),
        mine_unresolved=lambda **kw: read("unresolved-names"))),
        "prompt_manager": SimpleNamespace(
            discovery_gaps=SimpleNamespace(discovery_gaps=lambda **kw: read("discovery-gaps")),
            skill_usage=SimpleNamespace(skill_usage=lambda **kw: read("skill-usage"))),
        "meta_optimization_manager": SimpleNamespace(focus=SimpleNamespace(next=lambda **kw: read("focus-next")))}


def test_ai_gateway_optional_reads_keep_partial_and_all_failed_policy():
    path = "scenarios/ai-gateway/.vrooli/program-runtime/setpoint-read.py"
    result = execute(path, {}, ai_gateway_bindings(), python_repr=True)
    assert result["status"] == "ok"
    assert result["signals"]["readable"] > 0

    partial = execute(path, {}, ai_gateway_bindings(broken=("cost",)), python_repr=True)
    assert partial["status"] == "partial"
    assert partial["errors"][0]["class"] == "scenario_unreachable"

    unavailable = execute(path, {}, ai_gateway_bindings(broken=(
        "total", "success_rate", "fallback_rate", "failure_rate", "breaker_open", "capacity_rejections",
        "latency_p95", "cost", "tokens", "local_share", "evidence_list")), python_repr=True)
    assert unavailable["status"] == "unavailable"
    invalid = execute(path, {"window_token": "bad"}, ai_gateway_bindings(), python_repr=True)
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"


def test_search_hub_setpoint_keeps_partial_and_all_failed_policy():
    path = "scenarios/search-hub/.vrooli/program-runtime/setpoint-read.py"
    result = execute(path, {}, search_hub_setpoint_bindings())
    assert result["status"] == "ok"
    assert result["signals"]["rows"]

    partial = execute(path, {}, search_hub_setpoint_bindings(broken=("pdr",)))
    assert partial["status"] == "partial"
    assert partial["errors"][0]["class"] == "scenario_unreachable"

    unavailable = execute(path, {}, search_hub_setpoint_bindings(broken=("routing", "pdr", "dqr", "lat", "ins", "fed", "suites")))
    assert unavailable["status"] == "unavailable"
    invalid = execute(path, {"window_token": "bad"}, search_hub_setpoint_bindings())
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"


def test_search_hub_provider_quality_keeps_nested_run_failure_policy():
    path = "scenarios/search-hub/.vrooli/program-runtime/provider-quality-read.py"
    result = execute(path, {}, provider_quality_bindings())
    assert result["status"] == "ok"
    assert result["signals"]["provider_count"] == 0

    partial = execute(path, {}, provider_quality_bindings(run_error=ValueError("malformed eval response")))
    assert partial["status"] == "partial"
    assert partial["errors"][0]["class"] == "binding_error"

    unavailable = execute(path, {}, provider_quality_bindings(run_error=RuntimeError("connection refused")))
    assert unavailable["status"] == "unavailable"
    invalid = execute(path, {"max_rows": 0}, provider_quality_bindings())
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"


def test_prompt_manager_sensor_guard_keeps_success_partial_and_unavailable_policy():
    path = "scenarios/prompt-manager/.vrooli/program-runtime/meta-optimization-reads.py"
    result = execute(path, {}, meta_optimization_bindings(), python_repr=True)
    assert result["status"] == "ok"
    assert result["signals"]["readable"] == 5
    assert result["signals"]["rows"][-1]["reading"][0]["gap"] == "fixture-gap"

    partial = execute(path, {}, meta_optimization_bindings(broken=("discovery-gaps",)), python_repr=True)
    assert partial["status"] == "partial"
    assert partial["errors"][0]["class"] == "scenario_unreachable"

    unavailable = execute(path, {}, meta_optimization_bindings(broken=(
        "governance-share", "unresolved-names", "discovery-gaps", "skill-usage", "focus-next")), python_repr=True)
    assert unavailable["status"] == "unavailable"
    invalid = execute(path, {"window_seconds": 10}, meta_optimization_bindings(), python_repr=True)
    assert invalid["status"] == "failed"
    assert invalid["errors"][0]["class"] == "invalid_input"
