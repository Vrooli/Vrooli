#!/usr/bin/env python3
"""Inlines fixture references (@fixture/<slug>) inside workflow JSON definitions."""
from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any, Dict, Set


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Resolve BAS workflow fixtures")
    parser.add_argument("--workflow", required=True, help="Path to workflow JSON file")
    parser.add_argument("--scenario", help="Scenario directory (defaults to workflow dir)")
    parser.add_argument("--fixtures", help="Override fixtures directory")
    return parser.parse_args()


def load_json(path: Path) -> Dict[str, Any]:
    try:
        return json.loads(path.read_text())
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"Failed to parse JSON from {path}: {exc}") from exc


def clone(obj: Any) -> Any:
    return json.loads(json.dumps(obj))


def load_fixtures(directory: Path) -> Dict[str, Dict[str, Any]]:
    fixtures: Dict[str, Dict[str, Any]] = {}
    if not directory.exists():
        return fixtures
    for root, _dirs, files in os.walk(directory):
        for name in files:
            if not name.endswith(".json"):
                continue
            path = Path(root) / name
            data = load_json(path)
            metadata = data.get("metadata") or {}
            fixture_id = metadata.get("fixture_id") or metadata.get("fixtureId")
            if not fixture_id or not isinstance(fixture_id, str):
                raise RuntimeError(f"Fixture {path} is missing metadata.fixture_id")
            if fixture_id in fixtures:
                other = fixtures[fixture_id]["source"]
                raise RuntimeError(f"Duplicate fixture id '{fixture_id}' in {path} (already defined in {other})")
            definition = {k: v for k, v in data.items() if k != "metadata"}
            nodes = definition.get("nodes")
            if not isinstance(nodes, list) or not nodes:
                raise RuntimeError(f"Fixture {path} must define at least one node")
            if not isinstance(definition.get("edges"), list):
                definition["edges"] = []
            fixtures[fixture_id] = {"definition": definition, "source": str(path)}
    return fixtures


def resolve_definition(definition: Dict[str, Any], fixtures: Dict[str, Dict[str, Any]], used: Set[str]) -> None:
    nodes = definition.get("nodes")
    if not isinstance(nodes, list):
        return
    for node in nodes:
        if not isinstance(node, dict):
            continue
        data = node.get("data")
        if not isinstance(data, dict):
            continue
        workflow_id = data.get("workflowId")
        if isinstance(workflow_id, str) and workflow_id.startswith("@fixture/"):
            slug = workflow_id.split("/", 1)[1]
            fixture = fixtures.get(slug)
            if not fixture:
                raise RuntimeError(f"Workflow references unknown fixture '{slug}'")
            used.add(slug)
            nested = clone(fixture["definition"])
            resolve_definition(nested, fixtures, used)
            data["workflowDefinition"] = nested
            data.pop("workflowId", None)
        elif isinstance(data.get("workflowDefinition"), dict):
            resolve_definition(data["workflowDefinition"], fixtures, used)


def main() -> None:
    args = parse_args()
    workflow_path = Path(args.workflow).resolve()
    if args.scenario:
        scenario_dir = Path(args.scenario).resolve()
    else:
        scenario_dir = workflow_path.parent
    fixtures_dir = Path(args.fixtures).resolve() if args.fixtures else scenario_dir / "test" / "playbooks" / "__subflows"

    fixtures = load_fixtures(fixtures_dir)
    workflow = load_json(workflow_path)
    used: Set[str] = set()
    resolve_definition(workflow, fixtures, used)
    json.dump(workflow, sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
