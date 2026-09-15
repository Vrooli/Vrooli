"""Classify the runtime example corpus through AI Gateway's shared workflow."""
import json

envelope = {"program": "program-runtime.batch-inference", "version": "2", "status": "failed",
            "phase": "validate", "inputs": {}, "signals": {}, "errors": [], "evidence": []}
try:
    corpus = inputs.get("corpus", ['The provider timed out during a retry.', 'The user supplied an invalid request field.', 'The deployment lost its database connection.'])
    labels = inputs.get("labels", ["infra", "user"])
    if not isinstance(corpus, list) or not 1 <= len(corpus) <= 32:
        raise ValueError("corpus must contain 1 to 32 texts")
    envelope["inputs"] = {"documents": len(corpus), "labels": labels}
    envelope["phase"] = "classify"
    child = lib.ai_gateway.classify_batch(
        corpus=corpus, labels=labels,
        instruction=inputs.get("instruction", "Choose the primary failure class."))
    result = child.head(1)[0]
    envelope["status"] = result["status"]
    envelope["signals"] = result["signals"]
    envelope["errors"] = result["errors"]
    envelope["evidence"] = result["evidence"] + [{"program": "ai-gateway.classify-batch", "artifact": child.meta()}]
except Exception as exc:
    envelope["status"] = "failed"
    envelope["errors"].append({"class": "invalid_input" if envelope["phase"] == "validate" else "kernel_runtime",
                               "detail": str(exc)[:160], "where": envelope["phase"]})
finally:
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
