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
