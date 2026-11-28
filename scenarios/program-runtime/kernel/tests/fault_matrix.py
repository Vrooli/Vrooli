"""Generate helper and parent-source child-envelope fault matrices.

Rows come from checked-in source/contract pairs, not a hand-maintained program
list. The helper matrix calls the production helper with synthetic handles.
The parent-source matrix executes each actual program source in SessionKernel,
injecting at its library boundary while stubbing every declared direct binding.
Both matrices are fixture-injected and do not qualify real provider behavior.
"""
from __future__ import annotations

import ast
import contextlib
import hashlib
import io
import json
import os
import re
import sys
import tempfile
from types import SimpleNamespace
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4]
KERNEL = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(KERNEL / "host"))
sys.path.insert(0, str(KERNEL))
from host.engine import Handle as RuntimeHandle, SessionKernel  # noqa: E402
from host.program_helper import ProgramHelper  # noqa: E402
from program_helper import ProgramHelper as RuntimeProgramHelper  # noqa: E402


class Handle:
    def __init__(self, rows, contract, program):
        self.rows = rows
        self.contract = contract
        self.program = program

    def count(self):
        return len(self.rows)

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return {"program": self.program, "version": "fixture", "digest": "sha256:generated", "output_contract": self.contract}


def capability_requirements(contract: dict, source_code: str) -> list[dict]:
    """Project declared binding and nested-library capabilities for the receipt."""
    required = [
        {"binding": item["id"], "effect": item["effect"]}
        for item in contract.get("bindings", [])
        if item.get("id") and item.get("effect")
    ]
    calls = sorted(set(re.findall(r"\blib\.([A-Za-z0-9_]+)\.([A-Za-z0-9_]+)", source_code)))
    required.extend({"library_program": f"{scenario.replace('_', '-')}.{name.replace('_', '-')}"} for scenario, name in calls)
    return required


def library_calls(source_code: str) -> list[tuple[str, str]]:
    tree = ast.parse(source_code)
    calls: set[tuple[str, str]] = set()
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        parts = []
        value = node.func
        while isinstance(value, ast.Attribute):
            parts.append(value.attr)
            value = value.value
        if not isinstance(value, ast.Name) or value.id != "lib":
            continue
        parts.reverse()
        if len(parts) >= 2:
            calls.add((parts[0].replace("_", "-"), parts[1].replace("_", "-")))
    return sorted(calls)


