"""Bounded Unit Health cohort selection and advisory classification."""
import hashlib, json, re
inputs = program.inputs()

LABELS = {"behavioral", "weak_oracle", "missing_negative_case", "implementation_coupled", "valid_exception", "insufficient_context", "uncertain"}
MAX_ROWS = 64
env = {"program":"unit-health.test-quality-sample", "version":"1", "status":"failed", "phase":"validate", "signals":{}, "errors":[], "evidence":[]}

def fail(status, klass, detail, where):
    env["status"] = status
    env["errors"].append({"class": klass, "detail": str(detail)[:200], "where": where})
    return "report"

def identity(row):
    return ":".join(str(row.get(k, "")) for k in ("workspace", "file", "testId", "framework"))

STATUS_RANK = {"violation": 0, "unknown": 1, "checked_clean": 2}

def candidates_from_results(payload, workspace):
    """Fold per-rule test-quality results into one candidate per test; not_applicable rows are excluded."""
    quality = payload.get("testQuality", payload.get("test_quality", {}))
    results = quality.get("results", []) if isinstance(quality, dict) else []
    folded = {}
    for item in results:
        if not isinstance(item, dict): continue
        target = item.get("target", {}) if isinstance(item.get("target"), dict) else {}
        status = str(item.get("status", "")).replace("QUALITY_CHECK_STATUS_", "").lower()
        if status not in STATUS_RANK: continue
        ws = str(target.get("workspace", ""))
        if workspace and ws != workspace: continue
        profile = str(item.get("supportProfile", item.get("support_profile", "")))
        framework = "go" if profile.startswith("go") else ("vitest" if "vitest" in profile else profile or "unknown")
        row = {"workspace": ws, "file": str(target.get("file", "")), "testId": str(target.get("testId", target.get("test_id", ""))),
               "framework": framework, "testKind": str(item.get("testKind", item.get("test_kind", ""))), "staticStatus": status}
        key = identity(row)
        current = folded.get(key)
        if current is None or STATUS_RANK[status] < STATUS_RANK[current["staticStatus"]]:
            folded[key] = row
        if len(folded) > MAX_ROWS * 4: break
    return sorted(folded.values(), key=identity)[:MAX_ROWS]

