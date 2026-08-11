"""Small, isolated Python execution host for one program-runtime session.

The host deliberately exposes bounded handles instead of printing result rows.
An IPython adapter can be layered on this stable protocol when the host
requirement is available; the standard-library engine keeps development and
health checks deterministic on minimal installations.
"""
from __future__ import annotations

import contextlib
import ast
import asyncio
import builtins
import contextvars
import concurrent.futures
import io
import json
import os
import sys
import traceback
import urllib.error
import urllib.request
from typing import Any, Iterable


_OPEN = builtins.open


def _guarded_open(file: Any, mode: str = "r", *args: Any, **kwargs: Any):
    """Keep ordinary program writes inside the supervisor-pinned cwd.

    This is a cooperative defense-in-depth layer. The process boundary and
    resource limits remain authoritative; a hostile local program can still
    deliberately bypass Python-level helpers through native subprocesses.
    """
    if any(flag in mode for flag in ("w", "a", "x", "+")):
        candidate = os.path.abspath(os.fspath(file))
        root = os.path.abspath(os.getcwd())
        if os.path.commonpath((candidate, root)) != root:
            raise PermissionError(f"program write is outside the pinned workspace: {candidate}")
    return _OPEN(file, mode, *args, **kwargs)


builtins.open = _guarded_open

_INVOCATION_CONTEXT: contextvars.ContextVar[dict[str, str]] = contextvars.ContextVar(
    "program_runtime_invocation_context", default={}
)


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

    def __await__(self):
        async def return_self():
            return self

        return return_self().__await__()


