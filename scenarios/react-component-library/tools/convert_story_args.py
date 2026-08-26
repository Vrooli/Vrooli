#!/usr/bin/env python3
"""Move React-valued story arguments into version-local specimen exports.

The story contract is intentionally JSON data.  React nodes, icons, and event
callbacks belong in the executable specimen module instead of being encoded as
an untyped JSON pseudo-AST.  This migration is deterministic and idempotent:
it only visits story.json files containing one of the legacy `$node`, `$icon`,
or `$handler` values and appends generated exports to story.tsx.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


SPECIAL_KEYS = {"$node", "$icon", "$handler"}
ICON_FALLBACK = "Circle"


def has_special(value: Any) -> bool:
    if isinstance(value, dict):
        return any(key in SPECIAL_KEYS for key in value) or any(has_special(v) for v in value.values())
    if isinstance(value, list):
        return any(has_special(v) for v in value)
    return False


def text_from_node(value: Any) -> str:
    if isinstance(value, dict):
        if isinstance(value.get("$text"), str):
            return value["$text"]
        return " ".join(text_from_node(v) for v in value.get("children", []))
    if isinstance(value, list):
        return " ".join(text_from_node(v) for v in value)
    return str(value) if value is not None else ""


def safe_identifier(value: str, fallback: str) -> str:
    value = re.sub(r"[^A-Za-z0-9_$]", "", value)
    if not value or not re.match(r"[A-Za-z_$]", value):
        return fallback
    return value


def pascal(value: str) -> str:
    parts = re.split(r"[^A-Za-z0-9]+", value)
    return "".join(part[:1].upper() + part[1:] for part in parts if part) or "Story"


def js(value: Any, icons: set[str], handlers: set[str], indent: int = 0) -> str:
    """Serialize a JSON-ish value to a small, readable TypeScript expression."""
    if value is None or isinstance(value, (bool, int, float, str)):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, list):
        if not value:
            return "[]"
        pad = " " * indent
        child_pad = " " * (indent + 2)
        return "[\n" + ",\n".join(child_pad + js(item, icons, handlers, indent + 2) for item in value) + "\n" + pad + "]"
    if not isinstance(value, dict):
        return "undefined"
    if "$text" in value:
        return json.dumps(str(value["$text"]), ensure_ascii=False)
    if "$icon" in value:
        name = safe_identifier(str(value["$icon"]), ICON_FALLBACK)
        icons.add(name)
        return f"createElement({name}, {{ 'aria-hidden': true }})"
    if "$handler" in value:
        name = str(value["$handler"])
        handlers.add(name)
        return f"(...eventArgs: unknown[]) => log({json.dumps(name)}, ...eventArgs)"
    if "$node" in value:
        tag = str(value["$node"])
        props = {key: child for key, child in value.items() if key not in {"$node", "children"}}
        prop_expr = js_object(props, icons, handlers, indent + 2)
        children = value.get("children", [])
        if not isinstance(children, list):
            children = [children]
        child_exprs = [js(child, icons, handlers, indent + 2) for child in children]
        args = [json.dumps(tag), prop_expr] + child_exprs
        return "createElement(" + ", ".join(args) + ")"
    return js_object(value, icons, handlers, indent)


def js_object(value: dict[str, Any], icons: set[str], handlers: set[str], indent: int) -> str:
    if not value:
        return "{}"
    pad = " " * indent
    child_pad = " " * (indent + 2)
    entries = []
    for key, child in value.items():
        normalized = {"class": "className", "for": "htmlFor"}.get(key, key)
        entries.append(f"{child_pad}{json.dumps(normalized)}: {js(child, icons, handlers, indent + 2)}")
    return "{\n" + ",\n".join(entries) + "\n" + pad + "}"


def cast_generated_props(source: str) -> str:
    """Cast generated prop objects at the React boundary.

    Story controls are deliberately open-ended at runtime.  The catalog
    compiler still checks the component implementation itself, while this
    generated adapter supplies the required props from each story's fixture.
    """
    marker = "return createElement("
    cursor = 0
    out = source
    while True:
        start = out.find(marker, cursor)
        if start < 0:
            break
        comma = out.find(",", start + len(marker))
        if comma < 0:
            break
        opening = out.find("{", comma)
        if opening < 0:
            break
        depth = 0
        quote = ""
        escaped = False
        closing = -1
        for index in range(opening, len(out)):
            char = out[index]
            if quote:
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == quote:
                    quote = ""
                continue
            if char in "'\"`":
                quote = char
            elif char == "{":
                depth += 1
            elif char == "}":
                depth -= 1
                if depth == 0:
                    closing = index
                    break
        if closing < 0:
            break
        tail = out[closing + 1 :]
        if not re.match(r"\s*\);", tail) or out[closing - 3 : closing + 1] == " as never":
            cursor = closing + 1
            continue
        out = out[:closing] + "} as never" + out[closing + 1 :]
        cursor = closing + len("} as never")
    return out


def plain_schema_default(value: Any) -> Any:
    """Keep control metadata useful without retaining executable pseudo-nodes."""
    if not has_special(value):
        return value
    if isinstance(value, dict):
        if "$icon" in value:
            return ""
        if "$handler" in value:
            return None
        if "$node" in value:
            return text_from_node(value) or "Content"
        return {key: plain_schema_default(child) for key, child in value.items() if not has_special(child)}
    if isinstance(value, list):
        return [plain_schema_default(child) for child in value if not has_special(child)]
    return value


def remove_special_args(value: Any) -> Any:
    """Return scalar story args; a prop containing any React value is omitted."""
    if not isinstance(value, dict):
        return value
    return {key: child for key, child in value.items() if not has_special(child)}


def component_name(version_dir: Path, document: dict[str, Any]) -> str:
    title = str(document.get("title") or version_dir.parent.parent.name)
    source = version_dir / f"{title}.tsx"
    if source.exists():
        return title
    candidates = sorted(version_dir.glob("*.tsx"))
    for candidate in candidates:
        if candidate.name != "story.tsx":
            return candidate.stem
    return title


def migrate(path: Path) -> bool:
    document = json.loads(path.read_text())
    story_path = path.parent / "story.tsx"
    existing = story_path.read_text() if story_path.exists() else ""
    repaired = existing
    generated_components = set(re.findall(r"return createElement\(([A-Za-z_$][A-Za-z0-9_$]*)\s*,", existing))
    for component in sorted(generated_components):
        if f'from "./{component}"' not in repaired:
            repaired = f'import {{ {component} }} from "./{component}";\n' + repaired
    repaired = re.sub(
        r"(export function [A-Za-z_$][A-Za-z0-9_$]*\([\s\S]*?args: any[\s\S]*?\)\s*\{\n)(?!\s*void log;)",
        r"\1  void log;\n",
        repaired,
    )
    repaired = repaired.replace("args: any;", "args: Record<string, never>;")
    repaired = cast_generated_props(repaired)
    repaired_changed = repaired != existing
    if repaired_changed:
        story_path.write_text(repaired)
        existing = repaired
    if not any(has_special(story.get("args", {})) for story in document.get("stories", [])):
        return repaired_changed
    version_dir = path.parent
    component = component_name(version_dir, document)
    generated: list[str] = []
    used_names: set[str] = set(re.findall(r"export\s+(?:function|const|class)\s+([A-Za-z_$][A-Za-z0-9_$]*)", existing))
    all_icons: set[str] = set()
    all_handlers: set[str] = set()
    for story in document.get("stories", []):
        original_args = story.get("args", {})
        if not has_special(original_args):
            continue
        export_name = pascal(str(story.get("id", "story")))
        if export_name in used_names:
            export_name = export_name + "Specimen"
        while export_name in used_names:
            export_name += "Specimen"
        used_names.add(export_name)
        expressions: dict[str, str] = {}
        for key, value in original_args.items():
            if has_special(value):
                expressions[key] = js(value, all_icons, all_handlers)
        story["args"] = remove_special_args(original_args)
        story["composition"] = {"specimen": {"module": "./story.tsx", "export": export_name}}
        props = "{...args"
        for key, expression in expressions.items():
            props += f", {key}: {expression}"
        props += "}"
        generated.append(
            f"export function {export_name}({{ args, log }}: {{ args: Record<string, never>; log: (name: string, ...eventArgs: unknown[]) => void }}) {{\n"
            "  void log;\n"
            f"  return createElement({component}, {props} as never);\n"
            "}\n"
        )
    # Remove React pseudo-values from control defaults and field options too.
    fields = document.get("args", {}).get("fields", [])
    for field in fields:
        if "default" in field and has_special(field["default"]):
            field["default"] = plain_schema_default(field["default"])
        if "options" in field:
            field["options"] = [plain_schema_default(option) for option in field["options"]]
    prefix = "import { createElement } from \"react\";\n"
    if all_icons:
        names = sorted(all_icons)
        prefix += "import { " + ", ".join(names) + " } from \"lucide-react\";\n"
    component_import = f'import {{ {component} }} from "./{component}";\n'
    if generated:
        body = existing.rstrip() + ("\n\n" if existing.strip() else "") + "\n".join(generated)
        if "import { createElement } from \"react\";" not in body:
            body = prefix + body
        elif all_icons and 'from "lucide-react"' not in body:
            body = prefix.split("\n", 1)[0] + "\n" + "\n".join(prefix.split("\n")[1:]) + body
        if f'from "./{component}"' not in body:
            body = component_import + body
        story_path.write_text(body.rstrip() + "\n")
    path.write_text(json.dumps(document, indent=2, ensure_ascii=False) + "\n")
    return True


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", type=Path, help="library root containing component version directories")
    parser.add_argument("--limit", type=int, default=0, help="convert at most this many files (0 = all)")
    args = parser.parse_args()
    files = sorted(args.root.glob("**/versions/*/story.json"))
    changed = 0
    for path in files:
        if args.limit and changed >= args.limit:
            break
        if migrate(path):
            changed += 1
            print(path)
    print(f"converted {changed} story contract(s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
