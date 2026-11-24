"""Exact Browser Use 1.0.3 optional adapter contract tests (no provider calls)."""

from __future__ import annotations

import importlib.metadata
import json
from pathlib import Path
import unittest
from typing import Any
from unittest.mock import AsyncMock, patch

try:
    _version = importlib.metadata.version("browser-use")
except importlib.metadata.PackageNotFoundError as exc:
    raise unittest.SkipTest("optional browser-use==1.0.3 is not installed") from exc

if _version != "1.0.3":
    raise unittest.SkipTest(f"tests require browser-use==1.0.3, found {_version}")

from pydantic_ai.messages import ModelResponse, ToolCallPart
from pydantic_ai.models.function import FunctionModel

from .browser_use_adapter import ALLOWED_MODEL_TOOLS, BrowserUseBrokerAdapter

VECTOR = json.loads((Path(__file__).parents[3] / "testdata/harness-conformance/v1.json").read_text())


class FakeBroker:
    def __init__(self) -> None:
        self.active = True
        self.in_scope = True
        self.clicked = False
        self.input_value = ""
        self.url = "https://allowed.example/start"
        self.trace: list[dict[str, Any]] = []
        self.calls: list[tuple[str, dict[str, Any]]] = []
        self.draft_calls: list[tuple[str, dict[str, Any]]] = []

    async def draft_trace(self, session_id: str, request: dict[str, Any]) -> dict[str, Any]:
        self.draft_calls.append((session_id, request))
        if session_id != "session-1" or "project_id" in request:
            raise PermissionError("BAS draft owner/session denied")
        return {"draft_digest": "sha256:preview", "workflow_id": ""}

    async def run_task(self, session_id: str, task: dict[str, Any]) -> dict[str, Any]:
        self.calls.append((session_id, task))
        operation = task["operation"]
        self.trace.append({"sequence": len(self.trace) + 1, "session_id": session_id, "owner_id": "agent-1", "grant_id": "grant-1", "operation": operation, "permission_class": {"read": "observe", "click": "act", "fill": "act", "navigate": "navigate", "receipt": "billing-read"}[operation], "effect_class": {"read": "none", "click": "ui", "fill": "ui", "navigate": "navigation", "receipt": "none"}[operation], "effect_applied": False})
        if session_id != "session-1" or not self.active or not self.in_scope:
            raise PermissionError("BAS owner/grant/scope denied task")
        if operation == "receipt":
            return {"settlement_status": "settled", "receipt": {"reservation_id": "hold-1", "status": "settled", "browser_session_minutes": 2}}
        if task["operation"] == "read":
            return {"url": self.url, "tree": {"root": {"role": "RootWebArea", "name": "Broker fixture", "children": [{"role": "button", "name": "Opened" if self.clicked else "Continue", "ref": "e123"}, {"role": "textbox", "name": "Search", "value": self.input_value, "ref": "e456"}]}}}
        if task["operation"] == "click" and task["ref"] == "e123":
            self.clicked = True
            self.trace[-1]["effect_applied"] = True
            return {"accepted": True}
        if task["operation"] == "fill" and task["ref"] == "e456":
            self.input_value = task["value"]
            self.trace[-1]["effect_applied"] = True
            return {"accepted": True}
        if task["operation"] == "navigate" and task["url"] == "https://allowed.example/next":
            self.url = task["url"]
            self.trace[-1]["effect_applied"] = True
            return {"url": self.url}
        raise PermissionError("unsupported or stale BAS operation")


def task_model(broker: FakeBroker, revoke_before_action: str | None = None) -> FunctionModel:
    step = 0

    def respond(_messages: Any, info: Any) -> ModelResponse:
        nonlocal step
        visible = {tool.name for tool in info.function_tools}
        visible.update(tool.name for tool in info.output_tools)
        forbidden = {"python", "finish_from_repl", "browser_close", "browser_start", "browser_evaluate", "browser_cdp"}
        assert not (visible & forbidden), f"forbidden model-visible tools: {sorted(visible & forbidden)}"
        if revoke_before_action and step == VECTOR["expected_revocation"]["denied_step_index"]:
            if revoke_before_action == "owner": broker.active = False
            if revoke_before_action == "scope": broker.in_scope = False
        if (not broker.active or not broker.in_scope) and revoke_before_action and step > VECTOR["expected_revocation"]["denied_step_index"]:
            name, args = "finish", {"result": "BAS denied the action before effect", "success": False}
        elif step < len(VECTOR["steps"]):
            current = VECTOR["steps"][step]
            name, args = current["tool"], current["arguments"]
        else:
            name, args = "finish", {"result": "verified", "success": True}
        step += 1
        return ModelResponse(parts=[ToolCallPart(tool_name=name, args=args)])

    return FunctionModel(function=respond)


