#!/usr/bin/env python3
"""Inlines fixture references (@fixture/<slug>) inside workflow JSON definitions."""
from __future__ import annotations

import argparse
import ast
import json
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional, Set, Tuple


SELECTOR_CACHE: Dict[str, Dict[str, Any]] = {}
MANIFEST_FILENAME = "selectors.manifest.json"
SELECTOR_TOKEN_PATTERN = re.compile(r"@selector/([A-Za-z0-9_.-]+)(\([^)]*\))?")
FIXTURE_TOKEN_PATTERN = re.compile(r"^@fixture/([A-Za-z0-9_.-]+)(\s*\(.*\))?$")
SEED_TOKEN_PATTERN = re.compile(r"^@seed/([A-Za-z0-9_.-]+)$")
DATA_TESTID_PATTERN = re.compile(r"\[data-testid[^\]]*\]")
NUMBER_PATTERN = re.compile(r"^-?\d+(?:\.\d+)?$")
BOOLEAN_TRUE = {"true", "1", "yes", "on"}
BOOLEAN_FALSE = {"false", "0", "no", "off"}
SEED_STATE_RELATIVE_PATH = Path("test") / "artifacts" / "runtime" / "seed-state.json"
RESET_LEVELS = {"none": 0, "project": 1, "global": 2}
DEFAULT_RESET_LEVEL = "project"


@dataclass(frozen=True)
class FixtureParameterDefinition:
    name: str
    type: str = "string"
    required: bool = False
    default: Any = None
    enum_values: Optional[List[str]] = None
    description: str = ""


@dataclass
class FixtureMetadata:
    description: str
    fixture_id: str
    requirements: List[str]
    parameters: List[FixtureParameterDefinition]
    reset: Optional[str]


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


def load_seed_state(scenario_dir: Path) -> Dict[str, Any]:
    seed_path = scenario_dir / SEED_STATE_RELATIVE_PATH
    if not seed_path.exists():
        return {}
    data = load_json(seed_path)
    if not isinstance(data, dict):
        raise RuntimeError(f"Seed state at {seed_path} must be a JSON object")
    return data


def normalize_fixture_parameters(metadata: Dict[str, Any], source: Path) -> List[FixtureParameterDefinition]:
    raw_params = metadata.get("parameters") or []
    normalized: List[FixtureParameterDefinition] = []
    if not raw_params:
        return normalized
    if not isinstance(raw_params, list):
        raise RuntimeError(f"Fixture {source} metadata.parameters must be an array")
    seen: Set[str] = set()
    for idx, entry in enumerate(raw_params):
        pointer = f"metadata.parameters[{idx}]"
        if not isinstance(entry, dict):
            raise RuntimeError(f"Fixture {source} {pointer} must be an object")
        name = entry.get("name")
        if not name or not isinstance(name, str):
            raise RuntimeError(f"Fixture {source} {pointer} is missing a string 'name'")
        if name in seen:
            raise RuntimeError(f"Fixture {source} defines duplicate parameter '{name}'")
        seen.add(name)
        param_type = (entry.get("type") or "string").lower()
        if param_type not in {"string", "number", "boolean", "enum"}:
            raise RuntimeError(
                f"Fixture {source} parameter '{name}' has unsupported type '{param_type}'"
            )
        enum_values: Optional[List[str]] = None
        if param_type == "enum":
            values = entry.get("enumValues") or entry.get("enum_values")
            if not isinstance(values, list) or not values:
                raise RuntimeError(
                    f"Fixture {source} parameter '{name}' requires non-empty enumValues list"
                )
            enum_values = []
            for raw_value in values:
                if not isinstance(raw_value, (str, int, float)):
                    raise RuntimeError(
                        f"Fixture {source} parameter '{name}' enumValues must be strings or numbers"
                    )
                enum_values.append(str(raw_value))
        required = bool(entry.get("required", False))
        default = entry.get("default")
        if required and default is not None:
            required = False
        if default is not None:
            default = coerce_parameter_value(param_type, default, name, str(source), enum_values)
        description = entry.get("description") if isinstance(entry.get("description"), str) else ""
        normalized.append(
            FixtureParameterDefinition(
                name=name,
                type=param_type,
                required=required,
                default=default,
                enum_values=enum_values,
                description=description,
            )
        )
    return normalized


