"""The `program` helper and the typed bridge exceptions.

Sixty-one contract programs carried a verbatim copy of `classify_transport`,
`fail`, the `inputs` guard, `guarded` and the driver loop because a program is
one file and the kernel withholds shared imports. This file pins the behaviour
the copies encoded, now owned by the kernel:

- every bridge failure is a typed `BindingError` carrying the closed class;
- `program.classify` reads the class and never relabels a missing name;
- `program.run` guarantees one envelope on every path;
- `program.report` prints exactly once, through the environment's `print`,
  so a nested library call's capture still works.
"""
import io
import json
import sys
import urllib.error
from pathlib import Path
from unittest.mock import patch

import pytest

sys.path.insert(0, str(Path(__file__).parents[1]))

from host import engine, program_helper  # noqa: E402
from host.engine import SessionKernel  # noqa: E402
from host.program_helper import (  # noqa: E402
    AmbiguousResponse,
    BindingError,
    DeadlineExceeded,
    InvalidInput,
    NoGovernedBinding,
    ProgramHelper,
    Refused,
    RemoteError,
    ScenarioUnreachable,
    bridge_error,
    classify_message,
)

BRIDGE = "http://127.0.0.1:1/internal/program-runtime/bindings/execute"
BINDINGS = [{
    "id": "demo/work/run", "namespace": "demo", "scenario": "demo", "group": "work",
    "command": "run", "effect": "read", "reachable": True,
}]


def http_error(code, body):
    payload = body if isinstance(body, (bytes, str)) else json.dumps(body)
    if isinstance(payload, str):
        payload = payload.encode()
    return urllib.error.HTTPError(BRIDGE, code, "error", {}, io.BytesIO(payload))


class _Reachable:
    """The reachability probe's answer: the scenario is up, so the execute call is what fails."""

    def __enter__(self):
        return self

    def __exit__(self, *_exc):
        return False

    def read(self):
        return json.dumps({"demo": {"reachable": True, "reason": "scenario API resolved"}}).encode()


def failing_execute(error):
    """A urlopen stand-in that answers the reachability probe and raises `error` for the execute call."""

    def urlopen(request, timeout=None):
        if request.full_url.endswith("/reachability"):
            return _Reachable()
        raise error

    return urlopen


# -- typed exceptions from bridge bodies -----------------------------------

@pytest.mark.parametrize("body, exc_type, status, klass, http_status", [
    ({"error": "binding demo/work/run is unreachable: x", "class": "scenario_unreachable", "status": "unavailable"}, ScenarioUnreachable, "unavailable", "scenario_unreachable", 0),
    ({"error": "destructive binding requires an explicit grant", "class": "no_grant", "status": "refused"}, Refused, "refused", "no_grant", 0),
    ({"error": "inference_spend_exceeded: ceiling=1", "class": "inference_spend_exceeded", "status": "refused"}, Refused, "refused", "inference_spend_exceeded", 0),
    ({"error": "decode binding arguments: proto: syntax error", "class": "invalid_input", "status": "failed"}, InvalidInput, "failed", "invalid_input", 0),
    ({"error": "no determinable primary response field", "class": "ambiguous_response", "status": "failed"}, AmbiguousResponse, "failed", "ambiguous_response", 0),
    ({"error": "context deadline exceeded", "class": "deadline_exceeded", "status": "failed"}, DeadlineExceeded, "failed", "deadline_exceeded", 0),
    ({"error": "invoke x: remote status 500: boom", "class": "remote_error", "status": "failed", "http_status": 500}, RemoteError, "failed", "remote_error", 500),
    ({"error": "invoke x: remote status 503: draining", "class": "scenario_unreachable", "status": "unavailable", "http_status": 503}, ScenarioUnreachable, "unavailable", "scenario_unreachable", 503),
    ({"error": "binding does not exist: x", "class": "no_governed_binding", "status": "failed"}, NoGovernedBinding, "failed", "no_governed_binding", 0),
    ({"error": "something else", "class": "binding_error", "status": "failed"}, BindingError, "failed", "binding_error", 0),
])
def test_bridge_body_class_becomes_the_typed_exception(body, exc_type, status, klass, http_status):
    exc = bridge_error("demo/work/run", 400, body)
    assert type(exc) is exc_type
    assert exc.classification() == (status, klass)
    assert exc.binding_id == "demo/work/run"
    assert exc.http_status == http_status
    assert exc.detail == body["error"]


