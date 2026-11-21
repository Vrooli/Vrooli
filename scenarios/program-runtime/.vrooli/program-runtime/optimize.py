"""program-runtime.optimize v1 — submit once, wait once, and report the durable owner record.

The owner engine pins the immutable request and owns every optimization effect.
This entrypoint never polls, calls a local CLI, or reimplements target policy.
"""

import hashlib
import json
import math

inputs = program.inputs()
request = inputs.get("request")
request_json = ""
raw_wait_ms = inputs.get("wait_ms", 30000)
wait_ms_valid = isinstance(raw_wait_ms, int) and not isinstance(raw_wait_ms, bool)
wait_ms = raw_wait_ms if wait_ms_valid else 0
pinned_request = None

envelope = {
    "program": "program-runtime.optimize",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {"wait_ms": wait_ms},
    "signals": {
        "run_id": "",
        "state": "unknown",
        "stage": "",
        "revision": 0,
        "winner_id": "",
        "failure": "",
        "last_error": "",
        "retry_at": "",
        "stage_attempts": {},
        "candidate_artifact_digests": {},
        "candidate_results": {},
        "promoted_artifact_digest": "",
        "removal_receipts": {},
        "receipts": [],
    },
    "errors": [],
    "evidence": [],
}


def fail(status, klass, detail, phase):
    envelope["status"] = status
    envelope["phase"] = phase
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": phase})


def parse_record(record_json):
    if not isinstance(record_json, str) or len(record_json) > 65536:
        raise ValueError("owner record_json must be a string no larger than 65536 bytes")
    value = json.loads(record_json)
    if not isinstance(value, dict):
        raise ValueError("owner record_json must encode an object")
    return value


def case_digest(case_ids):
    encoded = json.dumps(case_ids, separators=(",", ":"), ensure_ascii=False)
    encoded = encoded.replace("&", "\\u0026").replace("<", "\\u003c").replace(">", "\\u003e")
    encoded = encoded.replace("\u2028", "\\u2028").replace("\u2029", "\\u2029")
    return hashlib.sha256(encoded.encode("utf-8")).hexdigest()


def utf8_size(value):
    return len(value.encode("utf-8"))


