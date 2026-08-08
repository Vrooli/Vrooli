import asyncio
import time

from host.engine import Handle, SessionKernel


def test_top_level_await_and_final_expression_display():
    kernel = SessionKernel()
    result = kernel.execute("import asyncio\nfrom host.engine import Handle\nawait asyncio.sleep(0)\nHandle([{'ok': True}])")
    assert result["ok"]
    assert "Handle(label='result', rows=1)" in result["stdout"]


def test_output_cap_preserves_true_context_bytes():
    kernel = SessionKernel()
    result = kernel.execute("print('x' * 200000)")
    assert result["ok"]
    assert result["context_bytes"] > len(result["stdout"].encode())
    assert result["output_limit_bytes"] == 4096
    expanded = kernel.execute("print('x' * 200000)", include_materialized=True)
    assert len(expanded["stdout"].encode()) <= 65537
    assert expanded["output_limit_bytes"] == 65536


def test_public_scenario_namespace_and_discovery():
    kernel = SessionKernel([
        {"id": "demo-service/search/query", "scenario": "demo-service", "group": "search", "command": "query", "effect": "read"},
        {"id": "other/health/status", "scenario": "other", "group": "health", "command": "status", "effect": "read"},
    ])
    assert "vrooli.demo_service.search.query" in kernel.globals["vrooli"].callable_namespace()
    assert "vrooli.other.health.status" in kernel.globals["vrooli"].callable_namespace()
    discovered = kernel.execute("print(vrooli.discover('demo search').head())")
    assert discovered["ok"]
    assert "demo_service" in discovered["stdout"]


def test_bare_binding_call_executes_after_top_level_await():
    class FakeBinding:
        def __call__(self):
            return Handle([{"ok": True}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    result = kernel.execute("import asyncio\nawait asyncio.sleep(0)\nresult = vrooli.demo.work()\nprint(result.count())")
    assert result["ok"]
    assert "1" in result["stdout"]
    assert len(result["invocations"]) == 1


def test_await_returns_the_same_eager_handle():
    class FakeBinding:
        def __call__(self):
            return Handle([{"ok": True}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    result = kernel.execute("handle = vrooli.demo.work()\nawaited = await handle\nprint(handle is awaited)")
    assert result["ok"]
    assert "True" in result["stdout"]


def test_gather_runs_binding_callables_in_parallel():
    class FakeBinding:
        def __call__(self, delay=0.05):
            time.sleep(delay)
            return Handle([{"delay": delay}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    started = time.perf_counter()
    result = kernel.execute("results = vrooli.gather(*[lambda: vrooli.demo.work(delay=0.05) for _ in range(10)])\nprint(len(results))")
    elapsed = time.perf_counter() - started
    assert result["ok"]
    assert "10" in result["stdout"]
    assert elapsed < 0.35