def test_legacy_bridge_body_without_a_class_is_classified_from_its_message():
    exc = bridge_error("demo/work/run", 400, {"error": "binding demo/work/run is unreachable: scenario API is unavailable"})
    assert isinstance(exc, ScenarioUnreachable)
    exc = bridge_error("demo/work/run", 400, "invoke demo/work/run: remote status 502 Bad Gateway: x")
    assert isinstance(exc, ScenarioUnreachable) and exc.http_status == 502
    exc = bridge_error("demo/work/run", 400, "invoke demo/work/run: remote status 404 Not Found: x")
    assert isinstance(exc, RemoteError) and exc.http_status == 404


def test_unknown_class_degrades_to_the_generic_binding_error():
    exc = bridge_error("demo/work/run", 400, {"error": "x", "class": "made_up", "status": "failed"})
    assert type(exc) is BindingError and exc.klass == "binding_error"


def test_typed_exceptions_keep_their_python_ancestry():
    """Programs written `except TypeError` or `except ValueError` still catch the typed forms."""
    assert issubclass(InvalidInput, TypeError)
    assert issubclass(AmbiguousResponse, ValueError)
    assert issubclass(DeadlineExceeded, TimeoutError)
    for cls in (ScenarioUnreachable, Refused, InvalidInput, AmbiguousResponse, DeadlineExceeded, RemoteError, NoGovernedBinding):
        assert issubclass(cls, BindingError) and issubclass(cls, RuntimeError)


def test_classify_message_covers_the_canon_table():
    table = {
        "binding x is unreachable: y": ("unavailable", "scenario_unreachable"),
        "binding bridge unavailable: <urlopen error>": ("unavailable", "scenario_unreachable"),
        "dial tcp 127.0.0.1:1: connection refused": ("unavailable", "scenario_unreachable"),
        'destructive binding "x" requires an explicit grant': ("refused", "no_grant"),
        "manifest governance declares run_eligible=false": ("refused", "not_run_eligible"),
        "inference_spend_exceeded: ceiling=1": ("refused", "inference_spend_exceeded"),
        "delegated_run_spend_exceeded: ceiling=1": ("refused", "delegated_run_spend_exceeded"),
        "binding x has no determinable primary response field": ("failed", "ambiguous_response"),
        "binding x rows must be one of: a, b": ("failed", "ambiguous_response"),
        "x accepts named proto fields, not positional arguments": ("failed", "invalid_input"),
        "decode binding arguments: proto: syntax error": ("failed", "invalid_input"),
        "binding does not exist: x": ("failed", "no_governed_binding"),
        "invoke x: remote status 500 Internal Server Error": ("failed", "remote_error"),
        "context deadline exceeded": ("failed", "deadline_exceeded"),
        "wall budget exhausted: ceiling=1s": ("failed", "deadline_exceeded"),
        "anything else": ("failed", "binding_error"),
    }
    for message, expected in table.items():
        assert classify_message(message) == expected, message


# -- the kernel raises typed exceptions ------------------------------------

def test_bridge_binding_raises_typed_exception_from_structured_body():
    kernel = SessionKernel(BINDINGS, "sess", BRIDGE, "", "", [])
    error = http_error(400, {"error": "invoke demo/work/run: remote status 500 Internal Server Error: boom", "class": "remote_error", "status": "failed", "http_status": 500})
    source = (
        "try:\n    demo.work.run()\nexcept program.RemoteError as exc:\n"
        "    print(json.dumps({'k': exc.klass, 's': exc.status, 'h': exc.http_status, 'b': exc.binding_id}))\n"
    )
    with patch("host.engine.urllib.request.urlopen", side_effect=failing_execute(error)):
        result = kernel.execute("import json\n" + source)
    assert result["ok"], result
    assert json.loads(result["stdout"]) == {"k": "remote_error", "s": "failed", "h": 500, "b": "demo/work/run"}


