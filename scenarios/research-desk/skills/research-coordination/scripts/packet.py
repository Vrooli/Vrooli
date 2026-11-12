#!/usr/bin/env python3
"""Build, preview and check research commission packets.

Usage:
  packet.py preview --home DIR --spec FILE|-     render in memory; print the result with texts
  packet.py build   --home DIR --spec FILE|-     render, gate, write cohorts/<id>/ atomically
  packet.py check   --home DIR COMMISSION_ID     gate an existing packet folder; write nothing

A packet is the charter (README.md), STANCE-MESSAGE.txt, KERNEL.txt and PACKET.json, rendered
from ../templates/ and gated by ../templates/limits.json. Prints one JSON object
(schema research-packet-result/v1). Exit 0 ok; 1 a gate failed; 2 invalid spec, unknown topic
or mission error; 3 filesystem error or folder exists. Contract:
scenarios/research-desk/docs/reference/FILE-CONTRACT.md §2.
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import shutil
import sys
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent
sys.dont_write_bytecode = True  # the desk runs this inside the repository; leave no __pycache__
sys.path.insert(0, str(SCRIPTS))
import mission_check as mc  # noqa: E402

SKILL = SCRIPTS.parent
TEMPLATES = SKILL / "templates"
REPO = SKILL.parents[3]
PATHS = {
    "footprint": SCRIPTS / "footprint.py",
    "adaptive_research": SKILL.parent / "adaptive-research" / "SKILL.md",
    "long_effort": REPO / "docs" / "agent-system" / "LONG_EFFORT_DESIGN.md",
    "operator_attention": SKILL / "references" / "operator-attention.md",
    "owner_brief": TEMPLATES / "owner-brief.md",
    "effort_anatomy": REPO / "scenarios/prompt-manager/store/skills/packs/core/effort-diagnosis/scripts/effort_anatomy.py",
}
SPEC_KEYS = {
    "topics", "allowance_hours", "emphasis", "stance_quotes", "supervisor_model", "supervisor_effort",
    "owner_model_tier", "run_alongside", "read_first", "supersedes", "carry_baseline_from", "slug", "window",
    "order", "card_notes",
}
MAX_CARD_WORKSTREAMS = 4  # references/direction.md: a card names two to four workstreams


def default_count(topics: int) -> int:
    """Workstreams on a card without an operator order: four, fewer as topics grow (12 in all)."""
    return max(2, min(MAX_CARD_WORKSTREAMS, 12 // topics))
NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}")  # topic folder or commission id
SLUG = re.compile(r"[a-z0-9][a-z0-9-]{0,63}")
PLACEHOLDER = re.compile(r"\{\{([a-z_]+)\}\}")
# An absolute path starts after a delimiter; it ends at whitespace, quotes, brackets or punctuation.
ABS_PATH = re.compile(r"(?<![^\s`'\"(\[])(/[^\s`'\"()\[\]<>|,;*{}$]+)")
CARD_HEADING = re.compile(r"^### Card \d+\b.*$", re.MULTILINE)


class Fail(Exception):
    def __init__(self, code: str, message: str, exit_code: int):
        super().__init__(message)
        self.code, self.message, self.exit_code = code, message, exit_code


def today() -> dt.date:
    return dt.date.today()


def now_utc() -> str:
    return dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def load_limits() -> dict:
    return json.loads((TEMPLATES / "limits.json").read_text(encoding="utf-8"))


def render(template: str, values: dict[str, str]) -> str:
    def sub(m: re.Match) -> str:
        name = m.group(1)
        if name not in values:
            raise Fail("unknown_placeholder", f"template placeholder {{{{{name}}}}} has no value", 2)
        if values[name] is None:
            raise Fail("unfilled_placeholder", f"template placeholder {{{{{name}}}}} is unfilled", 2)
        return values[name]

    return PLACEHOLDER.sub(sub, template)


# ---------------------------------------------------------------- spec


def normalize_spec(raw: object, home: Path, limits: dict) -> dict:
    def bad(msg: str, code: str = "invalid_spec") -> Fail:
        return Fail(code, msg, 2)

    if not isinstance(raw, dict):
        raise bad("spec must be a JSON object")
    unknown = sorted(set(raw) - SPEC_KEYS)
    if unknown:
        raise bad(f"unknown spec fields: {', '.join(unknown)}")
    spec = {k: raw.get(k) for k in sorted(SPEC_KEYS)}
    topics = spec["topics"]
    if not isinstance(topics, list) or not 1 <= len(topics) <= 6 or not all(isinstance(t, str) for t in topics):
        raise bad("topics must be a list of 1-6 topic names")
    if len(set(topics)) != len(topics):
        raise bad("topics repeat")
    for t in topics:
        if not NAME.fullmatch(t) or not (home / "topics" / t / "MISSION.md").is_file():
            raise bad(f"topic {t!r} has no topics/{t}/MISSION.md", "unknown_topic")
    hours = spec["allowance_hours"]
    if not isinstance(hours, int) or isinstance(hours, bool) or not 1 <= hours <= 72:
        raise bad("allowance_hours must be an integer from 1 to 72")
    quotes = spec["stance_quotes"]
    if not isinstance(quotes, list) or not quotes or not all(isinstance(q, str) and q.strip() for q in quotes):
        raise bad("stance_quotes must hold at least one non-empty quote")
    spec["emphasis"] = spec["emphasis"] or ""
    if not isinstance(spec["emphasis"], str):
        raise bad("emphasis must be a string")
    for key, default in limits["defaults"].items():
        spec[key] = spec[key] or default
        allowed = limits["models"][key]
        if spec[key] not in allowed:
            raise bad(f"{key} must be one of {', '.join(allowed)} (templates/limits.json models)")
    spec["run_alongside"] = spec["run_alongside"] or []
    if not isinstance(spec["run_alongside"], list) or not all(isinstance(c, str) and NAME.fullmatch(c) for c in spec["run_alongside"]):
        raise bad("run_alongside must be a list of commission ids")
    spec["read_first"] = spec["read_first"] or {}
    if not isinstance(spec["read_first"], dict):
        raise bad("read_first must map topic to an absolute path")
    for t, p in spec["read_first"].items():
        if t not in topics:
            raise bad(f"read_first names {t!r}, which is not a selected topic")
        if not isinstance(p, str) or not p.startswith("/") or not Path(p).is_file() \
                or not Path(p).resolve().is_relative_to(home.resolve()):
            raise bad(f"read_first for {t} must be an existing file under {home}: {p}")
    spec["order"] = spec["order"] or {}
    if not isinstance(spec["order"], dict):
        raise bad("order must map topic to a list of workstream ids")
    for t, ids in spec["order"].items():
        if t not in topics:
            raise bad(f"order names {t!r}, which is not a selected topic")
        if not isinstance(ids, list) or not 1 <= len(ids) <= MAX_CARD_WORKSTREAMS or len(set(ids)) != len(ids):
            raise bad(f"order for {t} must list 1-{MAX_CARD_WORKSTREAMS} distinct workstream ids")
        live = [w.ident for w in mc.check_mission(t, home / "topics" / t / "MISSION.md").workstreams if w.status in mc.LIVE]
        for i in ids:
            if i not in live:
                raise bad(f"order for {t} names {i!r}, which is not an open, active or approved workstream "
                          f"(live: {', '.join(live) or 'none'})")
    spec["card_notes"] = spec["card_notes"] or {}
    if not isinstance(spec["card_notes"], dict):
        raise bad("card_notes must map topic to text")
    for t, text in spec["card_notes"].items():
        if t not in topics:
            raise bad(f"card_notes names {t!r}, which is not a selected topic")
        if not isinstance(text, str) or not text.strip():
            raise bad(f"card_notes for {t} must be non-empty text")
        if re.search(r"^#", text, re.MULTILINE):
            raise bad(f"card_notes for {t} must not contain a heading line (it would split the charter)")
    sup = spec["supersedes"] or []
    spec["supersedes"] = [sup] if isinstance(sup, str) else sup
    if not isinstance(spec["supersedes"], list) or not all(isinstance(c, str) and NAME.fullmatch(c) for c in spec["supersedes"]) \
            or len(set(spec["supersedes"])) != len(spec["supersedes"]):
        raise bad("supersedes must be a commission id or a list of distinct ones")
    for c in spec["supersedes"]:
        if not (home / "cohorts" / c).is_dir():
            raise bad(f"supersedes: cohorts/{c} does not exist")
    val = spec["carry_baseline_from"]
    if val is not None and (not isinstance(val, str) or not NAME.fullmatch(val)):
        raise bad("carry_baseline_from must be a commission id or null")
    if spec["carry_baseline_from"] and not (home / "cohorts" / spec["carry_baseline_from"] / "BASELINE.json").is_file():
        raise bad(f"carry_baseline_from: cohorts/{spec['carry_baseline_from']}/BASELINE.json does not exist")
    spec["slug"] = spec["slug"] or "research"
    if not isinstance(spec["slug"], str) or not SLUG.fullmatch(spec["slug"]):
        raise bad("slug must match [a-z0-9][a-z0-9-]{0,63}")
    spec["window"] = spec["window"] or ""
    if not isinstance(spec["window"], str) or "\n" in spec["window"]:
        raise bad("window must be one line")
    return spec


def commission_id(home: Path, slug: str) -> str:
    base = f"{slug}-{today():%Y%m%d}"
    cid, n = base, 1
    while (home / "cohorts" / cid).exists():
        n += 1
        cid = f"{base}-{n}"
    return cid


# ---------------------------------------------------------------- missions and cards


def mission_title(text: str, topic: str) -> str:
    m = re.search(r"^# (.+)$", text, re.MULTILINE)
    if not m:
        return topic
    return re.sub(r"\s*[—-]\s*mission\s*$", "", m.group(1).strip(), flags=re.IGNORECASE) or topic


def list_files(topic_dir: Path) -> list[Path]:
    out = []
    for p in sorted(topic_dir.glob("*.md")):
        if p.name in ("MISSION.md", "README.md"):
            continue
        try:
            if re.search(r"^## Wanted\b", p.read_text(encoding="utf-8", errors="replace"), re.MULTILINE):
                out.append(p)
        except OSError:
            continue
    return out


def read_mission(home: Path, topic: str) -> tuple[mc.TopicReport, str]:
    path = home / "topics" / topic / "MISSION.md"
    text = path.read_text(encoding="utf-8")
    return mc.check_mission(topic, path), mission_title(text, topic)


def and_join(items: list[str]) -> str:
    return items[0] if len(items) == 1 else ", ".join(items[:-1]) + " and " + items[-1]


def card_workstreams(rep: mc.TopicReport, order: list[str] | None, count: int) -> list:
    """The operator's order, else active, then approved, then open rows, capped (S1)."""
    rows = {w.ident: w for w in rep.workstreams}
    if order:
        return [rows[i] for i in order]
    ranked = [w for s in ("active", "approved", "open") for w in rep.workstreams if w.status == s]
    return ranked[:count]


