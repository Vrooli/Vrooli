#!/usr/bin/env python3
"""Fail-closed, source-grouped evaluator for a pinned labeled detector dataset.

The tool reads only CSV metadata and score values into a temporary SQLite DB.
It never emits sample text or IDs. Keep dataset and scratch paths outside the
Vrooli repository and ~/private/presence/; the JSON report contains aggregates.
"""
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import math
import os
import random
import re
import sqlite3
import sys
import tempfile
from collections import defaultdict
from pathlib import Path
from typing import Any

REPO_ROOT = Path(__file__).resolve().parents[3]
PRIVATE_PRESENCE = Path.home() / "private" / "presence"
REVISION_RE = re.compile(r"^(?:[0-9a-f]{40}|[0-9a-f]{64})$")
REQUIRED_MANIFEST = {
    "schema_version", "dataset_id", "revision", "split", "file_name",
    "file_sha256", "score_file_name", "score_file_sha256", "detector", "columns", "human_model", "ai_models", "domains",
    "attack_classes", "clean_attack_value", "empty_attack_is_clean", "threshold",
    "threshold_provenance", "bootstrap", "license",
}
REQUIRED_COLUMN_MAP = {
    "sample_id", "source_id", "adv_source_id", "model", "domain", "attack",
}


class EvaluationError(Exception):
    """Input does not satisfy the frozen evaluation contract."""


def _inside(path: Path, parent: Path) -> bool:
    try:
        path.resolve().relative_to(parent.resolve())
        return True
    except ValueError:
        return False


def _safe_public_path(path: Path, label: str) -> Path:
    resolved = path.expanduser().resolve()
    if _inside(resolved, REPO_ROOT):
        raise EvaluationError(f"{label} must be outside the Vrooli repository")
    if _inside(resolved, PRIVATE_PRESENCE):
        raise EvaluationError(f"{label} must be outside ~/private/presence")
    return resolved


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def load_manifest(path: Path) -> dict[str, Any]:
    try:
        doc = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise EvaluationError(f"cannot read manifest: {exc}") from exc
    if not isinstance(doc, dict):
        raise EvaluationError("manifest must be an object")
    missing = sorted(REQUIRED_MANIFEST - set(doc))
    extra = sorted(set(doc) - REQUIRED_MANIFEST)
    if missing or extra:
        raise EvaluationError(f"manifest keys invalid; missing={missing}, extra={extra}")
    if doc["schema_version"] != 1:
        raise EvaluationError("unsupported manifest schema_version")
    if not isinstance(doc["dataset_id"], str) or not doc["dataset_id"].strip():
        raise EvaluationError("dataset_id must be nonempty")
    if not isinstance(doc["human_model"], str) or not doc["human_model"].strip():
        raise EvaluationError("human_model must be nonempty")
    if not isinstance(doc["revision"], str) or not REVISION_RE.fullmatch(doc["revision"]):
        raise EvaluationError("revision must be an immutable 40- or 64-character lowercase hex commit")
    if not isinstance(doc["split"], str) or not doc["split"].strip():
        raise EvaluationError("split must be nonempty")
    if not isinstance(doc["file_name"], str) or Path(doc["file_name"]).name != doc["file_name"]:
        raise EvaluationError("file_name must be a basename")
    if not isinstance(doc["file_sha256"], str) or not re.fullmatch(r"[0-9a-f]{64}", doc["file_sha256"]):
        raise EvaluationError("file_sha256 must be a lowercase SHA-256 digest")
    if not isinstance(doc["score_file_name"], str) or Path(doc["score_file_name"]).name != doc["score_file_name"]:
        raise EvaluationError("score_file_name must be a basename")
    if not isinstance(doc["score_file_sha256"], str) or not re.fullmatch(r"[0-9a-f]{64}", doc["score_file_sha256"]):
        raise EvaluationError("score_file_sha256 must be a lowercase SHA-256 digest")
    detector = doc["detector"]
    if not isinstance(detector, dict) or set(detector) != {"id", "revision"} or any(not isinstance(v, str) or not v.strip() for v in detector.values()):
        raise EvaluationError("detector must record nonempty id and revision")
    if not isinstance(doc["columns"], dict) or set(doc["columns"]) != REQUIRED_COLUMN_MAP:
        raise EvaluationError("columns must map exactly the required semantic fields")
    for semantic, column in doc["columns"].items():
        if not isinstance(column, str) or not column:
            raise EvaluationError(f"column mapping for {semantic} must be nonempty")
    if len(set(doc["columns"].values())) != len(REQUIRED_COLUMN_MAP):
        raise EvaluationError("semantic field mappings must use distinct CSV columns")
    for name in ("ai_models", "domains", "attack_classes"):
        values = doc[name]
        if not isinstance(values, list) or not values or any(not isinstance(v, str) or not v for v in values):
            raise EvaluationError(f"{name} must be a nonempty list of nonempty strings")
        if len(set(values)) != len(values):
            raise EvaluationError(f"{name} contains duplicates")
    if doc["human_model"] in doc["ai_models"]:
        raise EvaluationError("human_model must not appear in ai_models")
    if doc["clean_attack_value"] not in doc["attack_classes"]:
        raise EvaluationError("clean_attack_value must be in attack_classes")
    if not isinstance(doc["empty_attack_is_clean"], bool):
        raise EvaluationError("empty_attack_is_clean must be an explicit boolean from pinned schema review")
    threshold = doc["threshold"]
    if isinstance(threshold, bool) or not isinstance(threshold, (int, float)) or not math.isfinite(threshold):
        raise EvaluationError("threshold must be a finite number frozen before evaluation")
    if not isinstance(doc["threshold_provenance"], str) or not doc["threshold_provenance"].strip():
        raise EvaluationError("threshold_provenance must state the calibration method")
    bootstrap = doc["bootstrap"]
    if not isinstance(bootstrap, dict) or set(bootstrap) != {"replicates", "seed"}:
        raise EvaluationError("bootstrap must contain exactly replicates and seed")
    reps, seed = bootstrap["replicates"], bootstrap["seed"]
    if isinstance(reps, bool) or not isinstance(reps, int) or not 100 <= reps <= 10000:
        raise EvaluationError("bootstrap.replicates must be between 100 and 10000")
    if isinstance(seed, bool) or not isinstance(seed, int) or seed < 0:
        raise EvaluationError("bootstrap.seed must be a nonnegative integer")
    license_info = doc["license"]
    if not isinstance(license_info, dict) or set(license_info) != {"declared", "source_url", "per_source_rights"}:
        raise EvaluationError("license must record declared, source_url and per_source_rights")
    if not isinstance(license_info["declared"], str) or not license_info["declared"].strip():
        raise EvaluationError("license.declared must be nonempty")
    if not isinstance(license_info["source_url"], str) or not license_info["source_url"].startswith("https://"):
        raise EvaluationError("license.source_url must be an HTTPS source")
    if not isinstance(license_info["per_source_rights"], str) or license_info["per_source_rights"] not in {"unresolved", "reviewed"}:
        raise EvaluationError("license.per_source_rights must be unresolved or reviewed")
    return doc