def step_validate():
    if not isinstance(inputs, dict): return fail("failed", "invalid_input", "inputs must be an object", "validate")
    scenario = inputs.get("scenario")
    if not isinstance(scenario, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", scenario): return fail("failed", "invalid_input", "scenario must be an exact scenario id", "validate")
    if inputs.get("review_mode", "advisory") != "advisory" or inputs.get("selection", "stratified") != "stratified": return fail("failed", "invalid_input", "only stratified advisory review is supported", "validate")
    size = inputs.get("sample_size", 12)
    if type(size) is not int or size < 1 or size > MAX_ROWS: return fail("failed", "invalid_input", "sample_size must be within [1,64]", "validate")
    for key in ("seed", "source_identity"):
        if not isinstance(inputs.get(key), str) or not inputs[key].strip() or len(inputs[key]) > 256: return fail("failed", "invalid_input", key + " is required and bounded", "validate")
    rows = inputs.get("candidates", [])
    if not isinstance(rows, list) or len(rows) > MAX_ROWS: return fail("failed", "invalid_input", "candidates must be a bounded array", "validate")
    if any(not isinstance(r, dict) for r in rows): return fail("failed", "invalid_input", "candidate rows must be objects", "validate")
    env["inputs"] = {"scenario":scenario, "workspace":str(inputs.get("workspace", "api")), "sample_size":size, "seed":inputs["seed"], "source_identity":inputs["source_identity"], "selection":"stratified", "review_mode":"advisory", "include_controls":bool(inputs.get("include_controls", True))}
    return "collect"

def step_collect():
    env["phase"] = "collect"
    rows = list(inputs.get("candidates", []))
    if not rows:
        # Live path: the owner's validation read is the inventory. Each test-quality result row names one
        # (workspace, file, test) target; the worst rule status becomes the candidate's staticStatus.
        try:
            owner = unit_health.validate.scenario(scenario=env["inputs"]["scenario"], include_execution=False, rows="findings")
            payload = owner.raw() if hasattr(owner, "raw") else {}
            rows = candidates_from_results(payload if isinstance(payload, dict) else {}, env["inputs"]["workspace"])
            env["evidence"].append("unit-health:validate:" + str(payload.get("runId", payload.get("run_id", "")) if isinstance(payload, dict) else "")[:96])
        except NameError:
            return fail("unavailable", "scenario_unreachable", "Unit Health inventory unavailable", "collect")
        except Exception as exc:
            status, klass = program.classify(exc)
            return fail("unavailable" if status == "unavailable" else "failed", klass, str(exc), "collect")
    if not rows: return fail("unavailable", "scenario_unreachable", "inventory unavailable or empty; no clean cohort inferred", "collect")
    digests = {str(r.get("sourceDigest")) for r in rows if r.get("sourceDigest")}
    if digests and any(d != env["inputs"]["source_identity"] for d in digests):
        return fail("unavailable", "source_drift", "candidate source identity differs from requested source", "collect")
    # Stable hash ordering is deterministic for a fixed source identity and seed.
    rows = sorted(rows, key=lambda r: hashlib.sha256((env["inputs"]["seed"] + "\0" + identity(r)).encode()).hexdigest())
    priority = lambda r: 0 if r.get("staticStatus") == "violation" else (1 if r.get("staticStatus") in ("unknown", "") or r.get("runtimeStatus") == "unknown" else 2)
    rows = sorted(rows, key=lambda r: (priority(r), hashlib.sha256((env["inputs"]["seed"] + identity(r)).encode()).hexdigest()))
    chosen = []
    for row in rows:
        if priority(row) < 2 and len(chosen) < env["inputs"]["sample_size"]: chosen.append(row)
    if env["inputs"]["include_controls"]:
        for row in rows:
            if priority(row) == 2 and len(chosen) < env["inputs"]["sample_size"]: chosen.append(row)
    for row in rows:
        if len(chosen) >= env["inputs"]["sample_size"]: break
        if row not in chosen: chosen.append(row)
    env["signals"]["cohort"] = {"schemaVersion":"test-quality-cohort/v1", "policyVersion":"test-quality-stratified/v1", "seed":env["inputs"]["seed"], "sourceIdentity":env["inputs"]["source_identity"], "denominator":len(rows), "selected":[{k:r.get(k) for k in ("workspace","file","testId","framework","testKind","staticStatus","sourceDigest")} for r in chosen], "excluded":max(0, len(rows)-len(chosen)), "controls":sum(1 for r in chosen if priority(r)==2)}
    env["evidence"] += ["unit-health:inventory", "selection:" + env["inputs"]["seed"]]
    env["signals"]["_selected_rows"] = chosen
    return "classify"

def step_classify():
    env["phase"] = "classify"
    chosen = env["signals"].pop("_selected_rows", [])
    supplied = {str(x.get("testIdentity")): x for x in inputs.get("classifications", []) if isinstance(x, dict)}
    review_rows = []
    body_failures = {}
    body_bytes = 0
    redactions = 0
    if inputs.get("review_ai") and not supplied:
        for row in chosen:
            tid = identity(row)
            if bool(row.get("sensitive")) or re.search(r"(^|/)(\.env|secrets?|credentials?)", str(row.get("file", "")), re.I):
                body_failures[tid] = "privacy_refused"
                env["errors"].append({"class":"privacy_refused", "detail":"candidate source path was refused by the owner", "where":"classify"})
                continue
            try:
                body_handle = unit_health.validate.test_body(
                    scenario=env["inputs"]["scenario"], workspace=row.get("workspace", ""),
                    file=row.get("file", ""), test_id=row.get("testId", ""), max_bytes=4096,
                )
                body = body_handle.raw() if hasattr(body_handle, "raw") else {}
                if not isinstance(body, dict) or body.get("refused"):
                    reason = str(body.get("refusalReason", body.get("refusal_reason", "privacy_refused"))) if isinstance(body, dict) else "body_unavailable"
                    body_failures[tid] = reason
                    env["errors"].append({"class":"privacy_refused", "detail":reason[:200], "where":"classify"})
                    continue
                excerpt = str(body.get("bodyExcerpt", body.get("body_excerpt", "")))
                if len(excerpt.encode()) > 4096:
                    body_failures[tid] = "body_over_limit"
                    env["errors"].append({"class":"body_over_limit", "detail":"owner returned an excerpt over the hard ceiling", "where":"classify"})
                    continue
                body_bytes += int(body.get("bodyBytes", body.get("body_bytes", 0)) or 0)
                redactions += int(body.get("redactions", 0) or 0)
                review_rows.append({"row": row, "testIdentity": tid, "bodyExcerpt": excerpt, "bodyBytes": body.get("bodyBytes", body.get("body_bytes", 0)), "redactions": body.get("redactions", 0)})
            except NameError:
                body_failures[tid] = "body_read_unavailable"
                env["errors"].append({"class":"no_governed_binding", "detail":"ReadTestBody binding unavailable", "where":"classify"})
            except Exception as exc:
                status, klass = program.classify(exc)
                body_failures[tid] = "body_read_unavailable"
                env["errors"].append({"class":klass, "detail":str(exc)[:200], "where":"classify"})
        env["evidence"].append({"binding":"unit-health/validate/test-body", "rows":len(review_rows), "body_bytes":body_bytes, "redactions":redactions, "max_bytes":4096})
        env["signals"]["review"] = {"body_rows":len(review_rows), "refused_rows":len(body_failures), "body_bytes":body_bytes, "redactions":redactions}
    elif inputs.get("review_ai"):
        review_rows = [{"row": row, "testIdentity": identity(row), "bodyExcerpt":"", "bodyBytes":0, "redactions":0} for row in chosen]
    if inputs.get("review_ai") and not supplied:
        try:
            corpus = [json.dumps({"testIdentity": item["testIdentity"], "framework": item["row"].get("framework"), "testKind": item["row"].get("testKind"), "staticStatus": item["row"].get("staticStatus"), "bodyExcerpt": item["bodyExcerpt"]}, sort_keys=True) for item in review_rows if item["testIdentity"] not in body_failures]
            if not corpus:
                raise NameError("no review excerpts available")
            child = lib.ai_gateway.classify_batch(corpus=corpus, labels=sorted(LABELS), instruction="Label only the supplied redacted test excerpts. Return uncertain when context is insufficient.")
            batch = child.head(1) if hasattr(child, "head") else []
            payload = batch[0] if batch else {}
            available = [candidate for candidate in review_rows if candidate["testIdentity"] not in body_failures]
            for item in (payload.get("signals", {}).get("results", []) if isinstance(payload, dict) else []):
                idx = item.get("index"); label = item.get("label")
                if isinstance(idx, int) and 0 <= idx < len(available): supplied[available[idx]["testIdentity"]] = {"label": label}
            env["evidence"].append({"binding":"ai-gateway/inference/run-batch", "promptVersion":"test-quality-review/v2", "schemaVersion":"test-quality-labels/v1", "body_excerpts":len(corpus)})
        except NameError:
            env["errors"].append({"class":"insufficient_context", "detail":"No governed body excerpt or AI Gateway helper was available", "where":"classify"})
        except Exception as exc:
            env["errors"].append({"class":"partial_results", "detail":str(exc)[:200], "where":"classify"})
    observations = []
    for row in chosen:
        tid = identity(row); value = supplied.get(tid)
        if value:
            label = value.get("label"); status = "observed" if label in LABELS else "invalid"
            if status == "invalid": label = "uncertain"
            observations.append({"testIdentity":tid, "ruleVersion":"test-quality/v1", "sourceIdentity":env["inputs"]["source_identity"], "label":label, "status":status, "limitations":["advisory; independently review before promotion"]})
        else:
            reason = body_failures.get(tid, "review_not_requested" if not inputs.get("review_ai") else "insufficient_context")
            observations.append({"testIdentity":tid, "ruleVersion":"test-quality/v1", "sourceIdentity":env["inputs"]["source_identity"], "label":"insufficient_context", "status":"uncertain", "limitations":[reason, "advisory; independently review before promotion"]})
    env["signals"]["observations"] = observations
    env["signals"]["advisory"] = True
    env["status"] = "partial" if env["errors"] or any(o["status"] != "observed" for o in observations) else "ok"
    return "report"

def step_report():
    env["phase"] = "report"
    encoded = json.dumps(env, sort_keys=True, separators=(",", ":"))
    if len(encoded.encode()) > 16384:
        env["status"] = "failed"
        env["signals"] = {}
        env["errors"] = [{"class":"output_budget_exhausted", "detail":"cohort envelope exceeds output bound", "where":"report"}]
    print(json.dumps(env, sort_keys=True, separators=(",", ":")))
    return None

states = {"validate":step_validate, "collect":step_collect, "classify":step_classify, "report":step_report}
state = "validate"
while state:
    try: state = states[state]()
    except Exception as exc: state = fail("failed", "kernel_runtime", exc, state)
