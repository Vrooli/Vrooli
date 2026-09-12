"""The runtime verb surface: uniformity, addressing, and honest discovery rows.

Two defects motivate this file.

First, `vrooli` was a full `Namespace`, and `Namespace` carries seven of the ten
runtime verbs as *methods*. Ordinary attribute lookup found those methods before
`__getattr__` ran, so `vrooli.discover` worked while `vrooli.recall` raised —
a 7-of-10 split nobody chose. It is the worst shape for a learned surface: it
works often enough to be internalised, then fails unpredictably. Two of twelve
authoring-eval cases failed on exactly this.

Second, `discover` returned one row shape for "the judge is down" and for
"nothing serves this intent". The skill tells an agent an empty binding_id is a
stop, so a degraded dependency silently taught agents the fleet was empty.
"""
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[1]))

import pytest  # noqa: E402

from host import engine  # noqa: E402


BINDINGS = [
    {
        "id": "search-hub/query/query",
        "namespace": "search_hub",
        "scenario": "search-hub",
        "group": "query",
        "command": "query",
        "effect": "read",
        "reachable": True,
    },
    {
        "id": "vrooli/scenario/status",
        "namespace": "vrooli",
        "scenario": "vrooli",
        "group": "scenario",
        "command": "status",
        "effect": "read",
        "reachable": True,
    },
]


def build_kernel():
    return engine.SessionKernel(BINDINGS, "sess_test", "", "", "", [])


def test_every_declared_verb_is_bound_at_the_top_level():
    """The surface must cover exactly the declared verb tuple."""
    kernel = build_kernel()
    for name in engine._RUNTIME_VERB_NAMES:
        assert name in kernel.globals, f"{name} is declared but not reachable at the top level"


def test_no_runtime_verb_leaks_onto_the_project_namespace():
    """`vrooli.` addresses the project control plane and never a verb.

    This is the regression for the 7-of-10 split. It asserts the property for
    every verb rather than the three that used to fail, so a verb added as a
    method in future cannot reintroduce the asymmetry.
    """
    kernel = build_kernel()
    project = kernel.globals["vrooli"]
    for name in engine._RUNTIME_VERB_NAMES:
        with pytest.raises(AttributeError) as caught:
            getattr(project, name)
        assert name in str(caught.value)
        # The refusal must teach the correct form, not merely deny.
        assert "top level" in str(caught.value)


def test_project_namespace_still_resolves_project_commands():
    """Suppressing verbs must not break what `vrooli.` is actually for."""
    kernel = build_kernel()
    project = kernel.globals["vrooli"]
    assert project.scenario is not None
    assert callable(project.scenario.status)


def test_project_namespace_reports_an_unknown_group_plainly():
    kernel = build_kernel()
    with pytest.raises(AttributeError) as caught:
        kernel.globals["vrooli"].not_a_group
    assert "no project command group" in str(caught.value)


def test_escape_hatch_still_reaches_scenario_bindings():
    """__vrooli__ remains the stable root when a name is shadowed."""
    kernel = build_kernel()
    root = kernel.globals["__vrooli__"]
    assert callable(root.search_hub.query.query)


def test_protected_names_are_derived_from_the_verb_tuple():
    """The protected set and the surface cannot drift, because both derive
    from `_RUNTIME_VERB_NAMES`."""
    kernel = build_kernel()
    protected = kernel.globals.protected if hasattr(kernel.globals, "protected") else kernel.globals._protected
    for name in engine._RUNTIME_VERB_NAMES:
        assert name in protected
    assert "vrooli" in protected
    assert "__vrooli__" in protected


@pytest.mark.parametrize("mode", ["fast", "judged", "deep"])
def test_discover_accepts_every_documented_mode(mode):
    """The kernel hardcoded `judged`, contradicting the skill's three modes and
    making a degraded judge unavoidable from inside a program."""
    kernel = build_kernel()
    namespace = kernel.globals["__vrooli__"]
    # No discovery URL is configured, so this reaches the unavailable path
    # rather than the network — which is the row shape under test anyway.
    handle = namespace.discover("an intent", mode=mode)
    row = handle.head(1)[0]
    assert row["mode"] == mode


def test_discover_rejects_an_unknown_mode():
    kernel = build_kernel()
    with pytest.raises(ValueError) as caught:
        kernel.globals["__vrooli__"].discover("an intent", mode="clever")
    assert "fast" in str(caught.value)


def test_unavailable_discovery_is_distinguishable_from_a_null_verdict():
    """Both carry an empty binding_id; only one means 'stop'."""
    row = engine._discovery_unavailable_row("discovery unavailable: timed out", "judged")
    assert row["binding_id"] == ""
    assert row["null_verdict"] is True
    assert row["unavailable"] is True, "a failed dependency must not read as an empty fleet"
    assert "timed out" in row["reason"]


def test_discovery_bridge_reports_unavailable_when_unconfigured():
    kernel = build_kernel()
    handle = kernel.globals["__vrooli__"].discover("an intent")
    row = handle.head(1)[0]
    assert row["unavailable"] is True
    assert row["null_verdict"] is True


def test_budgets_match_go_authority():
    """The kernel's standalone fallbacks must equal the shipped Go ladder.

    A test that ran against different budgets than production would exercise a
    contract the runtime does not have.
    """
    ladder = (Path(__file__).parents[2] / "api" / "internal" / "budgets" / "budgets.go").read_text()
    expected = {
        "KernelTelemetry": engine._Budgets.telemetry,
        "KernelDescribe": engine._Budgets.describe,
        "KernelInvoke": engine._Budgets.invoke,
    }
    for constant, seconds in expected.items():
        assert constant in ladder, f"{constant} is missing from the Go budget authority"
        # The Go source states minutes or seconds; normalise both.
        for line in ladder.splitlines():
            if line.strip().startswith(constant + " ="):
                declared = line.split("=", 1)[1].strip()
                if "time.Minute" in declared:
                    value = float(declared.split("*")[0].strip()) * 60 if "*" in declared else 60.0
                else:
                    value = float(declared.split("*")[0].strip()) if "*" in declared else 1.0
                assert value == seconds, f"{constant} is {value}s in Go but {seconds}s in the kernel fallback"
                break


def test_budgets_load_overrides_the_fallbacks():
    """Go is the authority at runtime; the fallbacks are for standalone use."""
    original = (engine._Budgets.telemetry, engine._Budgets.describe, engine._Budgets.invoke)
    try:
        engine._Budgets.load(json.dumps({"telemetry_seconds": 1, "describe_seconds": 2, "invoke_seconds": 3}))
        assert (engine._Budgets.telemetry, engine._Budgets.describe, engine._Budgets.invoke) == (1.0, 2.0, 3.0)
    finally:
        engine._Budgets.telemetry, engine._Budgets.describe, engine._Budgets.invoke = original


def test_budgets_ignore_malformed_input():
    original = engine._Budgets.invoke
    engine._Budgets.load("not json")
    assert engine._Budgets.invoke == original
    engine._Budgets.load(json.dumps({"invoke_seconds": -5}))
    assert engine._Budgets.invoke == original