def summarize_owner_receipts(request, record):
    evidence = record.get("evidence") if isinstance(record.get("evidence"), list) else []
    by_stage = {}
    for item in evidence:
        if isinstance(item, dict):
            by_stage[str(item.get("stage", "") or "")] = item
    candidates = request.get("candidates", []) if isinstance(request.get("candidates"), list) else []
    candidate_ids = {item["id"] for item in candidates if isinstance(item, dict) and item.get("id")}
    expected_digests = {item["id"]: item["artifact_digest"] for item in candidates if isinstance(item, dict) and item.get("id") and item.get("artifact_digest")}
    summary = {"candidate_artifact_digests": {}, "candidate_results": {}, "promoted_artifact_digest": "", "removal_receipts": {}}
    errors = []

    acquisition = by_stage.get("acquire", {}).get("details", {})
    acquired = acquisition.get("candidate_artifact_digests") if isinstance(acquisition, dict) else None
    if isinstance(acquired, dict):
        summary["candidate_artifact_digests"] = {str(key): value for key, value in acquired.items() if isinstance(value, str)}

    evaluation = by_stage.get("evaluate", {}).get("details", {})
    results = evaluation.get("candidate_results") if isinstance(evaluation, dict) else None
    if isinstance(results, dict):
        summary["candidate_results"] = results

    promotion = by_stage.get("promote", {}).get("details", {})
    promoted_digest = promotion.get("artifact_digest") if isinstance(promotion, dict) else None
    if isinstance(promoted_digest, str):
        summary["promoted_artifact_digest"] = promoted_digest

    cleanup = by_stage.get("cleanup", {}).get("details", {})
    removed = cleanup.get("removal_receipts") if isinstance(cleanup, dict) else None
    if isinstance(removed, dict):
        summary["removal_receipts"] = {str(key): value for key, value in removed.items() if isinstance(value, str)}

    stage = str(record.get("stage", "") or "")
    expected_stages = {
        "preflight_disk": ("preflight_rights",),
        "acquire": ("preflight_rights", "preflight_disk"),
        "evaluate": ("preflight_rights", "preflight_disk", "acquire"),
        "rights": ("preflight_rights", "preflight_disk", "acquire", "evaluate"),
        "disk": ("preflight_rights", "preflight_disk", "acquire", "evaluate", "rights"),
        "promote": ("preflight_rights", "preflight_disk", "acquire", "evaluate", "rights", "disk"),
        "cleanup": ("preflight_rights", "preflight_disk", "acquire", "evaluate", "rights", "disk", "promote"),
        "complete": ("preflight_rights", "preflight_disk", "acquire", "evaluate", "rights", "disk", "promote", "cleanup"),
    }.get(stage, ())
    if stage != "refused":
        for required_stage in expected_stages:
            if required_stage not in by_stage:
                errors.append("durable record stage " + stage + " is missing " + required_stage + " receipt")
        if "acquire" in expected_stages:
            acquired = summary["candidate_artifact_digests"]
            if set(acquired) != candidate_ids or any(not isinstance(value, str) or not value.strip() for value in acquired.values()):
                errors.append("acquisition candidate_artifact_digests must identify every candidate with a non-empty owner digest")
            elif any(acquired.get(candidate_id) != digest for candidate_id, digest in expected_digests.items()):
                errors.append("acquisition candidate_artifact_digests do not match supplied digest expectations")
        if "evaluate" in expected_stages:
            expected_digest = case_digest(request.get("case_ids", []))
            count = evaluation.get("case_count") if isinstance(evaluation, dict) else 0
            required_case_count = max(int(request.get("minimum_cases", 1)), len(request.get("case_ids", [])))
            if not isinstance(count, int) or count < required_case_count or evaluation.get("suite_id") != request.get("suite_id") or evaluation.get("case_digest") != expected_digest:
                errors.append("evaluation receipt does not match the pinned suite and case denominator")
            if not isinstance(results, dict) or set(results) != candidate_ids:
                errors.append("evaluation candidate_results does not cover every pinned candidate")
            else:
                for candidate_id in candidate_ids:
                    result = results.get(candidate_id)
                    if not isinstance(result, dict):
                        errors.append("evaluation candidate_results is malformed for " + candidate_id)
                        continue
                    result_count = result.get("case_count")
                    score = result.get("score")
                    if not isinstance(result_count, int) or result_count < count or result.get("case_digest") != expected_digest or isinstance(score, bool) or not isinstance(score, (int, float)) or not math.isfinite(score):
                        errors.append("evaluation candidate result is incomplete for " + candidate_id)
        if "promote" in expected_stages:
            winner_id = str(promotion.get("winner_id", "") or "") if isinstance(promotion, dict) else ""
            if winner_id not in candidate_ids or summary["promoted_artifact_digest"] != summary["candidate_artifact_digests"].get(winner_id):
                errors.append("promotion receipt does not name the exact pinned winner artifact digest")
        if "cleanup" in expected_stages:
            winner_id = str(promotion.get("winner_id", "") or "") if isinstance(promotion, dict) else ""
            expected_removed = {candidate_id for candidate_id in candidate_ids if candidate_id != winner_id}
            predecessor_id = str(request.get("predecessor_id", "") or "")
            if predecessor_id and predecessor_id != winner_id:
                expected_removed.add(predecessor_id)
            if set(summary["removal_receipts"]) != expected_removed or any(not receipt.strip() for receipt in summary["removal_receipts"].values()):
                errors.append("cleanup removal_receipts do not cover every losing candidate and predecessor")
    return summary, errors


