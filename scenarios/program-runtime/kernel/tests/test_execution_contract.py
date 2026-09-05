import asyncio
import io
import time
from unittest.mock import patch

import pytest

from host.engine import Handle, Namespace, SessionKernel, _InferenceSurface


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
    assert "demo_service.search.query" in kernel.globals["__vrooli__"].callable_namespace()
    assert "other.health.status" in kernel.globals["__vrooli__"].callable_namespace()
    discovered = kernel.execute("print(discover('demo search').head())")
    assert discovered["ok"]
    assert "bridge is not configured" in discovered["stdout"]


def test_reachable_reports_scenario_reason_and_blocks_unreachable_binding():
    kernel = SessionKernel([
        {"id": "offline/health/status", "scenario": "offline", "group": "health", "command": "status", "effect": "read", "reachable": False, "reachability_reason": "scenario API is not running"},
        {"id": "online/health/status", "scenario": "online", "group": "health", "command": "status", "effect": "read", "reachable": True, "reachability_reason": "scenario API resolved"},
    ], bridge_url="http://127.0.0.1:1/internal/program-runtime/bindings/execute")
    reachability = kernel.execute("print(reachable().materialize())")
    assert reachability["ok"]
    assert "offline" in reachability["stdout"]
    assert "scenario API is not running" in reachability["stdout"]
    blocked = kernel.execute("offline.health.status()")
    assert not blocked["ok"]
    assert "unreachable" in blocked["error"]
    assert blocked["invocations"] == []