def child_envelope_dependencies(source_code: str) -> list[tuple[str, str]]:
    """Resolve library handles that actual owner source passes to child_envelope."""
    tree = ast.parse(source_code)

    def dependency(call: ast.AST) -> tuple[str, str] | None:
        if not isinstance(call, ast.Call):
            return None
        parts = []
        value = call.func
        while isinstance(value, ast.Attribute):
            parts.append(value.attr)
            value = value.value
        if not isinstance(value, ast.Name) or value.id != "lib":
            return None
        parts.reverse()
        if len(parts) < 2:
            return None
        return parts[0].replace("_", "-"), parts[1].replace("_", "-")

    handles: dict[str, tuple[str, str]] = {}
    aliases: dict[str, str] = {}
    dictionary_calls: dict[str, dict[str, tuple[str, str]]] = {}
    dictionary_sources: dict[str, str] = {}

    def dependencies_in(node: ast.AST) -> set[tuple[str, str]]:
        return {dep for item in ast.walk(node) if isinstance(item, ast.Call)
                for dep in [dependency(item)] if dep}

    def assignment_target(node: ast.AST) -> str | None:
        if isinstance(node, ast.Name):
            return node.id
        if isinstance(node, ast.Subscript) and isinstance(node.value, ast.Name):
            return node.value.id
        return None

    assigned_dependencies: dict[str, tuple[str, str]] = {}
    wrapper_names: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)) and any(
                isinstance(item, ast.Call) and isinstance(item.func, ast.Attribute)
                and isinstance(item.func.value, ast.Name) and item.func.value.id == "program"
                and item.func.attr == "child_envelope" for item in ast.walk(node)):
            wrapper_names.add(node.name)
        if isinstance(node, ast.Assign) and isinstance(node.value, ast.Name):
            target = next((item for item in node.targets if isinstance(item, ast.Name)), None)
            if target:
                aliases[target.id] = node.value.id
        if isinstance(node, ast.Assign) and isinstance(node.value, ast.Call):
            target = next((item for item in node.targets if isinstance(item, ast.Name)), None)
            dep = dependency(node.value)
            if target and dep:
                handles[target.id] = dep
            elif target:
                nested = dependencies_in(node.value)
                if len(nested) == 1:
                    assigned_dependencies[target.id] = next(iter(nested))
        if isinstance(node, ast.Assign) and isinstance(node.value, ast.Dict):
            target = next((item for item in node.targets if isinstance(item, ast.Name)), None)
            if target:
                mapping = {}
                for key, value in zip(node.value.keys, node.value.values):
                    if isinstance(key, ast.Constant) and isinstance(key.value, str):
                        calls = [item for item in ast.walk(value) if isinstance(item, ast.Call)]
                        dep = next((candidate for candidate in (dependency(item) for item in calls) if candidate), None)
                        if dep:
                            mapping[key.value] = dep
                if mapping:
                    dictionary_calls[target.id] = mapping
        if isinstance(node, (ast.For, ast.AsyncFor)) and isinstance(node.target, ast.Tuple) and len(node.target.elts) == 2:
            target_key, target_value = node.target.elts
            if not (isinstance(target_key, ast.Name) and isinstance(target_value, ast.Name)):
                continue
            if not (isinstance(node.iter, ast.Call) and isinstance(node.iter.func, ast.Name) and node.iter.func.id == "zip" and node.iter.args):
                continue
            source_name = node.iter.args[0].id if isinstance(node.iter.args[0], ast.Name) else ""
            for item in ast.walk(node):
                if not isinstance(item, ast.Assign) or not isinstance(item.value, ast.Name) or item.value.id != target_value.id:
                    continue
                for assigned in item.targets:
                    if (isinstance(assigned, ast.Subscript) and isinstance(assigned.value, ast.Name)
                            and isinstance(assigned.slice, ast.Name) and assigned.slice.id == target_key.id and source_name):
                        dictionary_sources[assigned.value.id] = source_name

    def resolve_alias(name: str) -> str:
        seen = set()
        while name in aliases and name not in seen:
            seen.add(name)
            name = aliases[name]
        return name

    found = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id in wrapper_names:
            for arg in node.args:
                found.update(dependencies_in(arg))
                if isinstance(arg, ast.Name) and arg.id in assigned_dependencies:
                    found.add(assigned_dependencies[arg.id])
            for keyword in node.keywords:
                found.update(dependencies_in(keyword.value))
        if not (isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                and isinstance(node.func.value, ast.Name) and node.func.value.id == "program"
                and node.func.attr == "child_envelope" and node.args):
            continue
        value = node.args[0]
        dep = dependency(value)
        if dep:
            found.add(dep)
        elif isinstance(value, ast.Name):
            handle = resolve_alias(value.id)
            if handle in handles:
                found.add(handles[handle])
            if value.id in assigned_dependencies:
                found.add(assigned_dependencies[value.id])
        elif isinstance(value, ast.Subscript) and isinstance(value.value, ast.Name) and isinstance(value.slice, ast.Constant) and isinstance(value.slice.value, str):
            handle = resolve_alias(value.value.id)
            source = dictionary_sources.get(handle, handle)
            dep = dictionary_calls.get(source, {}).get(value.slice.value)
            if dep:
                found.add(dep)

        # Some readers gather guarded handles into a keyed result map and only
        # send one selected key through child_envelope. Resolve the selected key
        # back through gather(zip(...)) so unrelated guarded reads are not faulted.
        if isinstance(value, ast.Name):
            for function in (item for item in ast.walk(tree) if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef))):
                if not any(item is node for item in ast.walk(function)):
                    continue
                for loop in (item for item in ast.walk(function) if isinstance(item, (ast.For, ast.AsyncFor))
                             and isinstance(item.target, ast.Tuple) and len(item.target.elts) == 2):
                    keys, result_value = loop.target.elts
                    if not (isinstance(keys, ast.Name) and isinstance(result_value, ast.Name)):
                        continue
                    if result_value.id != value.id:
                        continue
                    selected = {item.comparators[0].value for item in ast.walk(function)
                                if isinstance(item, ast.Compare) and isinstance(item.left, ast.Name)
                                and item.left.id == keys.id and len(item.comparators) == 1
                                and isinstance(item.comparators[0], ast.Constant) and isinstance(item.comparators[0].value, str)}
                    if not selected:
                        continue
                    results_name = next((target.value.id for item in ast.walk(function)
                                         if isinstance(item, ast.Assign) and isinstance(item.value, ast.Name)
                                         and item.value.id == result_value.id
                                         for target in item.targets if isinstance(target, ast.Subscript)
                                         and isinstance(target.value, ast.Name) and isinstance(target.slice, ast.Name)
                                         and target.slice.id == keys.id), "")
                    if not results_name:
                        continue
                    zipped = loop.iter
                    if not (isinstance(zipped, ast.Call) and isinstance(zipped.func, ast.Name) and zipped.func.id == "zip" and len(zipped.args) >= 2):
                        continue
                    key_values = next((arg for arg in zipped.args if isinstance(arg, (ast.List, ast.Tuple))), None)
                    if key_values is None:
                        continue
                    key_names = [item.value for item in key_values.elts if isinstance(item, ast.Constant) and isinstance(item.value, str)]
                    result_name = next((arg.id for arg in zipped.args if isinstance(arg, ast.Name)
                                        and any(isinstance(item, ast.Assign) and isinstance(item.value, ast.Name)
                                                and item.value.id == arg.id and any(isinstance(target, ast.Name)
                                                and target.id == "values" for target in item.targets)
                                                for item in ast.walk(function))), "")
                    if not result_name or len(key_names) != len(key_values.elts):
                        continue
                    for selected_key in selected:
                        if selected_key not in key_names:
                            continue
                        index = key_names.index(selected_key)
                        for assignment in ast.walk(function):
                            if not isinstance(assignment, ast.Assign) or not any(isinstance(target, ast.Name) and target.id == result_name for target in assignment.targets):
                                continue
                            calls = [item for item in ast.walk(assignment.value) if isinstance(item, ast.Call)
                                     and isinstance(item.func, ast.Name) and item.func.id == "gather"]
                            if not calls or not calls[0].args:
                                continue
                            call_list = next((item for item in ast.walk(calls[0].args[0]) if isinstance(item, ast.List)), None)
                            if call_list and index < len(call_list.elts):
                                found.update(dependencies_in(call_list.elts[index]))
                                continue
                            comp = next((item for item in ast.walk(calls[0].args[0]) if isinstance(item, ast.ListComp)), None)
                            loop_name = comp.generators[0].iter.id if comp and isinstance(comp.generators[0].iter, ast.Name) else ""
                            actual_calls = next((item.value for item in ast.walk(function) if isinstance(item, ast.Assign)
                                                 and any(isinstance(target, ast.Name) and target.id == loop_name for target in item.targets)
                                                 and isinstance(item.value, ast.List)), None)
                            if actual_calls and index < len(actual_calls.elts):
                                found.update(dependencies_in(actual_calls.elts[index]))
                # Same selected-key flow when the result map itself is built by
                # a zip over a list of guarded callables (for example, diagnostic
                # readers that envelope only the skill-registry child).
                for loop in (item for item in ast.walk(function) if isinstance(item, (ast.For, ast.AsyncFor))
                             and isinstance(item.target, ast.Tuple) and len(item.target.elts) == 2):
                    key_var, result_var = loop.target.elts
                    if not (isinstance(key_var, ast.Name) and isinstance(result_var, ast.Name) and result_var.id == value.id):
                        continue
                    if not (isinstance(loop.iter, ast.Call) and isinstance(loop.iter.func, ast.Name)
                            and loop.iter.func.id == "zip" and len(loop.iter.args) == 2
                            and isinstance(loop.iter.args[0], (ast.List, ast.Tuple))
                            and isinstance(loop.iter.args[1], ast.Name)):
                        continue
                    labels = [item.value for item in loop.iter.args[0].elts
                              if isinstance(item, ast.Constant) and isinstance(item.value, str)]
                    selected = {item.comparators[0].value for item in ast.walk(function)
                                if isinstance(item, ast.Compare) and isinstance(item.left, ast.Name)
                                and item.left.id == key_var.id and len(item.comparators) == 1
                                and isinstance(item.comparators[0], ast.Constant)
                                and isinstance(item.comparators[0].value, str)}
                    source_values = loop.iter.args[1].id
                    gather_assignment = next((item for item in ast.walk(function) if isinstance(item, ast.Assign)
                                              and any(isinstance(target, ast.Name) and target.id == source_values for target in item.targets)
                                              and isinstance(item.value, ast.Call) and isinstance(item.value.func, ast.Name)
                                              and item.value.func.id == "gather" and item.value.args), None)
                    if not gather_assignment:
                        continue
                    comp = next((item for item in ast.walk(gather_assignment.value.args[0]) if isinstance(item, ast.ListComp)), None)
                    callable_name = comp.generators[0].iter.id if comp and isinstance(comp.generators[0].iter, ast.Name) else ""
                    callable_list = next((item.value for item in ast.walk(function) if isinstance(item, ast.Assign)
                                          and any(isinstance(target, ast.Name) and target.id == callable_name for target in item.targets)
                                          and isinstance(item.value, ast.List)), None)
                    for selected_key in selected:
                        if selected_key in labels and callable_list and labels.index(selected_key) < len(callable_list.elts):
                            found.update(dependencies_in(callable_list.elts[labels.index(selected_key)]))
            # A phase may collect in one function and classify in another, so
            # resolve keyed child handles across the module, not only within the
            # function that contains child_envelope.
            for loop in (item for item in ast.walk(tree) if isinstance(item, (ast.For, ast.AsyncFor))
                         and isinstance(item.target, ast.Tuple) and len(item.target.elts) == 2):
                key_var, result_var = loop.target.elts
                if not (isinstance(key_var, ast.Name) and isinstance(result_var, ast.Name) and result_var.id == value.id):
                    continue
                if not (isinstance(loop.iter, ast.Call) and isinstance(loop.iter.func, ast.Name)
                        and loop.iter.func.id == "zip" and len(loop.iter.args) == 2
                        and isinstance(loop.iter.args[0], (ast.List, ast.Tuple))
                        and isinstance(loop.iter.args[1], ast.Name)):
                    continue
                labels = [item.value for item in loop.iter.args[0].elts
                          if isinstance(item, ast.Constant) and isinstance(item.value, str)]
                selected = {item.comparators[0].value for item in ast.walk(tree)
                            if isinstance(item, ast.Compare) and isinstance(item.left, ast.Name)
                            and item.left.id == key_var.id and len(item.comparators) == 1
                            and isinstance(item.comparators[0], ast.Constant)
                            and isinstance(item.comparators[0].value, str)}
                if not selected:
                    selected = set(labels)
                else:
                    selected = set(labels) - selected
                source_values = loop.iter.args[1].id
                gather_assignment = next((item for item in ast.walk(tree) if isinstance(item, ast.Assign)
                                          and any(isinstance(target, ast.Name) and target.id == source_values for target in item.targets)
                                          and isinstance(item.value, ast.Call) and isinstance(item.value.func, ast.Name)
                                          and item.value.func.id == "gather" and item.value.args), None)
                if not gather_assignment:
                    continue
                comp = next((item for item in ast.walk(gather_assignment.value.args[0]) if isinstance(item, ast.ListComp)), None)
                callable_name = comp.generators[0].iter.id if comp and isinstance(comp.generators[0].iter, ast.Name) else ""
                callable_list = next((item.value for item in ast.walk(tree) if isinstance(item, ast.Assign)
                                      and any(isinstance(target, ast.Name) and target.id == callable_name for target in item.targets)
                                      and isinstance(item.value, ast.List)), None)
                for selected_key in selected:
                    if selected_key in labels and callable_list and labels.index(selected_key) < len(callable_list.elts):
                        found.update(dependencies_in(callable_list.elts[labels.index(selected_key)]))
    return sorted(found)