def add_record(request, record, terminal):
    record_request = record.get("request") if isinstance(record.get("request"), dict) else {}
    run_id = str(record_request.get("run_id", "") or "").strip()
    stage = str(record.get("stage", "") or "")
    evidence = record.get("evidence") if isinstance(record.get("evidence"), list) else []
    receipts = []
    for item in evidence[:32]:
        if not isinstance(item, dict):
            continue
        stage_name = str(item.get("stage", "") or "")
        receipt_id = str(item.get("receipt_id", "") or "")
        receipts.append({"stage": stage_name, "receipt_id": receipt_id})
        if receipt_id:
            envelope["evidence"].append(receipt_id)
    receipt_summary, receipt_errors = summarize_owner_receipts(request, record)
    envelope["signals"].update({
        "run_id": run_id,
        "state": "terminal" if terminal else "pending",
        "stage": stage,
        "revision": int(record.get("revision", 0) or 0),
        "winner_id": str(record.get("winner_id", "") or ""),
        "failure": str(record.get("failure", "") or ""),
        "last_error": str(record.get("last_error", "") or ""),
        "retry_at": str(record.get("retry_at", "") or ""),
        "stage_attempts": record.get("stage_attempts", {}) if isinstance(record.get("stage_attempts"), dict) else {},
        **receipt_summary,
        "receipts": receipts,
    })
    if receipt_errors and stage != "refused":
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "binding_error", "detail": "; ".join(receipt_errors)[:240], "where": "report"})
    elif stage == "complete":
        envelope["status"] = "ok"
    elif stage == "refused":
        envelope["status"] = "refused"
    else:
        envelope["status"] = "partial"


def classified_failure(exc, phase):
    try:
        status, klass = program.classify(exc)
    except Exception:
        status, klass = "failed", "kernel_runtime"
    if status not in ("unavailable", "failed", "refused"):
        status = "failed"
    if not klass:
        klass = "remote_error"
    fail(status, klass, exc, phase)


def deferred_wait_failure(exc):
    try:
        _, klass = program.classify(exc)
    except Exception:
        klass = "remote_error"
    if klass not in ("binding_error", "scenario_unreachable", "no_grant", "not_run_eligible", "deadline_exceeded", "remote_error", "kernel_runtime"):
        klass = "remote_error"
    envelope["status"] = "partial"
    envelope["phase"] = "report"
    envelope["errors"].append({"class": klass, "detail": str(exc)[:240], "where": "collect"})


