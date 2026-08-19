import sys
from unittest.mock import patch
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[1]))

from host.engine import Handle, SessionKernel


def test_default_repr_is_bounded():  # [REQ:PRT-P0-003]
    handle = Handle(({"id": i} for i in range(100_000)))
    assert len(repr(handle)) < 100
    assert "100000" in repr(handle)


def test_materialization_is_explicit():  # [REQ:PRT-P0-003]
    kernel = SessionKernel()
    result = kernel.execute("result = __import__('host.engine', fromlist=['Handle']).Handle(range(1000))\nprint(result)")
    assert result["ok"]
    assert "[0" not in result["stdout"]
    assert "rows=1000" in result["stdout"]


def test_binding_response_uses_declared_rows_field_and_preserves_metadata():
    class ReachabilityResponse:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"search-hub":{"reachable":true}}'

    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"corporaSearched":[{"name":"provider"}],"latencyMs":12,"ranked":[{"id":"one"},{"id":"two"}],"rerankerLeg":"direct"}'

    kernel = SessionKernel([
        {
            "id": "search-hub/query/query",
            "scenario": "search-hub",
            "group": "query",
            "command": "query",
            "effect": "read",
            "rows_field": "ranked",
            "meta_fields": ["corporaSearched", "latencyMs", "rerankerLeg"],
        }
    ], session_id="phase-1", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", side_effect=[ReachabilityResponse(), Response()]):
        result = kernel.execute("rows = search_hub.query.query(query='program runtime')\nprint(rows.count())\nprint(rows.meta()['latencyMs'])\nprint(rows.raw()['rerankerLeg'])")
    assert result["ok"]
    assert "2" in result["stdout"]
    assert "12" in result["stdout"]
    assert "direct" in result["stdout"]


def test_binding_response_refuses_ambiguous_repeated_fields_with_candidates():
    class ReachabilityResponse:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"fixture":{"reachable":true}}'

    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"first":[1],"second":[2]}'

    kernel = SessionKernel([
        {
            "id": "fixture/query/run",
            "scenario": "fixture",
            "group": "query",
            "command": "run",
            "effect": "read",
            "row_field_candidates": ["first", "second"],
        }
    ], session_id="phase-1", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", side_effect=[ReachabilityResponse(), Response()]):
        result = kernel.execute("fixture.query.run()")
    assert not result["ok"]
    assert "fixture/query/run" in result["error"]
    assert "first, second" in result["error"]


def test_binding_response_accepts_explicit_rows_projection():
    class ReachabilityResponse:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"fixture":{"reachable":true}}'

    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"first":[1],"second":[2],"metadata":"kept"}'

    kernel = SessionKernel([
        {"id": "fixture/query/run", "scenario": "fixture", "group": "query", "command": "run", "row_field_candidates": ["first", "second"]}
    ], session_id="phase-1", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", side_effect=[ReachabilityResponse(), Response()]):
        result = kernel.execute("rows = fixture.query.run(rows='second')\nprint(rows.count())\nprint(rows.raw()['metadata'])")
    assert result["ok"]
    assert "1" in result["stdout"]
    assert "kept" in result["stdout"]


def test_projection_verb_uses_operation_rows_and_preserves_metadata_and_raw():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'''{
                "verb":"recall",
                "binding_id":"search-hub/query/query",
                "owner":"search-hub",
                "rows_field":"ranked",
                "result":{
                    "ranked":[{"kind":"record","id":"one"},{"kind":"doc","id":"two"}],
                    "latencyMs":12,
                    "partial":false
                }
            }'''

    kernel = SessionKernel([], session_id="phase-2", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    source = """
rows = recall("retention")
print(rows.count())
print(rows.group_by("kind"))
print(rows.meta()["binding_id"])
print(rows.meta()["latencyMs"])
print(rows.raw()["partial"])
"""
    with patch("host.engine.urllib.request.urlopen", return_value=Response()):
        result = kernel.execute(source)
    assert result["ok"]
    assert "2" in result["stdout"]
    assert "'record': 1" in result["stdout"]
    assert "search-hub/query/query" in result["stdout"]
    assert "12" in result["stdout"]
    assert "False" in result["stdout"]


def test_projection_verb_without_repeated_rows_returns_one_response_row():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"verb":"capture","binding_id":"vrooli-memory/journal/note","owner":"vrooli-memory","result":{"entry":{"id":"note-1"}}}'

    kernel = SessionKernel([], session_id="phase-2", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()):
        result = kernel.execute('row = capture("evidence").head(1)[0]\nprint(row["entry"]["id"])')
    assert result["ok"]
    assert "note-1" in result["stdout"]


def test_projection_verb_treats_an_omitted_empty_repeated_field_as_zero_rows():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"verb":"recall","binding_id":"search-hub/query/query","owner":"search-hub","rows_field":"ranked","result":{"degraded":true,"groups":[]}}'

    kernel = SessionKernel([], session_id="phase-3", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()):
        result = kernel.execute('rows = recall(query="retention")\nprint(rows.count())\nprint(rows.meta()["degraded"])')
    assert result["ok"]
    assert result["stdout"].splitlines() == ["0", "True"]