def test_bare_binding_call_executes_after_top_level_await():
    class FakeBinding:
        def __call__(self):
            return Handle([{"ok": True}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    result = kernel.execute("import asyncio\nawait asyncio.sleep(0)\nresult = demo.work()\nprint(result.count())")
    assert result["ok"]
    assert "1" in result["stdout"]
    assert len(result["invocations"]) == 1


def test_await_returns_the_same_eager_handle():
    class FakeBinding:
        def __call__(self):
            return Handle([{"ok": True}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    result = kernel.execute("handle = demo.work()\nawaited = await handle\nprint(handle is awaited)")
    assert result["ok"]
    assert "True" in result["stdout"]


def test_gather_runs_binding_callables_in_parallel():
    class FakeBinding:
        def __call__(self, delay=0.05):
            time.sleep(delay)
            return Handle([{"delay": delay}])

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    started = time.perf_counter()
    result = kernel.execute("results = gather(*[lambda: demo.work(delay=0.05) for _ in range(10)], max_workers=10)\nprint(len(results))")
    elapsed = time.perf_counter() - started
    assert result["ok"]
    assert "10" in result["stdout"]
    assert elapsed < 0.35


def test_describe_reads_live_binding_contract_from_registry_bridge():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"arguments":[{"name":"scenario","protoPath":"scenario","kind":"raw_string","required":true}]}'

    kernel = SessionKernel(session_id="session-1", bridge_url="http://127.0.0.1:1/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()):
        result = kernel.execute('print(describe("test-genie/runs/list").head(1))')
    assert result["ok"]
    assert "scenario" in result["stdout"]


def test_describe_surfaces_named_registry_error():
    import urllib.error

    kernel = SessionKernel(session_id="session-1", bridge_url="http://127.0.0.1:1/internal/program-runtime/bindings/execute")
    error = urllib.error.HTTPError(
        "http://127.0.0.1:1",
        400,
        "bad request",
        {},
        io.BytesIO(b'{"error":"binding \\\"missing\\\" is not governed"}'),
    )
    with patch("host.engine.urllib.request.urlopen", side_effect=error):
        result = kernel.execute('describe("missing")')
    assert not result["ok"]
    assert "missing" in result["error"]


def test_gather_uses_default_concurrency_limit_without_rejecting_fanout():
    kernel = SessionKernel({"demo": {"work": lambda: Handle([{"ok": True}])}})
    result = kernel.execute("print(len(gather(*[lambda: demo.work() for _ in range(200)])))")
    assert result["ok"]
    assert "200" in result["stdout"]


def test_gather_preserves_order_and_caps_peak_concurrency():
    active = 0
    peak = 0
    lock = __import__("threading").Lock()

    def call(index):
        nonlocal active, peak
        with lock:
            active += 1
            peak = max(peak, active)
        time.sleep(0.002)
        with lock:
            active -= 1
        return Handle([{"index": index}])

    class FakeBinding:
        def __call__(self, index):
            return call(index)

    kernel = SessionKernel({"demo": {"work": FakeBinding()}})
    result = kernel.execute("print([item.head(1)[0]['index'] for item in gather(*[lambda i=i: demo.work(index=i) for i in range(200)], max_workers=8)])")
    assert result["ok"]
    assert "[0, 1, 2, 3" in result["stdout"]
    assert peak <= 8


def test_collision_requires_qualified_scenario_path():
    kernel = SessionKernel([
        {"id": "alpha/search/query", "scenario": "alpha", "group": "search", "command": "query"},
        {"id": "beta/search/query", "scenario": "beta", "group": "search", "command": "query"},
    ])
    result = kernel.execute("search.query()")
    assert result["ok"] is False
    assert "does not resolve" in result["error"]


def test_unique_bare_group_requires_qualified_scenario_path():
    kernel = SessionKernel([
        {"id": "demo/search/query", "scenario": "demo", "group": "search", "command": "query"},
    ])
    result = kernel.execute("search.query()")
    assert result["ok"] is False
    assert "does not resolve" in result["error"]


def test_flat_namespace_has_project_escape_hatch_and_protected_runtime_names():
    kernel = SessionKernel({"demo": {"work": lambda: Handle([{"ok": True}])}, "vrooli": {"scenario": {"start": lambda: Handle([{"started": True}])}}})
    result = kernel.execute("print(demo.work().count())\nprint(__vrooli__.demo.work().count())\nprint(vrooli.scenario.start().count())")
    assert result["ok"]
    assert result["stdout"].count("1") == 3
    refused = kernel.execute("discover = 1")
    assert not refused["ok"]
    assert "protected runtime name" in refused["error"]


def test_unknown_root_reports_nearest_match():
    result = SessionKernel({"search_hub": {"query": lambda: Handle([{"ok": True}])}}).execute("serach_hub.query()")
    assert not result["ok"]
    assert "does not resolve" in result["error"]


def test_inference_facade_forwards_optional_profile_and_batch_shape():
    class FakeBinding:
        def __init__(self):
            self.calls = []

        def __call__(self, **kwargs):
            self.calls.append(kwargs)
            return Handle([kwargs])

    run = FakeBinding()
    batch = FakeBinding()
    surface = _InferenceSurface("session", "configured", [])
    surface._run = run
    surface._run_batch = batch
    surface.classify("hello", profile={"locality": "local"}, turns=[{"role": "user", "text": "hi"}])
    surface.batch(["a", "b"], {"type": "string"}, "choose", role="classify.fast")
    assert run.calls[0]["profile"] == {"locality": "local"}
    assert run.calls[0]["turns"][0]["text"] == "hi"
    assert batch.calls[0]["items"] == [{"source": "a"}, {"source": "b"}]


def test_inference_and_describe_accept_additive_model_facing_aliases():
    class FakeBinding:
        def __init__(self):
            self.calls = []

        def __call__(self, **kwargs):
            self.calls.append(kwargs)
            return Handle([kwargs])

    run = FakeBinding()
    batch = FakeBinding()
    surface = _InferenceSurface("session", "configured", [])
    surface._run = run
    surface._run_batch = batch

    surface.classify(text="classify this")
    surface.extract(text="extract this")
    surface.judge(text="judge this")
    surface.write(text="write this")
    assert [call["source"] for call in run.calls[:4]] == ["classify this", "extract this", "judge this", "write this"]

    # The additive plural spelling resolves before the governed batch route.
    with pytest.raises(RuntimeError, match="incomplete result set"):
        surface.classify(texts=["one", "two"])
    assert batch.calls[-1]["items"] == [{"source": "one"}, {"source": "two"}]

    surface.extract(texts=["extract one", "extract two"])
    surface.judge(texts=["judge one", "judge two"])
    surface.write(texts=["write one", "write two"])
    assert [call["source"] for call in run.calls[-3:]] == [
        ["extract one", "extract two"],
        ["judge one", "judge two"],
        ["write one", "write two"],
    ]

    surface.classify(source="classify this", labels=["bug", "feature"])
    schema = __import__("json").loads(run.calls[-1]["schema_json"])
    assert schema == {
        "type": "object",
        "properties": {"label": {"type": "string", "enum": ["bug", "feature"]}},
        "required": ["label"],
    }

    with pytest.raises(TypeError, match=r"source=.*text=|text=.*source="):
        surface.classify(source="canonical", text="alias")
    with pytest.raises(TypeError, match=r"text=.*texts=|texts=.*text="):
        surface.classify(text="canonical", texts=["alias"])
    for operation in (surface.classify, surface.extract, surface.judge, surface.write):
        with pytest.raises(TypeError, match=r"source=.*texts=|texts=.*source="):
            operation(source="canonical", texts=["alias"])
    with pytest.raises(TypeError, match=r"schema=.*labels=|labels=.*schema="):
        surface.classify(text="both", schema={"type": "string"}, labels=["bug"])
    with pytest.raises(ValueError, match="labels"):
        surface.classify(text="empty", labels=[])
    with pytest.raises(TypeError, match="1"):
        surface.classify(text="non-string", labels=["bug", 1])


def test_inference_and_describe_unknown_keywords_list_the_accepted_set():
    surface = _InferenceSurface("session", "configured", [])
    for name, operation in (
        ("classify", surface.classify),
        ("extract", surface.extract),
        ("judge", surface.judge),
        ("write", surface.write),
    ):
        with pytest.raises(TypeError, match=rf"{name}.*mystery.*accepted keywords.*source.*text.*texts"):
            operation(mystery="value")

    namespace = Namespace(session_id="session", bridge_url="configured")
    with pytest.raises(TypeError, match=r"describe.*mystery.*accepted keywords.*binding.*binding_id"):
        namespace.describe(mystery="value")


def test_classify_accepts_small_string_batches_through_governed_batch_route():
    class FakeBatch:
        def __call__(self, **kwargs):
            assert kwargs["items"] == [{"source": "one"}, {"source": "two"}]
            return Handle(
                [{
                    "results": [
                        {"valueJson": '{"label":"bug"}'},
                        {"valueJson": '{"label":"feature"}'},
                    ]
                }],
                raw={
                    "results": [
                        {"valueJson": '{"label":"bug"}'},
                        {"valueJson": '{"label":"feature"}'},
                    ]
                },
            )

    surface = _InferenceSurface("session", "configured", [])
    surface._run_batch = FakeBatch()
    result = surface.classify(source=["one", "two"], labels=["bug", "feature"])
    assert result.materialize() == [
        {"label": "bug", "text": "one"},
        {"label": "feature", "text": "two"},
    ]
    assert result.meta()["role"] == "classify.fast"


def test_describe_accepts_binding_id_alias():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"arguments":[{"name":"scenario"}]}'

    namespace = Namespace(session_id="session", bridge_url="http://127.0.0.1:1/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()) as open_url:
        result = namespace.describe(binding_id="test-genie/runs/list")
    assert result.head(1)[0]["name"] == "scenario"
    assert open_url.call_args.args[0].data == b'{"session_id": "session", "binding": "test-genie/runs/list"}'

    with pytest.raises(TypeError, match=r"binding=.*binding_id=|binding_id=.*binding="):
        namespace.describe(binding="canonical", binding_id="alias")


def test_write_facade_carries_sampling_and_omits_it_when_unset():
    class FakeBinding:
        def __init__(self):
            self.calls = []

        def __call__(self, **kwargs):
            self.calls.append(kwargs)
            return Handle([kwargs])

    run = FakeBinding()
    surface = _InferenceSurface("session", "configured", [])
    surface._run = run

    surface.write("draft a paragraph")
    assert run.calls[0]["role"] == "write.default"
    # An absent temperature must stay off the request so the role's own declared
    # sampling applies. Sending a default here would silently pin every call.
    assert "sampling" not in run.calls[0]
    assert "max_output_tokens" not in run.calls[0]

    surface.write("draft a paragraph", temperature=1.2, max_output_tokens=4096)
    assert run.calls[1]["sampling"] == {"temperature": 1.2}
    assert run.calls[1]["max_output_tokens"] == 4096

    # 0.0 is a meaningful deterministic request, not an absence.
    surface.write("draft a paragraph", temperature=0.0)
    assert run.calls[2]["sampling"] == {"temperature": 0.0}