def step_validate():
    global pinned_request, request_json
    if not wait_ms_valid or wait_ms < 1 or wait_ms > 60000:
        fail("failed", "invalid_input", "wait_ms must be between 1 and 60000", "validate")
        return False
    if not isinstance(request, dict):
        fail("failed", "invalid_input", "request must be an optimization.Request object", "validate")
        return False
    try:
        request_json = json.dumps(request, separators=(",", ":"), ensure_ascii=False, allow_nan=False)
    except Exception as exc:
        fail("failed", "invalid_input", exc, "validate")
        return False
    if len(request_json.encode("utf-8")) > 65536:
        fail("failed", "invalid_input", "request must serialize to no more than 65536 UTF-8 bytes", "validate")
        return False
    allowed = {"run_id", "target", "subject", "candidates", "predecessor_id", "suite_id", "case_ids", "minimum_cases"}
    unknown = sorted(set(request) - allowed)
    if unknown:
        fail("failed", "invalid_input", "unsupported optimization.Request key(s): " + ", ".join(unknown[:8]), "validate")
        return False
    run_id_value = request.get("run_id")
    target_value = request.get("target")
    run_id = run_id_value.strip() if isinstance(run_id_value, str) else ""
    target = target_value.strip() if isinstance(target_value, str) else ""
    subject = request.get("subject")
    candidates = request.get("candidates")
    suite_value = request.get("suite_id")
    suite_id = suite_value.strip() if isinstance(suite_value, str) else ""
    case_ids = request.get("case_ids")
    minimum_cases = request.get("minimum_cases")
    valid_target = (
        bool(target)
        and len(target) <= 80
        and target.isascii()
        and target[0].islower()
        and all(char.islower() or char.isdigit() or char == "-" for char in target)
    )
    if not run_id:
        fail("failed", "invalid_input", "run_id is required", "validate")
        return False
    if utf8_size(run_id) > 128:
        fail("failed", "invalid_input", "run_id must contain no more than 128 characters", "validate")
        return False
    if not valid_target:
        fail("failed", "invalid_input", "target must be a lower-case owner target id no longer than 80 characters", "validate")
        return False
    if run_id_value != run_id or target_value != target:
        fail("failed", "invalid_input", "run_id and target must not contain leading or trailing whitespace", "validate")
        return False
    if not isinstance(subject, dict) or not subject or not isinstance(candidates, list) or not candidates:
        fail("failed", "invalid_input", "subject must be an object and candidates must be a non-empty array", "validate")
        return False
    if len(subject) > 32:
        fail("failed", "invalid_input", "subject must contain no more than 32 fields", "validate")
        return False
    if len(candidates) > 16:
        fail("failed", "invalid_input", "candidates must contain no more than 16 items", "validate")
        return False
    if not isinstance(suite_value, str) or not suite_id or suite_value != suite_id or utf8_size(suite_id) > 256 or not isinstance(case_ids, list) or not isinstance(minimum_cases, int) or isinstance(minimum_cases, bool):
        fail("failed", "invalid_input", "suite_id, case_ids, and integer minimum_cases are required", "validate")
        return False
    if minimum_cases < 1 or len(case_ids) < minimum_cases:
        fail("failed", "invalid_input", "minimum_cases must be positive and no greater than the pinned case_ids count", "validate")
        return False
    if len(case_ids) > 2000:
        fail("failed", "invalid_input", "case_ids must contain no more than 2000 items", "validate")
        return False
    candidate_ids = []
    for candidate in candidates:
        if not isinstance(candidate, dict) or set(candidate) - {"id", "artifact_digest", "predecessor_digest"}:
            fail("failed", "invalid_input", "candidate objects only support id, artifact_digest, and optional predecessor_digest", "validate")
            return False
        artifact_digest = candidate.get("artifact_digest", "")
        predecessor_digest = candidate.get("predecessor_digest", "")
        if (
            not isinstance(candidate.get("id"), str)
            or not candidate["id"].strip()
            or candidate["id"] != candidate["id"].strip()
            or utf8_size(candidate["id"]) > 256
            or not isinstance(artifact_digest, str)
            or utf8_size(artifact_digest) > 256
            or (artifact_digest and artifact_digest != artifact_digest.strip())
            or not isinstance(predecessor_digest, str)
            or utf8_size(predecessor_digest) > 256
            or (predecessor_digest and predecessor_digest != predecessor_digest.strip())
        ):
            fail("failed", "invalid_input", "each candidate needs a non-empty id and an optional normalized expected artifact_digest", "validate")
            return False
        candidate_ids.append(str(candidate["id"]).strip())
    if len(set(candidate_ids)) != len(candidate_ids):
        fail("failed", "invalid_input", "candidate IDs must be unique", "validate")
        return False
    predecessor_value = request.get("predecessor_id", "")
    if not isinstance(predecessor_value, str):
        fail("failed", "invalid_input", "predecessor_id must be a string when supplied", "validate")
        return False
    predecessor_id = predecessor_value.strip()
    if predecessor_value != predecessor_id or utf8_size(predecessor_id) > 256:
        fail("failed", "invalid_input", "predecessor_id must be canonical and no longer than 256 characters", "validate")
        return False
    if predecessor_id and predecessor_id in candidate_ids:
        fail("failed", "invalid_input", "predecessor_id must not be one of the candidate IDs", "validate")
        return False
    if any(not isinstance(case_id, str) or not case_id.strip() or case_id != case_id.strip() or utf8_size(case_id) > 256 for case_id in case_ids) or len(set(case_ids)) != len(case_ids):
        fail("failed", "invalid_input", "case_ids must be unique canonical strings no longer than 256 characters", "validate")
        return False
    if target == "ollama-role":
        if not isinstance(subject.get("role"), str) or not subject["role"].strip() or len(candidates) != 3 or minimum_cases < 20 or len(case_ids) < 20 or any(not candidate.get("artifact_digest") for candidate in candidates):
            fail("failed", "invalid_input", "ollama-role requires a named owner-policy role, three candidates pinned to expected artifact digests, and at least 20 fixed cases", "validate")
            return False
    if target == "desktop-startup":
        if len(candidates) != 1 or not predecessor_id or not candidates[0].get("artifact_digest"):
            fail("failed", "invalid_input", "desktop-startup requires one candidate and a predecessor_id", "validate")
            return False
        if minimum_cases < 30 or len(case_ids) < 30:
            fail("failed", "invalid_input", "desktop-startup requires at least 30 fixed class-phase-pair rows for the DP1 denominator", "validate")
            return False
        required_subject = ("scenario", "artifact_digest", "host_fingerprint", "display", "deployment_mode", "profiler_mode")
        if any(not isinstance(subject.get(field), str) or not subject[field].strip() for field in required_subject):
            fail("failed", "invalid_input", "desktop-startup subject is missing a required DP1 identity field", "validate")
            return False
    # Every other target is deliberately delegated to the owner adapter. This
    # keeps LM8 role sweeps, LM10 retrieval, and LM11 modality upgrades on the
    # same engine without granting the entrypoint authority to infer their
    # target-specific request shape.
    pinned_request = request
    envelope["signals"]["run_id"] = run_id
    return True


