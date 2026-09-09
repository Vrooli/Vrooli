"""program-runtime.learn-verbs-example — executable documentation for the ten learn verbs."""
import json

inputs = program.inputs()
value = str(inputs.get("value", "fixture-value"))
mode = inputs.get("mode", "baseline")
workflow_key = str(inputs.get("workflow_key", "agent-manager/investigate-typed"))
identity = learn.task(
    scope="program-runtime-usage",
    operation="program-runtime.learn-verbs-example",
    key={"fixture": "learn-verbs-example"},
)

envelope = {
    "program": "program-runtime.learn-verbs-example",
    "version": "1",
    "status": "partial",
    "phase": "run",
    "inputs": {"value": value, "workflow_key": workflow_key},
    "signals": {"identity": identity},
    "errors": [],
    "evidence": [],
}

with learn.step("recall-and-choose") as choice_step:
    recalled = learn.recall(query="preference/v1 option_id safe", kinds=["preference"], limit=3)
    choice = learn.choose(["safe", "fast"], "safe")
    choice_step.note("trace", {"bindings": [], "output": choice}, evidence=["choice:fixture"])

inferred = {"status": "not_requested"}
if mode == "all":
    try:
        inferred = learn.infer("label", "Classify this fixture as safe.", {"value": value},
                               {"type": "string", "enum": ["safe", "unsafe"]},
                               verify=lambda value: ("verified_success", ["label:fixture-safe"]) if value == "safe" else ("failed", []))
    except Exception as exc:
        inferred = {"status": "unavailable"}
        envelope["errors"].append({"class": "inference_failed", "detail": str(exc)[:160]})

with learn.step("find-account"):
    acted = learn.act(
        "find-account",
        "Return the input value in an object with the key value.",
        {"value": value},
        {"type": "object", "properties": {"value": {"type": "string"}}, "required": ["value"]},
        [],
        attempts=3,
        verify=lambda output: ("verified_success", ["value:exact-match"]) if output.get("value") == value else ("failed", ["value:mismatch"]),
        verifier_revision="value-equality/v1",
        baseline={} if mode == "adapt" else None,
        allow_ai=mode != "baseline",
    )

delegated = {"status": "not_requested"}
if mode == "all":
    try:
        delegated = learn.delegate(
            "delegate-check",
            "Return a bounded structured confirmation for this fixture.",
            {"value": value, "workflow_key": workflow_key},
            {"type": "object"},
            ["agent-manager/run/start"],
            capabilities={"code": False, "files": False, "web": False},
            verify=lambda result: ("verified_success", ["delegate:verified"]) if isinstance(result, dict) else ("failed", []),
            wait_seconds=5,
        )
        envelope["status"] = "ok"
    except Exception as exc:
        delegated = {"status": "unavailable", "error": str(exc)[:160]}
        envelope["errors"].append({"class": "delegate_unavailable", "detail": str(exc)[:160]})

if choice["selected_id"]:
    learn.note("preference", {"option_id": choice["selected_id"]}, evidence=["choice:fixture"])
# A later run grades an earlier run's recommendation by attempt id. Memory only accepts
# feedback for an attempt it already holds in this scope, so the example grades the most
# recent recalled attempt when one exists and records not_applicable on the first run.
prior = [row.get("attempt_id") or row.get("attemptId") for row in recalled.get("attempts", []) if isinstance(row, dict)]
prior = [item for item in prior if item]
if prior and mode == "all":
    graded = learn.feedback(prior[-1], "supported", ["learn-verbs-example:verified"])
else:
    graded = {"delivery": "not_applicable", "observation_id": ""}
assert acted["output"] == {"value": value}
learn.outcome("verified_success", ["learn-verbs-example:verified"], measurements={"reused_workflow": False})
envelope["signals"].update({
    "mode": mode,
    "choice": {"selected_id": choice["selected_id"], "source": choice["source"],
                "why": choice.get("why", [])[:3],
                "derived": any(item.get("derived") for item in choice.get("learning", {}).get("advice", []))},
    "recalled": {"status": recalled.get("recall_status"), "hits": len(recalled.get("hits", [])),
                 "preview": [{"kind": hit.get("kind", ""), "text": hit.get("text", "")[:160]} for hit in recalled.get("hits", [])[:2]]},
    "inferred": inferred,
    "acted": acted,
    "delegated": delegated,
    "graded": {"delivery": graded.get("delivery"), "observation_id": graded.get("observation_id")},
})
envelope["evidence"].append("all-learn-verbs")
envelope["status"] = "partial" if envelope["errors"] else "ok"
envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True, separators=(",", ":")))
