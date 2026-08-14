"""Small, isolated Python execution host for one program-runtime session.

The host deliberately exposes bounded handles instead of printing result rows.
The standard-library engine keeps the session protocol deterministic and
portable; data shaping remains inside bounded Handle operations.
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
import queue
import sys
import threading
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
    def __init__(self, rows: Iterable[Any], label: str = "result", *, metadata: dict[str, Any] | None = None, raw: Any = None) -> None:
        self._rows = list(rows)
        self.label = label
        self._metadata = dict(metadata or {})
        self._raw = raw if raw is not None else list(self._rows)

    def __repr__(self) -> str:
        return f"Handle(label={self.label!r}, rows={len(self._rows)})"

    def count(self) -> int:
        return len(self._rows)

    def head(self, n: int = 5) -> list[Any]:
        return self._rows[: max(0, n)]

    def filter(self, predicate) -> "Handle":
        return Handle((row for row in self._rows if predicate(row)), self.label)

    def map(self, transform) -> "Handle":
        return Handle((transform(row) for row in self._rows), f"{self.label}.map")

    def select(self, *fields: str) -> "Handle":
        names = _field_names(fields)
        return Handle(({name: _field(row, name, "select") for name in names} for row in self._rows), f"{self.label}.select")

    def sort(self, key: str, reverse: bool = False) -> "Handle":
        return Handle(sorted(self._rows, key=lambda row: _field(row, key, "sort"), reverse=reverse), f"{self.label}.sort")

    def unique(self, key: str | None = None) -> "Handle":
        seen: set[Any] = set()
        unique_rows = []
        for row in self._rows:
            value = _field(row, key, "unique") if key else row
            try:
                marker = value if hash(value) is not None else repr(value)
            except TypeError:
                marker = repr(value)
            if marker not in seen:
                seen.add(marker)
                unique_rows.append(row)
        return Handle(unique_rows, f"{self.label}.unique")

    def agg(self, key: str, operation: str) -> int | float:
        values = [_field(row, key, "agg") for row in self._rows]
        if not values:
            raise ValueError("agg requires at least one row")
        operation = operation.lower().strip()
        if operation == "sum":
            return sum(values)
        if operation == "min":
            return min(values)
        if operation == "max":
            return max(values)
        if operation == "mean":
            return sum(values) / len(values)
        raise ValueError("agg operation must be one of: sum, min, max, mean")

    def join(self, other: "Handle", key: str) -> "Handle":
        if not isinstance(other, Handle):
            raise TypeError("join requires another Handle")
        if len(self._rows) * max(1, len(other._rows)) > 100_000_000:
            raise MemoryError("join exceeds bounded comparison budget of 100000000 row pairs")
        right = {}
        for row in other._rows:
            right.setdefault(_field(row, key, "join"), []).append(row)
        joined = []
        for left in self._rows:
            for match in right.get(_field(left, key, "join"), []):
                if isinstance(left, dict) and isinstance(match, dict):
                    row = dict(left)
                    row.update(match)
                    joined.append(row)
                else:
                    joined.append((left, match))
        return Handle(joined, f"{self.label}.join")

    def group_by(self, key: str) -> dict[Any, int]:
        if not self._rows:
            return {}
        counts: dict[Any, int] = {}
        for row in self._rows:
            value = _field(row, key, "group_by")
            counts[value] = counts.get(value, 0) + 1
        return counts

    def materialize(self, limit: int | None = None) -> list[Any]:
        return list(self._rows if limit is None else self._rows[: max(0, limit)])

    def meta(self) -> dict[str, Any]:
        """Return non-row response fields without changing row materialization."""
        return dict(self._metadata)

    def raw(self) -> Any:
        """Return the decoded response message exactly as received."""
        return self._raw

    def __getitem__(self, item: int | slice) -> Any:
        if isinstance(item, slice):
            return Handle(self._rows[item], f"{self.label}[slice]")
        if isinstance(item, int):
            return self._rows[item]
        raise TypeError("Handle indices must be integers or slices")

    def __await__(self):
        async def return_self():
            return self

        return return_self().__await__()


class Namespace:
    def __init__(self, bindings: dict[str, Any] | None = None, invocations: list[dict[str, str]] | None = None, session_id: str = "", agent_bridge_url: str = "", bridge_url: str = "", discovery_url: str = "", reachability: dict[str, Any] | None = None, libraries: list[dict[str, Any]] | None = None) -> None:
        self._bindings = _normalize_bindings(bindings or {})
        self._invocations = invocations if invocations is not None else []
        self._session_id = session_id
        self._bridge_url = bridge_url.strip()
        self._bare_groups: dict[str, list[str]] = {}
        for scenario, groups in self._bindings.items():
            if not _is_scenario_map(groups):
                continue
            for group, commands in groups.items():
                paths = self._bare_groups.setdefault(group, [])
                paths.extend(f"vrooli.{scenario}.{group}.{command}" for command in commands)
        for paths in self._bare_groups.values():
            paths.sort()
        self._agent = _DelegationSurface(session_id, agent_bridge_url, self._invocations) if agent_bridge_url else _DeferredSurface("agent-manager delegation")
        self._inference = _InferenceSurface(session_id, bridge_url, self._invocations) if bridge_url else _DeferredSurface("ai-gateway inference")
        self._discovery_url = discovery_url.strip()
        self._reachability_url = self._bridge_url.rsplit("/", 1)[0] + "/reachability" if self._bridge_url else ""
        self._reachability = _normalize_bindings(reachability or {})
        self.lib = _LibraryNamespace(self, libraries or [])

    @property
    def ai(self) -> "_DeferredSurface":
        return self._inference

    @property
    def agent(self) -> "_DeferredSurface":
        return self._agent

    def describe(self, binding: str) -> Handle:
        """Read a binding contract through the registry's live descriptor path."""
        if not self._bridge_url:
            raise RuntimeError("program-runtime binding bridge is unavailable")
        endpoint = self._bridge_url.rsplit("/", 1)[0] + "/describe"
        request = urllib.request.Request(
            endpoint,
            data=json.dumps({"session_id": self._session_id, "binding": binding}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=10) as response:
                payload = json.loads(response.read().decode())
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode(errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise RuntimeError(str(detail)) from exc
        except (urllib.error.URLError, TimeoutError) as exc:
            raise RuntimeError(f"binding description bridge unavailable: {exc}") from exc
        rows = payload.get("arguments", []) if isinstance(payload, dict) else []
        return Handle(rows, f"vrooli.describe({binding})")

    def reachable(self) -> Handle:
        if self._reachability_url:
            request = urllib.request.Request(
                self._reachability_url,
                data=json.dumps({"session_id": self._session_id}).encode(),
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            try:
                with urllib.request.urlopen(request, timeout=10) as response:
                    live = json.loads(response.read().decode())
                if isinstance(live, dict):
                    rows = []
                    for scenario, status in sorted(live.items()):
                        status = status if isinstance(status, dict) else {}
                        rows.append({"scenario": scenario, "reachable": bool(status.get("reachable", False)), "reason": status.get("reason", ""), "checked_at": status.get("checked_at", "")})
                    return Handle(rows, "vrooli.reachable")
            except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError, OSError, json.JSONDecodeError):
                pass
        rows = []
        for scenario, status in sorted(self._reachability.items()):
            rows.append({
                "scenario": scenario,
                "reachable": bool(status.get("reachable", True)) if isinstance(status, dict) else True,
                "reason": status.get("reason", "") if isinstance(status, dict) else "",
            })
        return Handle(rows, "vrooli.reachable")

    def __getattr__(self, group: str) -> Any:
        if group in self._bindings:
            value = self._bindings[group]
            if _is_scenario_map(value):
                return _NamespaceScenario(group, value, self._invocations)
            return _NamespaceGroup(group, value, self._invocations)
        if group in self._bare_groups:
            alternatives = ", ".join(self._bare_groups[group][:4])
            omitted = len(self._bare_groups[group]) - 4
            if omitted > 0:
                alternatives += f", (+{omitted} more)"
            raise AttributeError(
                f"bare governed binding group {group!r} has no unqualified namespace; "
                f"owning scenario path required, use qualified alternatives: {alternatives}"
            )
        raise AttributeError(f"no governed binding scenario or group {group!r}")

    def discover(self, intent: str) -> Handle:
        """Resolve through the current seeded discover library facade."""
        if self.lib.available("discover"):
            return self.lib.discover(intent)
        return self._discover_bridge(intent)

    def _execute_library_source(self, spec: dict[str, Any], args: tuple[Any, ...], kwargs: dict[str, Any]) -> Handle:
        """Execute one operator-promoted source with only public runtime globals."""
        if args:
            raise TypeError("library programs accept named inputs")
        environment: dict[str, Any] = {
            "vrooli": self,
            "Handle": Handle,
            "__name__": "program_runtime_library",
            "intent": kwargs.pop("intent", ""),
            "text": kwargs.pop("text", ""),
            "__builtins__": {"len": len, "min": min, "max": max, "print": print, "range": range, "sorted": sorted},
        }
        if kwargs:
            raise TypeError(f"unknown library inputs: {', '.join(sorted(kwargs))}")
        exec(compile(str(spec.get("source", "")), f"<library:{spec.get('name', 'unknown')}>", "exec"), environment, environment)
        value = environment.get("result")
        if isinstance(value, Handle):
            return value
        return Handle([{"library": spec.get("name", ""), "version": spec.get("version", 0), "value": value}], f"vrooli.lib.{spec.get('name', '')}")

    def _discover_bridge(self, intent: str) -> Handle:
        """Private bridge used by the seeded discover facade."""
        if self._discovery_url:
            try:
                request = urllib.request.Request(
                    self._discovery_url,
                    data=json.dumps({"intent": intent, "limit": 20, "mode": "judged"}).encode(),
                    headers={"Content-Type": "application/json"},
                    method="POST",
                )
                with urllib.request.urlopen(request, timeout=10) as response:
                    payload = json.loads(response.read().decode())
                result = payload.get("result") or {}
                binding = result.get("binding") or {}
                binding_id = result.get("bindingId", result.get("binding_id", ""))
                row = {
                    "id": binding_id,
                    "binding_id": binding_id,
                    "scenario": _segment(binding.get("scenario", "")),
                    "group": _segment(binding.get("group", "")),
                    "command": _segment(binding.get("command", "")),
                    "effect": binding.get("effect", ""),
                    "confidence": result.get("confidence", ""),
                    "method": result.get("method", ""),
                    "reason": result.get("reason", payload.get("reason", "")),
                    "alternatives": result.get("alternatives", []),
                    "arguments": result.get("arguments", []),
                    "null_verdict": not bool(binding_id),
                }
                return Handle([row], "vrooli.discover")
            except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError, OSError, json.JSONDecodeError) as exc:
                return Handle([{"binding_id": "", "null_verdict": True, "reason": f"discovery unavailable: {exc}"}], "vrooli.discover")
        return Handle([{"binding_id": "", "null_verdict": True, "reason": "discovery unavailable: bridge is not configured"}], "vrooli.discover")

    def gather(self, *calls: Any, max_workers: int = 8) -> list[Handle]:
        """Run zero-argument binding callables concurrently and preserve order."""
        if not calls:
            return []
        if any(not callable(call) for call in calls):
            raise TypeError("vrooli.gather accepts zero-argument callables")
        if max_workers <= 0:
            raise ValueError("vrooli.gather max_workers must be positive")
        with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
            results: list[Handle] = []
            for offset in range(0, len(calls), max_workers):
                batch = calls[offset : offset + max_workers]
                results.extend(executor.map(lambda call: call(), batch))
            return results

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


def _field_names(fields: tuple[str, ...]) -> tuple[str, ...]:
    if not fields:
        raise ValueError("select requires at least one field")
    names = tuple(str(field) for field in fields)
    if any(not name for name in names):
        raise ValueError("select field names must not be empty")
    return names


def _field(row: Any, key: str, operation: str) -> Any:
    if isinstance(row, dict):
        if key in row:
            return row[key]
        available = ", ".join(sorted(str(name) for name in row)) or "<none>"
    else:
        try:
            return getattr(row, key)
        except AttributeError:
            available = ", ".join(sorted(vars(row))) if hasattr(row, "__dict__") else "<unknown>"
    raise KeyError(f"{operation} key {key!r} is missing; available keys: {available}")


class _NamespaceScenario:
    def __init__(self, scenario: str, groups: dict[str, Any], invocations: list[dict[str, str]]) -> None:
        self._scenario = scenario
        self._groups = groups
        self._invocations = invocations

    def __getattr__(self, group: str) -> Any:
        if group not in self._groups:
            raise AttributeError(f"no governed binding group {group!r} in scenario {self._scenario!r}")
        return _NamespaceGroup(f"{self._scenario}/{group}", self._groups[group], self._invocations)


class _LibraryNamespace:
    """Versioned, allowlisted facades for the promoted program library.

    Library source is retained as auditable metadata and is never evaluated as
    a second interpreter program. Each seeded entry maps to a narrow host
    facade, which keeps promotion from becoming an arbitrary code-loading
    escape hatch while still giving programs a stable ``vrooli.lib`` surface.
    """

    def __init__(self, owner: Namespace, libraries: list[dict[str, Any]]) -> None:
        self._owner = owner
        self._libraries = {
            _segment(str(item.get("name", ""))): item
            for item in libraries
            if item.get("name") and bool(item.get("current", True))
        }

    def available(self, name: str) -> bool:
        return name in self._libraries

    def __getattr__(self, name: str) -> Any:
        spec = self._libraries.get(name)
        if spec is None:
            raise AttributeError(f"library program {name!r} is not current and promoted")

        def invoke(*args: Any, **kwargs: Any) -> Handle:
            if name == "discover":
                intent = kwargs.pop("intent", args[0] if args else "")
                if kwargs or len(args) > 1 or not str(intent).strip():
                    raise TypeError("vrooli.lib.discover requires one non-empty intent")
                return self._owner._discover_bridge(str(intent))
            return self._owner._execute_library_source(spec, args, kwargs)

        invoke.__name__ = name
        return invoke


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

    def _request(self, endpoint: str, request: dict[str, Any], timeout: float) -> dict[str, Any]:
        body = json.dumps(request).encode()
        http_request = urllib.request.Request(endpoint, data=body, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(http_request, timeout=timeout) as response:
                return json.loads(response.read().decode())
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode(errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise RuntimeError(str(detail)) from exc
        except (urllib.error.URLError, TimeoutError) as exc:
            raise RuntimeError(f"agent-manager delegation unavailable: {exc}") from exc

    def start(self, **kwargs: Any) -> Handle:
        request = dict(kwargs)
        request["session_id"] = self._session_id
        payload = self._request(self._bridge_url.rsplit("/", 1)[0] + "/start", request, 15)
        self._invocations.append({"binding_id": "agent/start", "effect": "write"})
        return Handle([payload], "agent/start")

    def collect(self, handle: Handle, wait_seconds: int = 0) -> Handle:
        if not isinstance(handle, Handle):
            raise TypeError("agent.collect requires a delegation Handle")
        rows = handle.head(1)
        if not rows or not isinstance(rows[0], dict) or not rows[0].get("execution_id"):
            raise ValueError("agent.collect requires a Handle returned by agent.start")
        seconds = max(0, min(int(wait_seconds), 300))
        request = {"session_id": self._session_id, "execution_id": rows[0]["execution_id"], "wait_seconds": seconds}
        payload = self._request(self._bridge_url.rsplit("/", 1)[0] + "/collect", request, max(15, seconds + 15))
        self._invocations.append({"binding_id": "agent/collect", "effect": "read"})
        return Handle([payload], "agent/collect")

    def run(self, **kwargs: Any) -> Handle:
        started = self.start(**kwargs)
        return self.collect(started, 30)


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

    def __init__(self, binding_id: str, effect: str, session_id: str, bridge_url: str, invocations: list[dict[str, str]], reachable: bool = True, reachability_reason: str = "", rows_field: str = "", meta_fields: list[str] | None = None, row_field_candidates: list[str] | None = None, reachability_url: str = "") -> None:
        self.binding_id = binding_id
        self.effect = effect
        self.session_id = session_id
        self.bridge_url = bridge_url
        self.invocations = invocations
        self.reachable = reachable
        self.reachability_reason = reachability_reason
        self.rows_field = rows_field
        self.meta_fields = list(meta_fields or [])
        self.row_field_candidates = list(row_field_candidates or [])
        self.reachability_url = reachability_url
        self._records_invocation = True

    def __call__(self, *args: Any, **kwargs: Any) -> Handle:
        if args:
            raise TypeError(f"{self.binding_id} accepts named proto fields, not positional arguments")
        confirmed = bool(kwargs.pop("_confirm", False))
        if not self.bridge_url:
            raise RuntimeError("program-runtime binding bridge is unavailable")
        if self.reachability_url:
            self._check_live_reachability()
        elif not self.reachable:
            raise RuntimeError(f"binding {self.binding_id} is unreachable: {self.reachability_reason}")
        return self._invoke(kwargs, confirmed)

    def _check_live_reachability(self) -> None:
        request = urllib.request.Request(
            self.reachability_url,
            data=json.dumps({"session_id": self.session_id}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=10) as response:
                snapshot = json.loads(response.read().decode())
            scenario = self.binding_id.split("/", 1)[0]
            status = snapshot.get(scenario, {}) if isinstance(snapshot, dict) else {}
            if not status.get("reachable", False):
                raise RuntimeError(f"binding {self.binding_id} is unreachable: {status.get('reason', 'scenario API is unavailable')}")
        except RuntimeError:
            raise
        except (urllib.error.HTTPError, urllib.error.URLError, TimeoutError, OSError, json.JSONDecodeError) as exc:
            raise RuntimeError(f"binding {self.binding_id} is unreachable: live reachability unavailable: {exc}") from exc

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
        if not isinstance(payload, dict):
            raise RuntimeError(f"binding {self.binding_id} returned a non-object response")
        if self.row_field_candidates:
            candidates = ", ".join(self.row_field_candidates)
            raise RuntimeError(f"binding {self.binding_id} has no determinable primary response field; candidate repeated fields: {candidates}")
        if self.rows_field:
            rows = payload.get(self.rows_field, [])
            if not isinstance(rows, list):
                raise RuntimeError(f"binding {self.binding_id} response field {self.rows_field!r} is not a list")
        else:
            rows = [payload]
        metadata = {name: payload[name] for name in self.meta_fields if name in payload}
        return Handle(rows, self.binding_id, metadata=metadata, raw=payload)


class _ProgressBuffer(io.StringIO):
    def __init__(self, on_write) -> None:
        super().__init__()
        self._on_write = on_write

    def write(self, value: str) -> int:
        written = super().write(value)
        if self._on_write is not None:
            self._on_write(self.getvalue())
        return written


class SessionKernel:
    def __init__(self, bindings: dict[str, Any] | None = None, session_id: str = "", bridge_url: str = "", agent_bridge_url: str = "", discovery_url: str = "", libraries: list[dict[str, Any]] | None = None) -> None:
        self.invocations: list[dict[str, str]] = []
        self._loop = asyncio.new_event_loop()
        if bindings is None:
            bindings = {}
        if isinstance(bindings, list):
            mapped: dict[str, dict[str, dict[str, Any]]] = {}
            reachability: dict[str, dict[str, Any]] = {}
            for spec in bindings:
                scenario_name = spec.get("scenario", "")
                scenario = _segment(scenario_name)
                group = _segment(spec["group"])
                command = _segment(spec["command"])
                reachable = bool(spec.get("reachable", True))
                reason = spec.get("reachability_reason", "")
                reachability[scenario] = {"reachable": reachable, "reason": reason}
                reachability_url = bridge_url.rsplit("/", 1)[0] + "/reachability" if bridge_url else ""
                mapped.setdefault(scenario, {}).setdefault(group, {})[command] = BridgeBinding(spec["id"], spec.get("effect", ""), session_id, bridge_url, self.invocations, reachable, reason, spec.get("rows_field", ""), spec.get("meta_fields", []), spec.get("row_field_candidates", []), reachability_url)
            bindings = mapped
        else:
            reachability = {scenario: {"reachable": True, "reason": ""} for scenario in bindings}
        self.globals: dict[str, Any] = {
            "__name__": "program_runtime_session",
            # Handle is the public bounded-result constructor available to
            # submitted programs; the host module is intentionally not on the
            # isolated interpreter's import path.
            "Handle": Handle,
            "vrooli": Namespace(bindings, self.invocations, session_id, agent_bridge_url, bridge_url, discovery_url, reachability, libraries),
        }

    def execute(self, source: str, include_materialized: bool = False, program_id: str = "", provenance: str = "", progress=None) -> dict[str, Any]:
        output = _ProgressBuffer(progress)
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
            return {"type": "result", "ok": True, "stdout": agent_stdout, "context_bytes": len(raw_stdout.encode()), "agent_bytes": len(agent_stdout.encode()), "output_limit_bytes": limit, "invocations": list(self.invocations)}
        except Exception as exc:  # noqa: BLE001 - wire the program failure, never crash the host
            raw_stdout = output.getvalue()
            limit = MATERIALIZED_OUTPUT_LIMIT if include_materialized else DEFAULT_OUTPUT_LIMIT
            agent_stdout = _bounded_text(raw_stdout, limit)
            return {"type": "result", "ok": False, "stdout": agent_stdout, "context_bytes": len(raw_stdout.encode()), "agent_bytes": len(agent_stdout.encode()), "output_limit_bytes": limit, "error": f"{type(exc).__name__}: {exc}", "traceback": traceback.format_exc(limit=4), "invocations": list(self.invocations)}
        finally:
            _INVOCATION_CONTEXT.reset(context_token)


DEFAULT_OUTPUT_LIMIT = 4096
MATERIALIZED_OUTPUT_LIMIT = 65536


def _bounded_text(value: str, limit: int) -> str:
    if limit < 1:
        return ""
    encoded = value.encode("utf-8")
    if len(encoded) <= limit:
        return value
    suffix = "…"
    suffix_bytes = suffix.encode("utf-8")
    if len(suffix_bytes) >= limit:
        return encoded[:limit].decode("utf-8", errors="ignore")
    prefix = encoded[: limit - len(suffix_bytes)].decode("utf-8", errors="ignore")
    return prefix + suffix


def _output_response(raw_stdout: str, include_materialized: bool, response_type: str) -> dict[str, Any]:
    limit = MATERIALIZED_OUTPUT_LIMIT if include_materialized else DEFAULT_OUTPUT_LIMIT
    agent_stdout = _bounded_text(raw_stdout, limit)
    return {"type": response_type, "ok": True, "stdout": agent_stdout, "context_bytes": len(raw_stdout.encode()), "agent_bytes": len(agent_stdout.encode()), "output_limit_bytes": limit}


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
    libraries = []
    library_path = os.environ.get("PROGRAM_RUNTIME_LIBRARIES_FILE", "")
    if library_path:
        try:
            with open(library_path, encoding="utf-8") as handle:
                libraries = json.load(handle)
        except (OSError, json.JSONDecodeError):
            libraries = []
    kernel = SessionKernel(bindings, os.environ.get("PROGRAM_RUNTIME_SESSION_ID", ""), os.environ.get("PROGRAM_RUNTIME_BRIDGE_URL", ""), os.environ.get("PROGRAM_RUNTIME_AGENT_BRIDGE_URL", ""), os.environ.get("PROGRAM_RUNTIME_DISCOVERY_URL", ""), libraries)
    protocol_stdout = sys.stdout

    def send(response: dict[str, Any]) -> None:
        protocol_stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
        protocol_stdout.flush()

    for line in sys.stdin:
        if not line.strip():
            continue
        try:
            request = json.loads(line)
            include_materialized = bool(request.get("include_materialized", False))
            updates: queue.Queue[dict[str, Any]] = queue.Queue(maxsize=1)

            def progress(raw_stdout: str) -> None:
                response = _output_response(raw_stdout, include_materialized, "progress")
                try:
                    updates.put_nowait(response)
                except queue.Full:
                    try:
                        updates.get_nowait()
                    except queue.Empty:
                        pass
                    try:
                        updates.put_nowait(response)
                    except queue.Full:
                        pass

            result: dict[str, Any] = {}

            def run() -> None:
                result.update(kernel.execute(str(request.get("source", "")), include_materialized, str(request.get("program_id", "")), str(request.get("provenance", "")), progress))

            worker = threading.Thread(target=run, name="program-runtime-kernel", daemon=True)
            worker.start()
            while worker.is_alive() or not updates.empty():
                try:
                    send(updates.get(timeout=0.1))
                except queue.Empty:
                    pass
            send(result)
            continue
        except Exception as exc:  # malformed transport input is a request failure
            send({"type": "result", "ok": False, "error": f"protocol error: {exc}"})


if __name__ == "__main__":
    serve()
