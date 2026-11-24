"""Agent Manager task binding tests; no Browser Use install or service needed."""

from __future__ import annotations

import asyncio
import json
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
import unittest
from unittest.mock import patch

from . import browser_use_adapter
from .browser_use_adapter import BASAgentManagerTaskTransport


class _Response:
    def __init__(self, body: bytes):
        self.body = body

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return None

    def read(self, size: int = -1) -> bytes:
        result, self.body = self.body[:size], self.body[size:]
        return result


class BASAgentManagerTaskTransportTests(unittest.TestCase):
    def setUp(self):
        self.env = {
            "BAS_BROWSER_API_URL": "http://bas.local",
            "VROOLI_AGENT_IDENTITY_TOKEN": "identity-secret",
            "VROOLI_BROWSER_GRANT_RECEIPT": "grant-secret",
        }

    def test_binds_session_and_keeps_credentials_out_of_body(self):
        transport = BASAgentManagerTaskTransport("session/one", environment=self.env)
        captured = {}

        def fake_urlopen(request, timeout):
            captured["url"] = request.full_url
            captured["headers"] = request.headers
            captured["body"] = request.data
            captured["timeout"] = timeout
            return _Response(b'{"accepted":true}')

        with patch.object(browser_use_adapter, "_open_task_request", side_effect=fake_urlopen):
            result = asyncio.run(transport.run_task("session/one", {"operation": "read"}))
        self.assertEqual(result, {"accepted": True})
        self.assertEqual(captured["url"], "http://bas.local/api/v1/browser/sessions/session%2Fone/task")
        self.assertEqual(captured["headers"]["X-agent-identity-token"], "identity-secret")
        self.assertEqual(captured["headers"]["X-vrooli-browser-grant-receipt"], "grant-secret")
        self.assertEqual(json.loads(captured["body"]), {"operation": "read"})
        self.assertNotIn(b"identity-secret", captured["body"])
        self.assertNotIn(b"grant-secret", captured["body"])

    def test_receipt_operation_uses_same_authenticated_session_task_contract(self):
        transport = BASAgentManagerTaskTransport("session-1", environment=self.env)
        captured = {}

        def fake_urlopen(request, timeout):
            captured["url"] = request.full_url
            captured["headers"] = request.headers
            captured["body"] = request.data
            return _Response(b'{"settlement_status":"settled","receipt":{"reservation_id":"hold-1","status":"settled"}}')

        with patch.object(browser_use_adapter, "_open_task_request", side_effect=fake_urlopen):
            result = asyncio.run(transport.run_task("session-1", {"operation": "receipt"}))
        self.assertEqual(result["settlement_status"], "settled")
        self.assertEqual(result["receipt"]["reservation_id"], "hold-1")
        self.assertEqual(captured["url"], "http://bas.local/api/v1/browser/sessions/session-1/task")
        self.assertEqual(captured["headers"]["X-agent-identity-token"], "identity-secret")
        self.assertEqual(captured["headers"]["X-vrooli-browser-grant-receipt"], "grant-secret")
        self.assertEqual(json.loads(captured["body"]), {"operation": "receipt"})
        self.assertNotIn(b"identity-secret", captured["body"])
        self.assertNotIn(b"grant-secret", captured["body"])

    def test_trace_draft_uses_existing_session_route_without_project_authority(self):
        transport = BASAgentManagerTaskTransport("session-1", environment=self.env)
        captured = {}
        request_body = {
            "name": "Checkout preview",
            "description": "Unsaved draft",
            "assertions": [{"type": "ACTION_TYPE_ASSERT", "assert": {"selector": "#done", "mode": "ASSERTION_MODE_VISIBLE"}}],
        }

        def fake_urlopen(request, timeout):
            captured["url"] = request.full_url
            captured["headers"] = request.headers
            captured["body"] = request.data
            captured["timeout"] = timeout
            return _Response(b'{"session_id":"session-1","draft_digest":"sha256:preview","workflow_id":""}')

        with patch.object(browser_use_adapter, "_open_task_request", side_effect=fake_urlopen):
            result = asyncio.run(transport.draft_trace("session-1", request_body))
        self.assertEqual(result["draft_digest"], "sha256:preview")
        self.assertEqual(captured["url"], "http://bas.local/api/v1/browser/sessions/session-1/trace/draft")
        self.assertEqual(captured["headers"]["X-agent-identity-token"], "identity-secret")
        self.assertEqual(captured["headers"]["X-vrooli-browser-grant-receipt"], "grant-secret")
        self.assertEqual(json.loads(captured["body"]), request_body)
        self.assertNotIn("project_id", json.loads(captured["body"]))
        self.assertNotIn(b"identity-secret", captured["body"])
        self.assertNotIn(b"grant-secret", captured["body"])

    def test_trace_draft_rejects_other_sessions_and_project_fields_before_transport(self):
        transport = BASAgentManagerTaskTransport("session-1", environment=self.env)
        request = {"name": "Preview", "assertions": [{"type": "ACTION_TYPE_ASSERT"}]}
        with patch.object(browser_use_adapter, "_open_task_request") as send:
            with self.assertRaises(PermissionError):
                asyncio.run(transport.draft_trace("session-2", request))
            with self.assertRaises(ValueError):
                asyncio.run(transport.draft_trace("session-1", {**request, "project_id": "forged"}))
        send.assert_not_called()

    def test_requires_authority_and_fixed_session(self):
        with self.assertRaises(ValueError):
            BASAgentManagerTaskTransport("s", environment={"BAS_BROWSER_API_URL": "http://bas.local"})
        transport = BASAgentManagerTaskTransport("session-1", environment=self.env)
        with self.assertRaises(PermissionError):
            asyncio.run(transport.run_task("session-2", {"operation": "read"}))

    def test_rejects_unsupported_or_forged_task_before_authenticated_transport(self):
        transport = BASAgentManagerTaskTransport("session-1", environment=self.env)
        with patch.object(browser_use_adapter, "_open_task_request") as request:
            for task in (
                {"operation": "evaluate", "code": "document.cookie"},
                {"operation": "click", "ref": "e123", "grant_receipt": "forged"},
                {"operation": "click", "ref": "#continue"},
                {"operation": "fill", "ref": "e123"},
                {"operation": "navigate", "url": " "},
            ):
                with self.subTest(task=task), self.assertRaises(ValueError):
                    asyncio.run(transport.run_task("session-1", task))
        request.assert_not_called()

    def test_rejects_non_origin_base_url(self):
        for base in ("http://user:pass@bas.local", "http://bas.local/path", "http://bas.local/?token=x"):
            env = dict(self.env, BAS_BROWSER_API_URL=base)
            with self.subTest(base=base), self.assertRaises(ValueError):
                BASAgentManagerTaskTransport("s", environment=env)

    def test_redirect_is_rejected_without_forwarding_run_credentials(self):
        forwarded = []

        class DestinationHandler(BaseHTTPRequestHandler):
            def do_POST(self):
                forwarded.append(dict(self.headers.items()))
                self.send_response(200)
                self.end_headers()

            def log_message(self, *_args):
                return

        destination = ThreadingHTTPServer(("127.0.0.1", 0), DestinationHandler)
        destination_thread = Thread(target=destination.serve_forever, daemon=True)
        destination_thread.start()

        class RedirectHandler(BaseHTTPRequestHandler):
            def do_POST(self):
                self.send_response(307)
                self.send_header("Location", f"http://127.0.0.1:{destination.server_port}/collect")
                self.end_headers()

            def log_message(self, *_args):
                return

        redirector = ThreadingHTTPServer(("127.0.0.1", 0), RedirectHandler)
        redirect_thread = Thread(target=redirector.serve_forever, daemon=True)
        redirect_thread.start()
        env = dict(self.env, BAS_BROWSER_API_URL=f"http://127.0.0.1:{redirector.server_port}")
        transport = BASAgentManagerTaskTransport("session-1", environment=env)
        try:
            with self.assertRaises(PermissionError):
                asyncio.run(transport.run_task("session-1", {"operation": "read"}))
            self.assertEqual(forwarded, [])
        finally:
            redirector.shutdown()
            redirector.server_close()
            destination.shutdown()
            destination.server_close()
            redirect_thread.join(timeout=1)
            destination_thread.join(timeout=1)


if __name__ == "__main__":
    unittest.main()
