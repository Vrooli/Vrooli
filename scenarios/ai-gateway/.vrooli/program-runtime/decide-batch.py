"""Decide a bounded batch of states with a non-generative decision model.

The caller owns the questions. AI Gateway owns the candidate walk (local
decision model, credential-gated hosted models, LLM enum fallback), answer
validation and accounting. This composition makes one batch request and keeps
every item's answers, distribution, method and provider.
"""
import json

TYPES = {"choice": "DECISION_QUESTION_TYPE_CHOICE", "score": "DECISION_QUESTION_TYPE_SCORE",
         "noul": "DECISION_QUESTION_TYPE_NOUL"}
ROLES = ("decide.fast", "decide.accurate")
MAX_STATES, MAX_QUESTIONS, MAX_OPTIONS = 32, 4, 12

envelope = {
    "program": "ai-gateway.decide-batch", "version": "1", "phase": "validate",
    "status": "failed", "inputs": {},
    "signals": {"results": [], "by_choice": {}, "methods": {}, "answered": 0, "requested": 0,
                "usage": None, "candidates": {}},
    "errors": [], "evidence": [],
}


def error(kind, detail, where):
    return {"class": kind, "detail": str(detail)[:160], "where": where}


def text(value, limit, what):
    if not isinstance(value, str) or not value.strip() or len(value.encode("utf-8")) > limit:
        raise ValueError(f"{what} must be a nonempty string of at most {limit} UTF-8 bytes")
    return value


def wire_question(raw):
    """Turn a caller question into the DecisionQuestion proto shape and return
    it with the closed set the answer must fall in."""
    if not isinstance(raw, dict):
        raise ValueError("each question must be an object")
    qid, kind = text(raw.get("id"), 64, "question id"), raw.get("type")
    if kind not in TYPES:
        raise ValueError(f"question {qid}: type must be choice, score or noul")
    question = {"id": qid, "type": TYPES[kind]}
    if raw.get("instructions") is not None:
        question["instructions"] = text(raw["instructions"], 1024, "instructions")
    allowed = None
    if kind == "choice":
        options = raw.get("options")
        if isinstance(options, list):
            options = {label: "" for label in options}
        if not isinstance(options, dict) or not 2 <= len(options) <= MAX_OPTIONS:
            raise ValueError(f"question {qid}: choice needs 2 to {MAX_OPTIONS} options")
        question["options"] = [{"label": text(label, 32, "label"),
                                "description": "" if description is None else text(description, 512, "description")}
                               for label, description in options.items()]
        allowed = list(options)
    elif kind == "score":
        levels = raw.get("levels")
        if not isinstance(levels, list) or not 2 <= len(levels) <= 10:
            raise ValueError(f"question {qid}: score needs 2 to 10 levels")
        question["levels"] = [text(level, 512, "level") for level in levels]
        allowed = [str(level) for level in range(len(levels))]
    else:
        for side, field in (("true", "true_description"), ("false", "false_description")):
            if raw.get(side) is not None:
                question[field] = text(raw[side], 512, side)
    return question, allowed


def validate_inputs():
    states, questions, role = inputs.get("states"), inputs.get("questions"), inputs.get("role", "decide.fast")
    if not isinstance(states, list) or not 1 <= len(states) <= MAX_STATES:
        raise ValueError(f"states must contain 1 to {MAX_STATES} texts")
    for state in states:
        text(state, 16384, "each state")
    if sum(len(state.encode("utf-8")) for state in states) > 131072:
        raise ValueError("states exceed 131072 UTF-8 bytes")
    if not isinstance(questions, list) or not 1 <= len(questions) <= MAX_QUESTIONS:
        raise ValueError(f"questions must contain 1 to {MAX_QUESTIONS} questions")
    wired = [wire_question(question) for question in questions]
    if len({question["id"] for question, _ in wired}) != len(wired):
        raise ValueError("question ids must be distinct")
    if role not in ROLES:
        raise ValueError("role must be decide.fast or decide.accurate")
    # Option labels are listed once here; each answer's probabilities follow
    # this order, which keeps a full batch inside the output budget.
    envelope["inputs"] = {"states": len(states), "role": role,
                          "questions": {question["id"]: allowed for question, allowed in wired}}
    envelope["signals"]["requested"] = len(states)
    return states, wired, role


def number(value, low, high):
    """Protojson renders doubles as numbers and int64 as strings; anything else
    or out of range is not an answer."""
    if isinstance(value, bool) or not isinstance(value, (int, float, str)):
        return None
    try:
        value = float(value)
    except ValueError:
        return None
    return round(value, 4) if low <= value <= high else None


def usage(value):
    if not isinstance(value, dict):
        return None
    result = {}
    for field in ("inputTokens", "outputTokens", "costMicros"):
        count = value.get(field, 0)  # Protojson omits zero counters.
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


