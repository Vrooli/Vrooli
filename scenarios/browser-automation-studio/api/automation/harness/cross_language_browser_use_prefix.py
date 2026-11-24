"""Run the real Browser Use Broker adapter against a shared test BAS HTTP owner.

The adapter's pinned model loop is faked so this test does not require the
optional Browser Use package or an inference provider. The Broker transport is
the real Agent Manager environment-backed HTTP transport.
"""

from __future__ import annotations

import asyncio
import json

from . import browser_use_adapter as adapter_module
from .browser_use_adapter import ALLOWED_MODEL_TOOLS, BASAgentManagerTaskTransport, BrowserUseBrokerAdapter
from .test_harness_conformance import FakeLoop, FakeResult, FakeSession, FakeTool, VECTOR


async def main() -> None:
    session_id = VECTOR["session_id"]
    FakeLoop.conformance_steps = VECTOR["steps"][:3]
    original_imports = adapter_module._pinned_imports
    adapter_module._pinned_imports = lambda: (FakeLoop, FakeTool, FakeSession, FakeResult)
    try:
        transport = BASAgentManagerTaskTransport(session_id, timeout_s=10)
        adapter = BrowserUseBrokerAdapter(object(), transport, session_id)
        if set(adapter.model_tool_names()) != set(ALLOWED_MODEL_TOOLS):
            raise AssertionError("Browser Use Broker tool schema differs from its allowlist")
        result = await adapter.run("Execute the first half of the shared adapter vector")
        if result.status != "completed":
            raise AssertionError(f"Browser Use prefix failed: {result.status}: {result.text}")
        print(json.dumps({"status": result.status, "tools": sorted(adapter.model_tool_names())}))
    finally:
        adapter_module._pinned_imports = original_imports
        FakeLoop.conformance_steps = None


if __name__ == "__main__":
    asyncio.run(main())