ORDER_LABEL = "- **Order** (the operator's; land in this order):"
CHOICE_LABEL = "- **Workstreams** (active, approved, then open; not a priority order: choose by scorecard movement):"


def render_card(n: int, home: Path, cid: str, topic: str, rep: mc.TopicReport, title: str,
                read_first: str | None, order: list[str] | None, notes: str | None, count: int) -> tuple[str, dict]:
    tdir = home / "topics" / topic
    chosen = card_workstreams(rep, order, count)
    lists = ", ".join(f"`{p.name}`" for p in list_files(tdir))
    lines = [
        f"### Card {n} — {title} · mode `{rep.mode}`",
        "",
        f"- **Folder:** `{tdir}/` — `MISSION.md`{'; list files ' + lists if lists else ''}; run folder `runs/{cid}/`.",
    ]
    if read_first:
        lines.append(f"- **Read first:** `{read_first}`")
    lines.append(ORDER_LABEL if order else CHOICE_LABEL)
    lines += [f"  {i}. **{w.ident}** ({' '.join(w.magnitude.split())}): {' '.join(w.showcase.split())}"
              for i, w in enumerate(chosen, 1)]
    if notes:
        lines += ["", "Operator's notes for this card, verbatim:", "", notes.strip("\n")]
    card = {"topic": topic, "mode": rep.mode, "order": [w.ident for w in chosen],
            "order_source": "operator" if order else "mission", "read_first": read_first}
    return "\n".join(lines) + "\n", card