def test_projection_verbs_accept_measured_keyword_vocabulary():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"verb":"recall","binding_id":"search-hub/query/query","owner":"search-hub","rows_field":"ranked","result":{"ranked":[{"id":"one"}]}}'

    kernel = SessionKernel([], session_id="phase-3", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()) as open_url:
        result = kernel.execute('print(recall(query="retention").count())')
    assert result["ok"]
    assert result["stdout"].strip() == "1"
    assert b'"intent": "retention"' in open_url.call_args.args[0].data


def test_guide_returns_prompt_manager_discovery_rows_and_maps_aliases():
    class Response:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"verb":"guide","binding_id":"prompt-manager/discover/discover","owner":"prompt-manager","rows_field":"results","result":{"results":[{"name":"implementation-plan-authoring","type":"skill"}],"total":1}}'

    kernel = SessionKernel([], session_id="phase-guide", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", return_value=Response()) as open_url:
        result = kernel.execute('row = guide(intent="author a plan").head(1)[0]\nprint(row["name"])\nprint(guide(query="find tests").count())')
    assert result["ok"]
    assert "implementation-plan-authoring" in result["stdout"]
    assert result["stdout"].splitlines()[-1] == "1"
    assert b'"task": "find tests"' in open_url.call_args.args[0].data


def test_nested_manifest_command_paths_remain_callable():
    class ReachabilityResponse:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"vrooli":{"reachable":true}}'

    class ExecuteResponse:
        def __enter__(self):
            return self

        def __exit__(self, *_args):
            return False

        def read(self):
            return b'{"bindings":[{"id":"one"}]}'

    kernel = SessionKernel([
        {
            "id": "vrooli/bindings/audit/doctor",
            "scenario": "vrooli",
            "group": "bindings",
            "command": "audit/doctor",
            "rows_field": "bindings",
        }
    ], session_id="nested-manifest", bridge_url="http://bridge/internal/program-runtime/bindings/execute")
    with patch("host.engine.urllib.request.urlopen", side_effect=[ReachabilityResponse(), ExecuteResponse()]):
        result = kernel.execute("print(vrooli.bindings.audit.doctor().count())")
    assert result["ok"]
    assert result["stdout"].strip() == "1"


def test_projection_verbs_reject_alias_collisions_and_report_accepted_keywords():
    kernel = SessionKernel()

    collision = kernel.execute('recall(intent="canonical", query="alias")')
    assert not collision["ok"]
    assert "intent=" in collision["error"] and "query=" in collision["error"]

    unknown = kernel.execute('recall(topic="retention")')
    assert not unknown["ok"]
    assert "topic" in unknown["error"]
    for accepted in ("intent", "query", "depth", "rows"):
        assert accepted in unknown["error"]

    missing = kernel.execute("recall()")
    assert not missing["ok"]
    assert "accepted keywords" in missing["error"]
    assert "intent" in missing["error"] and "query" in missing["error"]


def test_handle_constructor_is_available_to_isolated_programs():
    result = SessionKernel().execute("rows = Handle([{'value': 7}])\nprint(rows.agg('value', 'sum'))")
    assert result["ok"]
    assert result["stdout"].strip() == "7"


def test_filter_and_aggregate_run_in_kernel():  # [REQ:PRT-P0-003]
    kernel = SessionKernel()
    result = kernel.execute("from host.engine import Handle\nrows = Handle([{'kind': 'a'}, {'kind': 'a'}, {'kind': 'b'}])\nprint(rows.filter(lambda x: x['kind'] == 'a').count())\nprint(rows.group_by('kind'))\nprint(rows.group_by('kind').count())")
    assert result["ok"]
    assert "2" in result["stdout"]
    assert "'a': 2" in result["stdout"]
    assert "\n3\n" in result["stdout"]


def test_group_counts_are_ints_with_a_bounded_count_helper():
    result = SessionKernel().execute("rows = Handle([{'kind': 'a'}, {'kind': 'a'}])\ngroups = rows.group_by('kind')\nprint(groups['a'] + 1)\nprint(groups['a'].count())")
    assert result["ok"]
    assert result["stdout"].splitlines() == ["3", "2"]


