#!/usr/bin/env python3
"""Generate the fail-closed OpenRGB 1.1 detector configuration."""

from __future__ import annotations

import argparse
import ast
import json
import re
import subprocess
import sys
from pathlib import Path


PINNED_TAG = "release_1.1"
PINNED_COMMIT = "8e16de27858cf2be6af54a33ce003df74cb9fd3e"
EXPECTED_KEYS = 1948
EXPECTED_STATIC_CALLS = 2160
ENABLED_KEYS = {
    "ASUS Aura SMBus Motherboard",
    "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC",
}
OUTPUT = Path(__file__).resolve().parents[1] / "config" / "detectors-release_1.1.json"

STATIC_TYPES = (
    "DeviceDetector",
    "I2CDeviceDetector",
    "I2CDRAMDeviceDetector",
    "I2CPCIDeviceDetector",
    "HIDDeviceDetector",
    "HIDWrappedDeviceDetector",
)
REGISTRATION_TOKEN = re.compile(r"\bREGISTER_[A-Z0-9_]*DETECTOR[A-Z0-9_]*\b")
DEFINE = re.compile(r"^\s*#\s*define\s+(REGISTER_[A-Z0-9_]*DETECTOR[A-Z0-9_]*)\s*\(")
STRING_DEFINE = re.compile(r'^\s*#\s*define\s+([A-Za-z_]\w*)\s+("(?:[^"\\]|\\.)*")\s*$')


class GenerationError(Exception):
    pass


def run_git(source: Path, *args: str) -> str:
    try:
        return subprocess.check_output(
            ["git", "-C", str(source), *args], text=True, stderr=subprocess.PIPE
        ).strip()
    except (OSError, subprocess.CalledProcessError) as exc:
        detail = getattr(exc, "stderr", "")
        raise GenerationError(f"cannot verify OpenRGB source checkout: {detail or exc}") from exc


def validate_checkout(source: Path) -> None:
    if not (source / "DetectionManager.h").is_file():
        raise GenerationError(f"{source} is not an OpenRGB source root (DetectionManager.h missing)")
    commit = run_git(source, "rev-parse", "HEAD")
    if commit != PINNED_COMMIT:
        raise GenerationError(f"source commit mismatch: expected {PINNED_COMMIT}, got {commit}")
    try:
        tag = run_git(source, "describe", "--tags", "--exact-match", "HEAD")
    except GenerationError as exc:
        raise GenerationError(f"pinned tag {PINNED_TAG} is not available at HEAD") from exc
    if tag != PINNED_TAG:
        raise GenerationError(f"source tag mismatch: expected {PINNED_TAG}, got {tag}")
    dirty = run_git(source, "status", "--porcelain")
    if dirty:
        raise GenerationError("OpenRGB source checkout is dirty; use an unmodified release_1.1 checkout")


def split_args(text: str) -> list[str]:
    args: list[str] = []
    start = 0
    depth = 0
    quote = False
    escaped = False
    for index, char in enumerate(text):
        if quote:
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == '"':
                quote = False
        elif char == '"':
            quote = True
        elif char in "([{":
            depth += 1
        elif char in ")]}" :
            depth -= 1
        elif char == "," and depth == 0:
            args.append(text[start:index].strip())
            start = index + 1
    if quote or depth != 0:
        raise GenerationError("unbalanced registration arguments")
    tail = text[start:].strip()
    if tail:
        args.append(tail)
    return args


def skip_space_and_comments(source: str, index: int) -> int:
    while index < len(source):
        if source[index].isspace():
            index += 1
        elif source.startswith("//", index):
            end = source.find("\n", index)
            return len(source) if end < 0 else skip_space_and_comments(source, end + 1)
        elif source.startswith("/*", index):
            end = source.find("*/", index + 2)
            if end < 0:
                raise GenerationError("unterminated C++ comment")
            index = end + 2
        else:
            break
    return index


def scan_calls(path: Path, source: str) -> list[tuple[str, list[str], int]]:
    calls: list[tuple[str, list[str], int]] = []
    index = 0
    line_start = 0
    while index < len(source):
        newline = source.find("\n", line_start)
        line_end = len(source) if newline < 0 else newline
        line = source[line_start:line_end]
        define_match = DEFINE.match(line)
        if define_match:
            # Macro bodies describe registration types; they are not registrations.
            index = line_end + (newline >= 0)
            line_start = index
            continue
        if source.startswith("//", index):
            index = line_end + (newline >= 0)
            line_start = index
            continue
        if source.startswith("/*", index):
            end = source.find("*/", index + 2)
            if end < 0:
                raise GenerationError(f"{path}: unterminated C++ comment")
            index = end + 2
            line_start = source.rfind("\n", 0, index) + 1
            continue
        if source[index] == '"':
            index += 1
            escaped = False
            while index < len(source):
                char = source[index]
                index += 1
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == '"':
                    break
            continue
        match = REGISTRATION_TOKEN.match(source, index)
        if not match:
            if source[index] == "\n":
                line_start = index + 1
            index += 1
            continue
        macro = match.group(0)
        opening = skip_space_and_comments(source, match.end())
        if opening >= len(source) or source[opening] != "(":
            index = match.end()
            continue
        depth = 1
        cursor = opening + 1
        quote = False
        escaped = False
        while cursor < len(source) and depth:
            if source.startswith("//", cursor) and not quote:
                end = source.find("\n", cursor)
                cursor = len(source) if end < 0 else end + 1
                continue
            if source.startswith("/*", cursor) and not quote:
                end = source.find("*/", cursor + 2)
                if end < 0:
                    raise GenerationError(f"{path}: unterminated C++ comment")
                cursor = end + 2
                continue
            char = source[cursor]
            if quote:
                if escaped:
                    escaped = False
                elif char == "\\":
                    escaped = True
                elif char == '"':
                    quote = False
            elif char == '"':
                quote = True
            elif char == "(":
                depth += 1
            elif char == ")":
                depth -= 1
            cursor += 1
        if depth:
            raise GenerationError(f"{path}: unterminated call to {macro}")
        calls.append((macro, split_args(source[opening + 1:cursor - 1]), source.count("\n", 0, index) + 1))
        index = cursor
    return calls