def _check_csv_header(reader: csv.DictReader, expected: set[str], label: str) -> None:
    if reader.fieldnames is None or len(set(reader.fieldnames)) != len(reader.fieldnames):
        raise EvaluationError(f"{label} CSV has missing or duplicate header")
    absent = sorted(expected - set(reader.fieldnames))
    if absent:
        raise EvaluationError(f"{label} CSV missing required columns: {absent}")


def _insert_data(db: sqlite3.Connection, data_path: Path, manifest: dict[str, Any]) -> int:
    fields = manifest["columns"]
    required = set(fields.values())
    count = 0
    with data_path.open("r", encoding="utf-8-sig", newline="") as source:
        reader = csv.DictReader(source)
        _check_csv_header(reader, required, "dataset")
        for line, row in enumerate(reader, start=2):
            sample_id = (row.get(fields["sample_id"]) or "").strip()
            source_id = (row.get(fields["source_id"]) or "").strip()
            adv_id = (row.get(fields["adv_source_id"]) or "").strip()
            model = (row.get(fields["model"]) or "").strip()
            domain = (row.get(fields["domain"]) or "").strip()
            attack = (row.get(fields["attack"]) or "").strip()
            if not all((sample_id, source_id, model, domain)):
                raise EvaluationError(f"dataset row {line} has missing sample/source/model/domain label")
            if model != manifest["human_model"] and model not in manifest["ai_models"]:
                raise EvaluationError(f"dataset row {line} has unknown model label")
            if domain not in manifest["domains"]:
                raise EvaluationError(f"dataset row {line} has unknown domain label")
            if not attack:
                if not manifest["empty_attack_is_clean"]:
                    raise EvaluationError(f"dataset row {line} has missing attack label")
                attack = manifest["clean_attack_value"]
            if attack not in manifest["attack_classes"]:
                raise EvaluationError(f"dataset row {line} has unsupported attack class")
            truth = 0 if model == manifest["human_model"] else 1
            try:
                db.execute(
                    "INSERT INTO samples(sample_id,source_id,adv_id,model,domain,attack,truth) VALUES(?,?,?,?,?,?,?)",
                    (sample_id, source_id, adv_id, model, domain, attack, truth),
                )
            except sqlite3.IntegrityError as exc:
                raise EvaluationError(f"dataset contains duplicate sample ID at row {line}") from exc
            count += 1
    if count == 0:
        raise EvaluationError("dataset has no data rows")
    return count


