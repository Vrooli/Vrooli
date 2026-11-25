"""Replay the labelled reviewer corpus at candidate thresholds.

`image-tools safety corpus --out corpus.jsonl` writes one labelled reviewer
verdict per line (see docs/internal/SECURITY.md). This script re-applies the
sexual/age decision to each imported evidence object, so a threshold change can
be measured before it is shipped:

  image-tools safety corpus --out /tmp/corpus.jsonl
  python3 scripts/safety-corpus-replay.py --corpus /tmp/corpus.jsonl \
      --sweep-age 0.10,0.15,0.20,0.25,0.30

The corpus contains only outputs the reviewer refused or flagged, so it measures
false positives and missed refusals on reviewed content. It is not a random
sample of all images and cannot estimate precision or recall over everything.

`finding.nsfw_scores` and `finding.faces[].p_under_20` are snapshotted at record
time precisely so this replay needs no model weights and no pixels.
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import defaultdict
from typing import Any, Dict, Iterable, List, Mapping, Optional, Sequence

SCHEMA_VERSION = 1
NON_CLASSIFIER_KEYS = {"age", "face.min_px", "face.score"}
LABELS = {"false_positive", "confirmed"}


def parse_thresholds(raw: Sequence[str]) -> Dict[str, float]:
    out: Dict[str, float] = {}
    for item in raw:
        if "=" not in item:
            raise SystemExit(f"--threshold {item!r}: expected id=value")
        key, value = item.split("=", 1)
        key = key.strip()
        if not key:
            raise SystemExit(f"--threshold {item!r}: empty classifier id")
        try:
            out[key] = float(value)
        except ValueError:
            raise SystemExit(f"--threshold {item!r}: {value!r} is not a number")
        if not 0 <= out[key] <= 1:
            raise SystemExit(f"--threshold {item!r}: want 0..1")
    return out


def evidence_of(case: Mapping[str, Any]) -> Mapping[str, Any]:
    evidence = case.get("evidence")
    return evidence if isinstance(evidence, dict) else {}


def finding_of(case: Mapping[str, Any]) -> Mapping[str, Any]:
    finding = case.get("finding")
    return finding if isinstance(finding, dict) else {}


def resolved_thresholds(case: Mapping[str, Any], overrides: Mapping[str, float]) -> Dict[str, float]:
    thresholds = evidence_of(case).get("thresholds") or {}
    out = {
        str(key): float(value)
        for key, value in thresholds.items()
        if key not in NON_CLASSIFIER_KEYS
    }
    out.update(overrides)
    return out


def decide(case: Mapping[str, Any], overrides: Mapping[str, float], age_threshold: Optional[float], min_face_px: Optional[int]) -> Dict[str, Any]:
    """Re-apply the reviewer's decision from the snapshotted measurements."""
    finding = finding_of(case)
    evidence = evidence_of(case)
    safety = case.get("safety") or {}
    reasons = set(str(safety.get("reason_code", "")).split(","))
    expected_refusal = str(safety.get("action", "")) in {"refuse", "partial_refused", "shadow_refused"} or reasons & {
        "sexual_minor_face",
        "sexual_unassessable",
    }
    if "model" in reasons:
        # The output was flagged because an NSFW-capable model produced it, not
        # because the classifier crossed a bar. The replay cannot infer that from
        # the scores, so it must not pretend to reproduce the flag.
        return {"replayable": False, "predicted": True, "model_flagged": True}
    if not finding:
        return {"replayable": False, "predicted": str(safety.get("action", "")) in {"refuse", "flag"}}

    scores = finding.get("nsfw_scores")
    if not isinstance(scores, dict) or not scores:
        return {"replayable": False, "predicted": bool(finding.get("sexual"))}

    thresholds = resolved_thresholds(case, overrides)
    sexual = any(float(scores.get(key, 0.0)) >= value for key, value in thresholds.items())
    if not thresholds:
        sexual = bool(finding.get("sexual"))

    age = float(age_threshold if age_threshold is not None else (evidence.get("thresholds") or {}).get("age", 0.2))
    min_px = int(min_face_px if min_face_px is not None else (evidence.get("thresholds") or {}).get("face.min_px", 40))
    faces = [face for face in (finding.get("faces") or []) if isinstance(face, dict)]
    assessed = [face for face in faces if float(face.get("size_px", 0)) >= min_px and "p_under_20" in face]
    minor_face = any(float(face.get("p_under_20", 0.0)) >= age for face in assessed)
    unassessable = sexual and not assessed
    refuse = sexual and (minor_face or unassessable)
    return {
        "replayable": True,
        "sexual": sexual,
        "minor_face": minor_face,
        "unassessable": unassessable,
        "refuse": refuse,
        # A case the reviewer refused is compared on the refusal decision; a
        # flagged case is compared on the sexual decision. Mixing the two would
        # count a threshold change that merely downgrades refuse->flag as a pass.
        "predicted": refuse if expected_refusal else sexual,
        "expected_refusal": expected_refusal,
        "age_threshold": age,
        "min_face_px": min_px,
    }


