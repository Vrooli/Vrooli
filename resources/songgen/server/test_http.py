"""Transport regressions use no model, CUDA import, or model installation."""
import http.client
from http.server import ThreadingHTTPServer
import json
import threading
import unittest

from server import handler


class FakeRuntime:
    def __init__(self):
        self.calls = 0

    def generate(self, request, disconnected, *, raw):
        self.calls += 1
        return {"status": "completed"}


class TransportTests(unittest.TestCase):
    def test_completed_post_closes_keepalive_and_does_not_strand_shutdown(self):
        runtime = FakeRuntime()
        server = ThreadingHTTPServer(("127.0.0.1", 0), handler(runtime))
        thread = threading.Thread(target=server.serve_forever, kwargs={"poll_interval": 0.01})
        thread.start()
        client = http.client.HTTPConnection(*server.server_address, timeout=2)
        try:
            client.request("POST", "/v1/compose", json.dumps({"fixture": True}), {"Connection": "keep-alive"})
            response = client.getresponse()
            self.assertEqual(response.status, 200)
            self.assertTrue(response.will_close)
            response.read()
            self.assertEqual(runtime.calls, 1)
            server.shutdown()
            closed = threading.Event()
            closer = threading.Thread(target=lambda: (server.server_close(), closed.set()))
            closer.start()
            self.assertTrue(closed.wait(2), "open client stranded resource shutdown")
            closer.join(2)
        finally:
            client.close()
            server.shutdown()
            server.server_close()
            thread.join(2)

    def test_oversized_body_is_refused_before_runtime(self):
        runtime = FakeRuntime()
        server = ThreadingHTTPServer(("127.0.0.1", 0), handler(runtime))
        thread = threading.Thread(target=server.serve_forever, kwargs={"poll_interval": 0.01})
        thread.start()
        client = http.client.HTTPConnection(*server.server_address, timeout=2)
        try:
            client.request("POST", "/v1/compose", headers={"Content-Length": "65537"})
            response = client.getresponse()
            self.assertEqual(response.status, 413)
            response.read()
            self.assertEqual(runtime.calls, 0)
        finally:
            client.close()
            server.shutdown()
            server.server_close()
            thread.join(2)