def requests_match(original, stored):
    if not isinstance(stored, dict):
        return False
    original_candidates = original.get("candidates", [])
    stored_candidates = stored.get("candidates", [])
    if not isinstance(stored_candidates, list) or len(original_candidates) != len(stored_candidates):
        return False
    candidate_keys = ("id", "artifact_digest", "predecessor_digest")
    for wanted, got in zip(original_candidates, stored_candidates):
        if not isinstance(got, dict) or any(str(wanted.get(key, "") or "") != str(got.get(key, "") or "") for key in candidate_keys):
            return False
    scalar_keys = ("run_id", "target", "predecessor_id", "suite_id", "minimum_cases")
    if any(original.get(key, "") != stored.get(key, "") for key in scalar_keys):
        return False
    return original.get("subject") == stored.get("subject") and original.get("case_ids") == stored.get("case_ids")


if step_validate():
    envelope["phase"] = "act"
    try:
        submitted = program_runtime.optimization.submit(request_json=request_json)
        record = parse_record(submitted.get("record_json", ""))
    except Exception as exc:
        classified_failure(exc, "act")
    else:
        run_id = str(submitted.get("run_id", "") or "").strip()
        record_request = record.get("request") if isinstance(record.get("request"), dict) else {}
        record_run_id = str(record_request.get("run_id", "") or "").strip()
        if not run_id or run_id != record_run_id or run_id != envelope["signals"]["run_id"] or not requests_match(pinned_request, record_request):
            fail("failed", "binding_error", "submit response record does not match the immutable optimization.Request", "act")
        else:
            stage = str(record.get("stage", "") or "")
            terminal = stage in ("complete", "refused")
            if terminal:
                add_record(pinned_request, record, True)
                envelope["phase"] = "report"
            else:
                envelope["phase"] = "collect"
                try:
                    waited = program_runtime.optimization.wait(run_id=run_id, wait_ms=wait_ms)
                    waited_record = parse_record(waited.get("record_json", ""))
                    waited_request = waited_record.get("request") if isinstance(waited_record.get("request"), dict) else {}
                    if not requests_match(pinned_request, waited_request):
                        raise ValueError("wait response record does not match the immutable submitted optimization.Request")
                    wait_terminal = bool(waited.get("terminal", False))
                    record_terminal = str(waited_record.get("stage", "") or "") in ("complete", "refused")
                    if wait_terminal != record_terminal:
                        raise ValueError("wait response terminal flag does not match the returned durable stage")
                    add_record(pinned_request, waited_record, wait_terminal)
                    envelope["signals"]["waited_ms"] = int(waited.get("waited_ms", 0) or 0)
                    envelope["phase"] = "report"
                except Exception as exc:
                    add_record(pinned_request, record, False)
                    deferred_wait_failure(exc)

print(json.dumps(envelope, separators=(",", ":")))