# ---------------------------------------------------------------- texts


def quote_text(q: str) -> str:
    q = q.strip()
    return q if q[:1] in "\"“" else f"\"{q}\""


def kernel_excerpt(source: str, budget: int) -> str:
    text = " ".join(source.strip().strip("\"“”").split())
    if len(text) <= budget:
        return text.rstrip(".")
    for part in re.split(r"(?<=[.!?])\s+", text):
        if 0 < len(part) <= budget:
            return part.rstrip(".")
    for part in re.split(r"(?<=[,;—])\s+", text):
        part = part.rstrip(",;— ")
        if 0 < len(part) <= budget:
            return part
    cut = text[: max(budget - 3, 0)].rsplit(" ", 1)[0]
    return cut + "..."


def compose(home: Path, spec: dict, cid: str) -> tuple[dict[str, str], list[dict], list[dict]]:
    """Render charter, stance and kernel. Returns texts, cards and mission errors."""
    limits = load_limits()
    errors: list[dict] = []
    cards_md, cards, titles, open_hours = [], [], [], []
    for n, topic in enumerate(spec["topics"], 1):
        rep, title = read_mission(home, topic)
        errors += [{"code": "mission_error", "message": f"{topic}: {e}"} for e in rep.errors]
        md, card = render_card(n, home, cid, topic, rep, title, spec["read_first"].get(topic),
                               spec["order"].get(topic), spec["card_notes"].get(topic),
                               default_count(len(spec["topics"])))
        cards_md.append(md)
        cards.append(card)
        titles.append(title)
        open_hours.append(str(rep.open_hours()))
    folder = home / "cohorts" / cid
    owners = len(spec["topics"])
    quotes = [quote_text(q) for q in spec["stance_quotes"]]
    emphasis = spec["emphasis"].strip()
    ordered = sum(t in spec["order"] for t in spec["topics"])
    phrase = ("in order" if ordered == owners else "in the order a card gives, else by scorecard movement" if ordered
              else "chosen by scorecard movement")
    values = {
        "order_phrase": phrase,
        "kernel_order": " in order" if ordered == owners else " (in order where given)" if ordered else "",
        "title": f"Research commission {cid} — {' · '.join(titles)}",
        "commission_id": cid,
        "research_home": str(home),
        "charter_path": str(folder / "README.md"),
        "topic_titles": " · ".join(titles),
        "window_phrase": f", {spec['window']} window" if spec["window"] else "",
        "stance_block": "\n\n".join("> " + q.replace("\n", "\n> ") for q in quotes) + "\n",
        "stance_quotes": "\n\n".join(quotes),
        "emphasis_block": f"\n## Emphasis for this window (operator's words)\n\n> {quote_text(emphasis)}\n" if emphasis else "",
        "emphasis_paragraph": f"\nThis window's emphasis, in my words: {quote_text(emphasis)}\n" if emphasis else "",
        "alongside_line": (f"Runs alongside {', '.join(f'`{c}`' for c in spec['run_alongside'])} by the operator's choice; "
                           "edit only this commission's items in shared files.\n") if spec["run_alongside"] else "",
        "supersedes_line": ("Supersedes " + and_join([f"`{c}`" for c in spec["supersedes"]])
                            + (f"; `BASELINE.json` carried from `{spec['carry_baseline_from']}`" if spec["carry_baseline_from"] else "")
                            + ".\n") if spec["supersedes"] else "",
        "open_hours": and_join(open_hours),
        "owner_hours": f"{owners} × {spec['allowance_hours']}",
        "allowance_hours": str(spec["allowance_hours"]),
        "cards": "\n".join(cards_md),
        "owner_count": str(owners),
        "held_slots": str(owners + 1),
        "slots": str(8 - owners - 1),
        "supervisor_model": spec["supervisor_model"],
        "supervisor_effort": spec["supervisor_effort"],
        "owner_model_line": ("`model` and `reasoning_effort` omitted (owners inherit this session's model)"
                             if spec["owner_model_tier"] == "inherit"
                             else f"`model` set to `{spec['owner_model_tier']}` and `reasoning_effort` omitted"),
        "owner_model": spec["supervisor_model"] if spec["owner_model_tier"] == "inherit" else spec["owner_model_tier"],
        "watch_args": " ".join(f"--watch {home / 'topics' / t / 'runs' / cid}" for t in spec["topics"]),
        "start_date": f"{today():%Y-%m-%d}",
        **{k: str(v) for k, v in PATHS.items()},
    }
    texts = {
        "charter": render((TEMPLATES / "charter.md").read_text(encoding="utf-8"), values),
        "stance": render((TEMPLATES / "stance.txt").read_text(encoding="utf-8"), values),
    }
    kernel_t = (TEMPLATES / "kernel.txt").read_text(encoding="utf-8")
    budget = limits["gates"]["kernel_chars"] - len(render(kernel_t, {**values, "kernel_excerpt": ""}))
    values["kernel_excerpt"] = kernel_excerpt(emphasis or spec["stance_quotes"][0], budget)
    texts["kernel"] = render(kernel_t, values)
    return texts, cards, errors


