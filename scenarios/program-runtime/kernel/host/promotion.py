"""Prepare reviewed portable assets while retaining the adaptive program source."""
import ast
try:
    from .fragments import baseline_artifact, wrap, ReplayBindings
except ImportError:
    from fragments import baseline_artifact, wrap, ReplayBindings


def promote_source(source, step_name, fragment, inputs, expected_output, *, compatibility=None,
                   reviewed_by="", evidence=None, fixtures=None):
    tree = ast.parse(source)
    matches = [node for node in ast.walk(tree) if isinstance(node, ast.Call)
               and isinstance(node.func, ast.Attribute) and isinstance(node.func.value, ast.Name)
               and node.func.value.id == "learn" and node.func.attr == "act" and node.args
               and isinstance(node.args[0], ast.Constant) and node.args[0].value == step_name]
    candidate = {"step_name": step_name, "source": source, "fragment": fragment,
                 "fixture": {"inputs": inputs, "expected": expected_output}, "reason": "review_required"}
    if len(matches) != 1:
        candidate["reason"] = "ambiguous_step"
        return {"promoted": False, "candidate": candidate}
    if not compatibility or not reviewed_by or not evidence:
        return {"promoted": False, "candidate": candidate}
    fixtures = fixtures or [{"inputs": inputs, "expected": expected_output}]
    artifact = baseline_artifact(fragment, compatibility, reviewed_by, evidence, fixtures)
    for fixture in fixtures:
        replay = ReplayBindings(fixture.get("calls", []))
        result = wrap(artifact["fragment"])(fixture["inputs"], replay)
        replay.assert_consumed()
        if result != fixture["expected"]:
            raise ValueError("baseline regression fixture failed")
    candidate.update(baseline=artifact, reason="reviewed_baseline")
    return {"promoted": True, "candidate": candidate}