def registration_macro_definitions(source_root: Path) -> tuple[set[str], set[str]]:
    """Classify every in-tree detector registration macro from its expansion."""
    static: set[str] = set()
    ignored: set[str] = set()
    definitions = re.compile(r"^\s*#\s*define\s+(REGISTER_[A-Z0-9_]*DETECTOR[A-Z0-9_]*)\s*\(")
    for path in sorted(source_root.rglob("*")):
        if path.suffix not in {".cpp", ".h"}:
            continue
        lines = path.read_text(encoding="utf-8").splitlines()
        index = 0
        while index < len(lines):
            line = lines[index]
            match = definitions.match(line)
            if not match:
                index += 1
                continue
            macro = match.group(1)
            body = line
            while body.rstrip().endswith("\\") and index + 1 < len(lines):
                index += 1
                body += "\n" + lines[index]
            if any(re.search(rf"\bstatic\s+{re.escape(type_name)}\s+", body) for type_name in STATIC_TYPES):
                static.add(macro)
            elif "DynamicDetector" in body or "DYNAMIC" in macro or "I2CBusDetector" in body:
                # Dynamic callbacks are not entries in Detectors.detectors, and
                # bus detectors initialize interfaces rather than device keys.
                ignored.add(macro)
            else:
                raise GenerationError(f"unclassified detector macro definition: {macro} ({path})")
            index += 1
    if not static:
        raise GenerationError("no static detector registration macros found")
    return static, ignored


def collect_names(source_root: Path) -> tuple[list[str], int]:
    static_macros, ignored_macros = registration_macro_definitions(source_root)

    names: set[str] = set()
    static_calls = 0
    for path in sorted(source_root.rglob("*")):
        if path.suffix not in {".cpp", ".h"}:
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except UnicodeDecodeError as exc:
            raise GenerationError(f"cannot decode source file {path}") from exc
        aliases = {
            match.group(1): ast.literal_eval(match.group(2))
            for line in text.splitlines()
            if (match := STRING_DEFINE.match(line))
        }
        for macro, args, line_number in scan_calls(path, text):
            if macro in ignored_macros:
                continue
            if macro not in static_macros:
                raise GenerationError(f"{path}:{line_number}: unknown detector registration macro {macro}")
            if not args:
                raise GenerationError(f"{path}:{line_number}: {macro} has no detector name")
            raw_name = args[0]
            if raw_name.startswith('"'):
                try:
                    name = ast.literal_eval(raw_name)
                except (SyntaxError, ValueError) as exc:
                    raise GenerationError(f"{path}:{line_number}: invalid string detector name") from exc
            elif re.fullmatch(r"[A-Za-z_]\w*", raw_name) and raw_name in aliases:
                name = aliases[raw_name]
            else:
                raise GenerationError(
                    f"{path}:{line_number}: unresolved detector name {raw_name!r} in {macro}"
                )
            if not isinstance(name, str) or not name.strip():
                raise GenerationError(f"{path}:{line_number}: empty or non-string detector name")
            names.add(name)
            static_calls += 1

    if static_calls != EXPECTED_STATIC_CALLS:
        raise GenerationError(
            f"static registration count changed: expected {EXPECTED_STATIC_CALLS}, got {static_calls}"
        )
    if len(names) != EXPECTED_KEYS:
        raise GenerationError(f"detector key count changed: expected {EXPECTED_KEYS}, got {len(names)}")
    missing = ENABLED_KEYS - names
    if missing:
        raise GenerationError(f"enabled detector names absent from pinned source: {sorted(missing)}")
    return sorted(names), static_calls


def expected_document(source_root: Path) -> tuple[bytes, int]:
    names, static_calls = collect_names(source_root)
    document = {"Detectors": {"detectors": {name: name in ENABLED_KEYS for name in names}}}
    return (json.dumps(document, ensure_ascii=False, indent=2) + "\n").encode(), static_calls


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true", help="verify the generated detector map is current")
    mode.add_argument("--write", action="store_true", help="write the generated detector map")
    parser.add_argument("--source", type=Path, required=True, help="pinned OpenRGB source checkout")
    parser.add_argument("--output", type=Path, default=OUTPUT, help=f"output JSON (default: {OUTPUT})")
    args = parser.parse_args()

    try:
        validate_checkout(args.source.resolve())
        expected, static_calls = expected_document(args.source.resolve())
        if args.write:
            args.output.parent.mkdir(parents=True, exist_ok=True)
            args.output.write_bytes(expected)
            print(f"wrote {args.output} ({EXPECTED_KEYS} detector keys from {static_calls} active static registrations)")
            return 0
        try:
            actual = args.output.read_bytes()
        except OSError as exc:
            raise GenerationError(f"cannot read generated map {args.output}: {exc}") from exc
        if actual != expected:
            raise GenerationError(f"generated map is stale: run {Path(__file__).name} --write --source <OpenRGB 1.1 source>")
        print(f"verified {args.output} ({EXPECTED_KEYS} detector keys from {static_calls} active static registrations; exactly two enabled)")
        return 0
    except GenerationError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
