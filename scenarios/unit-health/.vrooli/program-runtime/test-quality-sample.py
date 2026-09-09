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
        # The owner binding is optional in fixture mode, but required for a live run.
        try:
            owner = unit_health.validate.scenario(scenario=env["inputs"]["scenario"])
            rows = owner.head(MAX_ROWS) if hasattr(owner, "head") else list(owner)
        except NameError:
            return fail("unavailable", "scenario_unreachable", "Unit Health inventory unavailable", "collect")
        except Exception as exc:
            return fail("unavailable", "scenario_unreachable", str(exc), "collect")
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
    if any(bool(r.get("sensitive")) or re.search(r"(^|/)(\.env|secrets?|credentials?)", str(r.get("file", "")), re.I) for r in chosen):
        return fail("refused", "privacy_refused", "candidate metadata is marked sensitive", "classify")
    supplied = {str(x.get("testIdentity")): x for x in inputs.get("classifications", []) if isinstance(x, dict)}
    if inputs.get("review_ai") and not supplied:
        try:
            # Send metadata-only context through the governed helper. Never send source bodies.
            corpus = [json.dumps({"testIdentity": identity(r), "framework": r.get("framework"), "testKind": r.get("testKind"), "staticStatus": r.get("staticStatus")}, sort_keys=True) for r in chosen]
            child = lib.ai_gateway.classify_batch(corpus=corpus, labels=sorted(LABELS), instruction="Label only the supplied test metadata. Return uncertain when context is insufficient.")
            batch = child.head(1) if hasattr(child, "head") else []
            payload = batch[0] if batch else {}
            for item in (payload.get("signals", {}).get("results", []) if isinstance(payload, dict) else []):
                idx = item.get("index"); label = item.get("label")
                if isinstance(idx, int) and 0 <= idx < len(chosen): supplied[identity(chosen[idx])] = {"label": label}
            env["evidence"].append({"binding":"ai-gateway/inference/run-batch", "promptVersion":"test-quality-review/v1", "schemaVersion":"test-quality-labels/v1"})
        except NameError:
            env["errors"].append({"class":"scenario_unreachable", "detail":"AI Gateway helper unavailable", "where":"classify"})
        except Exception as exc:
            env["errors"].append({"class":"partial_results", "detail":str(exc)[:200], "where":"classify"})
    observations = []
    for row in chosen:
        tid = identity(row); value = supplied.get(tid)
        if value:
            label = value.get("label"); status = "observed" if label in LABELS else "invalid"
            if status == "invalid": label = "uncertain"
            observations.append({"testIdentity":tid, "sourceIdentity":env["inputs"]["source_identity"], "label":label, "status":status, "limitations":["advisory; independently review before promotion"]})
        else:
            observations.append({"testIdentity":tid, "sourceIdentity":env["inputs"]["source_identity"], "label":"uncertain", "status":"uncertain", "limitations":["AI classification not requested or unavailable"]})
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
