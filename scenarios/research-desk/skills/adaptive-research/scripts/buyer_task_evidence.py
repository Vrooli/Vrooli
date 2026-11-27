#!/usr/bin/env python3
"""Qualify captured buyer-task observations without transferring identity or task claims.

Input is a JSON object with ``candidates`` and ``observations``. This is a
read-only reducer: it retains every observation, emits field-level gaps, and
never aggregates unknown or contradictory reports into a score.
"""
from __future__ import annotations

import argparse
import json
import re
from collections import defaultdict
from pathlib import Path
from typing import Any


REQUIRED_PROVENANCE = (
    "source_url", "source_family", "source_record", "observed_date", "author_id",
    "work_id", "edition_id", "task_match", "firsthand", "purchase_status",
    "independence_basis",
)


def known(value: Any) -> bool:
    return str(value or "").strip().casefold() not in {"", "unknown", "u", "none", "unfixed", "not applicable"}


def classify(candidate: dict[str, Any], observation: dict[str, Any]) -> dict[str, Any]:
    missing = [field for field in REQUIRED_PROVENANCE
               if not known(observation.get(field))]
    task_match = observation.get("task_match", "unknown")
    candidate_id = candidate.get("edition_id")
    exact = bool(known(candidate_id) and known(candidate.get("work_id"))
                 and observation.get("edition_id") == candidate_id
                 and observation.get("work_id") == candidate.get("work_id"))
    firsthand = observation.get("firsthand") is True
    task_qualified = task_match == "exact"
    purchased = observation.get("purchase_status") == "verified_purchase"
    independent = observation.get("independence_basis") == "verified_distinct"
    source_traceable = (known(observation.get("source_url"))
                        and known(observation.get("source_family"))
                        and known(observation.get("source_record")))

    if exact and firsthand and task_qualified and purchased and independent and not missing:
        disposition, code = "accepted", "C"
        reason = "exact candidate work/edition, first-hand exact task, verified purchase and independent respondent"
    elif (known(observation.get("work_id")) or known(observation.get("edition_id"))) and firsthand and task_match in {"exact", "close"} and source_traceable:
        disposition, code = "near-match", "X"
        reason = "useful first-hand task observation; candidate identity, purchase, task, or respondent provenance is incomplete or mismatched"
    elif observation.get("status") in {"no_result", "unreadable"}:
        disposition, code = "rejected", "U"
        reason = "no classifiable observation; retain as unknown, not as absence of need"
    else:
        disposition, code = "rejected", "T"
        reason = "context only or provenance/task mismatch; not admitted as buyer outcome"

    gaps = list(missing)
    if not exact:
        gaps.append("exact_work_and_edition_match")
    if not firsthand:
        gaps.append("firsthand_use")
    if not task_qualified:
        gaps.append("exact_frozen_task_match")
    if not purchased:
        gaps.append("verified_purchase")
    if not independent:
        gaps.append("respondent_independence")

    return {
        "observation_id": observation.get("observation_id", ""),
        "row_id": candidate["row_id"],
        "disposition": disposition,
        "admission_code": code,
        "reason": reason,
        "gaps": sorted(set(gaps)),
        "capture_status": observation.get("status", "observed"),
        "capture_code": observation.get("capture_code", ""),
        "source_url": observation.get("source_url", ""),
        "source_family": observation.get("source_family", ""),
        "observed_date": observation.get("observed_date", "unknown"),
        "author_id": observation.get("author_id", "unknown"),
        "work_id": observation.get("work_id", ""),
        "edition_id": observation.get("edition_id", ""),
        "task_match": task_match,
        "firsthand": observation.get("firsthand", "unknown"),
        "purchase_status": observation.get("purchase_status", "unknown"),
        "independence_basis": observation.get("independence_basis", "unknown"),
        "outcome": observation.get("outcome", "unknown"),
        "claim_axis": observation.get("claim_axis", ""),
        "polarity": observation.get("polarity", ""),
        "mismatch": observation.get("mismatch", ""),
        "source_record": observation.get("source_record", ""),
    }


