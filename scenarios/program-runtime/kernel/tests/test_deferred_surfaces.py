from host.engine import SessionKernel
from host.engine import Handle, _DelegationSurface
from http.server import BaseHTTPRequestHandler, HTTPServer
import json
import threading


def test_inference_reports_ai_gateway_promotion_blocker():  # [REQ:PRT-P1-002]
    result = SessionKernel().execute("vrooli.ai.classify('hello')")
    assert result["ok"] is False
    assert "ai-gateway inference" in result["error"]
    assert "promotion" in result["error"]


def test_delegation_without_bridge_is_explicitly_unavailable():
    result = SessionKernel().execute("vrooli.agent.run(owner='missing', workflow_key='missing')")
    assert result["ok"] is False
    assert "agent-manager delegation" in result["error"]


def test_delegation_start_and_collect_are_separate_and_session_scoped():
    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            request = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
            if self.path.endswith("/start"):
                payload = {"execution_id": "exec-1", "status": "running"}
            elif self.path.endswith("/collect"):
                if request["session_id"] != "session-1":
                    self.send_response(403)
                    self.end_headers()
                    self.wfile.write(b'{"error":"delegation does not belong to session"}')
                    return
                payload = {"execution_id": request["execution_id"], "status": "succeeded", "evidence": {"ok": True}}
            else:
                self.send_response(404)
                self.end_headers()
                return
            data = json.dumps(payload).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)

        def log_message(self, *_args):
            return

    server = HTTPServer(("127.0.0.1", 0), Handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        surface = _DelegationSurface("session-1", f"http://127.0.0.1:{server.server_port}/agent/execute", [])
        first = surface.start(owner="owner", workflow_key="workflow")
        second = surface.start(owner="owner", workflow_key="workflow")
        assert first.head(1)[0]["status"] == "running"
        assert second.head(1)[0]["execution_id"] == "exec-1"
        collected = surface.collect(first, wait_seconds=1)
        assert collected.head(1)[0]["status"] == "succeeded"

        other = _DelegationSurface("session-2", f"http://127.0.0.1:{server.server_port}/agent/execute", [])
        try:
            other.collect(first)
            assert False, "cross-session collect should be refused"
        except RuntimeError as exc:
            assert "does not belong to session" in str(exc)
    finally:
        server.shutdown()
        thread.join(timeout=2)
