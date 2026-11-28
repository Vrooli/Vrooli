"""Browser Use draft-tool binding contract; no optional Browser Use package/provider needed."""

from __future__ import annotations

import unittest
from unittest.mock import patch
from typing import Any

from . import browser_use_adapter
from .browser_use_adapter import BrowserUseBrokerAdapter, _validate_broker_task


class _Tool:
    def __init__(self, function: Any, *, name: str, description: str):
        self.function = function
        self.name = name
        self.description = description


class _Session:
    def __init__(self, **_kwargs: Any):
        pass


class _Result:
    status = "completed"


class _Loop:
    def build_agent(self, model: Any, tools: list[_Tool]) -> Any:
        del model
        return type("Agent", (), {
            "_function_toolset": type("FunctionToolset", (), {"tools": {tool.name: tool for tool in tools} | {"python": object()}})(),
            "_output_toolset": type("OutputToolset", (), {"_tool_defs": [type("Definition", (), {"name": name})() for name in ("finish_from_repl", "finish")]})(),
        })()

    async def run(self, agent: Any, _task: str, _session: _Session, *, model_ref: str) -> _Result:
        assert model_ref == "bas-agentaccess"
        await agent._function_toolset.tools["browser_draft_trace"].function(
            "Checkout preview",
            [{"type": "ACTION_TYPE_ASSERT", "assert": {"selector": "#done", "mode": "ASSERTION_MODE_VISIBLE"}}],
            "Unsaved draft",
        )
        return _Result()


class _Transport:
    def __init__(self):
        self.drafts: list[tuple[str, dict[str, Any]]] = []
        self.tasks: list[tuple[str, dict[str, Any]]] = []

    async def run_task(self, session_id: str, task: dict[str, Any]) -> Any:
        self.tasks.append((session_id, task))
        return {"accepted": True}

    async def draft_trace(self, session_id: str, request: dict[str, Any]) -> Any:
        self.drafts.append((session_id, request))
        return {"draft_digest": "sha256:preview", "workflow_id": ""}


class BrowserUseDraftAdapterTests(unittest.IsolatedAsyncioTestCase):
    async def test_draft_tool_uses_borrowed_session_without_persistence_fields(self) -> None:
        transport = _Transport()
        imports = (_Loop(), _Tool, _Session, _Result)
        with patch.object(browser_use_adapter, "_pinned_imports", return_value=imports):
            adapter = BrowserUseBrokerAdapter(None, transport, "session-1")
            self.assertIn("browser_draft_trace", adapter.model_tool_names())
            result = await adapter.run("Compile the accepted trace")

        self.assertEqual(result.status, "completed")
        self.assertEqual(transport.drafts, [("session-1", {
            "name": "Checkout preview",
            "assertions": [{"type": "ACTION_TYPE_ASSERT", "assert": {"selector": "#done", "mode": "ASSERTION_MODE_VISIBLE"}}],
            "description": "Unsaved draft",
        })])
        self.assertNotIn("project_id", transport.drafts[0][1])

    async def test_bounded_keyboard_tool_uses_typed_broker_action(self) -> None:
        transport = _Transport()
        imports = (_Loop(), _Tool, _Session, _Result)
        with patch.object(browser_use_adapter, "_pinned_imports", return_value=imports):
            adapter = BrowserUseBrokerAdapter(None, transport, "session-1")
            tool = adapter._agent._function_toolset.tools["browser_press_key"].function
            await tool("Control+Enter")
            with self.assertRaises(ValueError):
                await tool("Alt+F4")
            _validate_broker_task({"operation": "keyboard", "key": "Enter", "modifiers": ["ctrl"]})
            with self.assertRaises(ValueError):
                _validate_broker_task({"operation": "keyboard", "key": "F4", "modifiers": ["alt"]})

        self.assertEqual(transport.tasks, [("session-1", {
            "operation": "keyboard", "key": "Enter", "modifiers": ["ctrl"],
        })])


if __name__ == "__main__":
    unittest.main()