def qualify(payload: dict[str, Any]) -> dict[str, Any]:
    validation = validate_input(payload)
    candidates = {str(row["row_id"]): row for row in payload.get("candidates", [])
                  if isinstance(row, dict) and known(row.get("row_id"))}
    by_row: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for observation in payload.get("observations", []):
        if not isinstance(observation, dict):
            continue
        row_id = str(observation.get("row_id", ""))
        if row_id not in candidates:
            by_row[row_id].append({"disposition": "rejected", "admission_code": "T",
                                   "reason": "observation references an unknown candidate row",
                                   "source_record": observation.get("source_record", "")})
            continue
        by_row[row_id].append(classify(candidates[row_id], observation))

    rows = []
    for row_id, candidate in candidates.items():
        observations = by_row.get(row_id, [])
        accepted = [item for item in observations if item.get("disposition") == "accepted"]
        near = [item for item in observations if item.get("disposition") == "near-match"]
        candidate_firsthand = [item for item in observations
                               if item.get("work_id") == candidate.get("work_id")
                               and item.get("edition_id") == candidate.get("edition_id")
                               and item.get("firsthand") is True]
        unmet_fields = sorted({gap for item in candidate_firsthand for gap in item.get("gaps", [])})
        if not candidate_firsthand and not accepted:
            unmet_fields = ["firsthand_exact_candidate_task_observation",
                            "verified_purchase", "respondent_independence"]
        polarities: dict[tuple[str, str, str], set[str]] = defaultdict(set)
        for item in observations:
            axis = item.get("claim_axis")
            polarity = item.get("polarity")
            scope = (item.get("work_id", ""), item.get("edition_id", ""))
            if axis and all(scope) and polarity in {"supports", "opposes"}:
                polarities[(axis, *scope)].add(polarity)
        rows.append({
            "row_id": row_id,
            "candidate": candidate.get("label", ""),
            "candidate_work_id": candidate.get("work_id", ""),
            "candidate_edition_id": candidate.get("edition_id", ""),
            "frozen_task": candidate.get("task", ""),
            "candidate_outcome": "observed" if accepted else "U",
            "accepted_count": len(accepted),
            "near_match_count": len(near),
            "contradictory_observations_preserved": any(
                len(values) > 1 for values in polarities.values()),
            "observations": observations,
            "unmet_fields": [] if accepted else unmet_fields,
        })
    return {"schema_version": 1, "read_only": True,
            "source_documents": payload.get("source_documents", []),
            "validation": validation, "rows": rows}


