"""Shared Broker vector exercised through the real Browser Use adapter with a fake pinned loop."""
from __future__ import annotations

import json
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch
from typing import Any

from . import browser_use_adapter as adapter_module
from .browser_use_adapter import ALLOWED_MODEL_TOOLS, BrowserUseBrokerAdapter

VECTOR = json.loads((Path(__file__).parents[3] / "testdata/harness-conformance/v1.json").read_text())


class FakeBroker:
    def __init__(self) -> None:
        self.active = True
        self.in_scope = True
        self.url = "https://allowed.example/start"
        self.clicked = False
        self.value = ""
        self.revoke_after_read = False
        self.tasks: list[dict[str, Any]] = []
        self.trace: list[dict[str, Any]] = []

    async def run_task(self, session_id: str, task: dict[str, Any]) -> dict[str, Any]:
        self.tasks.append(task)
        operation = task["operation"]
        entry = {
            "sequence": len(self.trace) + 1,
            "session_id": session_id,
            "owner_id": VECTOR["owner_id"],
            "grant_id": VECTOR["grant_id"],
            "operation": operation,
            "permission_class": {"read": "observe", "click": "act", "fill": "act", "navigate": "navigate"}[operation],
            "effect_class": {"read": "none", "click": "ui", "fill": "ui", "navigate": "navigation"}[operation],
            "effect_applied": False,
        }
        self.trace.append(entry)
        if session_id != VECTOR["session_id"] or not self.active or not self.in_scope:
            raise PermissionError("BAS owner/grant/scope denied task")
        if operation == "read":
            result = {"url": self.url, "tree": {"root": {"role": "RootWebArea", "name": "Broker fixture", "children": [
                {"role": "button", "name": "Opened" if self.clicked else "Continue", "ref": "e123"},
                {"role": "textbox", "name": "Search", "value": self.value, "ref": "e456"},
            ]}}}
            if self.revoke_after_read:
                self.active = False
            return result
        if operation == "click" and task["ref"] == "e123":
            self.clicked = True
            entry["effect_applied"] = True
            return {"accepted": True}
        if operation == "fill" and task["ref"] == "e456":
            self.value = task["value"]
            entry["effect_applied"] = True
            return {"accepted": True}
        if operation == "navigate" and task["url"] == "https://allowed.example/next":
            self.url = task["url"]
            entry["effect_applied"] = True
            return {"url": self.url}
        raise PermissionError("unsupported or stale BAS operation")


class FakeTool:
    def __init__(self, function: Any, name: str, description: str) -> None:
        self.function, self.name, self.description = function, name, description


class FakeSession:
    def __init__(self, **_: Any) -> None: pass


class FakeResult:
    def __init__(self, text: str = "verified", status: str = "completed") -> None:
        self.text, self.status = text, status


class FakeLoop:
    conformance_steps: list[dict[str, Any]] | None = None

    @staticmethod
    def build_agent(*, model: Any, tools: list[FakeTool]) -> Any:
        del model
        tools = [*tools, FakeTool(lambda *_args, **_kwargs: None, "python", "pinned REPL tool removed by adapter")]
        return SimpleNamespace(
            _function_toolset=SimpleNamespace(tools={tool.name: tool for tool in tools}),
            _output_toolset=SimpleNamespace(_tool_defs=[SimpleNamespace(name="finish_from_repl"), SimpleNamespace(name="finish")]),
        )

    @staticmethod
    async def run(agent: Any, task: str, session: Any, *, model_ref: str) -> FakeResult:
        del task, session, model_ref
        tools = agent._function_toolset.tools
        observations: list[str] = []
        try:
            steps = FakeLoop.conformance_steps or VECTOR["steps"]
            for index, step in enumerate(steps):
                if index == VECTOR["expected_revocation"]["denied_step_index"] and getattr(FakeLoop, "revoke_broker", None):
                    FakeLoop.revoke_broker.active = False
                name = step["tool"]
                args = step["arguments"]
                if name == "browser_snapshot": observations.append(await tools[name].function())
                elif name == "browser_click": await tools[name].function(ref=args["target"])
                elif name == "browser_type": await tools[name].function(ref=args["target"], text=args["text"])
                elif name == "browser_navigate": await tools[name].function(url=args["url"])
            return FakeResult(observations[-1])
        except PermissionError:
            return FakeResult("BAS denied the action before effect", "failed")


