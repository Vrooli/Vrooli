"""Trusted local HTTP transport shared by Runtime bridge clients."""
from __future__ import annotations

import contextvars
import urllib.parse
import urllib.request


AGENT_IDENTITY = contextvars.ContextVar("agent_identity", default="")


class _NoAgentRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def open_bridge_request(request, timeout):
    """Forward verified identity only to trusted local owners."""
    token = AGENT_IDENTITY.get()
    destination = urllib.parse.urlsplit(request.full_url)
    trusted = (
        destination.scheme in ("http", "https")
        and destination.hostname in ("localhost", "127.0.0.1", "::1")
        and destination.username is None
    )
    if token and trusted:
        request.add_header("X-Agent-Identity-Token", token)
        # Proof-bearing local calls cannot use an ambient proxy or follow a
        # redirect. The target owner verifies the original grant on each call.
        return urllib.request.build_opener(
            urllib.request.ProxyHandler({}), _NoAgentRedirect()
        ).open(request, timeout=timeout)
    request.remove_header("X-agent-identity-token")
    return urllib.request.urlopen(request, timeout=timeout)