def test_bridge_binding_transport_failure_is_scenario_unreachable():
    kernel = SessionKernel(BINDINGS, "sess", BRIDGE, "", "", [])
    with patch("host.engine.urllib.request.urlopen", side_effect=failing_execute(urllib.error.URLError("connection refused"))):
        result = kernel.execute("try:\n    demo.work.run()\nexcept program.ScenarioUnreachable as exc:\n    print(program.classify(exc))")
    assert result["ok"], result
    assert "('unavailable', 'scenario_unreachable')" in result["stdout"]


def test_positional_arguments_are_invalid_input_and_still_a_type_error():
    kernel = SessionKernel(BINDINGS, "sess", BRIDGE, "", "", [])
    result = kernel.execute("try:\n    demo.work.run(1)\nexcept TypeError as exc:\n    print(type(exc).__name__, program.classify(exc))")
    assert result["ok"], result
    assert "InvalidInput ('failed', 'invalid_input')" in result["stdout"]


def test_unbound_bridge_is_unreachable_not_a_bare_runtime_error():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    result = kernel.execute("try:\n    demo.work.run()\nexcept program.BindingError as exc:\n    print(program.classify(exc))")
    assert result["ok"], result
    assert "scenario_unreachable" in result["stdout"]


# -- the helper's own surface --------------------------------------------

def test_program_is_bound_protected_and_declared():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    assert "program" in engine._RUNTIME_VERB_NAMES
    # engine imports the helper by bare module name; compare against that copy.
    assert isinstance(kernel.globals["program"], engine.ProgramHelper)
    result = kernel.execute("program = 1")
    assert not result["ok"] and "protected runtime name 'program'" in result["error"]
    for member in program_helper.PUBLIC_MEMBERS:
        assert hasattr(kernel.globals["program"], member), member


def test_public_members_tuple_is_the_whole_public_surface():
    public = {name for name in dir(ProgramHelper) if not name.startswith("_")}
    assert public == set(program_helper.PUBLIC_MEMBERS)


def test_classify_never_relabels_a_missing_kernel_name():
    helper = ProgramHelper()
    with pytest.raises(NameError):
        helper.classify(NameError("name 'gather' is not defined"))
    with pytest.raises(AttributeError):
        helper.classify(AttributeError("x"))
    assert helper.classify(RuntimeError("binding x is unreachable")) == ("unavailable", "scenario_unreachable")
    assert helper.classify(KeyError("rate")) == ("failed", "binding_error")


def test_inputs_reads_the_injected_value_or_an_empty_dict():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    assert kernel.execute("print(program.inputs())")["stdout"].strip() == "{}"
    result = kernel.execute("inputs = {'device': 'tv'}\nprint(program.inputs())")
    assert result["stdout"].strip() == "{'device': 'tv'}"
    assert kernel.execute("print(program.inputs({'a': 1}))")["stdout"].strip() == "{'device': 'tv'}", "session inputs persist"


def test_envelope_fail_run_and_report_print_exactly_one_envelope():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    source = (
        "envelope = program.envelope('demo.read', '1')\n"
        "def step_validate():\n    envelope['phase'] = 'collect'\n    return 'collect'\n"
        "def step_collect():\n    return program.fail('unavailable', 'scenario_unreachable', 'demo is down', 'collect')\n"
        "def step_report():\n    envelope['phase'] = 'report'\n    program.report()\n"
        "program.run({'validate': step_validate, 'collect': step_collect, 'report': step_report})\n"
    )
    result = kernel.execute(source)
    assert result["ok"], result
    lines = [line for line in result["stdout"].splitlines() if line.strip()]
    assert len(lines) == 1
    envelope = json.loads(lines[0])
    assert envelope["program"] == "demo.read" and envelope["status"] == "unavailable" and envelope["phase"] == "report"
    assert envelope["errors"] == [{"class": "scenario_unreachable", "detail": "demo is down", "where": "collect"}]
    assert envelope["inputs"] == {} and envelope["signals"] == {} and envelope["evidence"] == []