def normalize_fixture_metadata(path: Path, metadata: Dict[str, Any]) -> FixtureMetadata:
    description = metadata.get("description") if isinstance(metadata.get("description"), str) else ""
    fixture_id = metadata.get("fixture_id") or metadata.get("fixtureId")
    if not fixture_id or not isinstance(fixture_id, str):
        raise RuntimeError(f"Fixture {path} is missing metadata.fixture_id")
    raw_requirements = metadata.get("requirements") or []
    requirements: List[str] = []
    if raw_requirements:
        if not isinstance(raw_requirements, list):
            raise RuntimeError(f"Fixture {path} metadata.requirements must be an array of strings")
        for idx, req in enumerate(raw_requirements):
            if not isinstance(req, str):
                raise RuntimeError(
                    f"Fixture {path} metadata.requirements[{idx}] must be a string requirement id"
                )
            requirements.append(req)
    parameters = normalize_fixture_parameters(metadata, path)
    reset_value = metadata.get("reset")
    if reset_value is not None and not isinstance(reset_value, str):
        raise RuntimeError(f"Fixture {path} metadata.reset must be a string if provided")
    if isinstance(reset_value, str):
        normalized_reset = reset_value.strip().lower()
        if normalized_reset not in RESET_LEVELS:
            allowed = ", ".join(sorted(RESET_LEVELS.keys()))
            raise RuntimeError(
                f"Fixture {path} metadata.reset must be one of: {allowed}"
            )
        reset_value = normalized_reset

    return FixtureMetadata(
        description=description,
        fixture_id=fixture_id,
        requirements=requirements,
        parameters=parameters,
        reset=reset_value,
    )


def normalize_reset_value(raw: Optional[str], context: str) -> Optional[str]:
    if raw is None:
        return None
    if not isinstance(raw, str):
        raise RuntimeError(f"Reset metadata at {context} must be a string")
    normalized = raw.strip().lower()
    if not normalized:
        return None
    if normalized not in RESET_LEVELS:
        allowed = ", ".join(sorted(RESET_LEVELS.keys()))
        raise RuntimeError(f"Reset metadata at {context} must be one of: {allowed}")
    return normalized


def merge_reset_values(current: Optional[str], candidate: Optional[str]) -> Optional[str]:
    if candidate is None:
        return current
    if current is None:
        return candidate
    if RESET_LEVELS[candidate] > RESET_LEVELS[current]:
        return candidate
    return current


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
            normalized_metadata = normalize_fixture_metadata(path, metadata)
            fixture_id = normalized_metadata.fixture_id
            if fixture_id in fixtures:
                other = fixtures[fixture_id]["source"]
                raise RuntimeError(f"Duplicate fixture id '{fixture_id}' in {path} (already defined in {other})")
            definition = {k: v for k, v in data.items() if k != "metadata"}
            nodes = definition.get("nodes")
            if not isinstance(nodes, list) or not nodes:
                raise RuntimeError(f"Fixture {path} must define at least one node")
            if not isinstance(definition.get("edges"), list):
                definition["edges"] = []
            fixtures[fixture_id] = {
                "definition": definition,
                "source": str(path),
                "metadata": normalized_metadata,
            }
    return fixtures


FIXTURE_TEMPLATE_PATTERN = re.compile(r"\$\{fixture\.([A-Za-z0-9_]+)\}")


