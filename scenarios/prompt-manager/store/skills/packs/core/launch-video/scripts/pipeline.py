#!/usr/bin/env python3
"""Fail-closed coordinator for a multi-cut launch-video delivery.

This composes the production plan, immutable snapshot, managed HyperFrames,
and exact-export gate owners. It does not capture product surfaces, install
tools, publish, or stage a landing-page document. HyperFrames itself owns the
fresh RAM/CPU and disk admission immediately before each render.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import shutil

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
from workflow import verify_render_bundle
from product_brief import digest as sha256
WORKFLOW = HERE / "workflow.py"
PRODUCTION = HERE / "production.py"
AUDITOR = HERE / "audit_sequence_packet.py"
PRODUCTS = {"aquila", "vega", "mizar", "rigel", "pan"}
CHANNELS = {"landing", "x", "reddit", "vertical"}
LPBS_MAX_MP4_BYTES = 256 * 1024**2
REPO_ROOT = HERE.parents[7]


def contained(root: Path, value: str, *, exists: bool = True) -> Path:
    rel = Path(value)
    if rel.is_absolute() or ".." in rel.parts or not value:
        raise ValueError(f"path must be relative to pipeline manifest: {value!r}")
    path = root / rel
    cursor = root
    for part in rel.parts:
        cursor = cursor / part
        if cursor.is_symlink():
            raise ValueError(f"symlink paths are not accepted: {value}")
    resolved = path.resolve(strict=exists)
    if not resolved.is_relative_to(root.resolve()):
        raise ValueError(f"path escapes manifest root: {value}")
    if exists and not resolved.is_file():
        raise ValueError(f"expected a regular file: {value}")
    return resolved


def command(argv: list[str], *, cwd: Path, env: dict[str, str] | None = None) -> dict:
    result = subprocess.run(argv, cwd=cwd, env=env, text=True, capture_output=True, check=False)
    record = {"argv": argv, "returncode": result.returncode,
              "stdout": result.stdout[-16000:], "stderr": result.stderr[-16000:]}
    if result.returncode:
        raise RuntimeError(json.dumps(record, indent=2))
    return record


def verify_packet_sources(packet: dict, packet_dir: Path) -> dict:
    """Recheck LV4 packet provenance and every cited repository byte range."""
    source_map = packet.get("source_map") or {}
    route = (packet.get("existing_route_candidate") or {}).get("route_file") or {}
    matrix = (packet.get("existing_route_candidate") or {}).get("evidence_decision_matrix") or {}
    for label, record in (("source map", source_map), ("route", route), ("evidence matrix", matrix)):
        if not record.get("path") or len(record.get("sha256", "")) != 64:
            raise ValueError(f"LV4 packet {label} file binding is incomplete")
        evidence = contained(packet_dir, record["path"])
        if sha256(evidence) != record["sha256"]:
            raise ValueError(f"LV4 packet frozen {label} hash mismatch")

    rows = packet.get("source_records")
    if not isinstance(rows, list) or not rows:
        raise ValueError("LV4 packet has no source records")
    checked = []
    for row in rows:
        relative = Path(row.get("path", ""))
        if not relative.parts or relative.is_absolute() or ".." in relative.parts:
            raise ValueError("LV4 packet has an unsafe repository source path")
        source = REPO_ROOT / relative
        cursor = REPO_ROOT
        for part in relative.parts:
            cursor = cursor / part
            if cursor.is_symlink():
                raise ValueError(f"LV4 packet source cannot traverse symlinks: {relative}")
        resolved = source.resolve(strict=True)
        if not resolved.is_relative_to(REPO_ROOT) or not resolved.is_file():
            raise ValueError(f"LV4 packet source is outside the repository or missing: {relative}")
        data = resolved.read_bytes()
        if len(data) != row.get("bytes") or sha256(resolved) != row.get("sha256"):
            raise ValueError(f"LV4 packet whole-file source hash changed: {relative}")
        spec = row.get("line_ranges")
        if spec:
            lines = data.decode("utf-8", errors="replace").splitlines(keepends=True)
            pieces = []
            for part in spec.split(","):
                match = re.fullmatch(r"\s*(\d+)(?:\s*-\s*(\d+))?\s*", part)
                if not match:
                    raise ValueError(f"LV4 packet has invalid source line range: {relative}:{spec}")
                start = int(match.group(1)); end = int(match.group(2) or match.group(1))
                if start < 1 or end < start or end > len(lines):
                    raise ValueError(f"LV4 packet source line range is out of bounds: {relative}:{spec}")
                pieces.extend(lines[start - 1:end])
            selected = "".join(pieces).encode("utf-8")
            if hashlib.sha256(selected).hexdigest() != row.get("selected_lines_sha256"):
                raise ValueError(f"LV4 packet cited lines changed: {relative}:{spec}")
        checked.append({"path": str(relative), "sha256": row["sha256"], "line_ranges": spec})
    return {
        "source_records": len(checked),
        "source_record_manifest_sha256": hashlib.sha256(
            json.dumps(checked, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
    }


def validate_source_packet(packet: dict, packet_dir: Path, expected_product: str) -> dict:
    if packet.get("packet_kind") != "source-bound-production-plan" or packet.get("product_id") != expected_product:
        raise ValueError(f"LV4 packet kind/product mismatch: {expected_product}")
    route = packet.get("existing_route_candidate") or {}
    route_claims = route.get("claim_ids")
    proof_claims = packet.get("product_proof", {}).get("claims")
    channels = packet.get("channels")
    proof_ids = {claim.get("id") for claim in proof_claims or [] if isinstance(claim, dict)}
    if not isinstance(route_claims, list) or not route_claims or not set(route_claims).issubset(proof_ids):
        raise ValueError(f"LV4 route claims are not all source-backed: {expected_product}")
    if not isinstance(channels, dict) or set(channels) != CHANNELS:
        raise ValueError(f"LV4 packet must provide exactly four channel plans: {expected_product}")
    bindings = {}
    for channel in sorted(CHANNELS):
        plan = channels[channel]
        ordered = plan.get("ordered_claims") if isinstance(plan, dict) else None
        script = plan.get("route_script") if isinstance(plan, dict) else None
        if not isinstance(ordered, list) or not ordered or not set(ordered).issubset(set(route_claims)):
            raise ValueError(f"LV4 {expected_product}/{channel} claim order is missing or out of route")
        if not isinstance(script, dict):
            raise ValueError(f"LV4 {expected_product}/{channel} has no channel-script content")
        bindings[channel] = {"ordered_claims": ordered,
                             "script_sha256": hashlib.sha256(json.dumps(
                                 script, sort_keys=True, separators=(",", ":")).encode()).hexdigest()}
    return {"source_verification": verify_packet_sources(packet, packet_dir),
            "channel_bindings": bindings}


def accept_source_packet_set(packet_dir: Path) -> dict:
    """Validate a multi-product planning set without accepting production inputs."""
    packet_dir = packet_dir.resolve(strict=True)
    index_path = contained(packet_dir, "index.json")
    index = json.loads(index_path.read_text())
    if index.get("schema_version") != 1 or not isinstance(index.get("packets"), list):
        raise ValueError("LV4 source-packet index must use schema 1 and contain packet records")
    entries = index["packets"]
    products = [entry.get("product_id") for entry in entries]
    if len(entries) != len(PRODUCTS) or set(products) != PRODUCTS or len(set(products)) != len(products):
        raise ValueError("LV4 intake requires exactly one packet for each supported product")
    frozen = index.get("frozen_inputs")
    if not isinstance(frozen, list) or not frozen:
        raise ValueError("LV4 source-packet index has no frozen upstream inputs")
    source_map_inputs = [item for item in frozen if Path(item.get("canonical_path", "")).name == "five-products.json"]
    if len(source_map_inputs) != 1 or index.get("source_map_sha256") != source_map_inputs[0].get("sha256"):
        raise ValueError("LV4 source-packet index map digest does not match its frozen five-product input")
    frozen_verified = []
    for item in frozen:
        path = contained(packet_dir, item.get("snapshot_path", ""))
        size = path.stat().st_size
        digest = sha256(path)
        if size != item.get("bytes") or digest != item.get("sha256"):
            raise ValueError(f"LV4 frozen packet input changed: {item.get('snapshot_path')}")
        frozen_verified.append({"path": item["snapshot_path"], "bytes": size, "sha256": digest})
    accepted = []
    for entry in entries:
        packet_path = contained(packet_dir, entry.get("path", ""))
        if sha256(packet_path) != entry.get("sha256"):
            raise ValueError(f"LV4 packet hash differs from its index: {entry.get('product_id')}")
        packet = json.loads(packet_path.read_text())
        product = entry["product_id"]
        validation = validate_source_packet(packet, packet_dir, product)
        if entry.get("source_records") != validation["source_verification"]["source_records"]:
            raise ValueError(f"LV4 source record count differs from index: {product}")
        if (not isinstance(entry.get("cuts_planned"), list)
                or len(entry["cuts_planned"]) != len(CHANNELS)
                or set(entry["cuts_planned"]) != CHANNELS):
            raise ValueError(f"LV4 channel set differs from index: {product}")
        accepted.append({"product_id": product, "packet_path": entry["path"],
                         "packet_sha256": entry["sha256"],
                         "route_id": packet["existing_route_candidate"].get("opening_id"),
                         "channel_bindings": validation["channel_bindings"],
                         **validation["source_verification"]})
    return {"schema": 1, "status": "source-packet-set-accepted-for-planning",
            "acceptance_scope": "packet identity, channel claim order, frozen source files and cited repository bytes only",
            "coordinator_sha256": sha256(HERE / "pipeline.py"),
            "index_sha256": sha256(index_path), "frozen_inputs": frozen_verified, "products": accepted,
            "production_status": "not-accepted",
            "production_blockers": ["no current exact-build product media", "no channel plan/snapshot/render bundle acceptance",
                                     "operator offer/CTA sign-off remains required", "capacity, capture and publication gates remain separate"]}


def verify_packet_intake_link(root: Path, manifest: dict, packet: dict, packet_dir: Path) -> dict:
    if packet.get("product_id") != manifest.get("product"):
        raise ValueError("source-bound packet product does not match pipeline manifest")
    link = manifest.get("source_packet_intake")
    if not isinstance(link, dict) or set(link) != {"path", "sha256"}:
        raise ValueError("LV4 packet manifests require an exact-hash source_packet_intake reference")
    receipt_path = contained(root, link["path"])
    receipt_hash = sha256(receipt_path)
    if link["sha256"] != receipt_hash:
        raise ValueError("source-packet intake receipt hash mismatch")
    receipt = json.loads(receipt_path.read_text())
    if (receipt.get("status") != "source-packet-set-accepted-for-planning"
            or receipt.get("production_status") != "not-accepted"
            or receipt.get("coordinator_sha256") != sha256(HERE / "pipeline.py")):
        raise ValueError("source-packet intake receipt is stale or has the wrong acceptance scope")
    if receipt != accept_source_packet_set(packet_dir):
        raise ValueError("source-packet intake receipt does not reproduce from the current indexed packet set")
    packet_hash = sha256(contained(root, manifest["source_map"]))
    matched = [row for row in receipt.get("products", [])
               if row.get("product_id") == packet.get("product_id")]
    if len(matched) != 1 or matched[0].get("packet_sha256") != packet_hash:
        raise ValueError("source-packet intake receipt does not include this exact product packet")
    row = matched[0]
    return {"path": link["path"], "sha256": receipt_hash,
            "index_sha256": receipt.get("index_sha256"),
            "coordinator_sha256": receipt.get("coordinator_sha256"),
            "product_id": row["product_id"], "packet_sha256": row["packet_sha256"],
            "source_records": row.get("source_records"),
            "source_record_manifest_sha256": row.get("source_record_manifest_sha256"),
            "channel_bindings": row.get("channel_bindings")}


def build_channel_handoff(*, product: str, channel: str, manifest_sha256: str,
                          source_map_path: str, source_map_sha256: str,
                          packet_intake: dict, route_id: str,
                          source_verification: dict, channel_binding: dict,
                          plan_sha256: str, gate_cues_sha256: str,
                          sequence_manifest_sha256: str, tools: dict) -> dict:
    """Create a deterministic, pre-render record joining packet claims to cut inputs."""
    return {
        "schema": 1,
        "kind": "launch-video-channel-input-handoff",
        "status": "inputs-bound-pre-render",
        "product": product,
        "channel": channel,
        "manifest_sha256": manifest_sha256,
        "source_packet": {"path": source_map_path, "sha256": source_map_sha256,
                          "route_id": route_id,
                          "source_records": source_verification["source_records"],
                          "source_record_manifest_sha256": source_verification["source_record_manifest_sha256"]},
        "packet_set_intake": packet_intake,
        "channel_binding": channel_binding,
        "cut_inputs": {"plan_sha256": plan_sha256, "gate_cues_sha256": gate_cues_sha256,
                       "sequence_manifest_sha256": sequence_manifest_sha256,
                       "tool_identities": {name: {"path": identity["path"], "sha256": identity["sha256"]}
                                           for name, identity in sorted(tools.items())}},
        "limits": ["No capture evidence is established", "No render or cleanup is established",
                   "No creative, audience, operator or publication acceptance is established"],
    }


def validate_cut_plan_binding(product: str, channel: str, plan_data: dict,
                              signoff: dict, channel_binding: dict | None) -> dict:
    """Validate plan product/offer context and packet-declared channel claim order."""
    context = plan_data.get("context", {})
    if (plan_data.get("product") != product
            or context.get("offer_ref") != signoff.get("offer_ref")
            or context.get("offer_revision") != signoff.get("offer_revision")):
        raise ValueError(f"cut {channel} plan does not match the operator-approved product/offer revision")
    if channel_binding is None:
        return {"claims": [claim.get("id") for claim in plan_data.get("claims", []) if isinstance(claim, dict)]}
    expected = channel_binding["ordered_claims"]
    actual = [claim.get("id") for claim in plan_data.get("claims", []) if isinstance(claim, dict)]
    if actual != expected:
        raise ValueError(f"cut {channel} plan claim order does not match the LV4 source-bound channel packet")
    return {"claims": actual, "channel_script_sha256": channel_binding["script_sha256"]}


def load_manifest(path: Path) -> tuple[dict, Path, dict, dict | None, dict | None]:
    if path.is_symlink():
        raise ValueError("pipeline manifest cannot be a symlink")
    path = path.resolve(strict=True)
    root = path.parent.resolve(strict=True)
    raw = json.loads(path.read_text())
    if raw.get("schema") != 2 or raw.get("product") not in PRODUCTS:
        raise ValueError("pipeline requires manifest schema 2 and one of the five launch-video products")
    cuts = raw.get("cuts")
    if not isinstance(cuts, list) or len(cuts) not in (3, 4):
        raise ValueError("the pipeline requires three cuts for the Vega slice or all four LV4 channels")
    names = [cut.get("channel") for cut in cuts]
    if len(set(names)) != len(names) or any(name not in CHANNELS for name in names):
        raise ValueError("cuts require unique channels from landing, x, reddit and vertical")
    if raw["product"] != "vega" and set(names) != CHANNELS:
        raise ValueError("Aquila, Mizar, Rigel and Pan require all four LV4 channel cuts")
    output_paths: list[Path] = []
    if not raw.get("source_map") or not raw.get("offer_signoff"):
        raise ValueError("source_map and operator offer_signoff references are required")
    for index, cut in enumerate(cuts):
        required = ("plan", "plan_sha256", "gate_cues", "gate_cues_sha256",
                    "sequence_manifest", "sequence_manifest_sha256", "snapshot_dir",
                    "render_bundle", "composition", "tools")
        missing = [key for key in required if key not in cut]
        if missing:
            raise ValueError(f"cut {names[index]} is missing fields: {', '.join(missing)}")
        for key in ("plan", "gate_cues", "sequence_manifest"):
            path_ref = contained(root, cut[key])
            expected = cut.get(key + "_sha256", "")
            if len(expected) != 64 or sha256(path_ref) != expected:
                raise ValueError(f"cut {names[index]} {key} exact SHA-256 is missing or mismatched")
        for key in ("snapshot_dir", "render_bundle"):
            output_paths.append(contained(root, cut[key], exists=False))
        if not isinstance(cut["tools"], dict) or set(cut["tools"]) != {"node", "browser", "ffmpeg", "ffprobe"}:
            raise ValueError(f"cut {names[index]} requires explicit node/browser/ffmpeg/ffprobe identities")
        for tool, identity in cut["tools"].items():
            tool_path = Path(identity.get("path", ""))
            expected = identity.get("sha256", "")
            if not tool_path.is_absolute() or not tool_path.is_file() or len(expected) != 64:
                raise ValueError(f"cut {names[index]} {tool} requires an existing absolute executable and SHA-256")
            if sha256(tool_path) != expected:
                raise ValueError(f"cut {names[index]} {tool} executable hash mismatch")
    for index, left in enumerate(output_paths):
        for right in output_paths[index + 1:]:
            if left == right or left.is_relative_to(right) or right.is_relative_to(left):
                raise ValueError("snapshot and render-bundle paths must be distinct and non-overlapping")
    for key in ("source_map", "offer_signoff"):
        evidence = contained(root, raw[key])
        if raw.get(key + "_sha256") != sha256(evidence):
            raise ValueError(f"{key} exact SHA-256 is missing or does not match")
    source_data = json.loads(contained(root, raw["source_map"]).read_text())
    if not isinstance(source_data, dict):
        raise ValueError("source_map must be a JSON object")
    product_packet = None
    source_verification = None
    if source_data.get("packet_kind") == "source-bound-production-plan":
        product_packet = source_data
        intake = verify_packet_intake_link(root, raw, source_data, contained(root, raw["source_map"]).parent)
        source_verification = {"source_records": intake["source_records"],
                               "source_record_manifest_sha256": intake["source_record_manifest_sha256"],
                               "packet_set_intake": intake,
                               "channel_bindings": intake["channel_bindings"]}
    elif raw["product"] != "vega":
        raise ValueError("Aquila, Mizar, Rigel and Pan require an LV4 source-bound production packet")
    try:
        signoff = json.loads(contained(root, raw["offer_signoff"]).read_text())
    except json.JSONDecodeError as exc:
        raise ValueError("offer_signoff must be an exact-hash JSON approval; a pending Markdown decision request is not authorization") from exc
    if (signoff.get("product") != raw["product"] or signoff.get("operator_approved") is not True
            or signoff.get("status") != "approved" or not signoff.get("offer_ref")
            or not signoff.get("offer_revision")):
        raise ValueError("offer sign-off must be an exact-hash operator-approved artifact for this product and offer revision")
    return raw, root, signoff, product_packet, source_verification


def inspection_environment(cut: dict) -> dict[str, str]:
    # production.py and export_gates.py resolve these owners by executable
    # name. Pin their resolution to the same binaries frozen for HyperFrames.
    expected = {name: Path(cut["tools"][name]["path"]).resolve()
                for name in ("ffmpeg", "ffprobe")}
    entries = [str(expected[name].parent) for name in ("ffmpeg", "ffprobe")]
    if os.environ.get("PATH"):
        entries.append(os.environ["PATH"])
    path_value = os.pathsep.join(entries)
    env = os.environ.copy()
    env["PATH"] = path_value
    for name, path in expected.items():
        found = shutil.which(name, path=path_value)
        if not found or Path(found).resolve() != path:
            raise ValueError(f"{name} cannot be resolved to the pinned HyperFrames executable")
    return env


def execute(manifest_path: Path, output: Path, render: bool) -> dict:
    manifest, root, signoff, product_packet, source_verification = load_manifest(manifest_path)
    if output.is_symlink():
        raise ValueError("pipeline output cannot be a symlink")
    output = output.resolve()
    if output.exists():
        raise ValueError(f"output path must be new: {output}")
    if not output.is_relative_to(root):
        raise ValueError("pipeline output must remain under the manifest root")
    output.mkdir(parents=True, exist_ok=False)
    report = {"schema": 1, "status": "blocked", "created_at": dt.datetime.now(dt.timezone.utc).isoformat(),
              "manifest": str(manifest_path.resolve()), "manifest_sha256": sha256(manifest_path),
              "product": manifest["product"], "render_requested": render, "cuts": [],
              "limits": ["No product capture orchestration", "No landing-page draft staging",
                         "No publication or aesthetic acceptance"]}
    if product_packet is not None:
        report["source_bound_packet"] = {
            "path": manifest["source_map"], "sha256": manifest["source_map_sha256"],
            "product_id": product_packet["product_id"],
            "route_id": product_packet["existing_route_candidate"]["opening_id"],
            "verification": source_verification,
        }
    host_cli = os.environ.get("VROOLI_BIN")
    if render:
        identity = manifest.get("host_cli", {})
        if not host_cli or not Path(host_cli).is_absolute() or not Path(host_cli).is_file():
            raise ValueError("render requires VROOLI_BIN to name the exact installed managed host CLI")
        if identity.get("path") != host_cli or identity.get("sha256") != sha256(Path(host_cli)):
            raise ValueError("pipeline manifest must pin the VROOLI_BIN path and its current SHA-256")

    prepared: list[dict] = []
    for cut in manifest["cuts"]:
        row = {"channel": cut["channel"], "status": "blocked", "steps": []}
        report["cuts"].append(row)
        plan = contained(root, cut["plan"])
        cues = contained(root, cut["gate_cues"])
        sequence = contained(root, cut["sequence_manifest"])
        snapshot = contained(root, cut["snapshot_dir"], exists=False)
        bundle = contained(root, cut["render_bundle"], exists=False)
        row["plan_sha256"] = sha256(plan)
        row["gate_cues_sha256"] = sha256(cues)
        row["sequence_manifest_sha256"] = sha256(sequence)
        for key, actual in (("plan_sha256", row["plan_sha256"]),
                            ("gate_cues_sha256", row["gate_cues_sha256"]),
                            ("sequence_manifest_sha256", row["sequence_manifest_sha256"])):
            if actual != cut[key]:
                raise ValueError(f"cut {cut['channel']} {key.removesuffix('_sha256')} changed after manifest validation")
        plan_data = json.loads(plan.read_text())
        row["inspection_tools"] = {name: dict(identity) for name, identity in cut["tools"].items()}
        owner_env = inspection_environment(cut)
        channel_plan = (source_verification["channel_bindings"].get(cut["channel"])
                        if product_packet is not None else None)
        plan_binding = validate_cut_plan_binding(manifest["product"], cut["channel"], plan_data,
                                                 signoff, channel_plan)
        if product_packet is not None:
            expected_claims = plan_binding["claims"]
            row["source_packet_sha256"] = sha256(contained(root, manifest["source_map"]))
            row["source_packet_claims"] = expected_claims
            row["channel_script_sha256"] = plan_binding["channel_script_sha256"]
            handoff = build_channel_handoff(
                product=manifest["product"], channel=cut["channel"],
                manifest_sha256=report["manifest_sha256"],
                source_map_path=manifest["source_map"], source_map_sha256=manifest["source_map_sha256"],
                packet_intake=source_verification["packet_set_intake"],
                route_id=product_packet["existing_route_candidate"]["opening_id"],
                source_verification=source_verification,
                channel_binding=channel_plan,
                plan_sha256=row["plan_sha256"], gate_cues_sha256=row["gate_cues_sha256"],
                sequence_manifest_sha256=row["sequence_manifest_sha256"], tools=cut["tools"])
            handoff_path = output / f"{cut['channel']}-input-handoff.json"
            handoff_path.write_text(json.dumps(handoff, indent=2) + "\n")
            row["input_handoff"] = {"path": handoff_path.name, "sha256": sha256(handoff_path),
                                    "status": handoff["status"]}
        row["steps"].append(command([sys.executable, str(PRODUCTION), "validate", str(plan), "--stage", "plan"], cwd=root))
        row["steps"].append(command([sys.executable, str(AUDITOR), str(sequence), "--out", str(output / f"{cut['channel']}-sequence-audit.json")], cwd=root))
        sequence_report = json.loads((output / f"{cut['channel']}-sequence-audit.json").read_text())
        if sequence_report.get("readiness") != "inputs-present":
            row["blocker"] = "sequence packet has missing inputs or integrity errors"
            report["blocker"] = f"{cut['channel']}: {row['blocker']}"
            break

        freeze_args = [sys.executable, str(WORKFLOW), "freeze", str(plan), "--out", str(snapshot)]
        music = cut.get("music")
        if music:
            for key in ("bank", "catalog", "selection"):
                if key not in music:
                    raise ValueError(f"cut {cut['channel']} music requires bank, catalog and selection")
            freeze_args += ["--music-bank", str(contained(root, music["bank"])),
                            "--music-catalog", str(contained(root, music["catalog"])),
                            "--music-selection", str(contained(root, music["selection"]))]
        row["steps"].append(command(freeze_args, cwd=root))
        receipt_path = snapshot / "render-inputs.json"
        freeze_receipt = json.loads(receipt_path.read_text())
        if freeze_receipt.get("plan_sha256") != row["plan_sha256"]:
            raise ValueError(f"cut {cut['channel']} frozen receipt is not bound to the source plan")
        composition = Path(cut["composition"])
        if composition.is_absolute() or ".." in composition.parts:
            raise ValueError(f"cut {cut['channel']} composition must be relative to plan root")
        composition_dir = snapshot / composition
        if not composition_dir.is_dir():
            raise ValueError(f"cut {cut['channel']} composition is not inside the frozen render snapshot")
        inputs = [{"path": item["file"], "sha256": item["sha256"]} for item in freeze_receipt["inputs"]]
        inputs.append({"path": "render-inputs.json", "sha256": sha256(receipt_path)})
        request = {"version": 1, "input_dir": str(snapshot), "composition": cut["composition"],
                   "inputs": inputs, "node": cut["tools"]["node"], "browser": cut["tools"]["browser"],
                   "ffmpeg": cut["tools"]["ffmpeg"], "ffprobe": cut["tools"]["ffprobe"],
                   "timeout_seconds": int(cut.get("timeout_seconds", 900)),
                   "memory_mib": int(cut.get("memory_mib", 4096))}
        request_path = output / f"{cut['channel']}-hyperframes-request.json"
        request_path.write_text(json.dumps(request, indent=2) + "\n")
        row["snapshot"] = str(snapshot)
        row["snapshot_sha256"] = sha256(receipt_path)
        row["request"] = str(request_path)
        row["request_sha256"] = sha256(request_path)
        if not render:
            row["status"] = "prepared"
        prepared.append({"cut": cut, "row": row, "plan": plan, "cues": cues,
                        "sequence": sequence,
                        "snapshot": snapshot, "bundle": bundle, "request": request,
                        "request_path": request_path, "owner_env": owner_env})

    if render and len(prepared) == len(manifest["cuts"]):
        for packet in prepared:
            cut, row = packet["cut"], packet["row"]
            plan, cues = packet["plan"], packet["cues"]
            snapshot, bundle = packet["snapshot"], packet["bundle"]
            request, request_path = packet["request"], packet["request_path"]
            owner_env = packet["owner_env"]

            # This exact managed call acquires its RAM/CPU lease and 120-GiB-floor
            # disk reservation before writes. It is never replaced by a direct
            # renderer or inferred from a point-in-time df reading.
            argv = [host_cli, "host", "hyperframes", "--request", str(request_path), "--out", str(bundle), "--json"]
            row["steps"].append(command(argv, cwd=root))
            render_receipt = json.loads((bundle / "receipt.json").read_text())
            admission = render_receipt.get("capacity_admission") or {}
            disk_job = admission.get("disk_job") or {}
            min_reserve = 120 * 1024**3
            floors = [fs.get("floorBytes", 0) for fs in disk_job.get("filesystems", [])]
            if (render_receipt.get("status") != "completed" or admission.get("verdict") != "grant"
                    or admission.get("ram_bytes") != request["memory_mib"] * 1024**2
                    or admission.get("cpu_milli") != 2000 or admission.get("release") != "released"
                    or disk_job.get("status") != "completed" or disk_job.get("cleanupVerified") is not True
                    or not floors or min(floors) < min_reserve):
                row["blocker"] = "managed render lacks a completed receipt with actual capacity admission"
                report["blocker"] = f"{cut['channel']}: {row['blocker']}"
                break
            bundle_errors = verify_render_bundle(snapshot, bundle)
            if bundle_errors:
                row["blocker"] = "managed bundle failed snapshot join: " + "; ".join(bundle_errors)
                report["blocker"] = f"{cut['channel']}: {row['blocker']}"
                break
            row["render_receipt_sha256"] = sha256(bundle / "receipt.json")
            row["render_sha256"] = sha256(bundle / "render.mp4")
            mp4 = bundle / "render.mp4"
            mp4_bytes = mp4.stat().st_size
            with mp4.open("rb") as stream:
                mp4_header = stream.read(12)
            inspection = output / f"{cut['channel']}-inspection"
            gates = output / f"{cut['channel']}-gates"
            cut_times = ",".join(str(scene["start"]) for scene in plan_data.get("scenes", [])[1:])
            row["steps"].append(command([sys.executable, str(PRODUCTION), "inspect", str(bundle / "render.mp4"),
                                         "--out", str(inspection), "--cuts", cut_times], cwd=root, env=owner_env))
            row["steps"].append(command([sys.executable, str(PRODUCTION), "gate", str(bundle / "render.mp4"),
                                         "--plan", str(plan), "--cues", str(cues), "--out", str(gates)], cwd=root, env=owner_env))
            technical = json.loads((inspection / "inspection.json").read_text())
            gate_receipt = json.loads((gates / "export-gates.json").read_text())
            if not technical.get("decode_ok") or not technical.get("inputs_stable") or gate_receipt.get("status") != "pass":
                row["blocker"] = "exact-export inspection or LV3 gates failed"
                report["blocker"] = f"{cut['channel']}: {row['blocker']}"
                break
            # Re-audit actual managed-owner evidence. This is a bounded cleanup
            # receipt for this invocation, not a claim about unrelated host jobs.
            copied_receipt = output / f"{cut['channel']}-render-receipt.json"
            shutil.copyfile(bundle / "receipt.json", copied_receipt)
            post_manifest = output / f"{cut['channel']}-completed-packet.json"
            post_manifest.write_text(json.dumps({"schema": 1, "packet_id": f"{manifest['product']}-{cut['channel']}",
                "root": ".", "artifacts": [{"path": copied_receipt.name, "role": "managed-render-cleanup",
                                          "sha256": sha256(copied_receipt), "required": True}],
                "operation": {"state": "completed", "cleanup_receipt": copied_receipt.name,
                              "owned_resources": [render_receipt.get("capacity_admission", {}).get("owner_id", "hyperframes-owner")]}}, indent=2) + "\n")
            row["steps"].append(command([sys.executable, str(AUDITOR), str(post_manifest),
                                         "--out", str(output / f"{cut['channel']}-cleanup-audit.json")], cwd=root))
            cleanup_audit = json.loads((output / f"{cut['channel']}-cleanup-audit.json").read_text())
            if cleanup_audit.get("operation_cleanup", {}).get("status") != "receipt-present-owner-verification-required":
                row["blocker"] = "managed owner cleanup receipt was not retained by packet audit"
                report["blocker"] = f"{cut['channel']}: {row['blocker']}"
                break
            row["status"] = "machine-gated-unreviewed"
            # LPBS owns local MP4 staging. The pipeline cannot use its step-up
            # admin session or make the separately authorized draft mutation;
            # emit the exact-byte handoff needed by that owner instead.
            media_ref = row["render_sha256"]
            upload_ready = 0 < mp4_bytes <= LPBS_MAX_MP4_BYTES and len(mp4_header) >= 8 and mp4_header[4:8] == b"ftyp"
            row["staging_handoff"] = {
                "status": "ready-for-admin-stage" if upload_ready else "blocked-media-constraint",
                "reason": None if upload_ready else f"LPBS requires 1..{LPBS_MAX_MP4_BYTES} bytes and an ISO BMFF ftyp signature.",
                "owner": "landing-page-business-suite.presentation-media",
                "upload": {
                    "method": "POST", "path": "/api/v1/admin/presentation-media",
                    "content_type": "video/mp4", "max_bytes": LPBS_MAX_MP4_BYTES,
                    "bytes": mp4_bytes, "ftyp_signature": mp4_header[4:8].decode("ascii", errors="replace"),
                    "media_ref": media_ref,
                    "requires": "Content-Length, step-up admin session and shared disk-broker admission"
                },
                "draft": {
                    "operation": "Use the authenticated local-film-stage admin UI and SaveDraft",
                    "playback": {"provider": "local", "url": "", "media_ref": media_ref},
                    "revision_binding": "Draft playback must use the current draft revision."
                },
                "private_preview_path_template": "/api/v1/admin/presentation-media/{variant}/{revision}/" + media_ref,
                "publication": "Not performed; activation/public playback remains operator-controlled."
            }

    manifest_changed = sha256(manifest_path) != report["manifest_sha256"]
    for packet in prepared:
        row = packet["row"]
        changed = (sha256(packet["plan"]) != row["plan_sha256"]
                   or sha256(packet["cues"]) != row["gate_cues_sha256"]
                   or sha256(packet["sequence"]) != row["sequence_manifest_sha256"])
        changed = changed or any(sha256(Path(identity["path"])) != identity["sha256"]
                                 for identity in packet["cut"]["tools"].values())
        if changed:
            report["status"] = "blocked"
            report["blocker"] = f"{row['channel']} source inputs changed during execution"
            row["status"] = "blocked"

    report["status"] = "machine-gated-unreviewed" if report["cuts"] and all(c["status"] == "machine-gated-unreviewed" for c in report["cuts"]) else (
        "prepared" if report["cuts"] and all(c["status"] in ("prepared", "machine-gated-unreviewed") for c in report["cuts"]) and not render else "blocked")
    if manifest_changed:
        report["status"] = "blocked"
        report["blocker"] = "pipeline manifest changed during execution"
    if (sha256(contained(root, manifest["source_map"])) != manifest["source_map_sha256"]
            or sha256(contained(root, manifest["offer_signoff"])) != manifest["offer_signoff_sha256"]):
        report["status"] = "blocked"
        report["blocker"] = "source map or operator offer sign-off changed during execution"
    if product_packet is not None:
        intake_path = contained(root, manifest["source_packet_intake"]["path"])
        current_intake = accept_source_packet_set(contained(root, manifest["source_map"]).parent)
        if (sha256(intake_path) != manifest["source_packet_intake"]["sha256"]
                or sha256(HERE / "pipeline.py") != source_verification["packet_set_intake"]["coordinator_sha256"]
                or json.loads(intake_path.read_text()) != current_intake):
            report["status"] = "blocked"
            report["blocker"] = "source packet, cited sources, intake receipt or coordinator identity changed during execution"
    if not render:
        report["staging"] = {"status": "pending-render", "owner": "landing-page-business-suite.presentation-media",
                              "reason": "No exact MP4 exists until the managed render completes; this invocation does not stage drafts."}
    elif report["status"] == "machine-gated-unreviewed" and all(c.get("staging_handoff", {}).get("status") == "ready-for-admin-stage" for c in report["cuts"]):
        report["staging"] = {"status": "admin-stage-required", "owner": "landing-page-business-suite.presentation-media",
                              "handoff": "Per-cut exact-hash upload and draft SaveDraft instructions are recorded in each cut's staging_handoff."}
    else:
        report["staging"] = {"status": "blocked", "owner": "landing-page-business-suite.presentation-media",
                              "reason": "Only exact MP4s that complete managed render, verification and LV3 gates receive staging handoffs."}
    (output / "pipeline.json").write_text(json.dumps(report, indent=2) + "\n")
    return report


def write_failure_receipt(output: Path, manifest_path: Path, error: Exception,
                          output_existed_before: bool, render_requested: bool) -> None:
    """Retain a failure envelope when execution leaves a newly-created partial run."""
    if output_existed_before or output.is_symlink() or not output.is_dir():
        return
    receipt = output / "pipeline.json"
    if receipt.exists():
        return
    artifacts = []
    failure_hash_limit = 64 * 1024**2
    failure_entry_limit = 5000
    inventory_truncated = False
    for current, dirs, files in os.walk(output):
        dirs[:] = sorted(name for name in dirs if not (Path(current) / name).is_symlink())
        for name in sorted(files):
            path = Path(current) / name
            if path.is_symlink() or not path.is_file():
                continue
            if len(artifacts) >= failure_entry_limit:
                inventory_truncated = True
                break
            size = path.stat().st_size
            item = {"path": str(path.relative_to(output)), "bytes": size}
            if size <= failure_hash_limit:
                item["sha256"] = sha256(path)
            else:
                item["hash_status"] = "omitted-over-64-MiB-bound"
            artifacts.append(item)
        if inventory_truncated:
            break
    failure_text = str(error)
    command_failure = None
    try:
        detail = json.loads(failure_text)
        if isinstance(detail, dict) and isinstance(detail.get("argv"), list) and isinstance(detail.get("returncode"), int):
            stdout = detail.get("stdout", "")
            stderr = detail.get("stderr", "")
            argv = [str(value) for value in detail["argv"]]
            joined = " ".join(argv)
            if "host hyperframes" in joined:
                phase = "managed-render"
            elif "production.py" in joined and " validate " in f" {joined} ":
                phase = "plan-validation"
            elif "workflow.py" in joined and " freeze " in f" {joined} ":
                phase = "snapshot-freeze"
            elif "production.py" in joined and " inspect " in f" {joined} ":
                phase = "exact-export-inspection"
            elif "production.py" in joined and " gate " in f" {joined} ":
                phase = "lv3-export-gates"
            elif "audit_sequence_packet.py" in joined:
                phase = "render-cleanup-audit" if "-cleanup-audit.json" in joined else "sequence-integrity-audit"
            else:
                phase = "owner-command"
            command_failure = {"argv": detail["argv"], "returncode": detail["returncode"],
                               "phase": phase,
                               "stdout_bytes": len(stdout.encode()), "stdout_sha256": hashlib.sha256(stdout.encode()).hexdigest(),
                               "stderr_bytes": len(stderr.encode()), "stderr_sha256": hashlib.sha256(stderr.encode()).hexdigest()}
    except (json.JSONDecodeError, TypeError, AttributeError):
        pass
    manifest_context = {}
    if manifest_path.is_file() and not manifest_path.is_symlink():
        try:
            manifest = json.loads(manifest_path.read_text())
            manifest_context["product"] = manifest.get("product")
            context_root = manifest_path.parent.resolve()
            for key in ("source_map", "offer_signoff"):
                value = manifest.get(key)
                evidence = {"path": value, "expected_sha256": manifest.get(key + "_sha256")}
                if value:
                    try:
                        path = contained(context_root, value)
                        evidence["observed_sha256"] = sha256(path)
                    except (ValueError, OSError):
                        evidence["observed_sha256"] = None
                manifest_context[key] = evidence
            link = manifest.get("source_packet_intake")
            if isinstance(link, dict):
                evidence = {"path": link.get("path"), "expected_sha256": link.get("sha256")}
                try:
                    evidence["observed_sha256"] = sha256(contained(context_root, link["path"]))
                except (ValueError, OSError, KeyError):
                    evidence["observed_sha256"] = None
                manifest_context["source_packet_intake"] = evidence
            cut_context = []
            for cut in manifest.get("cuts", []) if isinstance(manifest.get("cuts"), list) else []:
                row = {"channel": cut.get("channel")}
                for key in ("plan", "gate_cues", "sequence_manifest"):
                    value = cut.get(key)
                    evidence = {"path": value, "expected_sha256": cut.get(key + "_sha256")}
                    if value:
                        try:
                            evidence["observed_sha256"] = sha256(contained(context_root, value))
                        except (ValueError, OSError):
                            evidence["observed_sha256"] = None
                    row[key] = evidence
                cut_context.append(row)
            manifest_context["cuts"] = cut_context
        except (json.JSONDecodeError, OSError):
            manifest_context = {"status": "manifest-context-unreadable"}
    record = {
        "schema": 1,
        "status": "blocked",
        "phase": "execution-error",
        "failure_stage": "owner-command" if command_failure else "coordinator-step",
        "manifest": str(manifest_path.resolve()) if manifest_path.is_file() else str(manifest_path),
        "manifest_sha256": sha256(manifest_path) if manifest_path.is_file() and not manifest_path.is_symlink() else None,
        "manifest_context": manifest_context,
        "render_requested": render_requested,
        "blocker": failure_text,
        "command_failure": command_failure,
        "partial_artifacts": artifacts,
        "artifact_inventory": {"listed": len(artifacts), "truncated": inventory_truncated,
                               "max_entries": failure_entry_limit, "max_hashed_bytes_per_file": failure_hash_limit},
        "cleanup": "unknown; failure envelope does not establish owner cleanup",
        "limits": ["No automatic retry", "No automatic cleanup", "No creative or delivery acceptance"]
    }
    receipt.write_text(json.dumps(record, indent=2) + "\n")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path, nargs="?")
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--intake-source-packets", type=Path,
                        help="accept an exact multi-product source packet set for planning only; does not run the film pipeline")
    parser.add_argument("--render", action="store_true", help="invoke managed owner only after all per-cut packet gates pass")
    args = parser.parse_args()
    if args.intake_source_packets:
        if args.manifest or args.render:
            parser.error("--intake-source-packets cannot be combined with a production manifest or --render")
        if args.out.exists() or args.out.is_symlink():
            parser.error("intake receipt output must be new")
        try:
            receipt = accept_source_packet_set(args.intake_source_packets)
            args.out.parent.mkdir(parents=True, exist_ok=True)
            args.out.write_text(json.dumps(receipt, indent=2) + "\n")
            print(json.dumps({"status": receipt["status"], "products": len(receipt["products"]),
                              "production_status": receipt["production_status"], "receipt": str(args.out.resolve())}))
            return 0
        except Exception as exc:
            parser.error(str(exc))
    if not args.manifest:
        parser.error("a production manifest is required unless --intake-source-packets is used")
    output_existed_before = args.out.exists() or args.out.is_symlink()
    try:
        report = execute(args.manifest, args.out, args.render)
        print(json.dumps({"status": report["status"], "receipt": str(args.out.resolve()),
                          "blocker": report.get("blocker"), "staging": report["staging"]["status"]}))
        return 0 if report["status"] in ("prepared", "machine-gated-unreviewed") else 1
    except Exception as exc:
        try:
            write_failure_receipt(args.out, args.manifest, exc, output_existed_before, args.render)
        except OSError:
            pass
        parser.error(str(exc))


if __name__ == "__main__":
    raise SystemExit(main())