def import_a196(path: Path, candidates: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Import the six-row A196 route table and its bounded catalog no-result screens."""
    text = path.read_text(encoding="utf-8")
    imported: list[dict[str, Any]] = []
    route_rows: set[str] = set()
    table = False
    for line in text.splitlines():
        if line.startswith("| Frozen row |"):
            table = True
            continue
        if table and line.startswith("|---"):
            continue
        if table and line.startswith("|"):
            cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
            if len(cells) != 3:
                continue
            match = re.match(r"(E1|E2|E3|S1|S2|N23)\s*[·-]", cells[0])
            if not match:
                continue
            row_id = match.group(1)
            if row_id in route_rows:
                raise ValueError(f"A196 route table repeats frozen row {row_id}")
            route_rows.add(row_id)
            code_match = re.search(r"`([CXTU])`", cells[2])
            code = code_match.group(1) if code_match else "U"
            url_match = re.search(r"\[[^\]]+\]\((https?://[^)]+)\)", cells[1])
            source_url = url_match.group(1) if url_match else ""
            label_match = re.search(r"asked\s+(\d{4}-\d{2}-\d{2})", cells[1])
            observed_date = label_match.group(1) if label_match else "unknown"
            imported.append({
                "observation_id": f"A196-{row_id}-SE-import",
                "row_id": row_id,
                "source_record": f"{path.name} · Applied route screen · {row_id}",
                "source_url": source_url,
                "source_family": "Stack Exchange",
                "observed_date": observed_date,
                "author_id": "public poster; identity not verified" if source_url else "unknown",
                "work_id": "",
                "edition_id": "",
                "task_match": "close" if code == "T" else "unknown",
                "firsthand": "unknown",
                "purchase_status": "unknown",
                "independence_basis": "respondent and real-world uniqueness unverified",
                "outcome": re.sub(r"\[[^\]]+\]\((?:https?://[^)]+)\)", "linked public post", cells[1]),
                "mismatch": cells[2],
                "status": "no_result" if code == "U" else "observed",
                "capture_code": code,
            })
        elif table and line.startswith("## Result and current-field handling"):
            table = False

    expected_rows = {str(row.get("row_id", "")) for row in candidates if known(row.get("row_id"))}
    missing_route_rows = expected_rows - route_rows
    if missing_route_rows:
        raise ValueError("A196 applied route screen is missing candidate row(s): " + ", ".join(sorted(missing_route_rows)))

    candidate_by_isbn = {str(row.get("work_id", "")): row for row in candidates
                         if re.fullmatch(r"\d{13}", str(row.get("work_id", "")))}
    archive_section = text.split("Source family 2:", 1)[1].split("Source family 3:", 1)[0] if "Source family 2:" in text and "Source family 3:" in text else ""
    archive_isbns = set(re.findall(r"\b\d{13}\b", archive_section))
    for isbn, candidate in candidate_by_isbn.items():
        if isbn in archive_isbns:
            imported.append({
                "observation_id": f"A196-{candidate['row_id']}-IA-import",
                "row_id": candidate["row_id"],
                "source_record": f"{path.name} · Internet Archive exact-ISBN review screen",
                "source_family": "Internet Archive item reviews",
                "author_id": "no buyer author; bounded search returned no item review",
                "work_id": isbn,
                "edition_id": isbn,
                "task_match": "unknown",
                "firsthand": False,
                "purchase_status": "unknown",
                "independence_basis": "no buyer observation found",
                "outcome": "No matching item-review result in the bounded exact-ISBN search.",
                "mismatch": "No result is not evidence of no need or no demand.",
                "status": "no_result",
            })

    google_section = text.split("Source family 3:", 1)[1].split("## Result and current-field handling", 1)[0] if "Source family 3:" in text else ""
    google_match = re.search(r"\[([^\]]+)\]\((https?://[^)]+)\).*?E2's ISBN `?(\d{13})`?", google_section, re.S)
    google_isbn = ""
    if google_match:
        title, url, isbn = google_match.groups()
        google_isbn = isbn
        candidate = candidate_by_isbn.get(isbn)
        if candidate:
            imported.append({
                "observation_id": f"A196-{candidate['row_id']}-GB-import",
                "row_id": candidate["row_id"],
                "source_record": f"{path.name} · Google Books catalog crosswalk",
                "source_url": url,
                "source_family": "Google Books catalog",
                "observed_date": "2026-11-24",
                "author_id": "no buyer author",
                "work_id": isbn,
                "edition_id": isbn,
                "task_match": "unknown",
                "firsthand": False,
                "purchase_status": "not applicable: catalog record",
                "independence_basis": "not a buyer report",
                "outcome": f"Bibliographic crosswalk identifies {title}; no user review or task outcome.",
                "mismatch": "Catalog identity only; not buyer evidence.",
            })
    for isbn, candidate in candidate_by_isbn.items():
        if isbn == google_isbn:
            continue
        imported.append({
            "observation_id": f"A196-{candidate['row_id']}-GB-import",
            "row_id": candidate["row_id"],
            "source_record": f"{path.name} · Google Books exact-ISBN search",
            "source_family": "Google Books catalog",
            "work_id": isbn,
            "edition_id": isbn,
            "task_match": "unknown",
            "firsthand": False,
            "purchase_status": "unknown",
            "independence_basis": "no buyer observation found",
            "outcome": "No Google Books catalog result returned for this fixed ISBN in the bounded search.",
            "mismatch": "No result does not establish absence of demand.",
            "status": "no_result",
        })

    if "N23 has no fixed ISBN" in text:
        n23 = next((row for row in candidates if row.get("row_id") == "N23"), None)
        if n23:
            imported.append({
                "observation_id": "A196-N23-IA-import",
                "row_id": "N23",
                "source_record": f"{path.name} · Internet Archive identity constraint",
                "source_family": "Internet Archive item reviews",
                "author_id": "no buyer author",
                "work_id": n23.get("work_id", ""),
                "edition_id": n23.get("edition_id", ""),
                "task_match": "unknown",
                "firsthand": False,
                "purchase_status": "unknown",
                "independence_basis": "no item-keyed buyer observation available",
                "outcome": "No fixed ISBN exists for this hypothesis; an edition-keyed Archive lookup was not possible.",
                "mismatch": "Candidate identity is not edition-fixed.",
                "status": "unreadable",
            })
    return imported


def validate_input(payload: dict[str, Any]) -> dict[str, Any]:
    errors: list[str] = []
    warnings: list[str] = []
    candidate_rows = payload.get("candidates")
    observations = payload.get("observations")
    if not isinstance(candidate_rows, list) or not candidate_rows:
        errors.append("candidates must be a non-empty array")
        candidate_rows = []
    if not isinstance(observations, list):
        errors.append("observations must be an array")
        observations = []

    row_ids: set[str] = set()
    for index, candidate in enumerate(candidate_rows):
        if not isinstance(candidate, dict):
            errors.append(f"candidate[{index}] must be an object")
            continue
        for field in ("row_id", "label", "work_id", "edition_id", "task"):
            # Candidate records may explicitly retain U/unfixed edition fields.
            # The field itself must exist; its value can remain unknown.
            if candidate.get(field) in (None, ""):
                errors.append(f"candidate[{index}] missing {field}")
        row_id = str(candidate.get("row_id", ""))
        if row_id in row_ids:
            errors.append(f"duplicate candidate row_id: {row_id}")
        row_ids.add(row_id)

    observation_ids: set[str] = set()
    for index, observation in enumerate(observations):
        if not isinstance(observation, dict):
            errors.append(f"observation[{index}] must be an object")
            continue
        for field in ("observation_id", "row_id", "source_record", "source_family"):
            if not known(observation.get(field)):
                errors.append(f"observation[{index}] missing {field}")
        observation_id = str(observation.get("observation_id", ""))
        if observation_id in observation_ids:
            errors.append(f"duplicate observation_id: {observation_id}")
        observation_ids.add(observation_id)
        if observation.get("row_id") not in row_ids:
            errors.append(f"observation[{index}] references unknown row_id: {observation.get('row_id', '')}")
        if observation.get("status") not in {"no_result", "unreadable"} and not known(observation.get("source_url")):
            warnings.append(f"observation {observation_id} has no source_url and cannot be admitted")

    return {"status": "partial" if errors else "ok", "errors": errors, "warnings": warnings}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("--a196", type=Path, help="A196 markdown capture record to import directly")
    parser.add_argument("-o", "--output", type=Path)
    args = parser.parse_args()
    payload = json.loads(args.input.read_text(encoding="utf-8"))
    if args.a196:
        documents = payload.setdefault("source_documents", [])
        if not any(str(document).endswith(args.a196.name) for document in documents):
            documents.append(args.a196.as_posix())
        payload["observations"] = payload.get("observations", []) + import_a196(args.a196, payload.get("candidates", []))
    result = qualify(payload)
    serialized = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
    if args.output:
        args.output.write_text(serialized, encoding="utf-8")
    else:
        print(serialized, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
