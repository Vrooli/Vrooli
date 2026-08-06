"""Small, isolated Python execution host for one program-runtime session.

The host deliberately exposes bounded handles instead of printing result rows.
An IPython adapter can be layered on this stable protocol when the host
requirement is available; the standard-library engine keeps development and
health checks deterministic on minimal installations.
"""
from __future__ import annotations

import contextlib
import io
import json
import os
import sys
import traceback
import urllib.error
import urllib.request
from typing import Any, Iterable


class Handle:
    def __init__(self, rows: Iterable[Any], label: str = "result") -> None:
        self._rows = list(rows)
        self.label = label

    def __repr__(self) -> str:
        return f"Handle(label={self.label!r}, rows={len(self._rows)})"

    def count(self) -> int:
        return len(self._rows)

    def head(self, n: int = 5) -> list[Any]:
        return self._rows[: max(0, n)]

    def filter(self, predicate) -> "Handle":
        return Handle((row for row in self._rows if predicate(row)), self.label)

    def group_by(self, key: str) -> dict[Any, int]:
        counts: dict[Any, int] = {}
        for row in self._rows:
            value = row.get(key) if isinstance(row, dict) else getattr(row, key)
            counts[value] = counts.get(value, 0) + 1
        return counts

    def materialize(self, limit: int | None = None) -> list[Any]:
        return list(self._rows if limit is None else self._rows[: max(0, limit)])


class Namespace:
    def __init__(self, bindings: dict[str, Any] | None = None, invocations: list[dict[str, str]] | None = None, session_id: str = "", agent_bridge_url: str = "") -> None:
        self._bindings = bindings or {}
        self._invocations = invocations if invocations is not None else []
        self._agent = _DelegationSurface(session_id, agent_bridge_url, self._invocations) if agent_bridge_url else _DeferredSurface("agent-manager delegation")

    @property
    def ai(self) -> "_DeferredSurface":
        return _DeferredSurface("ai-gateway inference")

    @property
    def agent(self) -> "_DeferredSurface":
        return self._agent

    def __getattr__(self, group: str) -> Any:
        if group not in self._bindings:
            raise AttributeError(f"no governed binding group {group!r}")
        return _NamespaceGroup(group, self._bindings[group], self._invocations)


class _DeferredSurface:
    """Explicit placeholders for capabilities not promoted into this runtime."""

    def __init__(self, capability: str) -> None:
        self._capability = capability

    def __getattr__(self, operation: str) -> Any:
        def unavailable(*_args: Any, **_kwargs: Any) -> Any:
            raise RuntimeError(
                f"{self._capability} is unavailable: the required capability promotion is not complete; "
                "request an explicit governed binding"
            )

        unavailable.__name__ = operation
        return unavailable

    def __call__(self, *_args: Any, **_kwargs: Any) -> Any:
        raise RuntimeError(
            f"{self._capability} is unavailable: the required capability promotion is not complete; "
            "request an explicit governed binding"
        )