def _attach_scores(db: sqlite3.Connection, path: Path, manifest: dict[str, Any]) -> int:
    seen = 0
    with path.open("r", encoding="utf-8-sig", newline="") as source:
        reader = csv.DictReader(source)
        _check_csv_header(reader, {"sample_id", "score", "partition"}, "score")
        for line, row in enumerate(reader, start=2):
            sample_id = (row.get("sample_id") or "").strip()
            partition = (row.get("partition") or "").strip()
            if not sample_id:
                raise EvaluationError(f"score row {line} has empty sample_id")
            if partition not in ("calibration", "evaluation"):
                raise EvaluationError(f"score row {line} has unsupported partition")
            try:
                score = float(row.get("score", ""))
            except (TypeError, ValueError) as exc:
                raise EvaluationError(f"score row {line} has invalid score") from exc
            if not math.isfinite(score):
                raise EvaluationError(f"score row {line} has non-finite score")
            try:
                cursor = db.execute(
                    "UPDATE samples SET score=?,partition=? WHERE sample_id=? AND score IS NULL",
                    (score, partition, sample_id),
                )
            except sqlite3.Error as exc:
                raise EvaluationError(f"cannot match score row {line}") from exc
            if cursor.rowcount != 1:
                raise EvaluationError(f"score row {line} has duplicate or unknown sample ID")
            seen += 1
    data_count = db.execute("SELECT COUNT(*) FROM samples").fetchone()[0]
    missing = db.execute("SELECT COUNT(*) FROM samples WHERE score IS NULL").fetchone()[0]
    if seen != data_count or missing:
        raise EvaluationError("scores must cover each dataset sample ID exactly once")
    for label, query in (
        ("source_id", "SELECT source_id FROM samples GROUP BY source_id HAVING COUNT(DISTINCT partition)>1 LIMIT 1"),
        ("adv_source_id", "SELECT adv_id FROM samples WHERE adv_id<>'' GROUP BY adv_id HAVING COUNT(DISTINCT partition)>1 LIMIT 1"),
    ):
        if db.execute(query).fetchone() is not None:
            raise EvaluationError(f"{label} overlaps calibration and evaluation partitions")
    if not db.execute("SELECT 1 FROM samples WHERE partition='calibration' LIMIT 1").fetchone():
        raise EvaluationError("calibration partition is empty")
    if not db.execute("SELECT 1 FROM samples WHERE partition='evaluation' LIMIT 1").fetchone():
        raise EvaluationError("evaluation partition is empty")
    return seen


def _source_counts(db: sqlite3.Connection, threshold: float, axis: str | None = None,
                   value: str | None = None) -> list[tuple[int, int, int, int]]:
    where = "partition='evaluation'"
    params: tuple[Any, ...] = ()
    if axis is not None:
        if axis not in {"domain", "model", "attack"}:
            raise EvaluationError("internal error: unsupported metric axis")
        where += f" AND {axis}=?"
        params = (value,)
    pred = "score>=?"
    query = f"""SELECT source_id,
      SUM(CASE WHEN truth=1 AND {pred} THEN 1 ELSE 0 END),
      SUM(CASE WHEN truth=0 AND {pred} THEN 1 ELSE 0 END),
      SUM(CASE WHEN truth=0 AND NOT ({pred}) THEN 1 ELSE 0 END),
      SUM(CASE WHEN truth=1 AND NOT ({pred}) THEN 1 ELSE 0 END)
      FROM samples WHERE {where} GROUP BY source_id"""
    # CASE binds threshold once per predicate occurrence.
    thresholds = (threshold,) * 4 + params
    return [(int(tp or 0), int(fp or 0), int(tn or 0), int(fn or 0))
            for _, tp, fp, tn, fn in db.execute(query, thresholds)]


def _sum_counts(groups: list[tuple[int, int, int, int]]) -> tuple[int, int, int, int]:
    return tuple(sum(group[i] for group in groups) for i in range(4))  # type: ignore[return-value]