# ---------------------------------------------------------------- gates


def candidate_paths(text: str, roots: list[Path], cid: str) -> list[str]:
    out = []
    for m in ABS_PATH.finditer(text):
        if text[m.end(): m.end() + 1] in ("*", "{", "$", "<", "["):
            continue  # a pattern or placeholder, not a path
        p = m.group(1).split("#", 1)[0].rstrip(".:!?")
        if not p or cid in p or "..." in p:
            continue  # the commission creates its own folder and run folders
        if any(p == str(r) or p.startswith(str(r) + "/") for r in roots) and p not in out:
            out.append(p)
    return out


SECTION = re.compile(r"^#{2,3} .*$", re.MULTILINE)
NOTES_LABEL = "Operator's notes for this card, verbatim:"


def largest_parts(name: str, text: str) -> list[tuple[str, int]]:
    """Name the biggest parts of an oversized text, biggest first (S6)."""
    parts: list[tuple[str, int]] = []
    if name == "charter":
        fences = [m.start() for m in re.finditer(r"^```", text, re.MULTILINE)]
        heads = [h for h in SECTION.finditer(text) if sum(f < h.start() for f in fences) % 2 == 0]
        for i, h in enumerate(heads):
            sec = text[h.start(): heads[i + 1].start() if i + 1 < len(heads) else len(text)]
            label = h.group(0).lstrip("# ").split(" · mode")[0]
            if NOTES_LABEL in sec:
                notes = len(sec.split(NOTES_LABEL, 1)[1].encode("utf-8"))
                label += f" (notes {notes:,})"
            parts.append((label, len(sec.encode("utf-8"))))
    else:
        parts = [(f"paragraph {i}: {' '.join(p.split())[:40]}…", len(p))
                 for i, p in enumerate(re.split(r"\n\s*\n", text), 1) if p.strip()]
    return sorted(parts, key=lambda x: -x[1])[:4]


