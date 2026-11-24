"""Optional Browser Use 1.0.3 decision loop over a borrowed BAS session.

Browser Use 1.0.3's public Agent lifecycle starts/stops its Browser and creates
a Python REPL. This adapter uses the pinned package's model/decision loop only,
then removes those private lifecycle and REPL tools before model schema
construction. It never constructs Browser, Repl, a CDP client, or an MCP server.
"""

from __future__ import annotations

import asyncio
import importlib.metadata
import json
import os
import re
import tempfile
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Protocol
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener

EXPECTED_BROWSER_USE_VERSION = "1.0.3"
BROKER_FUNCTION_TOOLS = frozenset({
    "browser_snapshot",
    "browser_click",
    "browser_type",
    "browser_navigate",
    "browser_press_key",
    "browser_receipt",
    "browser_draft_trace",
})
ALLOWED_MODEL_TOOLS = BROKER_FUNCTION_TOOLS | {"finish"}
INITIAL_PINNED_FUNCTION_TOOLS = BROKER_FUNCTION_TOOLS | {"python"}
OPAQUE_REF = re.compile(r"^e[0-9]{1,90}$")
PRESSABLE_KEY = re.compile(
    r"^(?:[A-Za-z0-9]|Enter|Tab|Escape|Backspace|Delete|Arrow(?:Left|Right|Up|Down)|Home|End|PageUp|PageDown|Space|F(?:[1-9]|1[0-2]))$",
    re.IGNORECASE,
)


def _keyboard_task(key: str) -> dict[str, Any]:
    if not isinstance(key, str):
        raise ValueError("key must be text")
    parts = key.split("+")
    if len(parts) == 1:
        modifiers: list[str] = []
        pressed = parts[0]
    elif len(parts) == 2 and parts[0].lower() in ("control", "ctrl", "meta"):
        modifiers = ["meta" if parts[0].lower() == "meta" else "ctrl"]
        pressed = parts[1]
    else:
        raise ValueError("key must be one supported key with at most one Ctrl or Meta modifier")
    if not PRESSABLE_KEY.fullmatch(pressed):
        raise ValueError("key is outside the bounded BAS keyboard contract")
    task: dict[str, Any] = {"operation": "keyboard", "key": pressed}
    if modifiers:
        task["modifiers"] = modifiers
    return task


def _validate_broker_task(task: dict[str, Any]) -> None:
    """Reject task-shape drift before forwarding authenticated run authority."""
    if not isinstance(task, dict) or not isinstance(task.get("operation"), str):
        raise ValueError("BAS task must be an object with a typed operation")
    operation = task["operation"]
    if operation in ("read", "receipt"):
        valid = set(task) == {"operation"}
    elif operation == "click":
        valid = set(task) == {"operation", "ref"} and isinstance(task.get("ref"), str) and OPAQUE_REF.fullmatch(task["ref"]) is not None
    elif operation == "fill":
        valid = set(task) == {"operation", "ref", "value"} and isinstance(task.get("ref"), str) and OPAQUE_REF.fullmatch(task["ref"]) is not None and isinstance(task.get("value"), str)
    elif operation == "navigate":
        valid = set(task) == {"operation", "url"} and isinstance(task.get("url"), str) and bool(task["url"].strip())
    elif operation == "keyboard":
        valid = (
            set(task) in ({"operation", "key"}, {"operation", "key", "modifiers"})
            and isinstance(task.get("key"), str)
            and PRESSABLE_KEY.fullmatch(task["key"]) is not None
            and ("modifiers" not in task or task["modifiers"] in (["ctrl"], ["meta"]))
        )
    else:
        valid = False
    if not valid:
        raise ValueError("BAS task is outside the fixed typed adapter contract")


class _RejectRedirects(HTTPRedirectHandler):
    """Keep Agent Manager run credentials bound to the configured BAS origin."""

    def redirect_request(self, req: Any, fp: Any, code: int, msg: str, headers: Any, newurl: str) -> None:
        return None


def _open_task_request(request: Request, timeout: float) -> Any:
    return build_opener(_RejectRedirects()).open(request, timeout=timeout)