def _metrics(c: tuple[int, int, int, int]) -> dict[str, Any]:
    tp, fp, tn, fn = c
    def ratio(n: int, d: int) -> float | None:
        return n / d if d else None
    return {
        "n": tp + fp + tn + fn,
        "positive_ai": tp + fn,
        "negative_human": tn + fp,
        "tp": tp, "fp": fp, "tn": tn, "fn": fn,
        "precision": ratio(tp, tp + fp),
        "recall": ratio(tp, tp + fn),
        "false_positive_rate": ratio(fp, fp + tn),
        "f1": ratio(2 * tp, 2 * tp + fp + fn),
        "balanced_accuracy": (ratio(tp, tp + fn) + ratio(tn, tn + fp)) / 2
            if tp + fn and tn + fp else None,
    }


def _group_bootstrap(groups: list[tuple[int, int, int, int]], reps: int, seed: int) -> dict[str, Any]:
    rng = random.Random(seed)
    values: dict[str, list[float]] = {"precision": [], "recall": [], "false_positive_rate": [], "f1": [], "balanced_accuracy": []}
    valid: dict[str, int] = {name: 0 for name in values}
    for _ in range(reps):
        counts = [0, 0, 0, 0]
        for _draw in range(len(groups)):
            c = groups[rng.randrange(len(groups))]
            for i, value in enumerate(c):
                counts[i] += value
        m = _metrics(tuple(counts))
        for name in values:
            if m[name] is not None:
                values[name].append(m[name])
                valid[name] += 1
    intervals: dict[str, list[float] | None] = {}
    for name, items in values.items():
        if not items:
            intervals[name] = None
        else:
            items.sort()
            lo = items[max(0, int(0.025 * (len(items) - 1)))]
            hi = items[min(len(items) - 1, int(0.975 * (len(items) - 1)))]
            intervals[name] = [lo, hi]
    return {"confidence": 0.95, "method": "source-group bootstrap percentile", "replicates": reps,
            "valid_replicates": valid, "intervals": intervals}


def _auc(db: sqlite3.Connection) -> tuple[float | None, float | None]:
    positives, negatives = db.execute(
        "SELECT SUM(truth=1),SUM(truth=0) FROM samples WHERE partition='evaluation'"
    ).fetchone()
    if not positives or not negatives:
        return None, None
    # Mann-Whitney AUROC with half credit for tied scores; scan the SQLite index,
    # keeping only aggregate counters in process memory.
    rows = db.execute("SELECT score, SUM(truth=1), SUM(truth=0) FROM samples WHERE partition='evaluation' GROUP BY score ORDER BY score")
    below_neg = 0
    auc_twice = 0.0
    for _, pos, neg in rows:
        pos, neg = pos or 0, neg or 0
        auc_twice += pos * (2 * below_neg + neg)
        below_neg += neg
    auc = auc_twice / (2 * positives * negatives)
    # Average precision (stepwise precision over recall increments).
    tp = fp = 0
    ap = 0.0
    rows_desc = db.execute("SELECT score, SUM(truth=1), SUM(truth=0) FROM samples WHERE partition='evaluation' GROUP BY score ORDER BY score DESC")
    for _, pos, neg in rows_desc:
        tp += pos or 0
        fp += neg or 0
        ap += (pos or 0) / positives * (tp / (tp + fp))
    return auc, ap


def _evaluate(db: sqlite3.Connection, manifest: dict[str, Any]) -> dict[str, Any]:
    threshold = float(manifest["threshold"])
    reps, seed = manifest["bootstrap"]["replicates"], manifest["bootstrap"]["seed"]
    overall_groups = _source_counts(db, threshold)
    overall_conf = _sum_counts(overall_groups)
    report: dict[str, Any] = {
        "dataset": {"id": manifest["dataset_id"], "revision": manifest["revision"], "split": manifest["split"],
                    "file_sha256": manifest["file_sha256"]},
        "detector": manifest["detector"],
        "license": manifest["license"],
        "threshold": {"value": threshold, "provenance": manifest["threshold_provenance"],
                      "positive_class": "AI", "higher_score_means_ai": True},
        "evaluation": _metrics(overall_conf),
        "overall_source_group_bootstrap": _group_bootstrap(overall_groups, reps, seed),
        "by_domain": {}, "by_model": {}, "by_attack": {},
    }
    auc, ap = _auc(db)
    report["evaluation"]["auroc"] = auc
    report["evaluation"]["auprc_average_precision"] = ap
    for axis, output in (("domain", "by_domain"), ("model", "by_model"), ("attack", "by_attack")):
        vals = [row[0] for row in db.execute(f"SELECT DISTINCT {axis} FROM samples WHERE partition='evaluation' ORDER BY {axis}")]
        for index, value in enumerate(vals):
            groups = _source_counts(db, threshold, axis, value)
            c = _sum_counts(groups)
            report[output][value] = {**_metrics(c), "source_groups": len(groups),
                "source_group_bootstrap": _group_bootstrap(groups, reps, seed + index + 1)}
    return report