def library_pins(source_code: str) -> dict[str, str]:
    tree = ast.parse(source_code)
    constants: dict[str, str] = {}
    handles: dict[str, str] = {}
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign) and isinstance(node.value, ast.Constant) and isinstance(node.value.value, str):
            for target in node.targets:
                if isinstance(target, ast.Name):
                    constants[target.id] = node.value.value
        elif (isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name)
              and isinstance(node.value, ast.Constant) and isinstance(node.value.value, str)):
            constants[node.target.id] = node.value.value
        if isinstance(node, ast.Assign) and isinstance(node.value, ast.Call):
            target = next((item for item in node.targets if isinstance(item, ast.Name)), None)
            parts = []
            value = node.value.func
            while isinstance(value, ast.Attribute):
                parts.append(value.attr)
                value = value.value
            if target and isinstance(value, ast.Name) and value.id == "lib" and len(parts) >= 2:
                parts.reverse()
                handles[target.id] = f"{parts[0].replace('_', '-')}.{parts[1].replace('_', '-')}"
    pins = {}
    for node in ast.walk(tree):
        if not isinstance(node, ast.Call):
            continue
        parts = []
        value = node.func
        while isinstance(value, ast.Attribute):
            parts.append(value.attr)
            value = value.value
        pin = ""
        if isinstance(value, ast.Name) and value.id == "lib":
            parts.reverse()
            for keyword in node.keywords:
                if keyword.arg != "expected_digest":
                    continue
                if isinstance(keyword.value, ast.Constant) and isinstance(keyword.value.value, str):
                    pin = keyword.value.value
                elif isinstance(keyword.value, ast.Name):
                    pin = constants.get(keyword.value.id, "")
                break
            if pin and len(parts) >= 2:
                pins[f"{parts[0].replace('_', '-')}.{parts[1].replace('_', '-')}" ] = pin
        if (isinstance(node.func, ast.Attribute) and isinstance(node.func.value, ast.Name)
                and node.func.value.id == "program" and node.func.attr == "child_envelope"
                and node.args and isinstance(node.args[0], ast.Name)):
            expected = next((keyword.value for keyword in node.keywords if keyword.arg == "expected_digest"), None)
            if isinstance(expected, ast.Constant) and isinstance(expected.value, str):
                pin = expected.value
            elif isinstance(expected, ast.Name):
                pin = constants.get(expected.id, "")
            else:
                pin = ""
            dependency = handles.get(node.args[0].id, "")
            if pin and dependency:
                pins[dependency] = pin
    return pins


