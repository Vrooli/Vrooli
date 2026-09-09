import sys
import contextvars
from pathlib import Path
from types import SimpleNamespace

import pytest

sys.path.insert(0, str(Path(__file__).parents[1]))
from host.engine import Handle
from host.promotion import promote_source
from host.tasks import Tasks

SOURCE = 'result = learn.act("find-account", "find", inputs, schema, [])\n'
FRAGMENT = "def step(inputs, bindings):\n    return {'account': inputs['id']}"


def test_publication_retains_learning_and_checks_reviewed_fixtures():
    """[REQ:LV-11] Reviewed assets retain adaptive source and independently expected outputs."""
    result = promote_source(SOURCE, "find-account", FRAGMENT, {"id": "42"}, {"account": "42"},
        compatibility={"verifier_revision": "v1"}, reviewed_by="operator", evidence=["fixture:account"])
    assert result["promoted"]
    assert result["candidate"]["source"] == SOURCE
    baseline = result["candidate"]["baseline"]
    assert baseline["fixtures"][0]["expected"] == {"account": "42"}
    assert "digest" in baseline and "verified" not in baseline
    with pytest.raises(ValueError, match="regression fixture failed"):
        promote_source(SOURCE, "find-account", FRAGMENT, {"id": "wrong"}, {"account": "42"},
            compatibility={"verifier_revision": "v1"}, reviewed_by="operator", evidence=["fixture:account"])


def test_unreviewed_and_ambiguous_steps_remain_candidates():
    assert not promote_source(SOURCE, "find-account", FRAGMENT, {}, {})["promoted"]
    result = promote_source(SOURCE + SOURCE, "find-account", FRAGMENT, {}, {})
    assert not result["promoted"] and result["candidate"]["reason"] == "ambiguous_step"


def test_publication_uses_recorded_tools_without_live_effects():
    fragment = "def step(inputs, bindings):\n    return bindings.demo.read.run(id=inputs['id']).head(1)[0]"
    result = promote_source(SOURCE, "find-account", fragment, {}, None,
        compatibility={"verifier_revision": "v1"}, reviewed_by="operator", evidence=["fixture:recorded"],
        fixtures=[{"inputs": {"id": "42"}, "expected": {"account": "42"}, "calls": [
            {"binding_id": "demo/read/run", "arguments": {"id": "42"}, "rows": [{"account": "42"}]}]}])
    assert result["promoted"]


def test_publication_rejects_machine_specific_artifacts():
    with pytest.raises(ValueError, match="private or machine-specific"):
        promote_source(SOURCE, "find-account", FRAGMENT, {"id": "/home/alice/private"}, {"account": "/home/alice/private"},
            compatibility={"verifier_revision": "v1"}, reviewed_by="operator", evidence=["fixture:account"])


def test_tasks_publication_requires_explicit_publish_and_preserves_source(monkeypatch):
    calls = []
    task = Tasks(SimpleNamespace(_bridge_url="fixture"), contextvars.ContextVar("publication", default={}), Handle, 1)
    def bridge(action, **request):
        calls.append((action, request))
        if action == "fragment_get":
            return {"found": True, "fragment": {"verified": 5, "step_name": "find-account", "fragment": FRAGMENT,
                "compatibility": {"verifier_revision": "v1"}}}
        return {"published": True}
    monkeypatch.setattr(task, "_bridge", bridge)
    monkeypatch.setattr(task, "_spec", lambda *args: {"source": SOURCE, "digest": "expected", "declaration": {}})
    request = dict(step_key="fixture", operation="demo.program", reviewed_by="operator", evidence=["fixture:account"],
                   fixtures=[{"inputs": {"id": "42"}, "expected": {"account": "42"}}])
    candidate = task.fragment_promote(**request).head(1)[0]
    assert candidate["candidate"]["source"] == SOURCE
    assert [action for action, _ in calls] == ["fragment_get"]
    result = task.fragment_promote(**request, publish=True).head(1)[0]
    assert result["publication"]["published"]
    assert calls[-1][0] == "fragment_publish" and calls[-1][1]["digest"] == "expected"
