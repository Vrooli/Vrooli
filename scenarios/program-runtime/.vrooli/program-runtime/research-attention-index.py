"""Read the explicit actionable section of a caller-authorized research index."""
import hashlib
import json
import os
import re
import stat

inputs = program.inputs()
envelope = program.envelope("program-runtime.research-attention-index", "1")


def step_validate():
    root = inputs.get("research_root", "")
    if not isinstance(root, str) or not root or not os.path.isabs(root) or "\0" in root or len(root) > 1024:
        return program.fail("failed", "invalid_input", "research_root must be an authorized absolute directory", "validate")
    envelope["inputs"] = {"research_root": root}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    root = os.path.realpath(inputs["research_root"])
    path = os.path.join(root, "ATTENTION.md")
    try:
        nofollow = getattr(os, "O_NOFOLLOW", None)
        if nofollow is None:
            raise ValueError("safe non-symlink file reads are unavailable on this host")
        fd = os.open(path, os.O_RDONLY | nofollow)
        with os.fdopen(fd, "rb") as stream:
            before = os.fstat(stream.fileno())
            if not stat.S_ISREG(before.st_mode) or before.st_size > 1048576:
                raise ValueError("attention index is not a bounded regular file")
            data = stream.read(1048577)
            after = os.fstat(stream.fileno())
        current = os.stat(path, follow_symlinks=False)
        identity = lambda value: (value.st_dev, value.st_ino, value.st_size, value.st_mtime_ns)
        if len(data) > 1048576 or identity(before) != identity(after) or identity(after) != identity(current):
            raise ValueError("attention index changed during read or exceeds limit")
        text = data.decode("utf-8")
        lines = text.splitlines()
        starts = [i for i, line in enumerate(lines) if line.strip() == "## Actionable now"]
        if len(starts) != 1:
            raise ValueError("exactly one actionable section is required")
        start = starts[0] + 1
        end = next((i for i in range(start, len(lines)) if lines[i].startswith("## ")), len(lines))
        section = lines[start:end]
        headings = [line for line in section if line.startswith("### ")]
        ids = []
        for line in headings:
            match = re.fullmatch(r"### ([A-Z0-9][A-Z0-9_-]{0,127})\s+(?:—|–|-)\s+\S.*", line)
            if not match or match.group(1) in ids:
                raise ValueError("actionable heading has invalid or duplicate identity")
            ids.append(match.group(1))
        if len(ids) > 64:
            raise ValueError("too many actionable items")
        if not ids and [line.strip() for line in section if line.strip()] != ["No actionable items are recorded."]:
            raise ValueError("empty actionable section lacks explicit empty marker")
        # Nonblank prose before the first item is ambiguous when items exist.
        if ids:
            first = next(i for i, line in enumerate(section) if line.startswith("### "))
            if any(line.strip() for line in section[:first]):
                raise ValueError("unclassified actionable section preamble")
        envelope["signals"] = {"coverage": "parsed_snapshot", "actionable_ids": ids, "count": len(ids), "sha256": hashlib.sha256(data).hexdigest(), "mtime_ns": str(after.st_mtime_ns)}
        envelope["status"] = "ok"
        envelope["evidence"] = [{"path": path, "scope": "explicit actionable section only"}]
    except (OSError, UnicodeError, ValueError) as exc:
        envelope["signals"] = {"coverage": "unknown", "actionable_ids": []}
        return program.fail("unavailable", "index_unavailable", str(exc)[:200], "collect")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
