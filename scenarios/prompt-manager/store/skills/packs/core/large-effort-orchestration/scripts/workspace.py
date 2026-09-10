#!/usr/bin/env python3
"""Create and inspect effort artifacts; never launch work or grant authority."""
import argparse
import collections
import json
import re
from pathlib import Path


class Invalid(ValueError):
    pass


def read_json(path):
    return json.loads(path.read_text())


def require(condition, message):
    if not condition:
        raise Invalid(message)


def init(repo, slug):
    require(re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", slug), "invalid effort slug")
    contract = read_json(repo / ".vrooli/repo-contract.json")["runtime_home"]
    root = Path.home() / contract["dir_name"] / contract["entries"]["plan_artifacts"]["path"] / "efforts"
    folder = root / slug
    folder.mkdir(parents=True, exist_ok=True)
    files = {
        "effort.json": {"schema_version": 1, "slug": slug, "repository": str(repo.resolve()),
                        "stage": "intake", "owners": {},
                        "execution": {"status": "not-approved", "approval_ref": None,
                                      "schedule": {"enabled": False}},
                        "policy": {"repair_limits": {"fingerprint": 2, "component": 4, "effort": 12},
                                   "max_probe_attempts": 2, "max_transport_attempts": 2,
                                   "max_status_reads": 2}},
        "requirements.json": [], "capabilities.json": [],
    }
    for name, value in files.items():
        path = folder / name
        if not path.exists():
            path.write_text(json.dumps(value, indent=2) + "\n")
    (folder / "recovery.jsonl").touch(exist_ok=True)
    (folder / "sources").mkdir(exist_ok=True)
    return folder


def recovery(folder, policy):
    starts, finishes, observations, identities = {}, {}, {}, {}
    for line_number, line in enumerate((folder / "recovery.jsonl").read_text().splitlines(), 1):
        if not line.strip():
            continue
        event = json.loads(line)
        kind = event.get("event")
        require(kind in {"repair_started", "repair_finished", "probe", "transport_retry", "status_read", "workaround", "note"},
                f"recovery line {line_number}: unknown event")
        require(event.get("at") and event.get("evidence"), f"recovery line {line_number}: missing time/evidence")
        if kind in {"note", "workaround"}:
            continue
        for key in ("attempt_id", "component", "fingerprint"):
            require(event.get(key), f"recovery line {line_number}: missing {key}")
        identity = event["attempt_id"]
        identity_kind = "repair" if kind.startswith("repair_") else kind
        signature = (identity_kind, event["component"], event["fingerprint"])
        if identity in identities:
            require(identities[identity] == signature, f"conflicting operation identity: {identity}")
        identities[identity] = signature
        if kind in {"probe", "transport_retry", "status_read"}:
            observations[identity] = signature
            continue
        if kind == "repair_started":
            require(event.get("hypothesis") and event.get("new_evidence"),
                    f"repair {identity}: missing falsifiable hypothesis or evidence cut")
            signature = tuple(event[key] for key in ("component", "fingerprint", "hypothesis", "new_evidence"))
            if identity in starts:
                require(starts[identity]["signature"] == signature, f"conflicting attempt identity: {identity}")
            else:
                starts[identity] = {"signature": signature, "event": event}
        else:
            require(identity in starts, f"finish without persisted start: {identity}")
            start = starts[identity]["event"]
            require(all(event[key] == start[key] for key in ("component", "fingerprint")),
                    f"finish identity differs from start: {identity}")
            require(event.get("outcome") in {"verified_success", "failed", "unavailable", "unknown"},
                    f"invalid repair outcome: {identity}")
            if identity in finishes:
                require(finishes[identity] == event["outcome"], f"conflicting repair result: {identity}")
            finishes[identity] = event["outcome"]
    components, fingerprints = collections.Counter(), collections.Counter()
    evidence_cuts = collections.defaultdict(set)
    repeated_evidence = set()
    for attempt in starts.values():
        event = attempt["event"]
        component, fingerprint = event["component"], event["fingerprint"]
        components[component] += 1
        fingerprints[(component, fingerprint)] += 1
        key = (component, fingerprint)
        if event["new_evidence"] in evidence_cuts[key]:
            repeated_evidence.add(key)
        evidence_cuts[key].add(event["new_evidence"])
    limits = policy["repair_limits"]
    open_circuits = []
    if len(starts) >= limits["effort"]:
        open_circuits.append({"scope": "effort", "reason": "repair_limit", "used": len(starts)})
    for component, count in components.items():
        if count >= limits["component"]:
            open_circuits.append({"scope": "component", "component": component, "reason": "repair_limit", "used": count})
    for key, count in fingerprints.items():
        if count >= limits["fingerprint"] or key in repeated_evidence:
            open_circuits.append({"scope": "fingerprint", "component": key[0], "fingerprint": key[1],
                                  "reason": "unchanged_evidence" if key in repeated_evidence else "repair_limit", "used": count})
    observation_totals = collections.Counter(observations.values())
    observation_limits = {"probe": policy["max_probe_attempts"],
                          "transport_retry": policy.get("max_transport_attempts", 2),
                          "status_read": policy.get("max_status_reads", 2)}
    for (kind, component, fingerprint), count in observation_totals.items():
        if count >= observation_limits[kind]:
            open_circuits.append({"scope": kind, "component": component, "fingerprint": fingerprint,
                                  "reason": kind + "_limit", "used": count})
    return {"repair_attempts": len(starts), "unfinished_attempts": sorted(set(starts) - set(finishes)),
            "component_attempts": dict(components), "open_circuits": open_circuits,
            "enforcement": "local_admission_input_only"}


def inspect(folder):
    manifest = read_json(folder / "effort.json")
    require(manifest.get("schema_version") == 1, "unsupported effort schema")
    require(manifest.get("slug") == folder.name, "effort slug does not match folder")
    execution = manifest["execution"]
    require(execution["status"] in {"not-approved", "approved", "revoked", "complete"}, "invalid execution authority")
    if execution["status"] in {"approved", "complete"}:
        require(execution.get("approval_ref") and execution.get("approved_source_digest"), "missing approval evidence/digest")
    if execution["schedule"].get("enabled"):
        require(execution["status"] == "approved", "schedule enabled without active approval")
        require(execution["schedule"].get("qualification_ref"), "schedule enabled without qualification evidence")
    policy = manifest["policy"]
    for name in ("fingerprint", "component", "effort"):
        value = policy["repair_limits"][name]
        require(type(value) is int and value > 0, f"invalid repair limit: {name}")
    require(type(policy["max_probe_attempts"]) is int and policy["max_probe_attempts"] > 0, "invalid probe limit")
    for name in ("max_transport_attempts", "max_status_reads"):
        require(type(policy.get(name, 2)) is int and policy.get(name, 2) > 0, f"invalid observation limit: {name}")
    rows = read_json(folder / "requirements.json")
    require(isinstance(rows, list), "requirements must be a list")
    if manifest["stage"] != "intake":
        require(rows, "reviewable effort needs requirements")
    ids = [row["id"] for row in rows]
    require(len(ids) == len(set(ids)), "duplicate requirement IDs")
    assessments = collections.Counter()
    graph = {}
    for row in rows:
        for key in ("source", "statement", "deliverable", "acceptance", "owner_plan", "assessment"):
            require(row.get(key), f"{row['id']}: missing {key}")
        require(row["assessment"] in {"unverified", "met", "unmet", "waived"}, f"{row['id']}: invalid assessment")
        if row["assessment"] == "met":
            require(row.get("evidence"), f"{row['id']}: met without evidence")
        if row["assessment"] == "waived":
            require(row.get("waiver_ref"), f"{row['id']}: waiver without decision")
        graph[row["id"]] = row.get("depends_on", [])
        require(all(ref in ids for ref in graph[row["id"]]), f"{row['id']}: unknown requirement dependency")
        assessments[row["assessment"]] += 1
    visiting, visited = set(), set()
    def visit(node):
        require(node not in visiting, f"requirement dependency cycle at {node}")
        if node in visited:
            return
        visiting.add(node)
        for child in graph[node]:
            visit(child)
        visiting.remove(node)
        visited.add(node)
    for node in graph:
        visit(node)
    capabilities = read_json(folder / "capabilities.json")
    for row in capabilities:
        require(row.get("operation") and row.get("owner"), "capability observation missing operation/owner")
        require(row.get("assessment") in {"usable", "insufficient", "unavailable", "unverified"}, "invalid capability assessment")
        if row["assessment"] != "unverified":
            require(row.get("evidence"), "capability assessment missing evidence")
    recovery_report = recovery(folder, policy)
    if execution["status"] == "complete":
        require(rows, "complete without accepted requirements")
        require(not any(row["assessment"] in {"unverified", "unmet"} for row in rows), "complete with unmet/unverified requirements")
        require(not execution["schedule"].get("enabled"), "complete with active schedule")
        require(execution.get("closure_ref"), "missing closure evidence")
        require(not recovery_report["unfinished_attempts"], "complete with unfinished repair attempts")
        require(not manifest.get("pending_operations"), "complete with pending or uncertain owner operations")
    return {"slug": manifest["slug"], "stage": manifest["stage"], "execution": execution["status"],
            "requirements": len(rows), "assessments": dict(assessments), "owners": manifest.get("owners", {}),
            "recovery": recovery_report, "validation": "structural_only"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    create = commands.add_parser("init")
    create.add_argument("--repo", type=Path, required=True)
    create.add_argument("--slug", required=True)
    for name in ("validate", "report"):
        command = commands.add_parser(name)
        command.add_argument("folder", type=Path)
    args = parser.parse_args()
    try:
        if args.command == "init":
            print(init(args.repo, args.slug))
        else:
            result = inspect(args.folder)
            print(json.dumps(result, indent=2))
    except (ValueError, KeyError, OSError, TypeError) as error:
        parser.exit(1, f"Invalid effort workspace: {error}\n")


if __name__ == "__main__":
    main()
