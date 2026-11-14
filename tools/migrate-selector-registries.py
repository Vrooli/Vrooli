#!/usr/bin/env python3
"""Normalize generated adopter selector registries to semantic catalog ids."""

from pathlib import Path
import re


FILES = sorted(Path("scenarios").glob("*/ui/src/consts/selectors.library.ts"))
ENTRY = re.compile(r'^(\s*)(["\']?[^:]+["\']?)\s*:\s*\{\s*$')
PROPERTY = re.compile(r'^(\s*)(["\']?[^"\':]+["\']?)\s*:\s*(["\'])([^"\']*)\3(,?)([ \t]*)(\r?\n)?$')


def unquote(value: str) -> str:
    return value.strip().strip('"\'')


def canonical(root: str, value: str) -> str:
    root = unquote(root)
    value = value.replace("_", "-")
    if value == root or value == "":
        return root
    if value.startswith(root + "."):
        suffix = value[len(root) + 1 :]
        if suffix.startswith("shell") and len(suffix) > 5 and suffix[5].isupper():
            suffix = suffix[5].lower() + suffix[6:]
        return f"{root}.{suffix}"
    last = root.rsplit(".", 1)[-1]
    if value == last or value == last + "-shell":
        return root
    if value.startswith(last + "-"):
        value = value[len(last) + 1 :]
    if value.startswith("shell-"):
        value = value[len("shell-") :]
    if not value:
        return root
    suffix = re.sub(r"[^A-Za-z0-9]+", " ", value).strip().split()
    if not suffix:
        return root
    suffix = suffix[0].lower() + "".join(part[:1].upper() + part[1:] for part in suffix[1:])
    return f"{root}.{suffix}"


def field_name(root: str, value: str) -> str:
    value = canonical(root, value)
    if value == root:
        return "root"
    suffix = value[len(root) + 1 :] if value.startswith(root + ".") else value
    parts = re.split(r"[.\-_ ]+", suffix)
    return parts[0][:1].lower() + parts[0][1:] + "".join(part[:1].upper() + part[1:] for part in parts[1:] if part)


def migrate(path: Path) -> bool:
    lines = path.read_text().splitlines(keepends=True)
    inside = False
    root = None
    used = {}
    seen_values = set()
    changed = False
    output = []
    for line in lines:
        if "// vrooli:library-selectors start" in line:
            inside = True
        if inside and line.strip() == "} as const;":
            inside = False
            root = None
            used = {}
        match = ENTRY.match(line) if inside else None
        if match and not match.group(2).strip().startswith("export"):
            root = unquote(match.group(2))
            used = {}
            seen_values = set()
            output.append(line)
            continue
        if inside and not line.strip():
            continue
        match = PROPERTY.match(line) if inside else None
        if not match:
            output.append(line)
            continue
        key, value = unquote(match.group(2)), match.group(4)
        if root is None:
            # A legacy flat selector is still part of the managed registry.
            new_value = key
            new_key = key
        else:
            new_value = root if key == "root" else canonical(root, value)
            if new_value in seen_values:
                changed = True
                continue
            new_key = "root" if key == "root" else field_name(root, value)
            used[new_key] = used.get(new_key, 0) + 1
            if used[new_key] > 1:
                new_key = f"{new_key}{used[new_key]}"
        indent = match.group(1)
        quoted_key = f'"{new_key}"' if match.group(2).strip().startswith(('"', "'")) or root is not None else new_key
        seen_values.add(new_value)
        replacement = f'{indent}{quoted_key}: "{new_value}"{match.group(5)}{match.group(6)}{match.group(7) or ""}'
        output.append(replacement)
        changed |= replacement != line
    if changed:
        path.write_text("".join(output))
    return changed


changed = [str(path) for path in FILES if migrate(path)]
print(f"migrated {len(changed)} selector registries")
for path in changed:
    print(path)