def read_answer(question, allowed, answer):
    """Return the answer reduced to its value and distribution, or None when it
    falls outside the declared closed set."""
    kind = question["type"]
    if not isinstance(answer, dict) or answer.get("type") != kind:
        return None
    out = {}
    if kind == TYPES["choice"]:
        if answer.get("choice") not in allowed:
            return None
        out["choice"] = answer["choice"]
    elif kind == TYPES["score"]:
        out["score"] = number(answer.get("score", 0), 0, len(allowed) - 1)
        if out["score"] is None:
            return None
    else:
        out["noul"] = number(answer.get("noul", 0), 0, 1)
        if out["noul"] is None:
            return None
    if "confidence" in answer:
        out["confidence"] = number(answer["confidence"], 0, 1)
    probabilities = {}
    for row in answer.get("probabilities") or []:
        value = number(row.get("probability", 0), 0, 1) if isinstance(row, dict) else None
        if value is None or not isinstance(row.get("option"), str):
            return None
        probabilities[row["option"]] = value
    if probabilities:
        # A distribution must cover exactly the declared options; it is kept
        # as a list in the order envelope.inputs.questions lists them.
        if allowed is None or set(probabilities) != set(allowed):
            return None
        out["probabilities"] = [probabilities[option] for option in allowed]
    return out


def decide(states, wired, role):
    envelope["phase"] = "decide"
    try:
        handle = ai_gateway.inference.decide_batch(
            role=role, questions=[question for question, _ in wired], items=[{"state": state} for state in states])
        rows = handle.head(len(states) + 1)
        meta = handle.meta()
    except Exception as exc:
        envelope["status"], kind = program.classify(exc)
        envelope["errors"].append(error(kind, exc, "decide"))
        return
    signals = envelope["signals"]
    signals["usage"] = usage(meta.get("usage") if isinstance(meta, dict) else None)
    if len(rows) > len(states):
        envelope["errors"].append(error("invalid_response", "batch returned more results than states", "decide"))
        return
    for index in range(len(states)):
        row = rows[index] if index < len(rows) else None
        item = {"index": index, "answers": None, "method": None, "provider": None, "model": None,
                "locality": None, "latency_ms": None, "error": None}
        kind = "missing_result" if index >= len(rows) else "invalid_result"
        if isinstance(row, dict) and row.get("error"):
            failure = row["error"]
            item["error"] = {"class": "decision_failed",
                             "code": str(failure.get("code", "unknown"))[:80] if isinstance(failure, dict) else "unknown"}
            kind = "decision_failed"
        elif isinstance(row, dict):
            for field in ("provider", "model", "locality", "method"):
                value = row.get(field)
                item[field] = value.encode("utf-8")[:120].decode("utf-8", errors="ignore") if isinstance(value, str) else None
            latency = number(row.get("latencyMs", 0), 0, 9.0e15)
            item["latency_ms"] = None if latency is None else int(latency)
            by_id = {answer.get("questionId"): answer for answer in row.get("answers") or [] if isinstance(answer, dict)}
            answers = {question["id"]: read_answer(question, allowed, by_id.get(question["id"])) for question, allowed in wired}
            if all(value is not None for value in answers.values()):
                item["answers"] = answers
            if index == 0:
                for candidate in row.get("candidates") or []:
                    if isinstance(candidate, dict):
                        name = str(candidate.get("provider", ""))[:40] + "/" + str(candidate.get("resourceRole", ""))[:60]
                        signals["candidates"][name] = str(candidate.get("reason") or candidate.get("status", ""))[:60]
        if item["answers"] is None:
            item["error"] = item["error"] or {"class": kind}
            envelope["errors"].append(error(item["error"]["class"], "item has no valid answer set", "decide:" + str(index)))
        else:
            signals["answered"] += 1
            method = item["method"] or "unknown"
            signals["methods"][method] = signals["methods"].get(method, 0) + 1
            for question, _ in wired:
                choice = item["answers"][question["id"]].get("choice")
                if choice is not None:
                    counts = signals["by_choice"].setdefault(question["id"], {})
                    counts[choice] = counts.get(choice, 0) + 1
        signals["results"].append(item)
    envelope["evidence"].append({"binding": "ai-gateway/inference/decide-batch", "requests": 1,
                                 "items": len(states), "role": role})
    answered = signals["answered"]
    envelope["status"] = "ok" if answered == len(states) else "partial" if answered else "failed"


try:
    states, wired, role = validate_inputs()
    decide(states, wired, role)
except Exception as exc:
    envelope["status"] = "failed"
    envelope["errors"].append(error("invalid_input" if envelope["phase"] == "validate" else "kernel_runtime", exc, envelope["phase"]))
finally:
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
