import json
import os
import select
import socket
import struct
import subprocess
import sys
import tempfile
import threading
import urllib.request
from contextvars import ContextVar
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import pytest

sys.path.insert(0, str(Path(__file__).parents[1]))
from host.engine import SessionKernel, _open_bridge_request
from host.tasks import Tasks


class Response:
    def __init__(self, body):
        self.body = body

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def read(self, *_args):
        return json.dumps(self.body).encode()


def test_submitted_python_cannot_read_or_forward_agent_credentials():
    calls = []

    def opened(request, **_kwargs):
        calls.append((request.get_header("X-agent-identity-token"), request.get_header("X-agent-operation-token")))
        if request.full_url.endswith("reachability"):
            return Response({"demo": {"reachable": True}})
        return Response({"items": [{"id": "one"}]})

    kernel = SessionKernel([{"id": "demo/work/run", "scenario": "demo", "group": "work", "command": "run", "effect": "read", "rows_field": "items"}], session_id="session", bridge_url="http://127.0.0.1:1234/bridge/execute")
    with patch("host.engine.urllib.request.build_opener") as builder, patch("host.engine.urllib.request.urlopen", side_effect=opened):
        builder.return_value.open.side_effect = opened
        result = kernel.execute("import host.bridge_transport as transport; print(getattr(transport, 'AGENT_IDENTITY', 'missing')); print(getattr(transport, 'AGENT_OPERATION', 'missing')); print(demo.work.run().count())")
        assert result["ok"]
        assert "fixture-agent-proof" not in json.dumps(result)
        assert "fixture-operation-proof" not in json.dumps(result)
        assert "missing" in result["stdout"]
        assert calls and all(pair == (None, None) for pair in calls)
        calls.clear()
        again = kernel.execute("print(demo.work.run().count())")
        assert again["ok"]
        assert calls and all(pair == (None, None) for pair in calls)


def test_bridge_transport_strips_copied_credentials_from_all_destinations():
    for url in ["https://outside.example/bridge", "http://user@localhost/bridge", "http://localhost/bridge"]:
        request = urllib.request.Request(url, headers={"X-Agent-Identity-Token": "copied-proof", "X-Agent-Operation-Token": "copied-operation"})
        with patch("host.engine.urllib.request.urlopen", return_value=Response({})) as opened:
            if url == "http://localhost/bridge":
                with patch("host.engine.urllib.request.build_opener") as builder:
                    builder.return_value.open.return_value = Response({})
                    _open_bridge_request(request, timeout=1)
                    forwarded = builder.return_value.open.call_args.args[0]
            else:
                _open_bridge_request(request, timeout=1)
                forwarded = opened.call_args.args[0]
            assert forwarded.get_header("X-agent-identity-token") is None
            assert forwarded.get_header("X-agent-operation-token") is None


def test_local_bridge_calls_disable_redirects_and_ambient_proxies():
    with patch("host.engine.urllib.request.build_opener") as builder:
        _open_bridge_request(urllib.request.Request("http://localhost/bridge"), timeout=1)
        proxy, redirect = builder.call_args.args
        assert proxy.proxies == {}
        assert redirect.redirect_request(None, None, 302, "redirect", {}, "https://outside.example") is None


def test_parallel_owner_calls_do_not_receive_credentials_even_on_failure():
    calls = []

    def opened(request, **_kwargs):
        calls.append((request.get_header("X-agent-identity-token"), request.get_header("X-agent-operation-token")))
        if request.full_url.endswith("reachability"):
            return Response({"demo": {"reachable": True}})
        return Response({"items": [{"id": "one"}]})

    kernel = SessionKernel([{"id": "demo/work/run", "scenario": "demo", "group": "work", "command": "run", "effect": "read", "rows_field": "items"}], session_id="session", bridge_url="http://127.0.0.1:1234/bridge/execute")
    with patch("host.engine.urllib.request.build_opener") as builder, patch("host.engine.urllib.request.urlopen", side_effect=opened):
        builder.return_value.open.side_effect = opened
        result = kernel.execute("print([r.count() for r in gather(lambda: demo.work.run(), lambda: demo.work.run())]); raise ValueError('stop')")
        assert not result["ok"]
        assert result["stdout"].strip() == "[1, 1]"
        assert calls and all(pair == (None, None) for pair in calls)
        calls.clear()
        again = kernel.execute("print(demo.work.run().count())")
        assert again["ok"]
        assert calls and all(pair == (None, None) for pair in calls)


def test_learning_task_bridge_does_not_forward_caller_credentials():
    calls = []

    def opened(request, **_kwargs):
        calls.append(request)
        return Response({"task_id": "task-1"})

    owner = SimpleNamespace(
        _bridge_url="http://127.0.0.1:1234/internal/program-runtime/bindings/execute",
        _session_id="session-a",
    )
    invocation = ContextVar("test_learning_invocation", default={})
    invocation.set({"program_id": "program-a"})
    tasks = Tasks(owner, invocation, None, 1)
    with patch("host.engine.urllib.request.build_opener") as builder, patch("host.engine.urllib.request.urlopen", side_effect=opened):
        builder.return_value.open.side_effect = opened
        result = tasks._bridge("begin", operation="fixture.child")

    assert result == {"task_id": "task-1"}
    assert len(calls) == 1
    request = calls[0]
    assert request.full_url.endswith("/internal/program-runtime/tasks/begin")
    assert request.get_header("X-agent-identity-token") is None
    assert request.get_header("X-agent-operation-token") is None
    assert json.loads(request.data) == {
        "operation": "fixture.child",
        "session_id": "session-a",
        "program_id": "program-a",
    }