def test_run_catches_a_phase_exception_as_kernel_runtime_and_still_reports():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    source = (
        "envelope = program.envelope('demo.read', '1')\n"
        "def step_validate():\n    envelope['phase'] = 'validate'\n    return {}['missing']\n"
        "def step_report():\n    envelope['phase'] = 'report'\n    program.report()\n"
        "program.run({'validate': step_validate, 'report': step_report})\n"
    )
    result = kernel.execute(source)
    assert result["ok"], result
    envelope = json.loads(result["stdout"])
    assert envelope["status"] == "failed"
    assert envelope["errors"][0]["class"] == "kernel_runtime" and envelope["errors"][0]["where"] == "validate"


def test_run_reraises_a_failure_inside_report():
    """A report that cannot print has no envelope to fall back to; the kernel must see the real error."""
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    source = (
        "envelope = program.envelope('demo.read', '1')\n"
        "def step_report():\n    envelope['phase'] = 'report'\n    raise KeyError('boom')\n"
        "program.run({'validate': lambda: 'report', 'report': step_report})\n"
    )
    result = kernel.execute(source)
    assert not result["ok"] and "boom" in result["error"]


def test_report_refuses_a_second_envelope_and_resets_per_submission():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    result = kernel.execute("e = program.envelope('demo.read', '1')\nprogram.report()\nprogram.report()")
    assert not result["ok"] and "exactly one envelope" in result["error"]
    result = kernel.execute("e = program.envelope('demo.read', '1')\nprogram.report()")
    assert result["ok"], "a new submission reports again"


def test_fail_and_run_adopt_a_program_built_envelope():
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [])
    source = (
        "import json\n"
        "envelope = {'program': 'demo.read', 'version': '1', 'status': 'failed', 'phase': 'validate', 'inputs': {}, 'signals': {}, 'errors': [], 'evidence': []}\n"
        "def step_validate():\n    return program.fail('failed', 'invalid_input', 'missing device', 'validate')\n"
        "def step_report():\n    envelope['phase'] = 'report'\n    print(json.dumps(envelope, allow_nan=False))\n"
        "program.run({'validate': step_validate, 'report': step_report})\n"
    )
    result = kernel.execute(source)
    assert result["ok"], result
    assert json.loads(result["stdout"])["errors"][0]["class"] == "invalid_input"


def test_guarded_returns_the_exception_instead_of_raising():
    helper = ProgramHelper()
    boom = ScenarioUnreachable("down", binding_id="demo/work/run")

    def read():
        raise boom

    assert helper.guarded(read)() is boom
    assert helper.guarded(lambda: 3)() == 3


def test_current_without_an_envelope_names_the_fix():
    with pytest.raises(RuntimeError, match="program.envelope"):
        ProgramHelper().fail("failed", "x", "y", "z")


# -- nested library programs ------------------------------------------------

def test_library_child_report_is_captured_and_helper_version_is_checked():
    child = {
        "name": "read", "scenario": "demo", "contract": True,
        "declaration": {"name": "demo.read", "version": "1", "inputs": {}, "budget": {"output_bytes": 4096}, "helpers": {"program": "1"}},
        "source": (
            "envelope = program.envelope('demo.read', '1')\n"
            "envelope['status'] = 'ok'\n"
            "envelope['phase'] = 'report'\n"
            "program.report()\n"
        ),
    }
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [child])
    result = kernel.execute("row = lib.demo.read().head(1)[0]\nprint(row['status'], row['program'])")
    assert result["ok"], result
    assert result["stdout"].strip() == "ok demo.read", "the child's envelope is a row, not parent stdout"

    stale = dict(child, declaration=dict(child["declaration"], helpers={"program": "99"}))
    kernel = SessionKernel(BINDINGS, "sess", "", "", "", [stale])
    result = kernel.execute("lib.demo.read()")
    assert not result["ok"] and "program helper version '99'" in result["error"]
