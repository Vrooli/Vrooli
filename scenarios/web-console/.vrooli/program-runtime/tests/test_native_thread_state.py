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