class Namespace:
    def __init__(self, bindings: dict[str, Any] | None = None, invocations: list[dict[str, str]] | None = None, session_id: str = "", agent_bridge_url: str = "", bridge_url: str = "", discovery_url: str = "") -> None:
        self._bindings = _normalize_bindings(bindings or {})
        self._invocations = invocations if invocations is not None else []
        self._flat: dict[str, dict[str, Any]] = {}
        self._collisions: dict[str, list[str]] = {}
        owners: dict[str, list[Any]] = {}
        for scenario, groups in self._bindings.items():
            if not _is_scenario_map(groups):
                continue
            for group, commands in groups.items():
                for command, value in commands.items():
                    owners.setdefault(f"{group}.{command}", []).append((scenario, group, value))
        for key, values in owners.items():
            if len(values) == 1:
                scenario, group, value = values[0]
                self._flat.setdefault(group, {})[key.rsplit(".", 1)[1]] = value
            else:
                self._collisions[key] = sorted({scenario for scenario, _, _ in values})
        self._agent = _DelegationSurface(session_id, agent_bridge_url, self._invocations) if agent_bridge_url else _DeferredSurface("agent-manager delegation")
        self._inference = _InferenceSurface(session_id, bridge_url, self._invocations) if bridge_url else _DeferredSurface("ai-gateway inference")
        self._discovery_url = discovery_url.strip()

    @property
    def ai(self) -> "_DeferredSurface":
        return self._inference

    @property
    def agent(self) -> "_DeferredSurface":
        return self._agent

    def __getattr__(self, group: str) -> Any:
        if group in self._bindings:
            value = self._bindings[group]
            if _is_scenario_map(value):
                return _NamespaceScenario(group, value, self._invocations)
            return _NamespaceGroup(group, value, self._invocations)
        collision = next((owners for path, owners in self._collisions.items() if path.split(".", 1)[0] == group), None)
        if collision is not None:
            command = next(path.split(".", 1)[1] for path in self._collisions if path.split(".", 1)[0] == group)
            alternatives = ", ".join(f"vrooli.{owner}.{group}.{command}" for owner in collision)
            raise AttributeError(
                f"ambiguous governed binding group {group!r}; owning scenarios: {', '.join(collision)}; "
                f"use qualified alternatives: {alternatives}"
            )
        if group in self._flat:
            return _NamespaceGroup(group, self._flat[group], self._invocations)
        raise AttributeError(f"no governed binding scenario or group {group!r}")

    def discover(self, intent: str) -> Handle:
        if self._discovery_url:
            try:
                request = urllib.request.Request(
                    self._discovery_url,
                    data=json.dumps({"intent": intent, "limit": 20}).encode(),
                    headers={"Content-Type": "application/json"},
                    method="POST",
                )
                with urllib.request.urlopen(request, timeout=10) as response:
                    payload = json.loads(response.read().decode())
                rows = []
                for binding in payload.get("bindings", []):
                    row = {
                        "id": binding.get("id", ""),
                        "scenario": _segment(binding.get("scenario", "")),
                        "group": _segment(binding.get("group", "")),
                        "command": _segment(binding.get("command", "")),
                        "effect": binding.get("effect", ""),
                        "reason": payload.get("reason", ""),
                    }
                    rows.append(row)
                if not rows:
                    rows.append({"reason": payload.get("reason", "semantic discovery returned no governed bindings")})
                return Handle(rows, "vrooli.discover")
            except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError, OSError, json.JSONDecodeError) as exc:
                return self._local_discover(intent, f"local registry fallback: discovery bridge unavailable: {exc}")
        return self._local_discover(intent, "local registry fallback: semantic discovery bridge unavailable")

    def _local_discover(self, intent: str, reason: str) -> Handle:
        terms = [term.lower().replace("-", "_") for term in intent.split() if term.strip()]
        candidates: list[dict[str, str]] = []
        for scenario, groups in self._bindings.items():
            if not _is_scenario_map(groups):
                groups = {scenario: groups}
            for group, commands in groups.items():
                for command in commands:
                    label = f"{scenario} {group} {command}".lower()
                    if all(term in label for term in terms):
                        candidates.append({"scenario": scenario, "group": group, "command": command, "reason": reason})
        return Handle(candidates, "vrooli.discover")

    def gather(self, *calls: Any, max_workers: int = 8) -> list[Handle]:
        """Run zero-argument binding callables concurrently and preserve order."""
        if not calls:
            return []
        if any(not callable(call) for call in calls):
            raise TypeError("vrooli.gather accepts zero-argument callables")
        if max_workers <= 0:
            raise ValueError("vrooli.gather max_workers must be positive")
        if len(calls) > max_workers:
            raise ValueError(
                f"vrooli.gather concurrency ceiling exceeded: requested {len(calls)}, maximum {max_workers}"
            )
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            return list(executor.map(lambda call: call(), calls))

    def callable_namespace(self) -> list[str]:
        paths: list[str] = []
        for scenario, groups in self._bindings.items():
            if not _is_scenario_map(groups):
                groups = {"": groups}
            for group, commands in groups.items():
                for command in commands:
                    parts = [scenario]
                    if group:
                        parts.append(group)
                    parts.append(command)
                    paths.append("vrooli." + ".".join(parts))
        return sorted(paths)


def _segment(value: str) -> str:
    return value.replace("-", "_")


def _normalize_bindings(bindings: dict[str, Any]) -> dict[str, Any]:
    return {_segment(key): _normalize_bindings(value) if isinstance(value, dict) else value for key, value in bindings.items()}


def _is_scenario_map(value: Any) -> bool:
    return isinstance(value, dict) and any(isinstance(child, dict) for child in value.values())


class _NamespaceScenario:
    def __init__(self, scenario: str, groups: dict[str, Any], invocations: list[dict[str, str]]) -> None:
        self._scenario = scenario
        self._groups = groups
        self._invocations = invocations

    def __getattr__(self, group: str) -> Any:
        if group not in self._groups:
            raise AttributeError(f"no governed binding group {group!r} in scenario {self._scenario!r}")
        return _NamespaceGroup(f"{self._scenario}/{group}", self._groups[group], self._invocations)


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