class _DelegationSurface:
    def __init__(self, session_id: str, bridge_url: str, invocations: list[dict[str, str]]) -> None:
        self._session_id = session_id
        self._bridge_url = bridge_url
        self._invocations = invocations

    def run(self, **kwargs: Any) -> Handle:
        request = dict(kwargs)
        request["session_id"] = self._session_id
        body = json.dumps(request).encode()
        http_request = urllib.request.Request(self._bridge_url, data=body, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(http_request, timeout=45) as response:
                payload = json.loads(response.read().decode())
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode(errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise RuntimeError(str(detail)) from exc
        except (urllib.error.URLError, TimeoutError) as exc:
            raise RuntimeError(f"agent-manager delegation unavailable: {exc}") from exc
        self._invocations.append({"binding_id": "agent/delegate", "effect": "write"})
        return Handle([payload], "agent/delegate")


class _NamespaceGroup:
    def __init__(self, group: str, commands: dict[str, Any], invocations: list[dict[str, str]]) -> None:
        self._group = group
        self._commands = commands
        self._invocations = invocations

    def __getattr__(self, command: str) -> Any:
        if command not in self._commands:
            raise AttributeError(f"no governed binding command {command!r}")
        value = self._commands[command]
        if not callable(value):
            return value
        if getattr(value, "_records_invocation", False):
            return value

        def invoke(*args: Any, **kwargs: Any) -> Any:
            self._invocations.append({"binding_id": f"{self._group}/{command}"})
            return value(*args, **kwargs)

        invoke.__name__ = command
        return invoke


class BridgeBinding:
    """A callable that can only reach the Go governance bridge."""

    def __init__(self, binding_id: str, effect: str, session_id: str, bridge_url: str, invocations: list[dict[str, str]]) -> None:
        self.binding_id = binding_id
        self.effect = effect
        self.session_id = session_id
        self.bridge_url = bridge_url
        self.invocations = invocations
        self._records_invocation = True

    def __call__(self, *args: Any, **kwargs: Any) -> Handle:
        if args:
            raise TypeError(f"{self.binding_id} accepts named proto fields, not positional arguments")
        confirmed = bool(kwargs.pop("_confirm", False))
        if not self.bridge_url:
            raise RuntimeError("program-runtime binding bridge is unavailable")
        request = json.dumps({"session_id": self.session_id, "binding_id": self.binding_id, "args": kwargs, "confirmed": confirmed}).encode()
        http_request = urllib.request.Request(self.bridge_url, data=request, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(http_request, timeout=10) as response:
                payload = json.loads(response.read().decode())
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode(errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise RuntimeError(str(detail)) from exc
        except (urllib.error.URLError, TimeoutError) as exc:
            raise RuntimeError(f"binding bridge unavailable: {exc}") from exc
        self.invocations.append({"binding_id": self.binding_id, "effect": self.effect})
        rows = next((value for value in payload.values() if isinstance(value, list)), None) if isinstance(payload, dict) else None
        if rows is None:
            rows = [payload]
        return Handle(rows, self.binding_id)


class SessionKernel:
    def __init__(self, bindings: dict[str, Any] | None = None, session_id: str = "", bridge_url: str = "", agent_bridge_url: str = "") -> None:
        self.invocations: list[dict[str, str]] = []
        if bindings is None:
            bindings = {}
        if bridge_url and isinstance(bindings, list):
            mapped: dict[str, dict[str, Any]] = {}
            for spec in bindings:
                mapped.setdefault(spec["group"], {})[spec["command"]] = BridgeBinding(spec["id"], spec.get("effect", ""), session_id, bridge_url, self.invocations)
            bindings = mapped
        self.globals: dict[str, Any] = {"__name__": "program_runtime_session", "vrooli": Namespace(bindings, self.invocations, session_id, agent_bridge_url)}

    def execute(self, source: str) -> dict[str, Any]:
        output = io.StringIO()
        self.invocations.clear()
        try:
            with contextlib.redirect_stdout(output):
                exec(compile(source, "<program>", "exec"), self.globals, self.globals)
            return {"ok": True, "stdout": output.getvalue(), "context_bytes": len(output.getvalue().encode()), "invocations": list(self.invocations)}
        except Exception as exc:  # noqa: BLE001 - wire the program failure, never crash the host
            return {"ok": False, "stdout": output.getvalue(), "error": f"{type(exc).__name__}: {exc}", "traceback": traceback.format_exc(limit=4), "invocations": list(self.invocations)}


def serve() -> None:
    try:
        bindings = json.loads(os.environ.get("PROGRAM_RUNTIME_BINDINGS", "[]"))
    except json.JSONDecodeError:
        bindings = []
    kernel = SessionKernel(bindings, os.environ.get("PROGRAM_RUNTIME_SESSION_ID", ""), os.environ.get("PROGRAM_RUNTIME_BRIDGE_URL", ""), os.environ.get("PROGRAM_RUNTIME_AGENT_BRIDGE_URL", ""))
    for line in sys.stdin:
        if not line.strip():
            continue
        try:
            request = json.loads(line)
            response = kernel.execute(str(request.get("source", "")))
        except Exception as exc:  # malformed transport input is a request failure
            response = {"ok": False, "error": f"protocol error: {exc}"}
        sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    serve()