def overflow_error(name: str, text: str, value: int, limit: int, unit: str) -> dict:
    parts = "; ".join(f"{label} {size:,}" for label, size in largest_parts(name, text))
    return {"code": "gate_overflow",
            "message": f"{name} is {value:,} {unit}, {value - limit:,} over {limit:,}; largest parts: {parts}. "
                       "Cut card notes or a quote, or order fewer workstreams on the largest cards."}


def gate(texts: dict[str, str | None], limits: dict, home: Path, cid: str) -> tuple[list[dict], list[dict]]:
    g = limits["gates"]
    gates, errors = [], []

    def add(name, value, limit, ok):
        gates.append({"name": name, "value": value, "limit": limit, "ok": bool(ok)})

    charter, stance, kernel = texts.get("charter"), texts.get("stance"), texts.get("kernel")
    cb = len(charter.encode("utf-8")) if charter is not None else None
    add("charter_bytes", cb, g["charter_bytes"], cb is not None and cb <= g["charter_bytes"])
    sc = len(stance) if stance is not None else None
    add("stance_chars", sc, g["stance_chars"], sc is not None and sc <= g["stance_chars"])
    ending = stance.rstrip()[-len(g["stance_ending"]):] if stance is not None else None
    add("stance_ending", ending, g["stance_ending"], ending == g["stance_ending"])
    kc = len(kernel.rstrip("\n")) if kernel is not None else None
    add("kernel_chars", kc, g["kernel_chars"], kc is not None and kc <= g["kernel_chars"])
    for name, value, limit, unit in (("charter", cb, g["charter_bytes"], "bytes"), ("stance", sc, g["stance_chars"], "characters"),
                                     ("kernel", kc, g["kernel_chars"], "characters")):
        if value is not None and value > limit:
            errors.append(overflow_error(name, texts[name], value, limit, unit))
    roots = sorted({home.resolve(), home, REPO, Path.home()}, key=lambda p: -len(str(p)))
    missing = []
    for name in ("charter", "stance", "kernel"):
        for p in candidate_paths(texts.get(name) or "", roots, cid):
            if not Path(p).exists() and p not in missing:
                missing.append(p)
                errors.append({"code": "missing_path", "message": f"{name}: {p} does not exist"})
    add("paths_exist", len(missing), 0, not missing)
    return gates, errors