def coerce_parameter_value(
    param_type: str,
    value: Any,
    name: str,
    source: str,
    enum_values: Optional[List[str]] = None,
) -> Any:
    if param_type == "string":
        if isinstance(value, (str, int, float, bool)):
            return str(value)
        raise RuntimeError(
            f"Fixture parameter '{name}' in {source} must be a string-compatible value"
        )
    if param_type == "number":
        if isinstance(value, (int, float)):
            return value
        if isinstance(value, str) and NUMBER_PATTERN.fullmatch(value):
            if value.isdigit() or (value.startswith("-") and value[1:].isdigit()):
                return int(value)
            return float(value)
        raise RuntimeError(f"Fixture parameter '{name}' in {source} must be numeric")
    if param_type == "boolean":
        if isinstance(value, bool):
            return value
        if isinstance(value, str):
            normalized = value.strip().lower()
            if normalized in BOOLEAN_TRUE:
                return True
            if normalized in BOOLEAN_FALSE:
                return False
        raise RuntimeError(f"Fixture parameter '{name}' in {source} must be boolean (true/false)")
    if param_type == "enum":
        if not enum_values:
            raise RuntimeError(f"Fixture parameter '{name}' in {source} enumValues not configured")
        candidate = str(value)
        if candidate not in enum_values:
            raise RuntimeError(
                f"Fixture parameter '{name}' in {source} must be one of: {', '.join(enum_values)}"
            )
        return candidate
    return value


def parse_fixture_reference(raw: str, pointer: str) -> Tuple[str, Optional[str]]:
    match = FIXTURE_TOKEN_PATTERN.match(raw.strip())
    if not match:
        raise RuntimeError(f"Invalid fixture reference '{raw}' at {pointer}")
    slug = match.group(1)
    args_segment = match.group(2)
    return slug, args_segment


def _split_argument_entries(inner: str, pointer: str) -> List[str]:
    entries: List[str] = []
    current: List[str] = []
    quote: Optional[str] = None
    escape = False
    for char in inner:
        if escape:
            current.append(char)
            escape = False
            continue
        if quote:
            if char == "\\":
                escape = True
                continue
            if char == quote:
                quote = None
            current.append(char)
            continue
        if char in {'"', "'"}:
            quote = char
            current.append(char)
            continue
        if char == ',':
            entry = ''.join(current).strip()
            if entry:
                entries.append(entry)
            current = []
            continue
        current.append(char)
    if quote:
        raise RuntimeError(f"Unterminated quote in fixture parameters at {pointer}")
    tail = ''.join(current).strip()
    if tail:
        entries.append(tail)
    return entries


def parse_fixture_arguments(argument_segment: Optional[str], pointer: str) -> Dict[str, str]:
    if not argument_segment:
        return {}
    content = argument_segment.strip()
    if not content:
        return {}
    if not content.startswith("(") or not content.endswith(")"):
        raise RuntimeError(
            f"Fixture parameters must use parentheses syntax at {pointer}: {argument_segment}"
        )
    inner = content[1:-1].strip()
    if not inner:
        return {}
    assignments: Dict[str, str] = {}
    for entry in _split_argument_entries(inner, pointer):
        if "=" not in entry:
            raise RuntimeError(
                f"Invalid fixture parameter '{entry}' at {pointer}; expected key=value"
            )
        name, raw_value = entry.split("=", 1)
        clean_name = name.strip()
        if not clean_name:
            raise RuntimeError(f"Fixture parameter missing name at {pointer}")
        value = raw_value.strip()
        if not value:
            raise RuntimeError(
                f"Fixture parameter '{clean_name}' missing value at {pointer}"
            )
        if (value.startswith('"') and value.endswith('"')) or (value.startswith("'") and value.endswith("'")):
            try:
                value = ast.literal_eval(value)
            except Exception as exc:  # noqa: BLE001
                raise RuntimeError(
                    f"Fixture parameter '{clean_name}' has invalid quoted value at {pointer}: {exc}"
                ) from exc
        assignments[clean_name] = str(value)
    return assignments


