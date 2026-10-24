import json
import sys
import urllib.request
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parents[1]))
from host.engine import SessionKernel, _AGENT_IDENTITY, _open_bridge_request


class Response:
    def __init__(self, body):
        self.body = body

    def __enter__(self):
        return self

    def __exit__(self, *_args):
        return False

    def read(self):
        return json.dumps(self.body).encode()


def test_agent_proof_is_execution_local_and_absent_from_results():
    calls = []

    def opened(request, **_kwargs):
        calls.append(request.get_header("X-agent-identity-token"))
        if request.full_url.endswith("reachability"):
            return Response({"demo": {"reachable": True}})
        return Response({"items": [{"id": "one"}]})

    kernel = SessionKernel([{"id": "demo/work/run", "scenario": "demo", "group": "work", "command": "run", "effect": "read", "rows_field": "items"}], session_id="session", bridge_url="http://127.0.0.1:1234/bridge/execute")
    with patch("host.engine.urllib.request.build_opener") as builder, patch("host.engine.urllib.request.urlopen", side_effect=opened):
        builder.return_value.open.side_effect = opened
        result = kernel.execute("print(demo.work.run().count())", agent_identity_token="fixture-agent-proof")
        assert result["ok"]
        assert calls and all(value == "fixture-agent-proof" for value in calls)
        assert "fixture-agent-proof" not in json.dumps(result)
        calls.clear()
        again = kernel.execute("print(demo.work.run().count())")
        assert again["ok"]
        assert calls and all(value is None for value in calls)


def test_agent_proof_never_goes_to_an_external_or_userinfo_destination():
    saved = _AGENT_IDENTITY.set("fixture-agent-proof")
    try:
        for url in ["https://outside.example/bridge", "http://user@localhost/bridge"]:
            request = urllib.request.Request(url, headers={"X-Agent-Identity-Token": "copied-proof"})
            with patch("host.engine.urllib.request.urlopen", return_value=Response({})) as opened:
                _open_bridge_request(request, timeout=1)
                assert opened.call_args.args[0].get_header("X-agent-identity-token") is None
    finally:
        _AGENT_IDENTITY.reset(saved)


def test_proof_bearing_calls_disable_redirects_and_ambient_proxies():
    saved = _AGENT_IDENTITY.set("fixture-agent-proof")
    try:
        with patch("host.engine.urllib.request.build_opener") as builder:
            _open_bridge_request(urllib.request.Request("http://localhost/bridge"), timeout=1)
            proxy, redirect = builder.call_args.args
            assert proxy.proxies == {}
            assert redirect.redirect_request(None, None, 302, "redirect", {}, "https://outside.example") is None
    finally:
        _AGENT_IDENTITY.reset(saved)


def test_parallel_owner_calls_preserve_agent_proof_and_failure_clears_it():
    calls = []

    def opened(request, **_kwargs):
        calls.append(request.get_header("X-agent-identity-token"))
        if request.full_url.endswith("reachability"):
            return Response({"demo": {"reachable": True}})
        return Response({"items": [{"id": "one"}]})

    kernel = SessionKernel([{"id": "demo/work/run", "scenario": "demo", "group": "work", "command": "run", "effect": "read", "rows_field": "items"}], session_id="session", bridge_url="http://127.0.0.1:1234/bridge/execute")
    with patch("host.engine.urllib.request.build_opener") as builder, patch("host.engine.urllib.request.urlopen", side_effect=opened):
        builder.return_value.open.side_effect = opened
        result = kernel.execute("print([r.count() for r in gather(lambda: demo.work.run(), lambda: demo.work.run())]); raise ValueError('stop')", agent_identity_token="fixture-agent-proof")
        assert not result["ok"]
        assert result["stdout"].strip() == "[1, 1]"
        assert calls and all(value == "fixture-agent-proof" for value in calls)
        assert "fixture-agent-proof" not in json.dumps(result)
        assert _AGENT_IDENTITY.get() == ""
        calls.clear()
        again = kernel.execute("print(demo.work.run().count())")
        assert again["ok"]
        assert calls and all(value is None for value in calls)
