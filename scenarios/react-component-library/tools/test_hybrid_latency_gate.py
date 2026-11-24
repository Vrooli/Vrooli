import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import hybrid_latency_gate as gate


class MockHandler(BaseHTTPRequestHandler):
    response_factory = None

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length))
        payload, status = self.response_factory(body["query"])
        encoded = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def log_message(self, *_args):
        pass


class LatencyGateTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), MockHandler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base_url = "http://127.0.0.1:%d" % cls.server.server_port

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def test_percentile_interpolates_and_rejects_empty(self):
        self.assertEqual(gate.percentile([1, 2, 3, 4], 50), 2.5)
        self.assertEqual(gate.percentile([1, 2, 3, 4], 95), 3.85)
        with self.assertRaises(ValueError):
            gate.percentile([], 95)

    def test_target_is_read_from_search_config(self):
        self.assertEqual(gate.configured_p95_target(), 1500.0)
        config = Path(__file__).with_name("tmp-search-config.json")
        try:
            config.write_text(json.dumps({"providers": [{
                "provider_id": "react-component-library.workbench",
                "performance": {"p95_ms": 321},
            }]}))
            self.assertEqual(gate.configured_p95_target(config), 321.0)
        finally:
            config.unlink(missing_ok=True)

    def test_full_corpus_and_outcome_workspace_first_ranking_pass(self):
        MockHandler.response_factory = staticmethod(lambda query: ({
            "results": [{"title": "Marlow" if query == "landing page" else "Result", "kind": "workspace"}]
        }, 200))
        result = gate.run_gate(self.base_url, 2, 1500)
        self.assertEqual(len(gate.QUERY_CORPUS), 100)
        self.assertEqual(result["successful"], 100)
        self.assertEqual(result["failed"], 0)
        self.assertTrue(result["landing_page_workspace_first"])
        self.assertTrue(result["passed"])

    def test_http_failures_are_counted_and_fail_gate(self):
        MockHandler.response_factory = staticmethod(lambda query: (
            {"results": [{"title": "LandingPage"}]}, 503 if query == "plant care app" else 200)
        )
        result = gate.run_gate(self.base_url, 2, 1500)
        self.assertEqual(result["failed"], 4)  # deterministic corpus repeats every seed four times
        self.assertEqual(result["successful"], 96)
        self.assertFalse(result["passed"])

    def test_target_breach_and_ranking_regression_fail(self):
        fast = lambda _base, query, _timeout: {"results": [{"title": "Other"}]}
        bad_rank = gate.run_gate("unused", 1, 1500, request_fn=fast)
        self.assertFalse(bad_rank["landing_page_workspace_first"])
        self.assertFalse(bad_rank["passed"])

        # An injected request clock-like delay exercises the p95 gate with no network dependency.
        slow = lambda _base, query, _timeout: (threading.Event().wait(0.002) or {
            "results": [{"title": "Marlow", "kind": "workspace"}]
        })
        breached = gate.run_gate("unused", 1, 0.1, request_fn=slow)
        self.assertGreater(breached["p95_ms"], breached["p95_target_ms"])
        self.assertFalse(breached["passed"])


if __name__ == "__main__":
    unittest.main()
