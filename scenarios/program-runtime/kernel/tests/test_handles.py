import sys
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


def test_handle_constructor_is_available_to_isolated_programs():
    result = SessionKernel().execute("rows = Handle([{'value': 7}])\nprint(rows.agg('value', 'sum'))")
    assert result["ok"]
    assert result["stdout"].strip() == "7"


def test_filter_and_aggregate_run_in_kernel():  # [REQ:PRT-P0-003]
    kernel = SessionKernel()
    result = kernel.execute("from host.engine import Handle\nrows = Handle([{'kind': 'a'}, {'kind': 'a'}, {'kind': 'b'}])\nprint(rows.filter(lambda x: x['kind'] == 'a').count())\nprint(rows.group_by('kind'))")
    assert result["ok"]
    assert "2" in result["stdout"]
    assert "'a': 2" in result["stdout"]


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
    inference = kernel.execute("vrooli.ai.classify('hello')")
    delegation = kernel.execute("vrooli.agent.run('other-scenario')")
    assert not inference["ok"]
    assert "promotion" in inference["error"]
    assert not delegation["ok"]
    assert "explicit governed binding" in delegation["error"]