def _fixture_inputs(contract: dict, source_id: str, scratch: Path) -> dict:
    fixtures = contract.get("fixtures", [])
    candidates = []
    for fixture in fixtures:
        expected = fixture.get("expect", {})
        statuses = expected.get("status", [])
        if "admission_error" not in expected and any(status != "failed" for status in statuses):
            identity = str(fixture.get("id", "")).lower()
            preference = 0 if any(word in identity for word in ("live", "valid", "sample", "selection", "diagnostic")) else 1
            candidates.append((preference, fixture))
    if candidates:
        values = dict(min(candidates, key=lambda candidate: candidate[0])[1].get("inputs", {}))
    else:
        values = {}
        for name, spec in contract.get("inputs", {}).items():
            if "default" in spec:
                values[name] = spec["default"]
            elif spec.get("required"):
                type_defaults = {"string": "fixture", "integer": 1, "number": 1.0, "boolean": False, "array": [], "object": {}}
                values[name] = type_defaults.get(spec.get("type"), "fixture")
    # Use valid, bounded values where the checked-in negative fixture is the only
    # declaration currently present. These remain synthetic test inputs.
    if "campaign_id" in contract.get("inputs", {}):
        values["campaign_id"] = "11111111-1111-4111-8111-111111111111"
        values["run_ids"] = ["fixture-run-1"]
    if "caller_key" in values:
        values.update(execution_id="fixture-execution", phase_id="fixture-phase", phase_generation="1", question="Why did this fixture fail?", run_ids=["fixture-run-1"])
    if "workspace_root" in contract.get("inputs", {}):
        inventory = scratch / "TOOLS.md"
        inventory.write_text("## Fixture capability\n- **Monitoring scenarios:** `unit-health`\n", encoding="utf-8")
        values.update(workspace_root=str(scratch), inventory_relative_path="TOOLS.md")
    if source_id == "web-search.record-attempt":
        values["attempt"] = {"attempt_id": "fixture-attempt", "outcome": "failure", "approach": "bounded fault injection"}
    if source_id == "browser-automation-studio.do-task":
        values["task"] = "inspect current workspace"
    if source_id == "device-control.do-task":
        values.update(device="fixture-device", context_key="fixture-context", actor="fixture-actor")
    if source_id == "program-runtime.research-supervision-read":
        values.update(threads=["fixture-thread"], session_id="fixture-session")
    if source_id == "program-runtime.local-work-snapshot":
        values["research_root"] = str(ROOT)
    if source_id == "program-runtime.watch-set":
        values["requests"] = [{"owner": "fixture", "workflow_key": "fixture-flow", "input": {}}]
    if source_id == "tech-tree-designer.setpoint-read":
        values["collect_diagnostics"] = True
    if source_id == "unit-health.test-quality-sample":
        values["review_ai"] = True
    return values