def test_join_accepts_on_as_an_additive_key_alias():
    result = SessionKernel().execute("left = Handle([{'id': 1}])\nright = Handle([{'id': 1, 'ok': True}])\nprint(left.join(right, on='id').count())")
    assert result["ok"]
    assert result["stdout"].strip() == "1"


def test_join_rejects_key_and_on_together():
    result = SessionKernel().execute("Handle([{'id': 1}]).join(Handle([{'id': 1}]), 'id', on='id')")
    assert not result["ok"]
    assert "key" in result["error"] and "on" in result["error"]


def test_group_by_reports_missing_key_and_available_fields():
    result = SessionKernel().execute("from host.engine import Handle\nHandle([{'failureShape': 'runtimeerror'}]).group_by('missing')")
    assert not result["ok"]
    assert "missing" in result["error"]
    assert "failureShape" in result["error"]


def test_handle_shaping_operations_do_not_materialize_by_default():
    kernel = SessionKernel()
    source = """from host.engine import Handle
left = Handle([{'id': 2, 'kind': 'b', 'value': 20}, {'id': 1, 'kind': 'a', 'value': 10}, {'id': 2, 'kind': 'b', 'value': 20}])
right = Handle([{'id': 1, 'label': 'one'}, {'id': 2, 'label': 'two'}])
mapped = left.map(lambda row: {'id': row['id'], 'value': row['value'] + 1})
print(mapped.select('id', 'value').sort('id').unique('id').head())
print(left.agg('value', 'sum'))
print(left.agg('value', 'mean'))
print(left.join(right, 'id').sort('id')[0]['label'])
print(left[1:].count())
"""
    result = kernel.execute(source)
    assert result["ok"]
    assert "\none\n" in result["stdout"]
    assert "50" in result["stdout"]
    assert "16.666" in result["stdout"]
    assert "2" in result["stdout"]


def test_handle_operations_report_the_missing_key_and_available_fields():
    kernel = SessionKernel()
    result = kernel.execute("from host.engine import Handle\nHandle([{'id': 1, 'name': 'one'}]).sort('missing')")
    assert not result["ok"]
    assert "sort key 'missing' is missing" in result["error"]
    assert "id" in result["error"]
    assert "name" in result["error"]


def test_bounded_text_is_utf8_byte_safe():
    result = SessionKernel().execute("print('🙂' * 8000)")
    assert result["ok"]
    assert result["agent_bytes"] <= result["output_limit_bytes"]
    assert result["stdout"].endswith("…")
    result["stdout"].encode("utf-8").decode("utf-8")


def test_namespace_mirrors_manifest_groups(tmp_path):  # [REQ:PRT-P0-001]
    from bindings.namespace import namespace_from_manifest

    path = tmp_path / "manifest.json"
    path.write_text('{"groups":[{"name":"bindings","commands":[{"name":"list"}]}]}')
    assert namespace_from_manifest(path) == {"bindings": {"list": "list"}}


def test_inference_and_delegation_fail_closed():
    kernel = SessionKernel()
    inference = kernel.execute("ai.classify('hello')")
    delegation = kernel.execute("agent.run('other-scenario')")
    assert not inference["ok"]
    assert "promotion" in inference["error"]
    assert not delegation["ok"]
    assert "explicit governed binding" in delegation["error"]


def test_current_library_facades_are_versioned_and_bounded():
    kernel = SessionKernel(
        libraries=[{"name": "old-tool", "version": 1, "current": False}]
    )
    result = kernel.execute("print(lib.list().materialize())")
    assert result["ok"]
    assert "old-tool" not in result["stdout"]

    retired = kernel.execute("lib.old_tool()")
    assert not retired["ok"]
    assert "not current and promoted" in retired["error"]


def test_library_facade_rejects_ambiguous_arguments():
    kernel = SessionKernel(libraries=[{"name": "fleet-fanout", "version": 1, "current": True}])
    result = kernel.execute("lib.fleet_fanout('unexpected')")
    assert not result["ok"]
    assert "accept named inputs" in result["error"]


def test_promoted_library_source_runs_inside_the_same_governed_surface():
    kernel = SessionKernel(
        libraries=[
            {"name": "probe", "version": 2, "current": True, "source": "result = Handle([{'ok': True}])"},
        ]
    )
    result = kernel.execute("print(lib.probe().head(1))")
    assert result["ok"]
    assert "'ok': True" in result["stdout"]