class BrokerTaskTransport(Protocol):
    async def run_task(self, session_id: str, task: dict[str, Any]) -> Any: ...


class BASAgentManagerTaskTransport:
    """Session-bound HTTP transport using Agent Manager's injected run env."""

    def __init__(self, session_id: str, *, environment: dict[str, str] | None = None, timeout_s: float = 45):
        if not session_id.strip():
            raise ValueError("a pre-existing BAS session ID is required")
        env = os.environ if environment is None else environment
        base_url = env.get("BAS_BROWSER_API_URL", "").strip().rstrip("/")
        self._identity = env.get("VROOLI_AGENT_IDENTITY_TOKEN", "").strip()
        self._grant = env.get("VROOLI_BROWSER_GRANT_RECEIPT", "").strip()
        if not base_url or not self._identity or not self._grant:
            raise ValueError("BAS URL, Agent Manager identity, and browser grant are required from the run context")
        parsed = urlsplit(base_url)
        if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username or parsed.password or parsed.query or parsed.fragment:
            raise ValueError("BAS_BROWSER_API_URL must be an HTTP(S) origin without embedded credentials, query, or fragment")
        if parsed.path not in ("", "/"):
            raise ValueError("BAS_BROWSER_API_URL must be an origin")
        self._endpoint = f"{base_url}/api/v1/browser/sessions/{quote(session_id, safe='')}/task"
        self._session_id = session_id
        self._timeout_s = timeout_s

    async def run_task(self, session_id: str, task: dict[str, Any]) -> Any:
        if session_id != self._session_id:
            raise PermissionError("BAS task transport is bound to another session")
        _validate_broker_task(task)
        body = json.dumps(task, ensure_ascii=False, separators=(",", ":")).encode()

        def request() -> Any:
            req = Request(
                self._endpoint,
                data=body,
                headers={
                    "Content-Type": "application/json",
                    "Accept": "application/json",
                    "X-Agent-Identity-Token": self._identity,
                    "X-Vrooli-Browser-Grant-Receipt": self._grant,
                },
                method="POST",
            )
            try:
                with _open_task_request(req, timeout=self._timeout_s) as response:
                    payload = response.read(2 << 20)
                    if response.read(1):
                        raise RuntimeError("BAS task response exceeds the size limit")
                    return json.loads(payload) if payload else None
            except HTTPError as exc:
                payload = exc.read(1 << 16)
                code = "BROWSER_TASK_DENIED"
                try:
                    decoded = json.loads(payload)
                    candidate = decoded.get("error", {}).get("code")
                    if isinstance(candidate, str) and candidate:
                        code = candidate
                except (json.JSONDecodeError, AttributeError, TypeError):
                    pass
                raise PermissionError(code) from None
            except URLError as exc:
                raise RuntimeError("BAS task transport failed") from exc

        return await asyncio.to_thread(request)

    async def draft_trace(self, session_id: str, request_body: dict[str, Any]) -> Any:
        """Compile an unsaved draft through the same session-bound AM owner."""
        if session_id != self._session_id:
            raise PermissionError("BAS draft transport is bound to another session")
        if not isinstance(request_body, dict) or set(request_body) - {"name", "description", "assertions"}:
            raise ValueError("BAS trace draft is outside the fixed typed adapter contract")
        if not isinstance(request_body.get("name"), str) or not request_body["name"].strip():
            raise ValueError("BAS trace draft requires a name")
        if "description" in request_body and not isinstance(request_body["description"], str):
            raise ValueError("BAS trace draft description must be text")
        assertions = request_body.get("assertions")
        if not isinstance(assertions, list) or not assertions or any(not isinstance(item, dict) for item in assertions):
            raise ValueError("BAS trace draft requires typed assertion objects")
        body = json.dumps(request_body, ensure_ascii=False, separators=(",", ":")).encode()
        endpoint = self._endpoint.rsplit("/task", 1)[0] + "/trace/draft"

        def request() -> Any:
            req = Request(
                endpoint,
                data=body,
                headers={
                    "Content-Type": "application/json",
                    "Accept": "application/json",
                    "X-Agent-Identity-Token": self._identity,
                    "X-Vrooli-Browser-Grant-Receipt": self._grant,
                },
                method="POST",
            )
            try:
                with _open_task_request(req, timeout=self._timeout_s) as response:
                    payload = response.read(2 << 20)
                    if response.read(1):
                        raise RuntimeError("BAS draft response exceeds the size limit")
                    return json.loads(payload) if payload else None
            except HTTPError as exc:
                raise PermissionError("TRACE_DRAFT_DENIED") from None
            except URLError as exc:
                raise RuntimeError("BAS trace draft transport failed") from exc

        return await asyncio.to_thread(request)


