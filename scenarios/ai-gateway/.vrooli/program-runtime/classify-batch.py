"""Classify a bounded corpus, retaining each item's validation and usage evidence.

The caller owns the instruction and labels. AI Gateway owns provider selection,
schema validation and bounded repair. This composition makes one batch request.
"""
import json

envelope = {
    "program": "ai-gateway.classify-batch", "version": "1", "phase": "validate",
    "status": "failed", "inputs": {},
    "signals": {"results": [], "labels": [], "by_label": {}, "validated": 0,
                "usage": None, "requested": 0},
    "errors": [], "evidence": [],
}


def error(kind, detail, where):
    return {"class": kind, "detail": str(detail)[:160], "where": where}


def transport_failure(exc):
    text = str(exc)
    for needle, status, kind in (
        ("requires an explicit grant", "refused", "no_grant"),
        ("not run eligible", "refused", "not_run_eligible"),
        ("inference spend", "refused", "inference_spend_exceeded"),
        ("inference_spend_exceeded", "refused", "inference_spend_exceeded"),
        ("is unreachable", "unavailable", "scenario_unreachable"),
        ("bridge unavailable", "unavailable", "scenario_unreachable"),
        ("scenario_not_running", "unavailable", "scenario_unreachable"),
    ):
        if needle in text:
            return status, kind
    return "failed", "kernel_runtime" if isinstance(exc, (NameError, AttributeError)) else "binding_error"


def usage(value):
    if not isinstance(value, dict):
        return None
    result = {}
    for field in ("inputTokens", "outputTokens", "costMicros"):
        count = value.get(field, 0)  # Protojson omits zero counters in an available Usage.
        if isinstance(count, bool) or not isinstance(count, (int, str)):
            return None
        try:
            count = int(count)
        except ValueError:
            return None
        if count < 0 or count > 9223372036854775807:
            return None
        result[field] = count
    return result


def validate_inputs():
    corpus, labels, instruction = inputs.get("corpus"), inputs.get("labels"), inputs.get("instruction")
    if not isinstance(corpus, list) or not 1 <= len(corpus) <= 32:
        raise ValueError("corpus must contain 1 to 32 texts")
    if any(not isinstance(text, str) or not text.strip() or len(text.encode("utf-8")) > 16384 for text in corpus):
        raise ValueError("each text must be nonempty and at most 16384 UTF-8 bytes")
    if sum(len(text.encode("utf-8")) for text in corpus) > 65536:
        raise ValueError("corpus exceeds 65536 UTF-8 bytes")
    if not isinstance(labels, list) or not 1 <= len(labels) <= 32:
        raise ValueError("labels must contain 1 to 32 distinct strings")
    if any(not isinstance(label, str) or not label.strip() or len(label.encode("utf-8")) > 64 for label in labels):
        raise ValueError("each label must be nonempty and at most 64 UTF-8 bytes")
    if len(set(labels)) != len(labels):
        raise ValueError("labels must be distinct")
    if not isinstance(instruction, str) or not instruction.strip() or len(instruction.encode("utf-8")) > 4096:
        raise ValueError("instruction must be nonempty and at most 4096 UTF-8 bytes")
    envelope["inputs"] = {"documents": len(corpus), "labels": labels}
    envelope["signals"]["requested"] = len(corpus)
    return corpus, labels, instruction


def classify(corpus, labels, instruction):
    envelope["phase"] = "classify"
    try:
        handle = ai.batch(corpus, {"type": "string", "enum": labels}, instruction, role="classify.fast")
        batch = handle.head(1)
    except Exception as exc:
        envelope["status"], kind = transport_failure(exc)
        envelope["errors"].append(error(kind, exc, "classify"))
        return
    if len(batch) != 1 or not isinstance(batch[0], dict):
        envelope["errors"].append(error("invalid_response", "batch response must be one object", "classify"))
        return
    batch = batch[0]
    signals = envelope["signals"]
    signals["usage"] = usage(batch.get("usage"))
    rows = batch.get("results")
    if not isinstance(rows, list) or len(rows) > len(corpus):
        envelope["errors"].append(error("invalid_response", "batch result count exceeds request or results is not an array", "classify"))
        return
    for index in range(len(corpus)):
        row = rows[index] if index < len(rows) else None
        item = {"index": index, "label": None, "validated": False, "usage": None,
                "provider": None, "model": None, "error": None}
        kind = "missing_result" if index >= len(rows) else "invalid_result"
        if isinstance(row, dict):
            item["usage"] = usage(row.get("usage"))
            for field in ("provider", "model"):
                value = row.get(field)
                item[field] = value.encode("utf-8")[:120].decode("utf-8", errors="ignore") if isinstance(value, str) else None
            if row.get("error"):
                owner_error = row["error"]
                item["error"] = {"class": "inference_failed", "code": str(owner_error.get("code", "unknown"))[:80]
                                 if isinstance(owner_error, dict) else "unknown"}
                kind = "inference_failed"
            elif row.get("validated") is True:
                try:
                    label = json.loads(row.get("valueJson", ""))
                except (TypeError, ValueError):
                    label = None
                if isinstance(label, str) and label in labels:
                    item["label"], item["validated"] = label, True
                    signals["validated"] += 1
                    signals["by_label"][label] = signals["by_label"].get(label, 0) + 1
        if not item["validated"]:
            item["error"] = item["error"] or {"class": kind}
            envelope["errors"].append(error(kind, "item has no validated label", "classify:" + str(index)))
        signals["results"].append(item)
        signals["labels"].append(item["label"])
    envelope["evidence"].append({"binding": "ai-gateway/inference/run-batch", "requests": 1,
                                 "items": len(corpus), "role": "classify.fast"})
    envelope["status"] = "ok" if signals["validated"] == len(corpus) else "partial" if signals["validated"] else "failed"


try:
    corpus, labels, instruction = validate_inputs()
    classify(corpus, labels, instruction)
except Exception as exc:
    envelope["status"] = "failed"
    envelope["errors"].append(error("invalid_input" if envelope["phase"] == "validate" else "kernel_runtime", exc, envelope["phase"]))
finally:
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