def histogram(decisions: Iterable[Mapping[str, Any]]) -> Dict[str, Any]:
    counts = {"true_positive": 0, "false_positive": 0, "true_negative": 0, "false_negative": 0, "not_replayable": 0, "model_flagged": 0, "uncertain": 0}
    by_reason: Dict[str, Dict[str, int]] = defaultdict(lambda: defaultdict(int))
    second_opinion_dependent = 0
    for item in decisions:
        if item["label"] not in LABELS:
            counts["uncertain"] += 1
            continue
        if not item["replayable"]:
            counts["not_replayable"] += 1
            if item.get("model_flagged"):
                counts["model_flagged"] += 1
            continue
        reason = item["reason_code"] or "(none)"
        if reason == "sexual_unassessable":
            second_opinion_dependent += 1
        by_reason[reason]["cases"] += 1
        false_positive = item["label"] == "false_positive"
        if item["predicted"]:
            counts["false_positive" if false_positive else "true_positive"] += 1
            by_reason[reason]["false_positive" if false_positive else "true_positive"] += 1
        else:
            counts["true_negative" if false_positive else "false_negative"] += 1
            by_reason[reason]["true_negative" if false_positive else "false_negative"] += 1
    decisive = counts["false_positive"] + counts["true_positive"]
    errors = counts["false_positive"] + counts["false_negative"]
    error_denominator = decisive + counts["false_negative"]
    return {
        "counts": counts,
        "false_positive_rate": (counts["false_positive"] / decisive) if decisive else None,
        "error_rate": (errors / error_denominator) if error_denominator else None,
        "by_reason": {key: dict(value) for key, value in sorted(by_reason.items())},
        "second_opinion_dependent": second_opinion_dependent,
    }


def replay(cases: Sequence[Mapping[str, Any]], overrides: Mapping[str, float], age_threshold: Optional[float], min_face_px: Optional[int]) -> Dict[str, Any]:
    decisions = []
    for case in cases:
        decision = decide(case, overrides, age_threshold, min_face_px)
        decision.update(
            {
                "label": str(case.get("label") or ""),
                "reason_code": str((case.get("safety") or {}).get("reason_code") or ""),
            }
        )
        decisions.append(decision)
    return histogram(decisions)


def load_corpus(path: str) -> List[Dict[str, Any]]:
    source = sys.stdin if path == "-" else open(path, "r", encoding="utf-8")
    cases: List[Dict[str, Any]] = []
    try:
        for number, line in enumerate(source, 1):
            line = line.strip()
            if not line:
                continue
            try:
                case = json.loads(line)
            except json.JSONDecodeError as exc:
                raise SystemExit(f"{path}:{number}: not JSON: {exc}")
            if case.get("schema_version") != SCHEMA_VERSION:
                raise SystemExit(f"{path}:{number}: schema_version = {case.get('schema_version')!r}, want {SCHEMA_VERSION}")
            cases.append(case)
    finally:
        if source is not sys.stdin:
            source.close()
    return cases


def format_rate(value: Optional[float]) -> str:
    return "—" if value is None else f"{value * 100:.1f}%"


def print_report(report: Mapping[str, Any], label: str) -> None:
    counts = report["counts"]
    print(f"\n== {label} ==")
    print(f"  true positives:  {counts['true_positive']}")
    print(f"  false positives: {counts['false_positive']}  (rate {format_rate(report['false_positive_rate'])})")
    print(f"  false negatives: {counts['false_negative']}  (would let a confirmed case through)")
    print(f"  combined error:  {format_rate(report['error_rate'])}  ((FP+FN) / labelled)")
    print(f"  not replayable:  {counts['not_replayable']}  (no per-image scores/faces, or a model-capability flag)")
    print(f"  uncertain:       {counts['uncertain']}")
    if counts["model_flagged"]:
        print(f"  note: {counts['model_flagged']} case(s) were flagged because an NSFW-capable model produced them; the replay does not model that")
    if report["second_opinion_dependent"]:
        print(f"  note: {report['second_opinion_dependent']} case(s) were sexual_unassessable; the vision second opinion is not modelled")
    for reason, row in report["by_reason"].items():
        print(f"  {reason:<24} cases={row.get('cases', 0):<3} fp={row.get('false_positive', 0):<3} tp={row.get('true_positive', 0):<3} fn={row.get('false_negative', 0)}")


def main(argv: Optional[Sequence[str]] = None) -> None:
    p = argparse.ArgumentParser(description="Replay the labelled reviewer corpus at candidate thresholds")
    p.add_argument("--corpus", required=True, help="JSON Lines export from `image-tools safety corpus` (- for stdin)")
    p.add_argument("--threshold", action="append", default=[], metavar="ID=VALUE", help="override one classifier threshold (repeatable)")
    p.add_argument("--age-threshold", type=float, help="override P(age < 20) threshold")
    p.add_argument("--min-face-px", type=int, help="override the minimum readable face size")
    p.add_argument("--sweep-age", help="comma-separated age thresholds to compare (e.g. 0.10,0.15,0.20,0.25,0.30)")
    p.add_argument("--json", action="store_true", help="emit the report as JSON")
    args = p.parse_args(argv)

    overrides = parse_thresholds(args.threshold)
    cases = load_corpus(args.corpus)
    baseline = replay(cases, overrides, args.age_threshold, args.min_face_px)

    if args.json and not args.sweep_age:
        print(json.dumps({"baseline": baseline}, indent=2))
        return

    print(f"cases: {len(cases)}")
    if overrides:
        print("threshold overrides: " + ", ".join(f"{key}={value:.2f}" for key, value in sorted(overrides.items())))
    print_report(baseline, "current/candidate")
    if args.sweep_age:
        sweep = []
        for raw in args.sweep_age.split(","):
            raw = raw.strip()
            if not raw:
                continue
            value = float(raw)
            report = replay(cases, overrides, value, args.min_face_px)
            sweep.append({"age_threshold": value, **report})
            print_report(report, f"age threshold {value:.2f}")
        if args.json:
            print(json.dumps({"baseline": baseline, "sweep": sweep}, indent=2))


if __name__ == "__main__":
    main()
