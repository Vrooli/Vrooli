"""Thin HTTP transport; runtime.py owns jobs and model-cache mutations."""
import argparse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
import re
import signal
import threading

from runtime import Refused, Runtime, connection_closed


def handler(runtime):
    class Handler(BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"
        timeout = 10

        def reply(self, status, value):
            self.close_connection = True
            self.connection.settimeout(30)
            body = json.dumps(value, allow_nan=False).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Connection", "close")
            self.send_header("Cache-Control", "no-store")
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            self.connection.settimeout(30)
            try:
                if self.path == "/live":
                    return self.reply(200, {"status": "alive", "workload_active": runtime.active is not None})
                if self.path == "/health":
                    health = runtime.health()
                    return self.reply(200 if health["status"] == "ready" else 503, health)
                match = re.fullmatch(r"/v1/operations/([0-9a-f]{32})/(audio|receipt)", self.path)
                if not match:
                    return self.reply(404, {"error": "unknown resource route"})
                folder = runtime.outputs / match[1]
                path = folder / ("audio.wav" if match[2] == "audio" else "receipt.json")
                if not path.is_file() or folder.is_symlink():
                    return self.reply(404, {"error": "completed operation not found"})
                self.close_connection = True
                self.send_response(200)
                self.send_header("Connection", "close")
                self.send_header("Content-Type", "audio/wav" if match[2] == "audio" else "application/json")
                self.send_header("Content-Length", str(path.stat().st_size))
                self.send_header("Cache-Control", "no-store")
                self.end_headers()
                with path.open("rb") as stream:
                    while chunk := stream.read(65536):
                        self.wfile.write(chunk)
            except (BrokenPipeError, ConnectionResetError, TimeoutError):
                pass

        def do_POST(self):
            try:
                if self.headers.get("Transfer-Encoding"):
                    raise Refused("chunked requests are not supported")
                length = int(self.headers.get("Content-Length", "0"))
                if not 0 < length <= 65536:
                    raise Refused("request must have a body of 1..65536 bytes", 413)
                self.connection.settimeout(10)
                raw = self.rfile.read(length)
                if len(raw) != length:
                    raise Refused("incomplete request")
                request = json.loads(raw)
                self.connection.settimeout(None)
                if self.path == "/v1/compose":
                    return self.reply(200, runtime.generate(request, lambda: connection_closed(self.connection), raw=raw))
                if self.path == "/v1/models/retention":
                    return self.reply(200, runtime.models(request))
                self.reply(404, {"error": "unknown resource route"})
            except Refused as exc:
                self.reply(exc.status, {"error": str(exc)})
            except (ValueError, TypeError, KeyError) as exc:
                self.reply(400, {"error": str(exc)})
            except (BrokenPipeError, ConnectionResetError, TimeoutError):
                pass
            except Exception as exc:
                self.log_error("resource operation failed: %s", exc)
                self.reply(500, {"error": "resource operation failed; inspect resource logs"})
    return Handler


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, required=True)
    args = parser.parse_args()
    runtime = Runtime(os.environ["SONGGEN_DATA_DIR"], os.environ["SONGGEN_STATE_DIR"],
                      os.environ["SONGGEN_ARTIFACT_DIR"], os.environ["SONGGEN_DATA_LOCK_PATH"])
    server = ThreadingHTTPServer(("127.0.0.1", args.port), handler(runtime))
    server.daemon_threads = False
    def stop(*_):
        runtime.stopping.set()
        runtime.terminate()
        threading.Thread(target=server.shutdown, daemon=True).start()
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    try:
        server.serve_forever(poll_interval=0.2)
    finally:
        stop()
        server.server_close()


if __name__ == "__main__":
    main()