class BrowserUseBrokerAdapterTests(unittest.IsolatedAsyncioTestCase):
    async def test_settlement_receipt_uses_the_same_bound_broker_transport(self) -> None:
        broker = FakeBroker()
        adapter = BrowserUseBrokerAdapter(task_model(broker), broker, "session-1")
        receipt = await adapter.settlement_receipt()
        self.assertEqual(receipt["settlement_status"], "settled")
        self.assertEqual(receipt["receipt"]["reservation_id"], "hold-1")
        self.assertEqual(broker.calls, [("session-1", {"operation": "receipt"})])

    async def test_schema_and_same_session_observe_action_read(self) -> None:
        broker = FakeBroker()
        adapter = BrowserUseBrokerAdapter(task_model(broker), broker, "session-1")
        self.assertEqual(set(adapter.model_tool_names()), set(ALLOWED_MODEL_TOOLS))
        from browser_use.agent import loop, service
        from browser_use.browser import Browser

        with (
            patch.object(loop, "Repl", side_effect=AssertionError("REPL must not be constructed")) as repl,
            patch.object(service.Agent, "_run", new_callable=AsyncMock, side_effect=AssertionError("service Agent lifecycle must not run")) as lifecycle,
            patch.object(Browser, "start", new_callable=AsyncMock, side_effect=AssertionError("Browser.start must not run")) as start,
            patch.object(Browser, "stop", new_callable=AsyncMock, side_effect=AssertionError("Browser.stop must not run")) as stop,
        ):
            result = await adapter.run("Click Continue after inspecting the page")
        repl.assert_not_called()
        lifecycle.assert_not_awaited()
        start.assert_not_awaited()
        stop.assert_not_awaited()
        self.assertEqual(result.status, "completed")
        self.assertTrue(broker.clicked)
        self.assertEqual(broker.input_value, VECTOR["expected_final"]["input"])
        self.assertEqual(broker.url, VECTOR["expected_final"]["url"])
        self.assertEqual([sid for sid, _ in broker.calls], [VECTOR["session_id"]] * len(VECTOR["steps"]))
        self.assertEqual([task for _, task in broker.calls], [step["task"] for step in VECTOR["steps"]])
        self.assertEqual(broker.trace, [
            {"sequence": index + 1, "session_id": VECTOR["session_id"], "owner_id": VECTOR["owner_id"], "grant_id": VECTOR["grant_id"], "operation": step["task"]["operation"], "permission_class": step["permission_class"], "effect_class": step["effect_class"], "effect_applied": step["effect_class"] != "none"}
            for index, step in enumerate(VECTOR["steps"])
        ])

    async def test_owner_revoke_blocks_next_action_before_effect(self) -> None:
        broker = FakeBroker()
        adapter = BrowserUseBrokerAdapter(task_model(broker, "owner"), broker, "session-1")
        result = await adapter.run("Observe then click")
        self.assertFalse(broker.clicked)
        self.assertEqual([task["operation"] for _, task in broker.calls], ["read", "click"])
        self.assertFalse(broker.trace[-1]["effect_applied"])
        self.assertIn("denied", result.text.lower())

    async def test_scope_loss_blocks_next_action_before_effect(self) -> None:
        broker = FakeBroker()
        adapter = BrowserUseBrokerAdapter(task_model(broker, "scope"), broker, "session-1")
        result = await adapter.run("Observe then click")
        self.assertFalse(broker.clicked)
        self.assertEqual([task["operation"] for _, task in broker.calls], ["read", "click"])
        self.assertIn("denied", result.text.lower())

    async def test_draft_tool_uses_same_session_without_project_persistence(self) -> None:
        broker = FakeBroker()
        calls = 0

        def respond(_messages: Any, _info: Any) -> ModelResponse:
            nonlocal calls
            calls += 1
            if calls == 1:
                return ModelResponse(parts=[ToolCallPart(tool_name="browser_draft_trace", args={
                    "name": "Checkout preview",
                    "description": "Unsaved draft",
                    "assertions": [{"type": "ACTION_TYPE_ASSERT", "assert": {"selector": "#done", "mode": "ASSERTION_MODE_VISIBLE"}}],
                })])
            return ModelResponse(parts=[ToolCallPart(tool_name="finish", args={"result": "draft created", "success": True})])

        adapter = BrowserUseBrokerAdapter(FunctionModel(function=respond), broker, "session-1")
        result = await adapter.run("Compile the accepted trace as an unsaved draft")
        self.assertEqual(result.status, "completed")
        self.assertEqual(broker.draft_calls, [("session-1", {
            "name": "Checkout preview",
            "assertions": [{"type": "ACTION_TYPE_ASSERT", "assert": {"selector": "#done", "mode": "ASSERTION_MODE_VISIBLE"}}],
            "description": "Unsaved draft",
        })])