def evaluate(manifest_path: Path, data_path: Path, scores_path: Path, work_dir: Path) -> dict[str, Any]:
    data_path = _safe_public_path(data_path, "dataset")
    scores_path = _safe_public_path(scores_path, "score file")
    work_dir = _safe_public_path(work_dir, "scratch directory")
    if not data_path.is_file() or not scores_path.is_file():
        raise EvaluationError("dataset and score paths must be existing files")
    if not work_dir.is_dir():
        raise EvaluationError("scratch directory must already exist")
    manifest = load_manifest(manifest_path)
    if data_path.name != manifest["file_name"]:
        raise EvaluationError("dataset basename does not match manifest file_name")
    if scores_path.name != manifest["score_file_name"]:
        raise EvaluationError("score basename does not match manifest score_file_name")
    actual_hash = _sha256(data_path)
    if actual_hash != manifest["file_sha256"]:
        raise EvaluationError("dataset SHA-256 does not match pinned manifest")
    if _sha256(scores_path) != manifest["score_file_sha256"]:
        raise EvaluationError("score-file SHA-256 does not match pinned manifest")
    fd, db_name = tempfile.mkstemp(prefix="prose-raid-eval-", suffix=".sqlite", dir=work_dir)
    os.close(fd)
    db: sqlite3.Connection | None = None
    try:
        db = sqlite3.connect(db_name)
        db.execute("PRAGMA journal_mode=OFF")
        db.execute("PRAGMA synchronous=OFF")
        db.execute("CREATE TABLE samples (sample_id TEXT PRIMARY KEY, source_id TEXT NOT NULL, adv_id TEXT NOT NULL, model TEXT NOT NULL, domain TEXT NOT NULL, attack TEXT NOT NULL, truth INTEGER NOT NULL, score REAL, partition TEXT)")
        db.execute("CREATE INDEX samples_source_partition ON samples(source_id,partition)")
        db.execute("CREATE INDEX samples_adv_partition ON samples(adv_id,partition)")
        count = _insert_data(db, data_path, manifest)
        db.commit()
        _attach_scores(db, scores_path, manifest)
        db.commit()
        report = _evaluate(db, manifest)
        report["input_counts"] = {"samples": count, "scored_samples": count,
                                  "calibration_samples": db.execute("SELECT COUNT(*) FROM samples WHERE partition='calibration'").fetchone()[0],
                                  "calibration_ai_samples": db.execute("SELECT COUNT(*) FROM samples WHERE partition='calibration' AND truth=1").fetchone()[0],
                                  "calibration_human_samples": db.execute("SELECT COUNT(*) FROM samples WHERE partition='calibration' AND truth=0").fetchone()[0],
                                  "evaluation_samples": db.execute("SELECT COUNT(*) FROM samples WHERE partition='evaluation'").fetchone()[0],
                                  "evaluation_source_groups": db.execute("SELECT COUNT(DISTINCT source_id) FROM samples WHERE partition='evaluation'").fetchone()[0]}
        db.close()
        return report
    finally:
        if db is not None:
            db.close()
        try:
            os.unlink(db_name)
        except FileNotFoundError:
            pass


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", type=Path, required=True, help="pinned dataset metadata JSON")
    parser.add_argument("--data", type=Path, required=True, help="public dataset CSV outside repo/private presence")
    parser.add_argument("--scores", type=Path, required=True, help="CSV with sample_id,score,partition")
    parser.add_argument("--work-dir", type=Path, required=True, help="pre-created scratch directory outside repo/private presence")
    parser.add_argument("--report", type=Path, help="aggregate JSON report path; stdout if omitted")
    args = parser.parse_args(argv)
    try:
        report = evaluate(args.manifest, args.data, args.scores, args.work_dir)
    except EvaluationError as exc:
        print(json.dumps({"error": str(exc)}), file=sys.stderr)
        return 2
    encoded = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(encoded, encoding="utf-8")
    else:
        sys.stdout.write(encoded)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
