#!/usr/bin/env python3
"""Inlines fixture references (@fixture/<slug>) inside workflow JSON definitions."""
from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, Set


SELECTOR_CACHE: Dict[str, Dict[str, Any]] = {}
REGISTRY_SCRIPT = Path(__file__).resolve().parent / "selector-registry.js"
SELECTOR_TOKEN_PATTERN = re.compile(r"@selector/([A-Za-z0-9_.-]+)(\([^)]*\))?")
DATA_TESTID_PATTERN = re.compile(r"\[data-testid[^\]]*\]")


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


def load_selector_registry(scenario_dir: Path) -> Dict[str, Any]:
    selectors_map: Dict[str, Any] = {}
    selectors_path = scenario_dir / "ui" / "src" / "consts" / "selectors.ts"
    if not selectors_path.exists():
        return selectors_map
    cache_key = str(selectors_path)
    mtime = selectors_path.stat().st_mtime_ns
    cached = SELECTOR_CACHE.get(cache_key)
    if cached and cached.get("mtime") == mtime:
        return cached["data"]
    if not REGISTRY_SCRIPT.exists():
        raise RuntimeError(f"Selector registry helper missing at {REGISTRY_SCRIPT}")
    try:
        completed = subprocess.run(
            ["node", str(REGISTRY_SCRIPT), "--scenario", str(scenario_dir)],
            capture_output=True,
            text=True,
            check=True,
        )
    except subprocess.CalledProcessError as exc:
        raise RuntimeError(
            f"Failed to load selector registry: {exc.stderr.strip() or exc.stdout.strip() or exc}"
        ) from exc
    data = json.loads(completed.stdout)
    SELECTOR_CACHE[cache_key] = {"mtime": mtime, "data": data}
    return data


def resolve_selector_key(key: str, tree: Dict[str, Any]) -> str | None:
    if not isinstance(tree, dict):
        return None
    segments = [segment.strip() for segment in re.split(r"[./]", key) if segment.strip()]
    if not segments:
        return None
    current: Any = tree
    for segment in segments:
        if isinstance(current, dict) and segment in current:
            current = current[segment]
        else:
            return None
    return current if isinstance(current, str) else None


def parse_selector_arguments(argument_segment: str, pointer: str) -> Dict[str, str]:
    content = argument_segment.strip()
    if not content.startswith("(") or not content.endswith(")"):
        raise RuntimeError(
            f"Selector parameters must use parentheses syntax at {pointer}: {argument_segment}"
        )
    inner = content[1:-1].strip()
    if not inner:
        return {}
    assignments: Dict[str, str] = {}
    parts = [part.strip() for part in inner.split(",") if part.strip()]
    for part in parts:
        if "=" not in part:
            raise RuntimeError(
                f"Invalid selector parameter '{part}' at {pointer}; expected key=value"
            )
        name, value = part.split("=", 1)
        clean_name = name.strip()
        clean_value = value.strip()
        if (clean_value.startswith("\"") and clean_value.endswith("\"")) or (
            clean_value.startswith("'") and clean_value.endswith("'")
        ):
            clean_value = clean_value[1:-1]
        assignments[clean_name] = clean_value
    return assignments


TEMPLATE_PATTERN = re.compile(r"\$\{([^}]+)\}")


def substitute_template(template: str, values: Dict[str, str], pointer: str) -> str:
    def repl(match: re.Match[str]) -> str:
        key = match.group(1)
        if key not in values:
            raise RuntimeError(
                f"Selector template references parameter '{key}' that was not provided ({pointer})"
            )
        return values[key]

    return TEMPLATE_PATTERN.sub(repl, template)


