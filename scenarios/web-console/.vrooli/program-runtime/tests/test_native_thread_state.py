"""Native state is composable without interpreting observation as success."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS

SOURCE = Path(__file__).parents[1] / "native-thread-state.py"


def run(inputs, rows):
    output = io.StringIO()
    native = NS(session=NS(native_thread_state=lambda **_: NS(head=lambda _: rows)))
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {"program": NS(inputs=lambda: inputs), "web_console": native})
    return json.loads(output.getvalue())


def test_native_result_is_a_complete_child_envelope():
    result = run({"session_id": "s", "threads": ["t"]}, [{"status": "idle", "goalAvailable": True}])
    assert result["status"] == "ok"
    assert result["inputs"] == {"threads": ["t"]}
    assert result["evidence"] == ["web-console/session/native-thread-state"]


def test_unavailable_thread_is_partial_and_missing_input_fails():
    result = run({"session_id": "s", "threads": ["t"]}, [{"status": "unavailable"}])
    assert result["status"] == "partial"
    assert run({"threads": ["t"]}, [])['errors'][0]['class'] == 'invalid_input'


def test_native_usage_requires_time_bounds_before_calling_owner():
    source = Path(__file__).parents[1] / "native-usage-summary.py"
    called = []
    def read(**kwargs):
        called.append(kwargs)
        return NS(head=lambda _: [{"status": "ok"}], meta=lambda: {"status": "ok"})
    def run_usage(inputs):
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            exec(compile(source.read_text(), str(source), "exec"), {
                "program": NS(inputs=lambda: inputs),
                "web_console": NS(session=NS(native_usage=read)),
            })
        return json.loads(output.getvalue())
    base = {"session_id": "s", "threads": ["t"]}
    assert run_usage(base)["errors"][0]["class"] == "invalid_input"
    assert not called
    result = run_usage({**base, "start": "2026-10-30T00:00:00Z", "end": "2026-10-31T00:00:00Z"})
    assert result["status"] == "ok"
    assert called[0]["start"] == "2026-10-30T00:00:00Z"