def resolve_seed_reference(value: Any, seed_state: Dict[str, Any], pointer: str) -> Any:
    if isinstance(value, str):
        match = SEED_TOKEN_PATTERN.match(value.strip())
        if match:
            key = match.group(1)
            if key not in seed_state:
                raise RuntimeError(
                    f"Seed state missing key '{key}' required at {pointer}. Re-run seeds via test/playbooks/__seeds/apply.sh."
                )
            return seed_state[key]
    return value


def resolve_fixture_parameters(
    fixture: Dict[str, Any],
    provided: Dict[str, str],
    pointer: str,
    seed_state: Dict[str, Any],
) -> Dict[str, Any]:
    resolved: Dict[str, Any] = {}
    metadata: FixtureMetadata = fixture.get("metadata")
    parameter_defs = metadata.parameters if metadata else []
    known_names = {param.name for param in parameter_defs}
    extras = [name for name in provided.keys() if name not in known_names]
    if extras:
        raise RuntimeError(
            "Fixture '"
            + metadata.fixture_id
            + "' received unknown parameter(s): "
            + ", ".join(extras)
            + f" ({pointer})"
        )
    for param_def in parameter_defs:
        raw_value = provided.get(param_def.name)
        if raw_value is None:
            if param_def.default is not None:
                default_value = resolve_seed_reference(param_def.default, seed_state, pointer)
                resolved[param_def.name] = default_value
            elif param_def.required:
                raise RuntimeError(
                    f"Fixture '{metadata.fixture_id}' missing required parameter '{param_def.name}' ({pointer})"
                )
            continue
        normalized_value = resolve_seed_reference(raw_value, seed_state, pointer)
        resolved[param_def.name] = coerce_parameter_value(
            param_def.type,
            normalized_value,
            param_def.name,
            fixture.get("source", pointer),
            param_def.enum_values,
        )
    return resolved


def apply_fixture_templates(definition: Any, parameters: Dict[str, Any], pointer: str = "") -> Any:
    if isinstance(definition, dict):
        for key, value in definition.items():
            next_pointer = f"{pointer}/{key}" if pointer else f"/{key}"
            definition[key] = apply_fixture_templates(value, parameters, next_pointer)
        return definition
    if isinstance(definition, list):
        for idx, value in enumerate(definition):
            next_pointer = f"{pointer}/{idx}" if pointer else f"/{idx}"
            definition[idx] = apply_fixture_templates(value, parameters, next_pointer)
        return definition
    if isinstance(definition, str):
        def _replace(match: re.Match[str]) -> str:
            param_name = match.group(1)
            if param_name not in parameters:
                raise RuntimeError(
                    f"Fixture template references unknown parameter '{param_name}' at {pointer}"
                )
            return str(parameters[param_name])

        return FIXTURE_TEMPLATE_PATTERN.sub(_replace, definition)
    return definition


def annotate_workflow_metadata(root: Dict[str, Any], fixture_requirements: Set[str]) -> None:
    if not fixture_requirements:
        return
    metadata = root.get("metadata")
    if not isinstance(metadata, dict):
        metadata = {}
        root["metadata"] = metadata
    existing = metadata.get("requirementsFromFixtures")
    accumulator: Set[str] = set()
    if isinstance(existing, list):
        for req in existing:
            if isinstance(req, str):
                accumulator.add(req)
    accumulator.update(fixture_requirements)
    metadata["requirementsFromFixtures"] = sorted(accumulator)