# ---------------------------------------------------------------- commands


def result(cid, folder, written=False, gates=(), cards=(), texts=None, errors=()) -> dict:
    gates, errors = list(gates), list(errors)
    return {
        "schema": "research-packet-result/v1",
        "ok": all(x["ok"] for x in gates) and not errors and bool(gates),
        "commission_id": cid,
        "folder": str(folder) if folder else None,
        "written": written,
        "gates": gates,
        "cards": list(cards),
        "texts": texts,
        "errors": errors,
    }


def exit_code(res: dict) -> int:
    codes = {e["code"] for e in res["errors"]}
    if codes & {"invalid_spec", "unknown_topic", "mission_error", "unknown_placeholder", "unfilled_placeholder"}:
        return 2
    if codes & {"filesystem", "folder_exists", "not_found"}:
        return 3
    return 0 if res["ok"] else 1


def read_spec(arg: str) -> object:
    try:
        raw = sys.stdin.read() if arg == "-" else Path(arg).read_text(encoding="utf-8")
    except OSError as e:
        raise Fail("filesystem", f"cannot read spec: {e}", 3)
    try:
        return json.loads(raw)
    except json.JSONDecodeError as e:
        raise Fail("invalid_spec", f"spec is not JSON: {e}", 2)


def prepare(home: Path, spec_arg: str) -> tuple[dict, str, dict, list, list, list]:
    limits = load_limits()
    spec = normalize_spec(read_spec(spec_arg), home, limits)
    cid = commission_id(home, spec["slug"])
    texts, cards, mission_errors = compose(home, spec, cid)
    gates, gate_errors = gate(texts, limits, home, cid)
    gates.append({"name": "missions", "value": len({e["message"].split(":", 1)[0] for e in mission_errors}),
                  "limit": 0, "ok": not mission_errors})
    return spec, cid, texts, cards, gates, mission_errors + gate_errors


def cmd_preview(home: Path, spec_arg: str) -> dict:
    _, cid, texts, cards, gates, errors = prepare(home, spec_arg)
    return result(cid, home / "cohorts" / cid, False, gates, cards, texts, errors)