def _nested_binding_tree(contract: dict) -> dict:
    tree: dict = {}

    def inert(*_args, **_kwargs):
        return RuntimeHandle([], "fault-matrix-binding", metadata={}, raw={})

    for binding in contract.get("bindings", []):
        if binding.get("via") == "lib":
            continue
        binding_id = str(binding.get("id", ""))
        scenario, _, path = binding_id.partition("/")
        segments = [scenario.replace("-", "_")]
        segments.extend(part.replace("-", "_") for part in path.split("/") if part)
        if len(segments) < 2:
            continue
        current = tree
        for segment in segments[:-1]:
            current = current.setdefault(segment, {})
        current[segments[-1]] = inert
    return tree


def _fallback_markers(value: object, path: str = "signals") -> list[str]:
    """Find bounded owner-visible degradation in the actual envelope signals."""
    markers: list[str] = []
    if isinstance(value, dict):
        for key, child in value.items():
            child_path = f"{path}.{key}"
            if key == "unavailable" and child is True:
                markers.append(child_path)
            elif key in ("reason", "quality_causes") and child:
                markers.append(child_path)
            elif key == "status" and child in ("partial", "unavailable", "failed", "refused"):
                markers.append(child_path)
            else:
                markers.extend(_fallback_markers(child, child_path))
    elif isinstance(value, list):
        for index, child in enumerate(value):
            markers.extend(_fallback_markers(child, f"{path}[{index}]"))
    return markers[:20]


