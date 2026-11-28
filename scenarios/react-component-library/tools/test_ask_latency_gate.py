import json
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import ask_latency_gate as gate


class MockHandler(BaseHTTPRequestHandler):
    response_factory = None

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = json.loads(self.rfile.read(length))
        payload, status = self.response_factory(body)
        encoded = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def log_message(self, *_args):
        pass


class AskLatencyGateTests(unittest.TestCase):
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

    def test_config_declares_separate_ask_slo(self):
        self.assertEqual(gate.configured_slo(), (8000.0, 20))
        config = Path(__file__).with_name("tmp-ask-search-config.json")
        try:
            config.write_text(json.dumps({"providers": [{
                "provider_id": "react-component-library.workbench",
                "performance": {"ask_p95_ms": 321, "ask_minimum_samples": 1},
            }]}))
            self.assertEqual(gate.configured_slo(config), (321.0, 1))
        finally:
            config.unlink(missing_ok=True)

    def test_citation_contract_rejects_missing_or_unresolved_ids(self):
        self.assertEqual(gate.citation_contract_error({"answer": {"text": "ok"}}), "answer citations are missing")
        payload = {
            "results": [{"id": "asset:button"}],
            "answer": {"text": "ok", "citations": [{"resultId": "asset:other"}]},
        }
        self.assertIn("outside", gate.citation_contract_error(payload))

    def test_citation_contract_rejects_an_opaque_result_id_as_the_answer(self):
        payload = {
            "results": [{"id": "workspace:ws_curated_marlow"}],
            "answer": {
                "text": "workspace:ws_curated_marlow",
                "citations": [{"resultId": "workspace:ws_curated_marlow"}],
            },
        }
        self.assertEqual(gate.citation_contract_error(payload), "answer text is an opaque result ID")

    def test_citation_contract_reports_malformed_json_shapes_without_crashing(self):
        for payload in (None, [], "answer", {"answer": {"text": "ok", "citations": [None]}}):
            with self.subTest(payload=payload):
                self.assertIsInstance(gate.citation_contract_error(payload), str)
        self.assertEqual(
            gate.citation_contract_error({"answer": {"text": "ok", "citations": [{"resultId": "x"}]}}),
            "supplied results are missing",
        )

    def test_slo_rejects_boolean_values(self):
        config = Path(__file__).with_name("tmp-ask-search-config.json")
        try:
            for performance in ({"ask_p95_ms": True, "ask_minimum_samples": 1},
                                {"ask_p95_ms": 100, "ask_minimum_samples": True}):
                config.write_text(json.dumps({"providers": [{
                    "provider_id": "react-component-library.workbench", "performance": performance,
                }]}))
                with self.assertRaises(ValueError):
                    gate.configured_slo(config)
        finally:
            config.unlink(missing_ok=True)

    def test_service_timing_accepts_json_int64_and_rejects_invalid_values(self):
        self.assertEqual(gate.service_took_ms({"tookMs": "17"}), 17)
        self.assertEqual(gate.service_took_ms({"tookMs": 17}), 17)
        self.assertIsNone(gate.service_took_ms({"tookMs": True}))
        self.assertIsNone(gate.service_took_ms({"tookMs": "invalid"}))
        self.assertIsNone(gate.service_took_ms({"tookMs": -1}))

    def test_full_ask_corpus_passes_with_valid_unique_citations(self):
        def response(body):
            self.assertTrue(body.get("ask"))
            payload = {
                "results": [{"id": "asset:button"}],
                "answer": {"text": "Use the button component.", "citations": [{"resultId": "asset:button"}]},
                "tookMs": "12",
            }
            return payload, 200

        MockHandler.response_factory = staticmethod(response)
        result = gate.run_gate(self.base_url, 2, 8000, 20)
        self.assertEqual(result["samples"], 20)
        self.assertEqual(result["successful"], 20)
        self.assertEqual(result["failed"], 0)
        self.assertTrue(result["passed"])
        self.assertEqual(result["timing_breakdown"]["samples_with_service_timing"], 20)
        self.assertEqual(result["timing_breakdown"]["search_and_status"]["max_ms"], 12)
        self.assertEqual(result["measurements"][0]["search_and_status_ms"], 12)

    def test_contract_failure_and_latency_breach_fail_gate(self):
        invalid = lambda *_args: {"results": [], "answer": {"text": "unreferenced"}}
        failed = gate.run_gate("unused", 1, 8000, 1, request_fn=invalid)
        self.assertEqual(failed["failed"], len(gate.ASK_QUERIES))
        self.assertFalse(failed["passed"])

        slow = lambda *_args: (threading.Event().wait(0.002) or {
            "results": [{"id": "asset:button"}],
            "answer": {"text": "Use the button component.", "citations": [{"resultId": "asset:button"}]},
        })
        breached = gate.run_gate("unused", 1, 0.1, 20, request_fn=slow)
        self.assertGreater(breached["p95_ms"], breached["p95_target_ms"])
        self.assertFalse(breached["passed"])


if __name__ == "__main__":
    unittest.main()