def cmd_build(home: Path, spec_arg: str) -> dict:
    spec, cid, texts, cards, gates, errors = prepare(home, spec_arg)
    folder = home / "cohorts" / cid
    res = result(cid, folder, False, gates, cards, None, errors)
    if not res["ok"]:
        return res
    tmp = home / "cohorts" / f".{cid}.tmp"
    try:
        if tmp.exists():
            shutil.rmtree(tmp)  # a stale temp folder from an interrupted build of this id
        tmp.mkdir(parents=True)
        (tmp / "README.md").write_text(texts["charter"], encoding="utf-8")
        (tmp / "STANCE-MESSAGE.txt").write_text(texts["stance"], encoding="utf-8")
        (tmp / "KERNEL.txt").write_text(texts["kernel"], encoding="utf-8")
        packet = {"schema": "research-desk.packet/v1", "commission_id": cid, "built_at": now_utc(),
                  "template_version": load_limits()["template_version"], "spec": spec}
        (tmp / "PACKET.json").write_text(json.dumps(packet, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
        if spec["carry_baseline_from"]:
            shutil.copyfile(home / "cohorts" / spec["carry_baseline_from"] / "BASELINE.json", tmp / "BASELINE.json")
        if folder.exists():
            shutil.rmtree(tmp)
            return {**res, "ok": False, "errors": [{"code": "folder_exists", "message": f"{folder} exists"}]}
        tmp.rename(folder)
    except OSError as e:
        shutil.rmtree(tmp, ignore_errors=True)
        return {**res, "ok": False, "errors": [{"code": "filesystem", "message": str(e)}]}
    return {**res, "written": True}


def read_opt(path: Path) -> str | None:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except OSError:
        return None


def check_cards(home: Path, charter: str, topics: list[str]) -> list[dict]:
    cards = []
    heads = list(CARD_HEADING.finditer(charter))
    sections = [charter[h.start(): heads[i + 1].start() if i + 1 < len(heads) else len(charter)] for i, h in enumerate(heads)]
    for sec in sections:
        m = re.search(r"topics/([A-Za-z0-9][A-Za-z0-9._-]*)/(?:MISSION\.md|`)", sec)
        if not m:
            continue
        topic = m.group(1)
        mode = re.search(r"mode `([a-z-]+)`", sec)
        rf = re.search(r"Read first:\*\*\s*`?([^`\s]+)", sec)
        order = None
        path = home / "topics" / topic / "MISSION.md"
        source = "operator" if ORDER_LABEL in sec else "mission" if CHOICE_LABEL in sec else None
        if source:  # a card built from templates/: its list, not the notes, is the order
            listed = sec.split(ORDER_LABEL if source == "operator" else CHOICE_LABEL, 1)[1].split("\n\n", 1)[0]
            order = re.findall(r"^  \d+\. \*\*([A-Za-z0-9_-]+)\*\*", listed, re.MULTILINE)
        elif path.is_file():  # an older card: mission workstream IDs in order of first mention
            ids = [w.ident for w in mc.check_mission(topic, path).workstreams]
            first = {i: m.start() for i in ids
                     if (m := re.search(rf"(?<![A-Za-z0-9]){re.escape(i)}(?![A-Za-z0-9])", sec))}
            order = sorted(first, key=first.get)
        cards.append({"topic": topic, "mode": mode.group(1) if mode else None, "order": order,
                      "order_source": source, "read_first": rf.group(1) if rf else None})
    if not cards:
        cards = [{"topic": t, "mode": None, "order": None, "order_source": None, "read_first": None} for t in topics]
    return cards


def cmd_check(home: Path, cid: str) -> dict:
    folder = home / "cohorts" / cid
    if not NAME.fullmatch(cid) or not folder.is_dir():
        return result(cid, folder, errors=[{"code": "not_found", "message": f"{folder} is not a commission folder"}])
    limits = load_limits()
    texts = {"charter": read_opt(folder / "README.md"), "stance": read_opt(folder / "STANCE-MESSAGE.txt"),
             "kernel": read_opt(folder / "KERNEL.txt")}
    errors = [{"code": "missing_file", "message": f"{name} file is missing"} for name, t in texts.items() if t is None]
    gates, gate_errors = gate(texts, limits, home, cid)
    packet = read_opt(folder / "PACKET.json")
    topics: list[str] = []
    if packet:
        try:
            topics = list(json.loads(packet)["spec"]["topics"])
        except (ValueError, KeyError, TypeError):
            errors.append({"code": "invalid_packet", "message": "PACKET.json is not research-desk.packet/v1"})
    if not topics:
        for t in re.findall(r"topics/([a-z0-9][a-z0-9._-]*)/MISSION\.md", texts["charter"] or ""):
            if t not in topics:
                topics.append(t)
    mission_errors = []
    for t in topics:
        path = home / "topics" / t / "MISSION.md"
        if not path.is_file():
            mission_errors.append({"code": "unknown_topic", "message": f"{t}: topics/{t}/MISSION.md does not exist"})
            continue
        mission_errors += [{"code": "mission_error", "message": f"{t}: {e}"} for e in mc.check_mission(t, path).errors]
    gates.append({"name": "missions", "value": len({e["message"].split(":", 1)[0] for e in mission_errors}),
                  "limit": 0, "ok": not mission_errors})
    cards = check_cards(home, texts["charter"] or "", topics)
    return result(cid, folder, False, gates, cards, None, errors + gate_errors + mission_errors)


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("preview", "build"):
        p = sub.add_parser(name)
        p.add_argument("--home", type=Path, required=True)
        p.add_argument("--spec", required=True, help="spec JSON file, or - for stdin")
    p = sub.add_parser("check")
    p.add_argument("--home", type=Path, required=True)
    p.add_argument("commission_id")
    args = ap.parse_args(argv)
    home = args.home.expanduser().absolute()
    try:
        if not (home / "topics").is_dir() or not (home / "cohorts").is_dir():
            raise Fail("filesystem", f"{home} is not a research home (needs topics/ and cohorts/)", 3)
        if args.cmd == "preview":
            res = cmd_preview(home, args.spec)
        elif args.cmd == "build":
            res = cmd_build(home, args.spec)
        else:
            res = cmd_check(home, args.commission_id)
    except Fail as f:
        res = result(None, None, errors=[{"code": f.code, "message": f.message}])
    except OSError as e:
        res = result(None, None, errors=[{"code": "filesystem", "message": str(e)}])
    print(json.dumps(res, indent=2, ensure_ascii=False))
    return exit_code(res)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