def execute_parent_fault(source: Path, contract: dict, fault: str, program_index: dict[str, tuple[Path, dict]]) -> dict:
    source_id = f"{source.parents[2].name}.{source.stem}"
    source_code = source.read_text()
    all_dependencies = library_calls(source_code)
    dependencies = child_envelope_dependencies(source_code)
    pins = library_pins(source_code)
    if not pins:
        digest_constants = [keyword.value.value for node in ast.walk(ast.parse(source_code))
                            if isinstance(node, ast.Call) and isinstance(node.func, ast.Attribute)
                            and isinstance(node.func.value, ast.Name) and node.func.value.id == "program"
                            and node.func.attr == "child_envelope"
                            for keyword in node.keywords if keyword.arg == "expected_digest"
                            and isinstance(keyword.value, ast.Constant) and isinstance(keyword.value.value, str)]
        if digest_constants and dependencies:
            pins[".".join(dependencies[0])] = digest_constants[0]
    if not dependencies:
        return {"program": source_id, "fault": fault, "passed": False, "reason": "source declares no library call to inject"}

    requested_child_status = {"unavailable": "unavailable", "degraded": "partial", "healthy-empty": "failed", "malformed": "failed", "partial": "partial"}[fault]
    observed_children: list[dict] = []
    injected_dependencies: list[str] = []
    expected_by_dependency: dict[str, str] = {}

    with tempfile.TemporaryDirectory(prefix="pr3-fault-matrix-") as directory:
        scratch = Path(directory)
        inputs = _fixture_inputs(contract, source_id, scratch)
        libraries = []
        output_contracts: dict[str, dict] = {}
        injected_dependencies_expected = {f"{scenario}.{name}" for scenario, name in dependencies}
        for scenario, name in all_dependencies:
            full_name = f"{scenario}.{name}"
            dep = program_index.get(full_name)
            if dep:
                dep_source, dep_contract = dep
                declaration = dict(dep_contract)
                if declaration.get("output_schema"):
                    schema_path = dep_source.parent / str(declaration["output_schema"])
                    if schema_path.is_file():
                        declaration["_runtime_output_schema"] = json.loads(schema_path.read_text())
                output_contract = dep_contract.get("outputs", {})
                schema_bytes = (dep_source.parent / str(dep_contract["output_schema"])).read_bytes() if dep_contract.get("output_schema") else b""
                contract_bytes = dep_source.with_suffix(".json").read_bytes()
                digest_bytes = contract_bytes + b"\0" + dep_source.read_bytes() + (b"\0" + schema_bytes if schema_bytes else b"")
                digest = hashlib.sha256(digest_bytes).hexdigest()
                version = dep_contract.get("version", "1")
            else:
                declaration = {"name": full_name, "version": "1", "helpers": {"program": "2"}, "inputs": {}, "outputs": {}}
                output_contract = {}
                digest = "sha256:fixture"
                version = "1"
            output_contracts[full_name] = output_contract
            allowed_statuses = output_contract.get("status", {}).get("enum", [])
            expected_by_dependency[full_name] = requested_child_status if requested_child_status in allowed_statuses else "failed"
            libraries.append({"name": name, "scenario": scenario, "contract": True, "current": True, "version": version,
                              "digest": digest, "declaration": declaration, "source": dep_source.read_text() if dep else ""})

        kernel = SessionKernel(bindings=_nested_binding_tree(contract), libraries=libraries)
        kernel.globals["inputs"] = inputs
        owner = kernel.globals["lib"]._owner
        original_execute = owner._execute_library_source
        original_child_envelope = RuntimeProgramHelper.child_envelope

        def injected_execute(spec, args, kwargs, *, _task_root=False):
            full_name = f"{spec.get('scenario', '')}.{spec.get('name', '')}"
            if full_name not in injected_dependencies_expected:
                return original_execute(spec, args, kwargs, _task_root=_task_root)
            injected_dependencies.append(full_name)
            contract_outputs = output_contracts[full_name]
            child_errors = contract_outputs.get("errors", {}).get("classes", [])
            injected_error = {"class": child_errors[0] if child_errors else "fixture_error",
                              "detail": f"injected {fault}", "where": "test-genie:fault-injection"}
            child = {"program": full_name, "version": str(spec.get("version", "1")), "status": "ok", "phase": "report",
                     "inputs": {}, "signals": {"fault": fault}, "errors": [], "evidence": ["fault-matrix:" + fault]}
            if fault == "unavailable":
                child.update(status="unavailable", errors=[injected_error])
            elif fault == "degraded":
                child.update(status="partial", errors=[injected_error])
            elif fault == "partial":
                child.update(status="partial", signals={"fault": fault, "partial": True})
            elif fault == "malformed":
                child["status"] = "not-a-status"
            rows = [] if fault == "healthy-empty" else [child]
            metadata = {"program": full_name, "version": str(spec.get("version", "1")), "digest": pins.get(full_name, spec.get("digest", "sha256:fixture")), "output_contract": contract_outputs}
            return RuntimeHandle(rows, "fault-matrix:" + full_name, metadata=metadata, raw=rows)

        def record_child(helper, handle, *args, **kwargs):
            try:
                metadata = handle.meta()
            except Exception:
                metadata = {}
            dependency = str(metadata.get("program", "")) if isinstance(metadata, dict) else ""
            try:
                result = original_child_envelope(helper, handle, *args, **kwargs)
            except Exception as exc:
                observed_children.append({"dependency": dependency, "status": "exception", "error": f"{type(exc).__name__}: {exc}"})
                raise
            observed_children.append({"dependency": dependency, "status": result.get("status"), "errors": result.get("errors", [])})
            return result

        owner._execute_library_source = injected_execute
        RuntimeProgramHelper.child_envelope = record_child
        try:
            if source_id == "program-runtime.watch-set":
                dict.__setitem__(kernel.globals, "agent", SimpleNamespace(start=lambda **request: request,
                                                                           collect=lambda _handle, wait_seconds: RuntimeHandle([{"execution_id": "fixture-run", "result": "completed"}])))
            result = kernel.execute(source.read_text(), include_materialized=True, program_id=source_id,
                                    declared_helper_contract={"version": str(contract.get("helpers", {}).get("program", "2")), "outputs": contract.get("outputs", {})})
        finally:
            RuntimeProgramHelper.child_envelope = original_child_envelope
            owner._execute_library_source = original_execute

    envelope = None
    for line in reversed(result.get("stdout", "").splitlines()):
        try:
            envelope = json.loads(line)
        except json.JSONDecodeError:
            try:
                envelope = ast.literal_eval(line)
            except (ValueError, SyntaxError):
                continue
        if isinstance(envelope, dict):
            break
    parent_status = envelope.get("status") if envelope else ""
    parent_phase = envelope.get("phase") if envelope else ""
    parent_errors = envelope.get("errors", []) if envelope else []
    parent_error_classes = [error.get("class") for error in parent_errors if isinstance(error, dict)]
    markers = _fallback_markers(envelope.get("signals", {})) if envelope else []
    parent_contract = contract.get("outputs", {})
    parent_statuses = parent_contract.get("status", {}).get("enum", [])
    parent_phases = parent_contract.get("phase", {}).get("enum", [])
    capabilities = capability_requirements(contract, source_code)
    observed_failure = parent_status != "ok" or bool(parent_error_classes) or bool(markers)
    no_retry = len(injected_dependencies) == len(set(injected_dependencies))
    observed_dependencies = {item.get("dependency") for item in observed_children}
    passed = bool(result.get("ok")) and isinstance(envelope, dict) and bool(injected_dependencies) and bool(observed_children)
    passed = passed and observed_dependencies == set(injected_dependencies) and no_retry and observed_failure
    passed = passed and parent_status in parent_statuses and parent_phase in parent_phases and bool(capabilities)
    passed = passed and all(item["status"] == expected_by_dependency.get(item.get("dependency"), requested_child_status) for item in observed_children)
    return {"program": source_id, "fault": fault, "execution_kind": "program_runtime_session_with_injected_library_handles",
            "native_qualification": False, "injected_dependencies": injected_dependencies,
            "capability_requirements": capabilities, "no_retry": no_retry,
            "expected_child_status": requested_child_status, "expected_status_by_dependency": expected_by_dependency, "observed_child_envelopes": observed_children,
            "parent_status": parent_status, "parent_status_enum": parent_statuses,
            "parent_phase": parent_phase, "parent_phase_enum": parent_phases,
            "observable_fallback": {"behavior": "owner_status_or_explicit_degraded_signal", "observed": observed_failure,
                                    "status": parent_status, "phase": parent_phase,
                                    "error_classes": parent_error_classes, "signal_markers": markers},
            "runtime_ok": bool(result.get("ok")), "passed": passed,
            **({"reason": result.get("error", "parent source did not exercise an injected child envelope"), "runtime_error": result.get("error", ""), "runtime_stdout": result.get("stdout", "")[:240]} if not passed else {})}