def build_dynamic_selector(
    selector_key: str,
    argument_segment: str | None,
    selectors: Dict[str, Any],
    selectors_path: Path,
    pointer: str,
) -> str | None:
    dynamic_map = selectors.get("dynamicSelectors") or {}
    definition = dynamic_map.get(selector_key)
    if not definition:
        return None
    params = definition.get("params") or []

    template = definition.get("selectorPattern") or definition.get("pattern")
    if not isinstance(template, str) or not template:
        raise RuntimeError(
            f"Selector '@selector/{selector_key}' is missing a selectorPattern in {selectors_path}"
        )

    has_args = bool(argument_segment and argument_segment.strip())

    if not params:
        if has_args and argument_segment.strip() not in ("", "()"): 
            provided_args = parse_selector_arguments(argument_segment, pointer)
            if provided_args:
                raise RuntimeError(
                    f"Selector '@selector/{selector_key}' does not accept parameters ({pointer})"
                )
        if TEMPLATE_PATTERN.search(template):
            raise RuntimeError(
                f"Selector '@selector/{selector_key}' requires parameters but none were provided ({pointer})"
            )
        return template

    if not has_args:
        raise RuntimeError(
            f"Selector '@selector/{selector_key}' requires parameters ({pointer}); define them as @selector/{selector_key}(param=value)"
        )

    provided = parse_selector_arguments(argument_segment, pointer)

    missing = [param for param in params if param not in provided]
    if missing:
        raise RuntimeError(
            "Selector '@selector/"
            + selector_key
            + "' is missing parameter(s): "
            + ", ".join(missing)
            + f" ({pointer})"
        )

    extra = sorted(set(provided.keys()) - set(params))
    if extra:
        raise RuntimeError(
            "Selector '@selector/"
            + selector_key
            + "' received unknown parameter(s): "
            + ", ".join(extra)
            + f" ({pointer})"
        )

    allowed = definition.get("allowedValues") or {}
    for param, allowed_values in allowed.items():
        if param in provided and allowed_values and provided[param] not in allowed_values:
            raise RuntimeError(
                "Selector '@selector/"
                + selector_key
                + "' parameter '"
                + param
                + "' must be one of: "
                + ", ".join(allowed_values)
                + f" ({pointer})"
            )

    value_types = definition.get("valueTypes") or {}
    for param, expected in value_types.items():
        if param not in provided:
            continue
        value = provided[param]
        if expected == "number" and not re.fullmatch(r"\d+", value):
            raise RuntimeError(
                "Selector '@selector/"
                + selector_key
                + "' parameter '"
                + param
                + "' must be a positive integer"
                + f" ({pointer})"
            )

    resolved = substitute_template(template, provided, pointer)
    return resolved


def inject_selector_references(
    definition: Any,
    selectors: Dict[str, Any],
    selectors_path: Path,
    pointer: str = "",
) -> Any:
    if isinstance(definition, dict):
        for key, value in definition.items():
            next_pointer = f"{pointer}/{key}" if pointer else f"/{key}"
            definition[key] = inject_selector_references(value, selectors, selectors_path, next_pointer)
        return definition
    if isinstance(definition, list):
        for idx, value in enumerate(definition):
            next_pointer = f"{pointer}/{idx}" if pointer else f"/{idx}"
            definition[idx] = inject_selector_references(value, selectors, selectors_path, next_pointer)
        return definition
    if isinstance(definition, str):
        selector_tree = selectors.get("workflowSelectors", {})
        if selectors and DATA_TESTID_PATTERN.search(definition):
            raise RuntimeError(
                "Raw data-testid selector detected at "
                + pointer
                + "; replace it with an @selector/<key> token registered in "
                + str(selectors_path)
            )

        def _replace(match: re.Match[str]) -> str:
            selector_key = match.group(1).strip()
            args_segment = match.group(2)
            if not selector_key:
                raise RuntimeError(f"Empty selector reference at {pointer}")
            resolved = None
            if args_segment:
                resolved = build_dynamic_selector(
                    selector_key,
                    args_segment,
                    selectors,
                    selectors_path,
                    pointer,
                )
            else:
                resolved = resolve_selector_key(selector_key, selector_tree)
                if resolved is None:
                    resolved = build_dynamic_selector(
                        selector_key,
                        args_segment,
                        selectors,
                        selectors_path,
                        pointer,
                    )

            if resolved is None:
                raise RuntimeError(
                    "Selector reference '@selector/"
                    + selector_key
                    + "' not found. Define it in "
                    + str(selectors_path)
                )
            return resolved

        replaced = SELECTOR_TOKEN_PATTERN.sub(_replace, definition)
        return replaced
    return definition


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
    selector_registry = load_selector_registry(scenario_dir)
    if selector_registry:
        selectors_path = Path(selector_registry.get("selectorsPath", "")) or scenario_dir / "ui" / "src" / "consts" / "selectors.ts"
        workflow = inject_selector_references(workflow, selector_registry, selectors_path)
    json.dump(workflow, sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