@dataclass
class _LoopContext:
    """Only non-browser fields Browser Use's loop bookkeeping needs."""

    state: Path
    workspace: Path
    secrets: tuple[()] = ()
    search: None = None


def _pinned_imports() -> tuple[Any, Any, Any, Any]:
    try:
        installed = importlib.metadata.version("browser-use")
    except importlib.metadata.PackageNotFoundError as exc:
        raise RuntimeError("optional browser-use==1.0.3 package is not installed") from exc
    if installed != EXPECTED_BROWSER_USE_VERSION:
        raise RuntimeError(f"Browser Use version drift: expected {EXPECTED_BROWSER_USE_VERSION}, found {installed}")

    from browser_use.agent import loop
    from pydantic_ai import Tool

    return loop, Tool, loop.Session, loop.AgentHistoryList


class BrowserUseBrokerAdapter:
    """Runs Browser Use's pinned Pydantic decision loop with Broker-only tools."""

    def __init__(self, model: Any, transport: BrokerTaskTransport, session_id: str):
        if not session_id.strip():
            raise ValueError("a pre-existing BAS session ID is required")
        loop, tool_type, session_type, result_type = _pinned_imports()
        self._loop = loop
        self._transport = transport
        self._session_id = session_id
        self._session_type = session_type
        self._result_type = result_type

        async def snapshot() -> str:
            result = await self._transport.run_task(self._session_id, {"operation": "read"})
            return _json_text(result)

        async def click(ref: str) -> str:
            if not OPAQUE_REF.fullmatch(ref):
                raise ValueError("only a BAS opaque accessibility reference is accepted")
            result = await self._transport.run_task(self._session_id, {"operation": "click", "ref": ref})
            return _json_text(result)

        async def type_text(ref: str, text: str) -> str:
            if not OPAQUE_REF.fullmatch(ref):
                raise ValueError("only a BAS opaque accessibility reference is accepted")
            # Both official harnesses express text entry as replacement. This
            # maps to the Broker's typed `fill` operation (ClearFirst=true).
            result = await self._transport.run_task(self._session_id, {"operation": "fill", "ref": ref, "value": text})
            return _json_text(result)

        async def navigate(url: str) -> str:
            result = await self._transport.run_task(self._session_id, {"operation": "navigate", "url": url})
            return _json_text(result)

        async def press_key(key: str) -> str:
            result = await self._transport.run_task(self._session_id, _keyboard_task(key))
            return _json_text(result)

        async def receipt() -> str:
            result = await self._transport.run_task(self._session_id, {"operation": "receipt"})
            return _json_text(result)

        async def draft_trace(name: str, assertions: list[dict[str, Any]], description: str = "") -> str:
            draft_method = getattr(self._transport, "draft_trace", None)
            if not callable(draft_method):
                raise RuntimeError("BAS trace draft transport is unavailable")
            request: dict[str, Any] = {"name": name, "assertions": assertions}
            if description:
                request["description"] = description
            result = await draft_method(self._session_id, request)
            return _json_text(result)

        broker_tools = [
            tool_type(snapshot, name="browser_snapshot", description="Read the current BAS accessibility tree."),
            tool_type(click, name="browser_click", description="Click one opaque ref from the current BAS tree."),
            tool_type(type_text, name="browser_type", description="Type text into one opaque ref from the current BAS tree."),
            tool_type(navigate, name="browser_navigate", description="Navigate the existing BAS session under its Broker policy."),
            tool_type(press_key, name="browser_press_key", description="Press one bounded key or Ctrl/Meta shortcut through the existing BAS session and Broker policy."),
            tool_type(receipt, name="browser_receipt", description="Read the LPBS settlement result for this exact BAS session after it closes."),
            tool_type(draft_trace, name="browser_draft_trace", description="Compile this existing BAS session's accepted trace into a deterministic unsaved workflow draft. It does not write a project workflow."),
        ]
        self._agent = loop.build_agent(model=model, tools=broker_tools)
        self._remove_unsafe_tools()

    def _remove_unsafe_tools(self) -> None:
        """Fail closed if the exact pinned private Pydantic registry moves."""
        toolset = getattr(self._agent, "_function_toolset", None)
        registry = getattr(toolset, "tools", None)
        if not isinstance(registry, dict):
            raise RuntimeError("Browser Use 1.0.3 function tool registry shape changed")
        if set(registry) != INITIAL_PINNED_FUNCTION_TOOLS:
            raise RuntimeError(f"Browser Use 1.0.3 tool registry changed: {sorted(registry)}")
        del registry["python"]
        if set(registry) != BROKER_FUNCTION_TOOLS:
            raise RuntimeError(f"Browser Use Broker tool schema drifted: {sorted(registry)}")

        output_toolset = getattr(self._agent, "_output_toolset", None)
        output_registry = getattr(output_toolset, "_tool_defs", None)
        output_names = [getattr(definition, "name", None) for definition in output_registry] if isinstance(output_registry, list) else None
        if output_names != ["finish_from_repl", "finish"]:
            shape = {
                "type": type(output_toolset).__name__,
                "attributes": sorted(getattr(output_toolset, "__dict__", {}).keys()),
                "tools_type": type(output_registry).__name__,
                "tools": output_names,
            }
            raise RuntimeError(f"Browser Use 1.0.3 output tool registry shape changed: {shape}")
        output_registry[:] = [definition for definition in output_registry if definition.name == "finish"]
        if [definition.name for definition in output_registry] != ["finish"]:
            raise RuntimeError("Browser Use Broker output schema contains an unsupported tool")

    def model_tool_names(self) -> tuple[str, ...]:
        toolset = getattr(self._agent, "_function_toolset", None)
        registry = getattr(toolset, "tools", None)
        if not isinstance(registry, dict):
            raise RuntimeError("Browser Use 1.0.3 function tool registry shape changed")
        output_toolset = getattr(self._agent, "_output_toolset", None)
        output_registry = getattr(output_toolset, "_tool_defs", None)
        if not isinstance(output_registry, list):
            raise RuntimeError("Browser Use 1.0.3 output tool registry shape changed")
        return tuple(sorted(set(registry) | {definition.name for definition in output_registry}))

    async def settlement_receipt(self) -> Any:
        """Read the receipt for the bound session through the same AM-authenticated BAS task route."""
        return await self._transport.run_task(self._session_id, {"operation": "receipt"})

    async def run(self, task: str, *, max_steps: int = 8, timeout_s: float = 30) -> Any:
        if not task.strip():
            raise ValueError("task is required")
        # Browser Use's `loop.run` expects this small bookkeeping context. It
        # does not expose a Python call method, browser env, page or CDP handle.
        with tempfile.TemporaryDirectory(prefix="bas-browser-use-loop-") as state_dir:
            path = Path(state_dir)
            loop_context = _LoopContext(state=path, workspace=path)
            session = self._session_type(repl=loop_context, max_steps=max_steps, timeout_s=timeout_s)
            result = await self._loop.run(self._agent, task, session, model_ref="bas-agentaccess")
            if not isinstance(result, self._result_type):
                raise RuntimeError("pinned Browser Use loop returned an unexpected result type")
            return result

    def run_sync(self, task: str, *, max_steps: int = 8, timeout_s: float = 30) -> Any:
        try:
            asyncio.get_running_loop()
        except RuntimeError:
            return asyncio.run(self.run(task, max_steps=max_steps, timeout_s=timeout_s))
        raise RuntimeError("run_sync cannot be used inside an active event loop")


def _json_text(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"), default=str)