def test_persistent_caller_diagnostic_is_per_submission_and_does_not_call_owner():
    kernel = SessionKernel(session_id="session", bridge_url="http://127.0.0.1:1234/internal/program-runtime/bindings/execute")
    with patch("tasks.open_bridge_request", return_value=Response({"state": "prepared"})) as opened:
        denied = kernel.execute("tasks.get(attempt_id='attempt')", persistent_task_authority_unavailable=True)
        assert not denied["ok"]
        assert "TaskAuthorityUnavailable" in denied["error"]
        assert "persistent Runtime sessions" in denied["error"]
        opened.assert_not_called()

        # The diagnostic must not carry over to the next submission. The
        # receiver still independently decides whether this call is allowed.
        next_submission = kernel.execute("tasks.get(attempt_id='attempt')")
        assert next_submission["ok"], next_submission
        opened.assert_called_once()


@pytest.mark.skipif(not sys.platform.startswith("linux"), reason="Linux SO_PEERCRED is needed for the process-identity observation")
def test_prior_submission_thread_has_same_peer_pid_during_later_submission():
    """A process-only relay cannot distinguish retained source threads by submission."""
    observed = []
    with tempfile.TemporaryDirectory(prefix="prt-peer-", dir="/tmp") as directory:
        path = str(Path(directory) / "relay.sock")
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as listener:
            listener.bind(path)
            listener.listen(1)
            listener.settimeout(5)

            def accept_one():
                connection, _ = listener.accept()
                with connection:
                    peer = connection.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, struct.calcsize("3i"))
                    observed.append((struct.unpack("3i", peer)[0], connection.recv(64)))
                    connection.sendall(b"ok")

            server = threading.Thread(target=accept_one, daemon=True)
            server.start()
            kernel_path = Path(__file__).parents[1] / "host" / "engine.py"
            process = subprocess.Popen([sys.executable, str(kernel_path)], stdin=subprocess.PIPE,
                                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            def submit(source, program_id):
                process.stdin.write(json.dumps({"source": source, "program_id": program_id}) + "\n")
                process.stdin.flush()
                assert select.select([process.stdout], [], [], 8)[0], "kernel response timed out"
                return json.loads(process.stdout.readline())

            try:
                first = submit('''import socket, threading
release_old = threading.Event()
old_done = threading.Event()
def old_worker():
    if release_old.wait(5):
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as conn:
            conn.connect(later_socket)
            conn.sendall(b"earlier-submission")
            conn.recv(2)
        old_done.set()
old_thread = threading.Thread(target=old_worker, daemon=True)
old_thread.start()
''', "earlier-program")
                assert first["ok"], first
                second = submit(f'''later_socket = {path!r}
release_old.set()
assert old_done.wait(5)
''', "later-program")
                assert second["ok"], second
                server.join(timeout=5)
                assert not server.is_alive()
                assert observed == [(process.pid, b"earlier-submission")]
                assert process.pid != os.getpid()
            finally:
                process.terminate()
                process.wait(timeout=5)
                process.stdin.close()
                process.stdout.close()
                process.stderr.close()


@pytest.mark.skipif(not sys.platform.startswith("linux"), reason="Linux SO_PEERCRED is needed for the process-identity observation")
def test_retained_finalizer_executes_in_later_submission_thread():
    """Per-thread isolation also fails when executable namespace state survives."""
    observed = []
    with tempfile.TemporaryDirectory(prefix="prt-finalizer-", dir="/tmp") as directory:
        path = str(Path(directory) / "relay.sock")
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as listener:
            listener.bind(path)
            listener.listen(1)
            listener.settimeout(5)

            def accept_one():
                connection, _ = listener.accept()
                with connection:
                    peer = connection.getsockopt(socket.SOL_SOCKET, socket.SO_PEERCRED, struct.calcsize("3i"))
                    observed.append((struct.unpack("3i", peer)[0], json.loads(connection.recv(1024))))
                    connection.sendall(b"ok")

            server = threading.Thread(target=accept_one, daemon=True)
            server.start()
            kernel_path = Path(__file__).parents[1] / "host" / "engine.py"
            process = subprocess.Popen([sys.executable, str(kernel_path)], stdin=subprocess.PIPE,
                                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

            def submit(source, program_id):
                process.stdin.write(json.dumps({"source": source, "program_id": program_id}) + "\n")
                process.stdin.flush()
                assert select.select([process.stdout], [], [], 8)[0], "kernel response timed out"
                return json.loads(process.stdout.readline())

            try:
                first = submit(f'''import json, socket, threading
class SavedState:
    def __del__(self):
        with socket.socket(socket.AF_UNIX, socket.SOCK_STREAM) as conn:
            conn.connect({path!r})
            conn.sendall(json.dumps({{"program": tasks.invocation_context.get().get("program_id"),
                                     "thread": threading.get_native_id(), "current_thread": current_thread}}).encode())
            conn.recv(2)
saved_state = SavedState()
''', "earlier-program")
                assert first["ok"], first
                second = submit('''current_thread = threading.get_native_id()
saved_state = None
''', "later-program")
                assert second["ok"], second
                server.join(timeout=5)
                assert not server.is_alive()
                assert len(observed) == 1
                pid, payload = observed[0]
                assert pid == process.pid
                assert payload["program"] == "later-program"
                assert payload["thread"] == payload["current_thread"]
            finally:
                process.terminate()
                process.wait(timeout=5)
                process.stdin.close()
                process.stdout.close()
                process.stderr.close()
