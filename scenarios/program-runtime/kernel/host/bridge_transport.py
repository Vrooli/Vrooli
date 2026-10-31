"""Trusted local HTTP transport shared by Runtime bridge clients."""
from __future__ import annotations

import urllib.parse
import urllib.request
import urllib.error
import http.client
import io
import os
import socket
import json

class _NoAgentRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


class _UnixConnection(http.client.HTTPConnection):
    def __init__(self, path, timeout):
        super().__init__("localhost", timeout=timeout)
        self._socket_path = path

    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(self.timeout)
        self.sock.connect(self._socket_path)


class _UnixResponse:
    def __init__(self, response, connection):
        self._response, self._connection = response, connection

    def read(self, *args):
        return self._response.read(*args)

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        self._response.close()
        self._connection.close()


def _open_relay(request, timeout, socket_path):
    destination = urllib.parse.urlsplit(request.full_url)
    connection = _UnixConnection(socket_path, timeout)
    target = urllib.parse.urlunsplit(("", "", destination.path, destination.query, ""))
    try:
        connection.request(request.get_method(), target, body=request.data, headers=dict(request.header_items()))
        response = connection.getresponse()
        if response.status >= 400:
            status, reason, headers, body = response.status, response.reason, response.headers, response.read(1024 * 1024)
            connection.close()
            raise urllib.error.HTTPError(request.full_url, status, reason, headers, io.BytesIO(body))
        return _UnixResponse(response, connection)
    except urllib.error.HTTPError:
        raise
    except OSError as exc:
        connection.close()
        raise urllib.error.URLError("verified owner relay unavailable") from exc
    except Exception:
        connection.close()
        raise


def open_bridge_request(request, timeout):
    """Open a bridge call without carrying caller credentials into Python."""
    destination = urllib.parse.urlsplit(request.full_url)
    trusted = (
        destination.scheme in ("http", "https")
        and destination.hostname in ("localhost", "127.0.0.1", "::1")
        and destination.username is None
    )
    request.remove_header("X-agent-identity-token")
    request.remove_header("X-agent-operation-token")
    if trusted:
        socket_path = os.environ.get("PROGRAM_RUNTIME_AUTHORITY_RELAY", "")
        checkpoint = destination.path in (
            "/internal/program-runtime/tasks/resolve",
            "/internal/program-runtime/tasks/begin",
            "/internal/program-runtime/tasks/start",
            "/internal/program-runtime/tasks/complete",
        )
        delegation = destination.path in (
            "/internal/program-runtime/agent/start",
            "/internal/program-runtime/agent/collect",
        )
        native_read = False
        if destination.path == "/internal/program-runtime/bindings/execute":
            try:
                native_read = json.loads(request.data).get("binding_id") in (
                    "web-console/session/native-thread-state", "web-console/session/native-usage",
                )
            except (TypeError, ValueError, AttributeError):
                pass
        if (checkpoint or delegation) and os.environ.get("PROGRAM_RUNTIME_AUTHORITY_RELAY_UNAVAILABLE") == "1":
            kind = "task checkpoint" if checkpoint else "agent delegation"
            raise urllib.error.URLError(f"verified {kind} relay unavailable")
        if socket_path and (checkpoint or delegation or native_read):
            return _open_relay(request, timeout, socket_path)
        return urllib.request.build_opener(
            urllib.request.ProxyHandler({}), _NoAgentRedirect()
        ).open(request, timeout=timeout)
    return urllib.request.urlopen(request, timeout=timeout)
