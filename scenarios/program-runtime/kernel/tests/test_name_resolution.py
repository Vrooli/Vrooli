"""Kernel-side name resolution: what counts as a capability miss, and what does not.

The dynamic `__missing__` hook is the fallback for names the static analyzer
cannot see. Its job is narrow: raise a useful error, and record an attempted
*capability* name without polluting the unresolved-attempt ledger with Python's
own vocabulary. That ledger is the Act denominator's feedback signal.
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[1]))

import pytest  # noqa: E402

from host import analyze, engine  # noqa: E402
from host.safebuiltins import SAFE_BUILTIN_NAMES  # noqa: E402


@pytest.mark.parametrize(
    "name",
    ["KeyError", "AttributeError", "row", "text", "self", "cls", "exc", "i", "x", "__name__", "__builtins__", "handle1", "handle_one", "prior_result", "data_store"],
)
def test_python_vocabulary_is_not_a_capability_attempt(name):
    """A language-level miss must never be recorded as an attempted binding."""
    assert engine._looks_like_binding_name(name) is False


@pytest.mark.parametrize("name", ["search_hub", "test_genie", "document_manager", "cli_health"])
def test_scenario_shaped_names_are_capability_attempts(name):
    assert engine._looks_like_binding_name(name) is True


def test_withheld_builtin_raises_plain_name_error():
    """A deliberately withheld builtin is a language miss, not a capability miss.

    `eval` is excluded from the program surface on purpose. Reporting it as an
    unresolved governed binding would both mislead the author and write a
    Python builtin into the Act denominator's evidence.
    """
    globals_ = engine.ProgramGlobals(protected=set(), known_names=["search_hub"])
    assert "eval" not in engine._SAFE_BUILTINS
    with pytest.raises(NameError) as excinfo:
        globals_["eval"]
    assert "governed binding" not in str(excinfo.value)


def test_capability_miss_names_the_nearest_match():
    globals_ = engine.ProgramGlobals(protected=set(), known_names=["search_hub", "test_genie"])
    with pytest.raises(NameError) as excinfo:
        globals_["test_geni"]
    message = str(excinfo.value)
    assert "governed binding" in message
    assert "test_genie" in message


def test_capability_miss_withholds_a_distant_suggestion():
    globals_ = engine.ProgramGlobals(protected=set(), known_names=["search_hub", "test_genie"])
    with pytest.raises(NameError) as excinfo:
        globals_["completely_unrelated_thing"]
    assert "nearest match" not in str(excinfo.value)


def test_protected_names_cannot_be_reassigned():
    globals_ = engine.ProgramGlobals(protected={"vrooli"}, known_names=[])
    globals_["vrooli"] = object()
    with pytest.raises(NameError, match="protected runtime name"):
        globals_["vrooli"] = 1


def test_shadowable_names_can_be_reassigned():
    globals_ = engine.ProgramGlobals(protected={"vrooli"}, known_names=[])
    globals_["search_hub"] = object()
    globals_["search_hub"] = 5
    assert globals_["search_hub"] == 5


def test_common_exceptions_are_available_to_programs():
    """`except KeyError:` is an ordinary program and must execute."""
    for name in ("KeyError", "AttributeError", "IndexError", "StopIteration", "ZeroDivisionError", "OSError"):
        assert name in SAFE_BUILTIN_NAMES


def test_analyzer_and_kernel_share_one_builtin_surface():
    """Two lists would let the analyzer refuse a name the kernel resolves."""
    assert set(engine._SAFE_BUILTINS) >= set(SAFE_BUILTIN_NAMES) - {"__spec__"}


def test_analyzer_reports_import_roots_without_changing_scope_analysis():
    result = analyze.analyze(
        "import vrooli\n"
        "from vrooli import recall\n"
        "import json as payload\n"
        "print(payload.dumps({}))\n"
    )

    assert result["imports"] == [
        {"name": "vrooli", "line": 1},
        {"name": "vrooli", "line": 2},
        {"name": "json", "line": 3},
    ]
    assert result["free"] == []
    assert {entry["name"] for entry in result["bound"]} >= {"vrooli", "recall", "payload"}