def resolve_definition(
    definition: Dict[str, Any],
    fixtures: Dict[str, Dict[str, Any]],
    used: Set[str],
    fixture_requirements: Set[str],
    seed_state: Dict[str, Any],
    pointer: str = "",
) -> None:
    nodes = definition.get("nodes")
    if not isinstance(nodes, list):
        return
    aggregated_reset = None
    for index, node in enumerate(nodes):
        if not isinstance(node, dict):
            continue
        node_pointer = f"{pointer}/nodes/{index}"
        data = node.get("data")
        if not isinstance(data, dict):
            continue
        data_pointer = f"{node_pointer}/data"
        workflow_id = data.get("workflowId")
        if isinstance(workflow_id, str) and workflow_id.startswith("@fixture/"):
            slug, args_segment = parse_fixture_reference(workflow_id, f"{data_pointer}/workflowId")
            fixture = fixtures.get(slug)
            if not fixture:
                raise RuntimeError(f"Workflow references unknown fixture '{slug}' at {data_pointer}")
            used.add(slug)
            provided_args = parse_fixture_arguments(args_segment, f"{data_pointer}/workflowId")
            parameters = resolve_fixture_parameters(
                fixture,
                provided_args,
                f"{data_pointer}/workflowId",
                seed_state,
            )
            nested = clone(fixture["definition"])
            apply_fixture_templates(nested, parameters, node_pointer)
            metadata: FixtureMetadata = fixture.get("metadata")
            if metadata and metadata.requirements:
                fixture_requirements.update(metadata.requirements)
            if metadata and metadata.reset:
                aggregated_reset = merge_reset_values(
                    aggregated_reset,
                    normalize_reset_value(metadata.reset, f"{node_pointer}/data/workflowId"),
                )
            resolve_definition(nested, fixtures, used, fixture_requirements, seed_state, node_pointer)
            data["workflowDefinition"] = nested
            data.pop("workflowId", None)
            nested_reset = None
            nested_metadata = nested.get("metadata")
            if isinstance(nested_metadata, dict):
                nested_reset = normalize_reset_value(
                    nested_metadata.get("reset"), f"{node_pointer}/data/workflowDefinition"
                )
            aggregated_reset = merge_reset_values(aggregated_reset, nested_reset)
        elif isinstance(data.get("workflowDefinition"), dict):
            resolve_definition(
                data["workflowDefinition"], fixtures, used, fixture_requirements, seed_state, node_pointer
            )
            nested = data.get("workflowDefinition")
            nested_metadata = nested.get("metadata") if isinstance(nested, dict) else None
            if isinstance(nested_metadata, dict):
                aggregated_reset = merge_reset_values(
                    aggregated_reset,
                    normalize_reset_value(
                        nested_metadata.get("reset"), f"{node_pointer}/data/workflowDefinition"
                    ),
                )

    metadata = definition.get("metadata")
    if not isinstance(metadata, dict):
        metadata = {}
        definition["metadata"] = metadata
    base_context = f"{pointer}/metadata/reset" if pointer else "/metadata/reset"
    base_reset = None
    if "reset" in metadata:
        base_reset = normalize_reset_value(metadata.get("reset"), base_context)
    final_reset = merge_reset_values(base_reset, aggregated_reset)
    if final_reset is None:
        final_reset = DEFAULT_RESET_LEVEL
    metadata["reset"] = final_reset


