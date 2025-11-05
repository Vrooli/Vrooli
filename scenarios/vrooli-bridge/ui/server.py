#!/usr/bin/env python3
"""Lightweight static file server with /health endpoint for Vrooli Bridge UI."""

import json
import os
from datetime import datetime, timezone
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler
from socketserver import ThreadingTCPServer

BASE_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "dist")
DEFAULT_PORT = 8080


class BridgeRequestHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=BASE_DIR, **kwargs)

    def log_message(self, format: str, *args) -> None:  # noqa: A003 - match base signature
        message = "%s - - [%s] %s\n" % (
            self.client_address[0],
            self.log_date_time_string(),
            format % args,
        )
        try:
            self.wfile.flush()
        except Exception:
            pass
        print(f"[UI] {message}", end="")

    def do_GET(self):  # noqa: N802 - base class signature
        if self.path.rstrip("/") == "":
            self.path = "/index.html"
        if self.path == "/health":
            self._handle_health()
            return
        super().do_GET()

    def _handle_health(self) -> None:
        payload = {
            "status": "healthy",
            "service": "vrooli-bridge-ui",
            "timestamp": datetime.now(timezone.utc).isoformat(),
        }
        encoded = json.dumps(payload).encode("utf-8")
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)


def run() -> None:
    port = int(os.environ.get("UI_PORT") or os.environ.get("PORT") or DEFAULT_PORT)
    address = ("0.0.0.0", port)
    ThreadingTCPServer.allow_reuse_address = True
    with ThreadingTCPServer(address, BridgeRequestHandler) as httpd:
        print(f"[UI] Serving Vrooli Bridge UI on port {port} (http://localhost:{port})")
        httpd.serve_forever()


if __name__ == "__main__":
    run()