def execute_program_source_matrix(sources: list[Path], classes: tuple[str, ...]) -> dict:
    program_index = {}
    for candidate in sorted(ROOT.glob("scenarios/*/.vrooli/program-runtime/*.py")):
        candidate_contract = candidate.with_suffix(".json")
        if not candidate_contract.exists():
            continue
        declaration = json.loads(candidate_contract.read_text())
        name = str(declaration.get("name", ""))
        if name:
            program_index[name.replace("_", "-")] = (candidate, declaration)
    rows = []
    for source in sources:
        if "program.child_envelope" not in source.read_text():
            continue
        contract = json.loads(source.with_suffix(".json").read_text())
        for fault in classes:
            rows.append(execute_parent_fault(source, contract, fault, program_index))
    failures = [row for row in rows if not row.get("passed")]
    return {"execution_kind": "program_runtime_session_with_injected_library_handles", "native_qualification": False,
            "program_count": len({row["program"] for row in rows}), "fault_classes": list(classes), "row_count": len(rows),
            "passed_rows": len(rows) - len(failures), "failed_rows": len(failures), "matrix": rows, "failures": failures}


def main() -> int:
    classes = ("unavailable", "degraded", "healthy-empty", "malformed", "partial")
    matrix = []
    failures = []
    sources = sorted(ROOT.glob("scenarios/*/.vrooli/program-runtime/*.py"))
    for source in sources:
        code = source.read_text()
        if "program.child_envelope" not in code:
            continue
        contract_path = source.with_suffix(".json")
        contract = json.loads(contract_path.read_text())
        helper = contract.get("helpers", {}).get("program")
        outputs = contract.get("outputs", {})
        errors = outputs.get("errors", {}).get("classes", [])
        source_id = f"{source.parents[2].name}.{source.stem}"
        capabilities = capability_requirements(contract, code)
        if not capabilities:
            failures.append({"program": source_id, "fault": "inventory", "reason": "no declared binding or library capability requirement"})
            continue
        digest = "sha256:" + hashlib.sha256(contract_path.read_bytes() + b"\0" + source.read_bytes()).hexdigest()
        if str(helper) != "2" or "invalid_response" not in errors or "failed" not in outputs.get("status", {}).get("enum", []):
            failures.append({"program": source_id, "fault": "inventory", "reason": "helper v2, invalid_response, or failed status is not declared"})
            continue
        envelope_contract = {
            "status": {"enum": ["ok", "partial", "unavailable", "refused", "failed"]},
            "phase": {"enum": ["report"]},
            "errors": {"classes": ["fixture_error"]},
        }
        for fault in classes:
            rows = []
            child = {"status": "ok", "phase": "report", "signals": {"fixture": fault}, "errors": [], "evidence": ["fixture:" + fault]}
            if fault == "unavailable":
                child.update(status="unavailable", errors=[{"class": "fixture_error", "detail": "generated unavailable", "where": "matrix"}])
                rows = [child]
            elif fault == "degraded":
                child.update(status="partial", errors=[{"class": "fixture_error", "detail": "generated degraded", "where": "matrix"}])
                rows = [child]
            elif fault == "partial":
                child.update(status="partial", signals={"fixture": fault, "partial": True})
                rows = [child]
            elif fault == "malformed":
                rows = [{"status": "not-a-status", "phase": "report", "signals": {}, "errors": [], "evidence": []}]
            handle = Handle(rows, envelope_contract, "fixture." + source_id)
            actual = ProgramHelper(contract["helpers"]["program"], outputs).child_envelope(handle, where="fault-matrix")
            expected_status = {"unavailable": "unavailable", "degraded": "partial", "partial": "partial"}.get(fault, "failed")
            if fault == "healthy-empty":
                expected_class = next((c for c in ("missing_result", "no_output") if c in errors), "invalid_response")
                passed = actual.get("status") == "failed" and any(e.get("class") == expected_class for e in actual.get("errors", []))
                fallback_behavior = "refuse_empty_child_result"
            elif fault == "malformed":
                passed = actual.get("status") == "failed" and any(e.get("class") == "invalid_response" for e in actual.get("errors", []))
                fallback_behavior = "refuse_malformed_child_envelope"
            else:
                passed = (
                    actual.get("status") == expected_status
                    and actual.get("signals", {}).get("fixture") == fault
                    and actual.get("errors") == child.get("errors")
                    and actual.get("evidence") == child.get("evidence")
                )
                fallback_behavior = "propagate_child_status_without_retry"
            row = {
                "program": source_id,
                "contract_digest": digest,
                "capability_requirements": capabilities,
                "fault": fault,
                "expected_status": expected_status,
                "observable_fallback": {
                    "behavior": fallback_behavior,
                    "status": actual.get("status"),
                    "error_classes": [error.get("class") for error in actual.get("errors", []) if isinstance(error, dict)],
                    "signal_keys": sorted(actual.get("signals", {}).keys()) if isinstance(actual.get("signals"), dict) else [],
                },
                "observed_status": actual.get("status"),
                "passed": passed,
            }
            matrix.append(row)
            if not passed:
                failures.append(row)
    source_execution = execute_program_source_matrix(sources, classes)
    failures.extend(source_execution["failures"])
    summary = {"schema_version": 1, "evidence_kind": "generated_fixture", "native_qualification": False,
               "program_count": len({row["program"] for row in matrix}),
               "fault_classes": list(classes), "row_count": len(matrix), "passed_rows": sum(bool(row["passed"]) for row in matrix),
               "failed_rows": len([row for row in matrix if not row.get("passed")]), "matrix": matrix, "failures": [row for row in matrix if not row.get("passed")],
               "program_source_execution": source_execution}
    print(json.dumps(summary, sort_keys=True))
    return 1 if failures or not matrix else 0


if __name__ == "__main__":
    raise SystemExit(main())