class _InferenceSurface:
    """Typed convenience facade over the governed ai-gateway Run binding."""

    def __init__(self, session_id: str, bridge_url: str, invocations: list[dict[str, str]]) -> None:
        self._run = BridgeBinding("ai-gateway/inference/run", "read", session_id, bridge_url, invocations)
        self._run_batch = BridgeBinding("ai-gateway/inference/run-batch", "read", session_id, bridge_url, invocations)

    def _invoke(self, role: str, source: str, schema: Any = None, instruction: str = "", *, turns: Any = None, attachments: Any = None, profile: str | None = None) -> Any:
        if schema is None:
            schema_json = ""
        elif isinstance(schema, str):
            schema_json = schema
        else:
            schema_json = json.dumps(schema, separators=(",", ":"))
        kwargs: dict[str, Any] = {"source": source, "schema_json": schema_json, "role": role}
        if instruction:
            kwargs["instruction"] = instruction
        if turns is not None:
            kwargs["turns"] = turns
        if attachments is not None:
            kwargs["attachments"] = attachments
        if profile is not None:
            kwargs["profile"] = profile
        return self._run(**kwargs)

    def classify(self, source: str, schema: Any = None, instruction: str = "", *, turns: Any = None, attachments: Any = None, profile: str | None = None) -> Any:
        return self._invoke("classify.fast", source, schema, instruction, turns=turns, attachments=attachments, profile=profile)

    def extract(self, source: str, schema: Any = None, instruction: str = "", *, turns: Any = None, attachments: Any = None, profile: str | None = None) -> Any:
        return self._invoke("extract.structured", source, schema, instruction, turns=turns, attachments=attachments, profile=profile)

    def judge(self, source: str, schema: Any = None, instruction: str = "", *, turns: Any = None, attachments: Any = None, profile: str | None = None) -> Any:
        return self._invoke("judge.default", source, schema, instruction, turns=turns, attachments=attachments, profile=profile)

    def batch(self, sources: Iterable[Any], schema: Any = None, instruction: str = "", role: str = "classify.fast") -> Any:
        if schema is None:
            schema_json = ""
        elif isinstance(schema, str):
            schema_json = schema
        else:
            schema_json = json.dumps(schema, separators=(",", ":"))
        items = [source if isinstance(source, dict) else {"source": str(source)} for source in sources]
        kwargs: dict[str, Any] = {"items": items, "schema_json": schema_json, "role": role}
        if instruction:
            kwargs["instruction"] = instruction
        return self._run_batch(**kwargs)


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
        return self._invoke(kwargs, confirmed)

    def _invoke(self, kwargs: dict[str, Any], confirmed: bool) -> Handle:
        context = _INVOCATION_CONTEXT.get()
        request = json.dumps({"session_id": self.session_id, "program_id": context.get("program_id", ""), "provenance": context.get("provenance", ""), "binding_id": self.binding_id, "args": kwargs, "confirmed": confirmed}).encode()
        http_request = urllib.request.Request(self.bridge_url, data=request, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(http_request, timeout=180) as response:
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
    def __init__(self, bindings: dict[str, Any] | None = None, session_id: str = "", bridge_url: str = "", agent_bridge_url: str = "", discovery_url: str = "") -> None:
        self.invocations: list[dict[str, str]] = []
        self._loop = asyncio.new_event_loop()
        if bindings is None:
            bindings = {}
        if isinstance(bindings, list):
            mapped: dict[str, dict[str, dict[str, Any]]] = {}
            for spec in bindings:
                scenario = _segment(spec.get("scenario", ""))
                group = _segment(spec["group"])
                command = _segment(spec["command"])
                mapped.setdefault(scenario, {}).setdefault(group, {})[command] = BridgeBinding(spec["id"], spec.get("effect", ""), session_id, bridge_url, self.invocations)
            bindings = mapped
        self.globals: dict[str, Any] = {
            "__name__": "program_runtime_session",
            "vrooli": Namespace(bindings, self.invocations, session_id, agent_bridge_url, bridge_url, discovery_url),
        }

    def execute(self, source: str, include_materialized: bool = False, program_id: str = "", provenance: str = "") -> dict[str, Any]:
        output = io.StringIO()
        self.invocations.clear()
        context_token = _INVOCATION_CONTEXT.set({"program_id": program_id, "provenance": provenance})
        try:
            with contextlib.redirect_stdout(output):
                tree = ast.parse(source, "<program>", "exec")
                last_expression = None
                if tree.body and isinstance(tree.body[-1], ast.Expr):
                    last_expression = tree.body[-1].value
                    tree.body[-1] = ast.Assign(targets=[ast.Name(id="__program_result", ctx=ast.Store())], value=last_expression)
                ast.fix_missing_locations(tree)
                code = compile(tree, "<program>", "exec", flags=ast.PyCF_ALLOW_TOP_LEVEL_AWAIT)
                result = eval(code, self.globals, self.globals)
                if result is not None:
                    self._loop.run_until_complete(result)
                if last_expression is not None:
                    value = self.globals.pop("__program_result", None)
                    if value is not None:
                        print(repr(value))
            raw_stdout = output.getvalue()
            limit = MATERIALIZED_OUTPUT_LIMIT if include_materialized else DEFAULT_OUTPUT_LIMIT
            agent_stdout = _bounded_text(raw_stdout, limit)
            return {"ok": True, "stdout": agent_stdout, "context_bytes": len(raw_stdout.encode()), "agent_bytes": len(agent_stdout.encode()), "output_limit_bytes": limit, "invocations": list(self.invocations)}
        except Exception as exc:  # noqa: BLE001 - wire the program failure, never crash the host
            raw_stdout = output.getvalue()
            limit = MATERIALIZED_OUTPUT_LIMIT if include_materialized else DEFAULT_OUTPUT_LIMIT
            agent_stdout = _bounded_text(raw_stdout, limit)
            return {"ok": False, "stdout": agent_stdout, "context_bytes": len(raw_stdout.encode()), "agent_bytes": len(agent_stdout.encode()), "output_limit_bytes": limit, "error": f"{type(exc).__name__}: {exc}", "traceback": traceback.format_exc(limit=4), "invocations": list(self.invocations)}
        finally:
            _INVOCATION_CONTEXT.reset(context_token)


DEFAULT_OUTPUT_LIMIT = 4096
MATERIALIZED_OUTPUT_LIMIT = 65536


def _bounded_text(value: str, limit: int) -> str:
    if len(value.encode()) <= limit:
        return value
    suffix = "…"
    return value[: max(0, limit - len(suffix.encode()))] + suffix


def serve() -> None:
    bindings = []
    binding_path = os.environ.get("PROGRAM_RUNTIME_BINDINGS_FILE", "")
    if binding_path:
        try:
            with open(binding_path, encoding="utf-8") as handle:
                bindings = json.load(handle)
        except (OSError, json.JSONDecodeError):
            bindings = []
    else:
        try:
            bindings = json.loads(os.environ.get("PROGRAM_RUNTIME_BINDINGS", "[]"))
        except json.JSONDecodeError:
            bindings = []
    kernel = SessionKernel(bindings, os.environ.get("PROGRAM_RUNTIME_SESSION_ID", ""), os.environ.get("PROGRAM_RUNTIME_BRIDGE_URL", ""), os.environ.get("PROGRAM_RUNTIME_AGENT_BRIDGE_URL", ""), os.environ.get("PROGRAM_RUNTIME_DISCOVERY_URL", ""))
    for line in sys.stdin:
        if not line.strip():
            continue
        try:
            request = json.loads(line)
            response = kernel.execute(str(request.get("source", "")), bool(request.get("include_materialized", False)), str(request.get("program_id", "")), str(request.get("provenance", "")))
        except Exception as exc:  # malformed transport input is a request failure
            response = {"ok": False, "error": f"protocol error: {exc}"}
        sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    serve()