def load_selector_registry(scenario_dir: Path) -> Dict[str, Any]:
    manifest_path = scenario_dir / "ui" / "src" / "consts" / MANIFEST_FILENAME
    if not manifest_path.exists():
        return {}
    cache_key = str(manifest_path)
    mtime = manifest_path.stat().st_mtime_ns
    cached = SELECTOR_CACHE.get(cache_key)
    if cached and cached.get("mtime") == mtime:
        return cached["data"]
    manifest = load_json(manifest_path)
    data = {"manifest": manifest, "path": str(manifest_path)}
    SELECTOR_CACHE[cache_key] = {"mtime": mtime, "data": data}
    return data


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
    manifest: Dict[str, Any],
    manifest_path: Path,
    pointer: str,
) -> str | None:
    dynamic_map = manifest.get("dynamicSelectors") or {}
    definition = dynamic_map.get(selector_key)
    if not definition:
        return None

    template = definition.get("selectorPattern")
    if not isinstance(template, str) or not template:
        raise RuntimeError(
            f"Selector '@selector/{selector_key}' is missing a selectorPattern in {manifest_path}"
        )

    param_configs = definition.get("params") or []
    has_args = bool(argument_segment and argument_segment.strip())

    if not param_configs:
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

    normalized: Dict[str, str] = {}
    expected_keys = {param.get("name"): param for param in param_configs if param.get("name")}

    missing = [name for name in expected_keys.keys() if name not in provided]
    if missing:
        raise RuntimeError(
            "Selector '@selector/"
            + selector_key
            + "' is missing parameter(s): "
            + ", ".join(missing)
            + f" ({pointer})"
        )

    extra = [name for name in provided.keys() if name not in expected_keys]
    if extra:
        raise RuntimeError(
            "Selector '@selector/"
            + selector_key
            + "' received unknown parameter(s): "
            + ", ".join(extra)
            + f" ({pointer})"
        )

    for name, config in expected_keys.items():
        value = provided.get(name, "")
        param_type = (config.get("type") or "string").lower()
        if param_type == "number" and not NUMBER_PATTERN.fullmatch(value):
            raise RuntimeError(
                "Selector '@selector/"
                + selector_key
                + "' parameter '"
                + name
                + "' must be numeric"
                + f" ({pointer})"
            )
        if param_type == "enum":
            allowed_values = config.get("values") or []
            if value not in allowed_values:
                raise RuntimeError(
                    "Selector '@selector/"
                    + selector_key
                    + "' parameter '"
                    + name
                    + "' must be one of: "
                    + ", ".join(map(str, allowed_values))
                    + f" ({pointer})"
                )
        normalized[name] = value

    resolved = substitute_template(template, normalized, pointer)
    return resolved


def inject_selector_references(
    definition: Any,
    manifest: Dict[str, Any],
    manifest_path: Path,
    pointer: str = "",
) -> Any:
    if isinstance(definition, dict):
        for key, value in definition.items():
            next_pointer = f"{pointer}/{key}" if pointer else f"/{key}"
            definition[key] = inject_selector_references(value, manifest, manifest_path, next_pointer)
        return definition
    if isinstance(definition, list):
        for idx, value in enumerate(definition):
            next_pointer = f"{pointer}/{idx}" if pointer else f"/{idx}"
            definition[idx] = inject_selector_references(value, manifest, manifest_path, next_pointer)
        return definition
    if isinstance(definition, str):
        selector_map = manifest.get("selectors", {})
        if manifest and DATA_TESTID_PATTERN.search(definition):
            raise RuntimeError(
                "Raw data-testid selector detected at "
                + pointer
                + "; replace it with an @selector/<key> token registered in "
                + str(manifest_path)
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
                    manifest,
                    manifest_path,
                    pointer,
                )
            else:
                entry = selector_map.get(selector_key)
                if entry is not None:
                    resolved = entry.get("selector")
                else:
                    resolved = build_dynamic_selector(
                        selector_key,
                        args_segment,
                        manifest,
                        manifest_path,
                        pointer,
                    )

            if resolved is None:
                raise RuntimeError(
                    "Selector reference '@selector/"
                    + selector_key
                    + "' not found. Define it in "
                    + str(manifest_path)
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
    fixture_requirements: Set[str] = set()
    seed_state = load_seed_state(scenario_dir)
    resolve_definition(workflow, fixtures, used, fixture_requirements, seed_state)
    annotate_workflow_metadata(workflow, fixture_requirements)
    selector_registry = load_selector_registry(scenario_dir)
    if selector_registry:
        manifest = selector_registry.get("manifest") or {}
        manifest_path = Path(selector_registry.get("path", "")) or scenario_dir / "ui" / "src" / "consts" / MANIFEST_FILENAME
        workflow = inject_selector_references(workflow, manifest, manifest_path)
    json.dump(workflow, sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