class HarnessConformanceTests(unittest.IsolatedAsyncioTestCase):
    async def test_shared_rejected_calls_fail_before_transport(self) -> None:
        broker = FakeBroker()
        with patch.object(adapter_module, "_pinned_imports", return_value=(FakeLoop, FakeTool, FakeSession, FakeResult)):
            adapter = BrowserUseBrokerAdapter(object(), broker, VECTOR["session_id"])
            tools = adapter._agent._function_toolset.tools
            for rejected in VECTOR["rejected_calls"]:
                function = tools[rejected["browser_use_tool"]].function
                with self.subTest(case=rejected["id"]):
                    with self.assertRaises((TypeError, ValueError)):
                        await function(**rejected["browser_use_arguments"])
                    self.assertEqual(broker.tasks, [], rejected["reason"])

    async def test_shared_vector_runs_in_browser_use_adapter(self) -> None:
        broker = FakeBroker()
        with patch.object(adapter_module, "_pinned_imports", return_value=(FakeLoop, FakeTool, FakeSession, FakeResult)):
            adapter = BrowserUseBrokerAdapter(object(), broker, VECTOR["session_id"])
            self.assertEqual(set(adapter.model_tool_names()), set(ALLOWED_MODEL_TOOLS))
            result = await adapter.run("Execute the shared conformance vector")
        self.assertEqual(result.status, "completed")
        final_observation = json.loads(result.text)
        children = final_observation["tree"]["root"]["children"]
        self.assertEqual({
            "url": final_observation["url"],
            "button": next(item["name"] for item in children if item["role"] == "button"),
            "textbox": next(item["name"] for item in children if item["role"] == "textbox"),
        }, VECTOR["expected_final_observation"])
        self.assertEqual(children[0]["name"], VECTOR["expected_final"]["button"])
        self.assertEqual(children[1]["value"], VECTOR["expected_final"]["input"])
        self.assertTrue(broker.clicked)
        self.assertEqual(broker.value, VECTOR["expected_final"]["input"])
        self.assertEqual(broker.url, VECTOR["expected_final"]["url"])
        self.assertEqual(broker.trace, VECTOR["expected_trace"])
        self.assertEqual(broker.tasks, [step["task"] for step in VECTOR["steps"]])
        self.assertEqual([row["session_id"] for row in broker.trace], [VECTOR["session_id"]] * len(VECTOR["steps"]))
        self.assertEqual([row["permission_class"] for row in broker.trace], [step["permission_class"] for step in VECTOR["steps"]])
        self.assertEqual([row["effect_class"] for row in broker.trace], [step["effect_class"] for step in VECTOR["steps"]])
        self.assertEqual([row["effect_applied"] for row in broker.trace], [step["effect_class"] != "none" for step in VECTOR["steps"]])

    async def test_revocation_vector_step_is_denied_without_effect(self) -> None:
        broker = FakeBroker()
        broker.revoke_after_read = True
        FakeLoop.revoke_broker = broker
        with patch.object(adapter_module, "_pinned_imports", return_value=(FakeLoop, FakeTool, FakeSession, FakeResult)):
            result = await BrowserUseBrokerAdapter(object(), broker, VECTOR["session_id"]).run("Revocation probe")
        self.assertEqual(result.status, "failed")
        self.assertEqual(broker.trace, VECTOR["expected_revocation"]["trace"])
        self.assertFalse(broker.trace[-1]["effect_applied"])
        del FakeLoop.revoke_broker

    async def test_shared_out_of_scope_navigation_is_owner_denied_before_effect(self) -> None:
        broker = FakeBroker()
        with patch.object(adapter_module, "_pinned_imports", return_value=(FakeLoop, FakeTool, FakeSession, FakeResult)):
            adapter = BrowserUseBrokerAdapter(object(), broker, VECTOR["session_id"])
            navigate = adapter._agent._function_toolset.tools["browser_navigate"].function
            with self.assertRaises(PermissionError):
                await navigate(url=VECTOR["expected_scope_denial"]["url"])
        self.assertEqual(broker.trace, VECTOR["expected_scope_denial"]["trace"])
        self.assertFalse(broker.trace[-1]["effect_applied"])


if __name__ == "__main__":
    unittest.main()
