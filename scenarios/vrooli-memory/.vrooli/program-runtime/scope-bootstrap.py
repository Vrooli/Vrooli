"""vrooli-memory.scope-bootstrap v1 — create a usage scope with facets and starter rules, idempotently.

Contract: scope-bootstrap.json.
Skill:    vrooli-memory §3 (setup, once per scenario).

Phases: validate -> collect -> decide -> act -> report.
Writes are limited to scopes/create, rules/create, rules/enable. An existing scope is
left untouched. A rule is enabled only after a dry run in this same submission.
Facets cannot be set through the binding (CLI-local flag); a request that names facets
for a scope that does not exist yet returns failed with class facets_not_settable (nothing was done).
Run each call in its own session: session variables persist across submissions.
"""

# ---- inputs: the caller binds a dict named `inputs`; contract defaults otherwise -------
try:
    inputs
except NameError:
    inputs = {}
scope = inputs.get("scope")
label = inputs.get("label") or (scope or "")
facets = list(inputs.get("facets", []))
wake_budget = int(inputs.get("wake_budget", 48))
max_entry_lines = int(inputs.get("max_entry_lines", 2))
frontier_target = int(inputs.get("frontier_target", 16))
rules = list(inputs.get("rules", []))

# ---- envelope: created first, printed once, on every path ---------------------------
envelope = {
    "program": "vrooli-memory.scope-bootstrap", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"scope": scope, "label": label, "facets": facets, "wake_budget": wake_budget,
               "max_entry_lines": max_entry_lines, "frontier_target": frontier_target, "rules": [r.get("id") for r in rules if isinstance(r, dict)]},
    "signals": {"scope_existed": None, "scope_created": False, "rules": [], "facets": facets},
    "errors": [], "evidence": [],
}
plan = {"create_scope": False, "rules_to_create": [], "rules_existing": []}


def fail(status, klass, detail, where):
    """The one place a bad path is recorded."""
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def step_validate():
    if not scope or not isinstance(scope, str):
        return fail("failed", "invalid_input", "scope is required", "validate")
    if any(ch for ch in scope if not (ch.isalnum() or ch in "-_:")):
        return fail("failed", "invalid_input", f"scope {scope!r} has characters outside [a-z0-9-_:]", "validate")
    for r in rules:
        if not isinstance(r, dict) or not r.get("id") or not r.get("facet") or not r.get("body_pattern"):
            return fail("failed", "invalid_input", "each rule needs id, facet, body_pattern", "validate")
    if wake_budget < 8 or max_entry_lines < 1 or frontier_target < 1:
        return fail("failed", "invalid_input", "wake_budget >= 8, max_entry_lines >= 1, frontier_target >= 1", "validate")
    return "collect"


def step_collect():  # COLLECT · read what exists
    envelope["phase"] = "collect"
    try:
        scopes, existing_rules = gather(
            lambda: vrooli_memory.scopes.list(),
            lambda: vrooli_memory.rules.list(scope=scope),
        )
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    # rules list ignores --scope (it answers for agent-memory whatever you pass), so the rows are
    # filtered here on their own scope field; without this a foreign rule id reads as already present.
    existing_rules = existing_rules.filter(lambda r: r.get("scope") == scope)
    existed = scopes.filter(lambda r: r.get("id") == scope).count() > 0
    envelope["signals"]["scope_existed"] = existed
    known = set()
    for r in existing_rules.head(200):
        known.add(r.get("id"))
    plan["create_scope"] = not existed
    for r in rules:
        (plan["rules_existing"] if r["id"] in known else plan["rules_to_create"]).append(r)
    return "decide"


def step_decide():  # DECIDE · pure; the plan is already explicit
    envelope["phase"] = "decide"
    if envelope["signals"]["scope_existed"]:
        envelope["signals"]["facets"] = "unknown: pre-existing scope; facets not re-read"
    for r in plan["rules_existing"]:
        envelope["signals"]["rules"].append({"id": r["id"], "existed": True, "created": False,
                                             "dry_run_matches": None, "enabled": None})
    return "act"


def step_act():  # ACT · declared writes only, in order: scope, then rules (create → dry-run → enable)
    envelope["phase"] = "act"
    try:
        if plan["create_scope"]:
            # The facet vocabulary is a CLI-local JSON flag (manifest bind_waiver on facets-json) with no
            # proto field reachable from a program; the scope is created without it and the caller sets
            # facets with `vrooli-memory scopes create --facets-json` before this runs, or accepts none.
            if facets:
                return fail("failed", "facets_not_settable",
                            "facets-json is CLI-local (manifest bind_waiver); create the scope with "
                            "`vrooli-memory scopes create <scope> --facets-json '[...]'` first, then rerun for rules",
                            "act")
            vrooli_memory.scopes.create(id=scope, label=label, frontier_target=frontier_target,
                                        wake_budget=wake_budget, max_entry_lines=max_entry_lines)
            envelope["signals"]["scope_created"] = True
            envelope["evidence"].append(f"scope:{scope}")
        for r in plan["rules_to_create"]:
            entry = {"id": r["id"], "existed": False, "created": False, "dry_run_matches": None, "enabled": False}
            kwargs = {"id": r["id"], "scope": scope, "facet_id": r["facet"], "body_pattern": r["body_pattern"]}  # proto field is facet_id (CLI flag --facet)
            if r.get("kind"):
                kwargs["kind"] = r["kind"]
            if r.get("priority") is not None:
                kwargs["priority"] = int(r["priority"])
            try:
                vrooli_memory.rules.create(**kwargs)
            except Exception as exc:
                if "unknown facet" in str(exc).lower():
                    # The scope's vocabulary lacks this facet; facets are set at scope creation (CLI).
                    envelope["errors"].append({"class": "unknown_facet",
                                               "detail": f"rule {r['id']}: facet {r['facet']!r} is not in scope {scope}", "where": "act"})
                    envelope["signals"]["rules"].append(entry)
                    continue
                raise
            entry["created"] = True
            dry = vrooli_memory.rules.dry_run(rule_id=r["id"], scope=scope)
            meta = dry.meta() or {}  # DryRunRuleResponse: rows are the sample strings; match_count lives in meta
            matches = int(meta.get("matchCount", meta.get("match_count", 0)) or 0)
            entry["dry_run_matches"] = matches
            # Enable when the rule matched something, or when the scope is new and has nothing to match yet.
            if matches or plan["create_scope"]:
                vrooli_memory.rules.enable(rule_id=r["id"], scope=scope)
                entry["enabled"] = True
            else:
                envelope["errors"].append({"class": "rule_dry_run_empty",
                                           "detail": f"rule {r['id']} matched 0 entries; left disabled", "where": "act"})
            envelope["signals"]["rules"].append(entry)
            envelope["evidence"].append(f"rule:{r['id']}")
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")
    envelope["status"] = "partial" if envelope["errors"] else "ok"
    return "report"


def step_report():  # REPORT · bounded, always
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "decide": step_decide,
          "act": step_act, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
