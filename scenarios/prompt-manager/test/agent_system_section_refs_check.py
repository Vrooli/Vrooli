#!/usr/bin/env python3
"""Every section citation into docs/agent-system resolves to a real heading.

Accepts `DOC.md` followed by a section sign and a quoted heading (prefix match,
case-insensitive), or a section number for LONG_EFFORT_DESIGN.md. A numeric
citation of any other doc fails: those docs carry no section numbers.
"""
import os
import re
import sys

ROOT = os.getcwd()
DOCS = ["SKILL_AUTHORING", "STANDING_TEAMS", "LEVELS", "LONG_EFFORT_DESIGN", "SCENARIO_DEVELOPMENT", "README", "RULES", "EFFORT_SUPERVISION"]
SKIP_DIRS = {"node_modules", "dist", ".git", "testdata", "retired-docs", "standing-team-route-investigation-2026-11-09", "research-experimentation"}
EXTS = (".md", ".go", ".json")
HEAD = re.compile(r"^#{1,6}\s+(.*)$", re.M)
REF = re.compile(r"(%s)\.md`?\)?\s*§\s*(?:\"([^\"]{3,90})\"|(\d+[a-z]?))" % "|".join(DOCS))


def norm(text):
    return re.sub(r"[`*]", "", text).strip().lower()


heads = {}
for name in DOCS:
    with open(os.path.join(ROOT, "docs/agent-system", name + ".md"), encoding="utf-8") as handle:
        heads[name] = [norm(h) for h in HEAD.findall(handle.read())]

bad = []
for top in ("scenarios", "docs", "packages", "cmd", "cli"):
    for base, dirs, files in os.walk(os.path.join(ROOT, top)):
        dirs[:] = [d for d in dirs if d not in SKIP_DIRS]
        for file in files:
            if not file.endswith(EXTS):
                continue
            path = os.path.join(base, file)
            try:
                lines = open(path, encoding="utf-8", errors="ignore").read().split("\n")
            except OSError:
                continue
            for number, line in enumerate(lines, 1):
                if "§" not in line:
                    continue
                for match in REF.finditer(line):
                    doc, quoted, num = match.groups()
                    if quoted:
                        want = norm(quoted).split(":")[0].strip()
                        ok = any(h.startswith(want) or want.startswith(h.split(":")[0]) for h in heads[doc])
                    else:
                        ok = doc == "LONG_EFFORT_DESIGN" and any(h.startswith(num + ".") for h in heads[doc])
                    if not ok:
                        bad.append("%s:%d: %s.md § %s" % (os.path.relpath(path, ROOT), number, doc, quoted or num))
for item in bad:
    print("    dangling: " + item)
sys.exit(1 if bad else 0)
