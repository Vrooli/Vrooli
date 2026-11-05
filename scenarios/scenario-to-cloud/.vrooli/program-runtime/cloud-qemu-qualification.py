import json
# scenario-to-cloud.cloud-qemu-qualification: admit one QEMU-lane qualification
# request, bind it to the owned guest's deployment, deploy the fixture through
# the operations API (plan -> apply -> wait), observe health and edge state,
# and return the qualification standing with one next action. Admission
# refuses before any read or mutation; every effect goes through a governed
# scenario-to-cloud binding, never a shell string. The owned guest itself
# (vps instance create / wait-for-ssh / snapshot / reset / destroy) is the
# operator's documented REST pre-step and post-step.
inputs = program.inputs()
LANE = "qemu"
MATRIX_REVISION = "cloud-launch-v1"
# Kept equal to certification/lanes/qemu.json by api/certification/lanes_test.go.
QEMU_CASES = ["RUN-01", "RUN-02", "RUN-03", "RUN-04", "RUN-05", "RUN-06", "RUN-07", "RUN-08", "RUN-09", "RUN-10", "REACH-01", "REACH-02", "REACH-03", "REACH-04", "REACH-05", "REACH-06", "REACH-07", "REACH-08", "RELEASE-01", "RELEASE-02", "RELEASE-03", "RELEASE-04", "RELEASE-05", "RELEASE-06", "RELEASE-07", "RELEASE-08", "RELEASE-09", "RELEASE-10", "DATA-01", "DATA-02", "DATA-03", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-08", "DATA-09", "DATA-10", "EDGE-01", "EDGE-02", "EDGE-03", "EDGE-04", "EDGE-05", "EDGE-06", "EDGE-07", "EDGE-08", "OPS-01", "OPS-02", "OPS-03", "OPS-04", "OPS-05", "OPS-06", "OPS-07", "OPS-08"]
FAULT_CAPABILITIES = ["worker_restart", "lease_expiry", "lost_reply", "transport_disconnect", "target_reboot", "data_interruption"]
CLEANUP_POLICIES = ["retain_failed_evidence_then_remove_owned_ephemeral_resources", "retain_all_owned_resources"]
OWNED_TARGET_PREFIX = "local-qemu:"
SPEND_CEILING_MAX = 0
OWNED_TARGET_MAX = 1
EXECUTION_SECONDS_MAX = 7200
# One kernel invoke is bounded well under the lane budget; a non-terminal
# operation after this wait is reported with its id, never as a failure.
WAIT_SECONDS = 90

def text(name, default=""):
    return str(inputs.get(name, default) or "").strip()

request_id = text("request_id"); release_ref = text("release_ref"); target_ref = text("target_ref")
environment_ref = text("environment_ref"); workload_ref = text("workload_ref"); authorization_ref = text("authorization_ref")
deployment_id = text("deployment_id")
matrix_revision = text("matrix_revision", MATRIX_REVISION); lane = text("lane", LANE)
cases = inputs.get("cases") or []
limits = inputs.get("limits") or {}
faults = inputs.get("faults") or {}
cleanup = inputs.get("cleanup") or {}
allowed_faults = faults.get("allowed") if isinstance(faults, dict) else None
cleanup_policy = str(cleanup.get("policy", "")).strip() if isinstance(cleanup, dict) else ""

envelope = program.envelope("scenario-to-cloud.cloud-qemu-qualification", "1")
envelope["signals"] = {"qualification_id": "qual:" + request_id if request_id else "", "standing": "refused", "cells": [], "release": {"release_ref": release_ref}, "target": {"target_ref": target_ref, "environment_ref": environment_ref, "workload_ref": workload_ref, "deployment_id": deployment_id}, "next_action": ""}
fail = program.fail
observed = {"deployment": None, "plan_digest": "", "plan_release": "", "closure_status": "", "validation_run": None, "operation": None, "health": None, "edge": None}

def refuse(klass, detail, where="validate"):
    envelope["signals"]["standing"] = "refused"
    envelope["signals"]["next_action"] = detail
    return fail("refused", klass, detail, where)

def invalid(detail):
    envelope["signals"]["standing"] = "refused"
    envelope["signals"]["next_action"] = detail
    return fail("failed", "invalid_input", detail, "validate")

def is_immutable_digest(value):
    if not value.startswith("sha256:"):
        return False
    digest = value[len("sha256:"):]
    return len(digest) == 64 and all(c in "0123456789abcdef" for c in digest)

def first_row(handle):
    rows = handle.head(1)
    return rows[0] if rows and isinstance(rows[0], dict) else {}

# --- validate: admission. Refuses before any read or mutation. ---
def step_validate():
    for name, value in (("request_id", request_id), ("release_ref", release_ref), ("target_ref", target_ref), ("environment_ref", environment_ref), ("workload_ref", workload_ref)):
        if not value:
            return invalid(name + " is required")
    if not authorization_ref:
        return refuse("no_authorization", "authorization grant absent: supply authorization_ref naming the operator grant that designates " + target_ref + " as a disposable owned fixture")
    if lane != LANE:
        return invalid("lane must be " + LANE)
    if matrix_revision != MATRIX_REVISION:
        return refuse("stale_artifact", "matrix_revision " + matrix_revision + " is not the frozen revision " + MATRIX_REVISION)
    if not is_immutable_digest(release_ref):
        return refuse("stale_artifact", "release_ref must be an immutable sha256:<64 hex> release manifest digest; mutable references are stale by definition")
    if not target_ref.startswith(OWNED_TARGET_PREFIX):
        return refuse("incompatible_target", "target_ref must name an owned disposable local-qemu guest (" + OWNED_TARGET_PREFIX + "<id>); operator machines and enrolled cloud targets are not fixtures")
    if not isinstance(cases, list) or not cases:
        return invalid("cases must be a non-empty list of matrix case ids")
    unknown = [c for c in cases if c not in QEMU_CASES]
    if unknown:
        return invalid("cases not required on the qemu lane: " + ", ".join(sorted(str(c) for c in unknown)[:8]))
    if not isinstance(limits, dict):
        return invalid("limits must be an object")
    if int(limits.get("spend_ceiling_minor_units", 0) or 0) > SPEND_CEILING_MAX:
        return refuse("no_authorization", "spend_ceiling_minor_units must be " + str(SPEND_CEILING_MAX) + ": no spend is authorised for the qemu lane (EXT-10)")
    if int(limits.get("owned_target_count", 1) or 1) > OWNED_TARGET_MAX:
        return refuse("incompatible_target", "owned_target_count exceeds the lane budget of " + str(OWNED_TARGET_MAX))
    if int(limits.get("execution_seconds", EXECUTION_SECONDS_MAX) or EXECUTION_SECONDS_MAX) > EXECUTION_SECONDS_MAX:
        return invalid("execution_seconds exceeds the qemu lane timeout of " + str(EXECUTION_SECONDS_MAX))
    if allowed_faults is None or not isinstance(allowed_faults, list):
        return invalid("faults.allowed must be a list (empty means no fault injection)")
    unknown_faults = [f for f in allowed_faults if f not in FAULT_CAPABILITIES]
    if unknown_faults:
        return refuse("unknown_fault_capability", "unknown fault capability: " + ", ".join(sorted(str(f) for f in unknown_faults)) + "; declared capabilities: " + ", ".join(FAULT_CAPABILITIES))
    if cleanup_policy not in CLEANUP_POLICIES:
        return invalid("cleanup.policy must be one of " + ", ".join(CLEANUP_POLICIES))
    envelope["signals"]["standing"] = "admitted"
    envelope["signals"]["cells"] = [{"case_id": c, "lane": LANE, "verdict": "pending"} for c in cases]
    envelope["evidence"].append("authorization:" + authorization_ref)
    return "collect"

# --- collect: governed reads only. Binds the request to one deployment on the
# owned guest and compiles the plan it would apply. ---
def step_collect():
    global deployment_id
    envelope["phase"] = "collect"
    try:
        if deployment_id:
            ref = first_row(scenario_to_cloud.deployment.get(id=deployment_id)).get("ref") or {}
        else:
            ref = first_row(scenario_to_cloud.deployment.resolve(selector={"scenario_id": workload_ref, "environment": environment_ref})).get("ref") or {}
    except program.RemoteError as exc:
        if exc.http_status == 404:
            return refuse("incompatible_target", "no deployment is bound to workload " + workload_ref + " in environment " + environment_ref + " (or deployment_id " + deployment_id + "); create the fixture deployment on the owned guest first", "collect")
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    deployment_id = str(ref.get("id") or "")
    target = ref.get("target") or {}
    machine_id = str(target.get("machineId") or "")
    observed["deployment"] = {"id": deployment_id, "scenario_id": ref.get("scenarioId"), "environment": ref.get("environment"), "machine_id": machine_id, "transport": target.get("transport")}
    envelope["signals"]["target"]["deployment_id"] = deployment_id
    if not deployment_id:
        return refuse("incompatible_target", "deployment reference carries no id", "collect")
    if machine_id != target_ref:
        return refuse("incompatible_target", "deployment " + deployment_id + " is bound to machine " + (machine_id or "<unbound>") + ", not to the owned guest " + target_ref + "; a qualification never mutates a target it does not own", "collect")
    if str(ref.get("environment") or "") != environment_ref:
        return refuse("incompatible_target", "deployment " + deployment_id + " lives in environment " + str(ref.get("environment")) + ", not " + environment_ref, "collect")
    try:
        plan = first_row(scenario_to_cloud.deployment.plan(deployment_id=deployment_id))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    observed["plan_digest"] = str(plan.get("planDigest") or "")
    observed["closure_status"] = str(plan.get("closureStatus") or "")
    observed["plan_release"] = str((plan.get("plan") or {}).get("releaseDigest") or "")
    # The latest recorded validation run for this scenario is evidence, not a
    # gate: an unreachable test-genie leaves the row unavailable.
    try:
        runs = validate("scenario-to-cloud").head(1)
        observed["validation_run"] = {"row": runs[0]} if runs else {"row": None}
    except Exception as exc:
        observed["validation_run"] = {"unavailable": str(exc)[:160]}
    return "decide"

# --- decide: pure function of inputs and collected rows. ---
def step_decide():
    envelope["phase"] = "decide"
    envelope["signals"]["deployment"] = observed["deployment"]
    envelope["signals"]["plan"] = {"plan_digest": observed["plan_digest"], "release_digest": observed["plan_release"], "closure_status": observed["closure_status"]}
    envelope["signals"]["faults_allowed"] = list(allowed_faults)
    envelope["signals"]["cleanup_policy"] = cleanup_policy
    envelope["evidence"].append("lane:certification/lanes/qemu.json")
    envelope["evidence"].append("images:certification/lanes/qemu-images.json")
    run = observed["validation_run"] or {}
    if isinstance(run.get("row"), dict):
        run_id = run["row"].get("id") or run["row"].get("runId") or ""
        if run_id:
            envelope["evidence"].append("test-genie-run:" + str(run_id))
    elif run.get("unavailable"):
        envelope["signals"]["validation_run"] = "unavailable"
    if not observed["plan_digest"]:
        return fail("failed", "ambiguous_response", "plan compile returned no plan_digest", "decide")
    if observed["closure_status"] != "derived":
        return refuse("stale_artifact", "closure is " + (observed["closure_status"] or "unknown") + ", not derived; the deployment's declaration closure must be derivable before qualification", "decide")
    if observed["plan_release"] != release_ref:
        return refuse("stale_artifact", "compiled plan carries release " + (observed["plan_release"] or "<none>") + " but the request names " + release_ref + "; refresh the deployment's release before qualifying", "decide")
    # Fault cells need faultinject arming on the owned guest, which has no
    # governed binding yet; they are named, not silently dropped.
    if allowed_faults:
        envelope["signals"]["fault_injection"] = "no_governed_binding: faultinject arming is not a scenario-to-cloud binding; fault cells stay pending after the deploy journey"
    envelope["signals"]["standing"] = "ready_to_act"
    return "act"

# --- act: one admitted plan digest, one request key, then observation. ---
def step_act():
    envelope["phase"] = "act"
    try:
        applied = first_row(scenario_to_cloud.deployment.apply(deployment_id=deployment_id, plan_digest=observed["plan_digest"], request_key=request_id, _confirm=True))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["signals"]["next_action"] = "grant the session effect:destructive (or the binding's permission) and resubmit with the same request_id; the apply is idempotent per request key" if klass == "no_grant" else str(exc)[:200]
        return fail(status, klass, exc, "act")
    operation_id = str(applied.get("operationId") or "")
    observed["operation"] = {"operation_id": operation_id, "state": applied.get("state"), "plan_digest": applied.get("planDigest")}
    envelope["signals"]["operation"] = observed["operation"]
    if operation_id:
        envelope["evidence"].append("operation:" + operation_id)
    if not operation_id:
        envelope["signals"]["standing"] = "no_op" if applied.get("state") == "no_op" else "applied_without_operation"
        return "classify"
    try:
        standing = first_row(scenario_to_cloud.operation.wait(operation_id=operation_id, timeout_seconds=WAIT_SECONDS))
    except program.BindingError as exc:
        status, klass = program.classify(exc)
        envelope["signals"]["next_action"] = "operation " + operation_id + " is server-owned; reattach with scenario-to-cloud operation wait " + operation_id
        return fail(status, klass, exc, "act")
    observed["operation"].update({"state": standing.get("state"), "terminal": bool(standing.get("terminal")), "fence": standing.get("fence")})
    envelope["signals"]["operation"] = observed["operation"]
    for name, call in (("health", lambda: scenario_to_cloud.deployment.health(deployment_id=deployment_id)), ("edge", lambda: scenario_to_cloud.edge.status(deployment_id=deployment_id))):
        try:
            observed[name] = first_row(call()).get("observation") or {}
        except program.BindingError as exc:
            status, klass = program.classify(exc)
            observed[name] = {"unavailable": klass}
    return "classify"

# --- classify: a deterministic table over the operation and observations. ---
def step_classify():
    envelope["phase"] = "classify"
    op = observed["operation"] or {}
    health = observed["health"] or {}
    edge = observed["edge"] or {}
    status = str(health.get("status") or "")
    freshness = str(health.get("freshness") or "")
    healthy = status.endswith("HEALTHY") and not status.endswith("UNHEALTHY") and freshness.endswith("CURRENT") and str(health.get("observedReleaseDigest") or "") == release_ref
    envelope["signals"]["health"] = {"status": status or "unknown", "freshness": freshness or "unknown", "observed_release_digest": health.get("observedReleaseDigest"), "unavailable": health.get("unavailable")}
    envelope["signals"]["edge"] = {"routes": len(edge.get("routes") or []), "private_listeners": len(edge.get("privateListeners") or []), "unavailable": edge.get("unavailable")}
    if not op.get("terminal"):
        envelope["signals"]["standing"] = "operation_pending"
        envelope["signals"]["next_action"] = "operation " + str(op.get("operation_id")) + " is still " + str(op.get("state")) + "; wait once with scenario-to-cloud operation wait, then re-observe health. A lost observer never fails a server-owned operation"
        return "report"
    if str(op.get("state")) != "succeeded":
        envelope["signals"]["standing"] = "journey_failed"
        envelope["signals"]["next_action"] = "operation " + str(op.get("operation_id")) + " ended " + str(op.get("state")) + "; read its receipts (scenario-to-cloud operation get) before any retry; failed evidence is retained per cleanup policy"
        return fail("failed", "journey_failed", "deploy journey operation ended " + str(op.get("state")), "classify")
    if not healthy:
        envelope["signals"]["standing"] = "journey_unverified"
        envelope["signals"]["next_action"] = "operation succeeded but the health observation is " + (status or "unknown") + "/" + (freshness or "unknown") + " for release " + str(health.get("observedReleaseDigest")) + "; do not write passed receipts until a current healthy observation names " + release_ref
        return fail("failed", "health_unverified", "health observation does not verify the deployed release", "classify")
    envelope["signals"]["standing"] = "journey_verified"
    envelope["signals"]["next_action"] = "deploy journey verified on " + target_ref + " for " + release_ref + ": collect target receipts and per-case assertions, then write certification/evidence/<CASE>.qemu.json receipts bound to operation " + str(op.get("operation_id"))
    for cell in envelope["signals"]["cells"]:
        cell["operation_ref"] = op.get("operation_id")
        cell["verdict"] = "pending_assertions"
    envelope["status"] = "ok"
    return "report"

def step_report():
    envelope["phase"] = "report"
    envelope["status"] = "ok" if not envelope["errors"] else envelope["status"]
    if len(json.dumps(envelope, allow_nan=False, separators=(",", ":")).encode()) > 60000:
        envelope["signals"]["cells"] = envelope["signals"]["cells"][:8]
        envelope["signals"]["cells_truncated"] = True
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None

STATES = {"validate": step_validate, "collect": step_collect, "decide": step_decide, "act": step_act, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
