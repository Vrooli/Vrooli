"""Attach later user or downstream evidence to an opaque result reference."""
import hashlib
import json

inputs = program.inputs()
envelope = program.envelope("program-runtime.learning-feedback", "1")
try:
    reference = inputs["feedback_ref"]
    learn.task(operation="program-runtime.learning-feedback",
               key={"result": hashlib.sha256(reference.encode()).hexdigest()})
    receipt = learn.feedback(reference, inputs["disposition"], inputs["evidence"],
        correction=inputs.get("correction", ""), dimension=inputs.get("dimension", "usefulness"))
    # The correction itself is gradeable: a later run can contradict a wrong correction.
    envelope["signals"] = {"receipt": receipt, "learning": {"feedback_ref": learn.result("correction")}}
    envelope["status"] = {"delivered": "ok", "rejected": "refused"}.get(receipt.get("delivery"), "partial")
    envelope["evidence"] = inputs["evidence"][:16]
    # Recording a correction does not re-verify the original domain task.
    learn.outcome("unknown", [])
except Exception as exc:
    status, kind = program.classify(exc)
    envelope["status"] = status
    envelope["errors"] = [{"class": kind, "detail": str(exc)[:240]}]
envelope["phase"] = "report"
print(json.dumps(envelope, sort_keys=True))
