#!/usr/bin/env python3
"""Build a phase-blind human review packet from frozen UI-health geometry claims.

The packet contains geometry and explicit acceptance limits only. It does not
contain stored findings, evaluator predictions, source scenario/route IDs, or
the source case IDs. Source linkage is kept in a mode-0600 sidecar.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import secrets
from collections import defaultdict
from pathlib import Path
from typing import Any


def read_jsonl(path: Path) -> list[dict[str, Any]]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def evidence_for(rule: str, observation: dict[str, Any]) -> dict[str, Any]:
    layout = json.loads(observation["layoutJson"])
    if rule == "visual_mobile_toolbar_oversized":
        return {
            "rule": "toolbar-height-limit",
            "criterion": "A declared compact-toolbar limit is exceeded only when observed height is more than 1 px above the declared limit.",
            "limit_px": layout.get("expectations", {}).get("headerMaxHeight"),
            "observed_height_px": layout.get("header", {}).get("height"),
        }
    if rule == "visual_drawable_canvas_unused":
        canvas = layout.get("cssViewport", {}).get("largeHeight")
        bottom = layout.get("shell", {}).get("bottom")
        screen = layout.get("screen", {}).get("height")
        return {
            "rule": "full-screen-canvas-coverage",
            "criterion": "For a standalone full-screen canvas, unused vertical space greater than 2 px below the shell is a defect.",
            "screen_height_px": screen,
            "drawable_canvas_height_px": canvas,
            "shell_bottom_px": bottom,
            "unused_space_px": canvas - bottom if isinstance(canvas, (int, float)) and isinstance(bottom, (int, float)) else None,
        }
    if rule == "visual_nav_control_unreachable":
        controls = [item for item in layout.get("controls", []) if isinstance(item, dict) and isinstance(item.get("centerHit"), bool)]
        return {
            "rule": "navigation-control-hit-target",
            "criterion": "A recorded navigation control is defective when its center hit test misses.",
            "recorded_control_count": len(controls),
            "center_hit_count": sum(item["centerHit"] for item in controls),
            "center_miss_count": sum(not item["centerHit"] for item in controls),
        }
    raise ValueError(f"unsupported rule: {rule}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    cases = read_jsonl(args.corpus)
    grouped: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for row in cases:
        digest = row.get("snapshot_sha256")
        if not isinstance(digest, str) or not digest:
            raise ValueError(f"{row.get('case_id')}: frozen snapshot_sha256 is required")
        grouped[digest].append(row)
    if not grouped:
        raise ValueError("corpus has no snapshot groups")

    out = args.output_dir
    out.mkdir(parents=True, exist_ok=True)
    opaque = secrets.token_hex(16)
    packet, crosswalk, templates = [], [], []
    for snapshot_digest, rows in sorted(grouped.items()):
        public_id = f"case-{secrets.token_hex(8)}"
        source = rows[0]
        observations = source.get("observation")
        if not isinstance(observations, dict):
            raise ValueError(f"{source.get('case_id')}: frozen observation missing")
        claims = []
        for row in sorted(rows, key=lambda item: item.get("rule_code", "")):
            claim_id = f"claim-{secrets.token_hex(8)}"
            claims.append({"claim_id": claim_id, "evidence": evidence_for(row["rule_code"], observations)})
            crosswalk.append({
                "packet_id": public_id,
                "claim_id": claim_id,
                "source_case_id": row["case_id"],
                "phase": row["phase"],
                "rule_code": row["rule_code"],
                "snapshot_sha256": snapshot_digest,
                "source_scenario": (observations.get("scenario") or "unknown"),
                "route_state_viewport": "not-recorded-in-source",
                "source_evidence": row.get("evidence", ""),
            })
            templates.append({"packet_id": public_id, "claim_id": claim_id, "truth": "", "reason": "", "confidence": "", "insufficient_reason": ""})
        packet.append({
            "packet_id": public_id,
            "artifact_kind": "frozen-ui-geometry-observation",
            "snapshot_sha256": snapshot_digest,
            "viewport": json.loads(observations["layoutJson"]).get("viewport", {}),
            "claims": claims,
        })
    encoded = "".join(json.dumps(row, sort_keys=True, ensure_ascii=False) + "\n" for row in packet).encode()
    (out / "review-packet.jsonl").write_bytes(encoded)
    for reviewer in ("reviewer-a", "reviewer-b"):
        (out / f"{reviewer}-template.jsonl").write_text("".join(json.dumps(row, sort_keys=True) + "\n" for row in templates), encoding="utf-8")
    private = out / "source-crosswalk.private.jsonl"
    private.write_text("".join(json.dumps(row, sort_keys=True, ensure_ascii=False) + "\n" for row in crosswalk), encoding="utf-8")
    private.chmod(0o600)
    rubric = """# Independent review: frozen UI geometry claims

Review each claim independently from the other reviewer. Do not consult source finding codes, stored verdicts, or evaluator output. Choose `defect`, `clean`, or `insufficient`; cite only the supplied measurements. `insufficient` is excluded from a scored denominator.

- Toolbar height: defect only when the observed height exceeds the stated compact-height limit by more than 1 px.
- Full-screen canvas: defect only when measured unused space below the shell is greater than 2 px.
- Navigation hit target: defect when any recorded navigation control misses its center hit test; clean when all recorded centers hit. Missing measurements are insufficient.

These rules cover only the listed objective geometry properties. They do not label overall UX quality. Two reviewers must submit separately. A third independent adjudicator resolves disagreement. Record reviewer identities outside the packet and do not place names in the returned files.
"""
    (out / "review-rubric.md").write_text(rubric, encoding="utf-8")
    manifest = {
        "schema": "experience-blind-review-packet/v1",
        "packet_sha256": sha256(encoded),
        "unique_snapshot_clusters": len(packet),
        "claims": len(crosswalk),
        "reviewer_count_required": 2,
        "source_case_rows": len(cases),
        "phase": "ui-health",
        "label_status": "pending independent human review",
        "prediction_data_in_packet": False,
        "limits": ["All source snapshots currently originate from git-control-tower.", "This packet contains only 45 unique source snapshots and cannot satisfy the 50-case minimum.", "Source observations lack route, UI state and served build identity; reviewers can validate the narrow geometry criteria but cannot establish broad experience quality.", "The source phase is ui-health; it must not be relabeled as canonical experience without a fresh replay through the current phase."],
    }
    (out / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"snapshot_clusters": len(packet), "claims": len(crosswalk), "packet_sha256": manifest["packet_sha256"], "output_dir": str(out)}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
