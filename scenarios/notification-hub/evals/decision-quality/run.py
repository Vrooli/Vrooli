#!/usr/bin/env python3
"""Decision-quality eval of the notification-hub skill, run by Codex Luna.

Each case in cases/ holds the facts of a real Hub decision. One Codex Luna run
per case gets only those facts and the skill, prepares the decision, and checks
it through check-decision (always a dry run: nothing reaches the operator).
This script launches the runs, collects what they made, re-runs the Hub's check
on the final attempt, photographs the page at 390 px in light and dark the way
the Hub's frame shows it, audits its contrast in the browser, and writes a
summary.

    python3 run.py all --round after              # launch, wait once, collect
    python3 run.py launch --round baseline --cases rebaseline,policy-batch
    python3 run.py collect --round baseline
    python3 run.py shoot page.html --out DIR      # photograph any page

Evidence is kept under ~/.vrooli/research/notification-hub-decision-eval/<round>/.
The baseline round reads the skill as it was before the decision-quality work
(skill-baseline/ there); the after round reads the canonical skill.

Launching needs Agent Manager run creation: an operator shell (--local-owner),
or an agent session that holds a run-creation grant.
"""

from __future__ import annotations

import argparse
import base64
import datetime as dt
import glob
import json
import mimetypes
import os
import re
import shlex
import shutil
import subprocess
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
SCENARIO = HERE.parent.parent
REPO = SCENARIO.parent.parent
KIT = SCENARIO / "ui/src/features/packet/explainer"
SKILL = SCENARIO / "skills/notification-hub"
ROOT = Path(os.environ.get("DECISION_EVAL_ROOT", Path.home() / ".vrooli/research/notification-hub-decision-eval"))
TRIALS_PROFILE_ID = "bfe024af-9ff7-4fdd-8d58-b89f2a7e0bc1"  # meta-optimization-manager/trials, role code.cheap
WIDTH = 390
HEIGHT = 844
MIN_CONTRAST = 3.0
KIT_BLOCKS = ("hub-proportion", "hub-groups", "hub-command", "hub-recurrence", "hub-matrix", "hub-compare",
              "hub-timeline", "hub-tradeoffs", "hub-callout", "hub-figure", "hub-pair", "hub-chip")


def die(message: str) -> None:
    print(f"run.py: {message}", file=sys.stderr)
    sys.exit(1)


def sh(args: list[str], *, cwd: Path | None = None, check: bool = True, timeout: int = 600) -> subprocess.CompletedProcess:
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    if check and result.returncode != 0:
        die(f"{' '.join(args[:4])} … failed ({result.returncode}): {(result.stderr or result.stdout).strip()[:800]}")
    return result


def as_json(text: str):
    text = text.strip()
    start = min([i for i in (text.find("{"), text.find("[")) if i >= 0], default=-1)
    return json.loads(text[start:]) if start >= 0 else None


def cases(selected: str | None) -> list[str]:
    names = sorted(Path(p).stem for p in glob.glob(str(HERE / "cases/*.md")))
    if selected:
        wanted = [name.strip() for name in selected.split(",") if name.strip()]
        unknown = sorted(set(wanted) - set(names))
        if unknown:
            die(f"unknown case(s) {unknown}; cases: {names}")
        return wanted
    return names


def round_name(value: str) -> str:
    if not re.fullmatch(r"baseline|after(-[0-9]+)?", value):
        raise argparse.ArgumentTypeError("a round is baseline, after or after-<n>")
    return value


def case_dir(round_name: str, case: str) -> Path:
    return ROOT / round_name / case


def skill_dir(round_name: str) -> Path:
    if round_name == "baseline":
        snapshot = ROOT / "skill-baseline"
        if not (snapshot / "SKILL.md").is_file():
            die(f"the baseline skill snapshot is missing: {snapshot}")
        return snapshot
    return SKILL


# --- launch ---------------------------------------------------------------------


def prompt_for(round_name: str, case: str, out: Path) -> str:
    template = (HERE / "prompt.md").read_text()
    facts = (HERE / "cases" / f"{case}.md").read_text()
    return (template.replace("{{SKILL_DIR}}", str(skill_dir(round_name)))
            .replace("{{CASE_DIR}}", str(out))
            .replace("{{WORK_DIR}}", str(out / "work"))
            .replace("{{CHECK}}", str(out / "check-decision"))
            .replace("{{CASE}}", case)
            .replace("{{FACTS}}", facts.strip()))


def prepare_case(round_name: str, case: str) -> Path:
    out = case_dir(round_name, case)
    if out.exists():
        stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
        out.rename(out.with_name(f"{case}.previous-{stamp}"))
    (out / "work").mkdir(parents=True)
    media = HERE / "cases" / case
    if media.is_dir():
        shutil.copytree(media, out / "work" / "evidence")
    wrapper = out / "check-decision"
    wrapper.write_text(f'#!/usr/bin/env bash\nDECISION_EVAL_OUT="{out}" exec "{HERE / "bin/check-decision"}" "$@"\n')
    wrapper.chmod(0o755)
    (out / "prompt.md").write_text(prompt_for(round_name, case, out))
    return out


def profile_id(explicit: str | None) -> str:
    """The eval runs on the code.cheap trials profile (GPT-6 Luna medium) unless
    --profile names another; each run's record confirms the model."""
    return explicit or TRIALS_PROFILE_ID


def launch(args) -> None:
    profile = profile_id(args.profile)
    stamp = dt.datetime.now().strftime("%Y%m%d-%H%M%S")
    for case in cases(args.cases):
        out = prepare_case(args.round, case)
        task = sh(["agent-manager", "task", "create", "--title", f"Decision eval {args.round}: {case}",
                   "--description", "Prepare one Notification Hub decision with a dry run; send nothing.",
                   "--project-root", str(REPO), "--scope-path", str(out), "--created-by", "notification-hub-decision-eval",
                   "--json"], timeout=120)
        task_id = (as_json(task.stdout) or {}).get("id") or ((as_json(task.stdout) or {}).get("task") or {}).get("id")
        if not task_id:
            die(f"task create returned no id: {task.stdout[:400]}")
        create = ["agent-manager", "run", "create", "--profile-id", profile, "--task-id", task_id,
                  "--prompt", (out / "prompt.md").read_text(), "--run-mode", "in_place",
                  # The case folder is outside the repository, so the run works in place
                  # with no sandbox, as the declared notification-hub/decision-eval profile says.
                  "--sandbox-config", json.dumps({"mode": "SANDBOX_MODE_OFF", "networkMode": "NETWORK_ACCESS_LOCALHOST"}),
                  "--workload-key", "notification-hub-decision-eval", "--workload-kind", "adhoc",
                  "--idempotency-key", f"nh-decision-eval-{args.round}-{case}-{stamp}", "--json"]
        if args.local_owner:
            create.append("--local-owner")
        run = sh(create, timeout=180)
        data = as_json(run.stdout) or {}
        run_id = data.get("id") or (data.get("run") or {}).get("id")
        if not run_id:
            die(f"run create returned no id: {run.stdout[:400]}")
        (out / "run.json").write_text(json.dumps({"round": args.round, "case": case, "task_id": task_id, "run_id": run_id,
                                                   "profile_id": profile, "launched_at": stamp}, indent=1))
        print(f"launched {case}: run {run_id}")


# --- collect --------------------------------------------------------------------


def find_models(value, path: str = "") -> list[str]:
    found = []
    if isinstance(value, dict):
        for key, inner in value.items():
            found += find_models(inner, f"{path}.{key}")
    elif isinstance(value, list):
        for inner in value:
            found += find_models(inner, path)
    elif isinstance(value, str) and "model" in path.lower() and value and not path.lower().endswith("reason"):
        found.append(f"{path.lstrip('.')}={value}")
    return found


def last_attempt(out: Path) -> Path | None:
    attempts = sorted(p for p in (out / "attempts").glob("*") if p.is_dir())
    return attempts[-1] if attempts else None


def flag_values(argv: list[str], flag: str) -> list[str]:
    return [value for name, value in zip(argv, argv[1:]) if name == flag]


def attempt_file(attempt: Path, cwd: str, value: str) -> Path | None:
    copy = attempt / "files" / os.path.basename(value.split("=", 1)[0])
    if copy.is_file():
        return copy
    path = Path(cwd, os.path.expanduser(value.split("=", 1)[0]))
    return path if path.is_file() else None


def collect(args) -> None:
    for case in cases(args.cases):
        out = case_dir(args.round, case)
        meta_path = out / "run.json"
        if not meta_path.is_file():
            print(f"{case}: not launched, skipped")
            continue
        meta = json.loads(meta_path.read_text())
        waited = sh(["agent-manager", "run", "wait", meta["run_id"], "--timeout", args.wait, "--json"], check=False, timeout=3 * 3600)
        (out / "run-wait.json").write_text(waited.stdout or waited.stderr)
        got = sh(["agent-manager", "run", "get", meta["run_id"], "--json"], check=False, timeout=120)
        (out / "run-record.json").write_text(got.stdout or got.stderr)
        facts = sh(["agent-manager", "run", "report", meta["run_id"], "--facts", "--json"], check=False, timeout=120)
        (out / "run-facts.json").write_text(facts.stdout or facts.stderr)
        models = []
        for text in (got.stdout, facts.stdout):
            try:
                models += find_models(as_json(text))
            except (ValueError, TypeError):
                pass
        meta["models"] = sorted(set(models))
        record = (as_json(got.stdout) or {}) if got.returncode == 0 else {}
        meta["status"] = ((record.get("run") or record).get("status") if got.returncode == 0 else f"unreadable: {got.stderr.strip()[:200]}")
        meta_path.write_text(json.dumps(meta, indent=1))
        judge(out, meta)
        contained = containment(out, meta)
        if contained["outside_writes"] or contained["sends"] or contained.get("problem"):
            print(f"{case}: CHECK CONTAINMENT {out / 'containment.json'}")


def judge(out: Path, meta: dict) -> dict:
    attempt = last_attempt(out)
    result = {"case": out.name, "round": out.parent.name, "status": meta.get("status"), "models": meta.get("models", []),
              "attempts": len(list((out / "attempts").glob("*"))) if (out / "attempts").is_dir() else 0}
    if attempt is None:
        result["problem"] = "no check-decision attempt"
        (out / "result.json").write_text(json.dumps(result, indent=1))
        print(f"{out.name}: no attempt")
        return result
    recorded = json.loads((attempt / "argv.json").read_text())
    argv, cwd = recorded["argv"], recorded["cwd"]
    result["final_attempt"] = attempt.name
    result["final_exit"] = (attempt / "exit").read_text().strip() if (attempt / "exit").is_file() else ""
    check = sh(["notification-hub", "conversations", "ask", *argv, "--dry-run", "--json"],
               cwd=Path(cwd) if Path(cwd).is_dir() else out, check=False, timeout=180)
    (out / "dryrun.json").write_text(check.stdout or "")
    (out / "dryrun.stderr.txt").write_text(check.stderr or "")
    try:
        report = as_json(check.stdout) or {}
    except ValueError:
        report = {}
    warnings = report.get("warnings") or []
    result["dry_run_exit"] = check.returncode
    if not report:
        # No packet check came back (for example the Hub could not verify the
        # shell), so there are no findings to report: say so instead of "none".
        result["recheck_error"] = (check.stderr or check.stdout).strip()[:400]
    result["blocking"] = sorted({w.get("code", "") for w in warnings if w.get("severity") == "blocking"})
    result["advice"] = sorted({w.get("code", "") for w in warnings if w.get("severity") == "advice"})
    result["dropped"] = sorted({w.get("code", "") for w in warnings if w.get("severity") == "dropped"})
    explainers = flag_values(argv, "--explainer")
    page_path = attempt_file(attempt, cwd, explainers[-1]) if explainers else None
    files = []
    for value in flag_values(argv, "--attach"):
        path = attempt_file(attempt, cwd, value)
        if path is not None:
            files.append(path)
    cover = cover_of(argv, attempt, cwd)
    (out / "cover.json").write_text(json.dumps(cover, indent=1))
    if page_path is not None:
        shutil.copy2(page_path, out / "page.html")
        html = page_path.read_text(errors="replace")
        result["page_bytes"] = len(html.encode())
        result["kit_blocks"] = [block for block in KIT_BLOCKS if re.search(rf'class="[^"]*\b{block}\b', html)]
        result["literal_colours"] = len(re.findall(r"#[0-9a-fA-F]{3,8}\b|\brgba?\(", html))
        result["shots"] = shoot(page_path, out / "shots", files, cover)
    else:
        result["problem"] = "no explainer page"
    (out / "result.json").write_text(json.dumps(result, indent=1))
    print(f"{out.name}: blocking={result['blocking']} advice={result['advice']} shots={'yes' if page_path else 'no'}")
    return result


def cover_of(argv: list[str], attempt: Path, cwd: str) -> dict:
    def one(flag: str) -> str:
        values = flag_values(argv, flag)
        return values[-1] if values else ""

    options = []
    for value in flag_values(argv, "--option"):
        key, _, label = value.partition("=")
        options.append({"key": key, "label": label or key})
    if "--yes-no" in argv:
        options = [{"key": "yes", "label": "Yes"}, {"key": "no", "label": "No"}]
    cover = {"question": one("--question"), "big_picture": one("--big-picture"), "from": one("--from"),
             "recommend": one("--recommend"), "reason": one("--recommend-reason"), "options": options,
             "technical_details": one("--technical-details"), "form": ""}
    body = one("--body-file")
    if body:
        path = attempt_file(attempt, cwd, body)
        if path is not None:
            text = path.read_text(errors="replace")
            text = re.sub(r"^# .*\n", "", text.lstrip())
            parts = re.split(r"^## +", text, flags=re.M)
            cover["question"] = cover["question"] or parts[0].strip().split("\n\n")[0]
            for part in parts[1:]:
                head, _, rest = part.partition("\n")
                if head.strip().lower() == "background":
                    cover["big_picture"] = rest.strip()
                if head.strip().lower() == "details":
                    cover["technical_details"] = rest.strip()
    form = one("--form")
    if form:
        path = attempt_file(attempt, cwd, form)
        cover["form"] = path.read_text(errors="replace") if path else ""
    return cover


# --- shoot ----------------------------------------------------------------------


def bundle() -> str:
    target = ROOT / "_render" / "frame.js"
    target.parent.mkdir(parents=True, exist_ok=True)
    sources = [HERE / "render/frame-entry.ts", KIT / "document.ts", KIT / "theme.ts"]
    if target.is_file() and all(target.stat().st_mtime >= s.stat().st_mtime for s in sources):
        return target.read_text()
    candidates = sorted(glob.glob(str(SCENARIO / "ui/node_modules/.pnpm/esbuild@*/node_modules/esbuild/bin/esbuild")))
    if not candidates:
        die("esbuild is not installed under the UI's node_modules; the eval bundles the Hub's own frame code with it")
    sh([candidates[-1], str(HERE / "render/frame-entry.ts"), "--bundle", "--format=iife", "--target=es2020",
        f"--outfile={target}", "--log-level=warning"], timeout=120)
    return target.read_text()


AUDIT_JS = r"""
() => {
  const parse = (value) => {
    let m = value.match(/^rgba?\(([^)]+)\)/);
    if (m) {
      const p = m[1].split(/[\s,\/]+/).filter(Boolean).map(Number);
      return { r: p[0], g: p[1], b: p[2], a: p.length > 3 ? p[3] : 1 };
    }
    m = value.match(/^color\(srgb ([^)]+)\)/);
    if (m) {
      const p = m[1].split(/[\s\/]+/).filter(Boolean).map(Number);
      return { r: p[0] * 255, g: p[1] * 255, b: p[2] * 255, a: p.length > 3 ? p[3] : 1 };
    }
    return null;
  };
  const over = (top, bottom) => ({
    r: top.r * top.a + bottom.r * (1 - top.a),
    g: top.g * top.a + bottom.g * (1 - top.a),
    b: top.b * top.a + bottom.b * (1 - top.a),
    a: 1,
  });
  const lum = (c) => {
    const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4); };
    return 0.2126 * f(c.r) + 0.7152 * f(c.g) + 0.0722 * f(c.b);
  };
  const ratio = (a, b) => { const x = lum(a), y = lum(b); return (Math.max(x, y) + 0.05) / (Math.min(x, y) + 0.05); };
  const hex = (c) => "#" + [c.r, c.g, c.b].map((v) => Math.round(v).toString(16).padStart(2, "0")).join("");
  const backgroundOf = (el) => {
    const layers = [];
    for (let node = el; node; node = node.parentElement) {
      const style = getComputedStyle(node);
      if (style.backgroundImage && style.backgroundImage !== "none") return { unknown: true };
      const c = parse(style.backgroundColor);
      if (c && c.a > 0) layers.push(c);
      if (c && c.a >= 1) break;
    }
    const canvas = parse(getComputedStyle(document.documentElement).backgroundColor);
    let bg = matchMedia("(prefers-color-scheme: dark)").matches && !(canvas && canvas.a > 0) ? { r: 18, g: 18, b: 18, a: 1 } : { r: 255, g: 255, b: 255, a: 1 };
    for (let i = layers.length - 1; i >= 0; i--) bg = over(layers[i], bg);
    return bg;
  };
  const failures = [];
  let checked = 0;
  for (const el of document.body.querySelectorAll("*")) {
    const own = [...el.childNodes].filter((n) => n.nodeType === 3 && n.textContent.trim()).map((n) => n.textContent.trim()).join(" ");
    if (!own) continue;
    const rect = el.getBoundingClientRect();
    const style = getComputedStyle(el);
    if (rect.width === 0 || rect.height === 0 || style.visibility === "hidden" || Number(style.opacity) === 0) continue;
    const bg = backgroundOf(el);
    if (bg.unknown) continue;
    const fg = parse(style.color);
    if (!fg) continue;
    checked++;
    const value = ratio(over(fg, bg), bg);
    if (value < 3) failures.push({ text: own.slice(0, 60), tag: el.tagName.toLowerCase(), fg: hex(over(fg, bg)), bg: hex(bg), ratio: Math.round(value * 100) / 100 });
  }
  return { checked, failures: failures.slice(0, 20), failureCount: failures.length,
           theme: document.documentElement.getAttribute("data-theme"), locked: document.documentElement.getAttribute("data-hub-scheme") };
}
"""


def host_page(cover: dict) -> str:
    def esc(text: str) -> str:
        return (text or "").replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

    options = "".join(
        f'<li class="opt{" rec" if o["key"] == cover.get("recommend") else ""}"><strong>{esc(o["label"])}</strong>'
        + (f'<span class="badge">Recommended</span><small>{esc(cover.get("reason", ""))}</small>' if o["key"] == cover.get("recommend") else "")
        + "</li>" for o in cover.get("options", []))
    sender = f'<p class="from">{esc(cover.get("from") or "Agent run")} asks</p>'
    picture = f'<p class="bp">{esc(cover.get("big_picture", ""))}</p>' if cover.get("big_picture") else ""
    form = '<p class="form">+ answer form</p>' if cover.get("form") else ""
    # The cover is an approximation of the Hub's decision screen around the
    # frame; the frame itself is the Hub's own sealing code.
    return f"""<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1">
<style>{{KIT}}</style>
<style>
  body {{ max-width: none; padding: 0; margin: 0; }}
  .cover {{ padding: 1rem 1rem 0.5rem; }}
  .from {{ margin: 0 0 0.25rem; color: var(--hub-muted); font-size: 0.8125rem; }}
  h1 {{ font-size: 1.25rem; margin: 0 0 0.5rem; }}
  .bp {{ margin: 0 0 0.75rem; color: var(--hub-fg); }}
  .caption {{ margin: 0; padding: 0.375rem 1rem; color: var(--hub-muted); font-size: 0.75rem; border-top: 1px solid var(--hub-border); }}
  iframe {{ display: block; width: 100%; border: 0; border-bottom: 1px solid var(--hub-border); }}
  .answers {{ list-style: none; margin: 0; padding: 0.75rem 1rem 1.5rem; display: grid; gap: 0.5rem; }}
  .opt {{ padding: 0.75rem; border: 1px solid var(--hub-border); border-radius: var(--hub-radius); background: var(--hub-surface); }}
  .opt.rec {{ border-color: var(--hub-primary); }}
  .opt small {{ display: block; color: var(--hub-muted); margin-top: 0.25rem; }}
  .badge {{ margin-left: 0.5rem; font-size: 0.75rem; color: var(--hub-primary); }}
  .form {{ margin: 0 1rem 1rem; color: var(--hub-muted); font-size: 0.8125rem; }}
</style></head>
<body><section class="cover">{sender}<h1>{esc(cover.get("question", ""))}</h1>{picture}</section>
<p class="caption">Agent's page, sealed</p><iframe id="page" sandbox="allow-scripts" title="page"></iframe>
<ul class="answers">{options}</ul>{form}</body></html>"""


def launch_browser(playwright):
    """Playwright's own Chromium when its version is installed, else Chrome, else
    any Chromium Playwright downloaded earlier."""
    attempts = [{}, {"channel": "chrome"}]
    attempts += [{"executable_path": path} for path in
                 sorted(glob.glob(str(Path.home() / ".cache/ms-playwright/chromium-*/chrome-linux/chrome")), reverse=True)]
    last = None
    for options in attempts:
        try:
            return playwright.chromium.launch(**options)
        except Exception as error:  # noqa: BLE001 - try the next browser
            last = error
    die(f"no headless Chromium could start: {last}")


def shoot(page: Path, out_dir: Path, files: list[Path], cover: dict | None = None) -> dict:
    try:
        from playwright.sync_api import sync_playwright
    except ImportError:
        die("Python Playwright is not installed; the eval photographs pages with it")
    out_dir.mkdir(parents=True, exist_ok=True)
    frame_js = bundle()
    kit_css = (KIT / "kit/kit.css").read_text()
    payload_files = []
    for path in files:
        media = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
        data = path.read_bytes()
        payload_files.append({"name": path.name, "mediaType": media, "sizeBytes": len(data),
                              "dataUrl": f"data:{media};base64,{base64.b64encode(data).decode()}"})
    base = {"html": page.read_text(errors="replace"), "kitCss": kit_css, "kitJs": (KIT / "kit/kit.js").read_text(),
            "bridgeJs": (KIT / "frameBridge.js").read_text(), "files": payload_files}
    host = host_page(cover or {"question": page.stem}).replace("{KIT}", kit_css.replace("</style", "<\\/style"))
    report = {}
    with sync_playwright() as p:
        browser = launch_browser(p)
        for theme in ("light", "dark"):
            context = browser.new_context(viewport={"width": WIDTH, "height": HEIGHT}, device_scale_factor=2, color_scheme=theme)
            tab = context.new_page()
            tab.set_content(host.replace("<html>", f'<html data-theme="{theme}">', 1))
            tab.add_script_tag(content=frame_js)
            built = tab.evaluate("""(input) => {
                const built = window.HubFrame.build(input);
                const frame = document.getElementById('page');
                frame.style.colorScheme = built.frameTheme;
                frame.style.height = '240px';
                window.__heights = [];
                window.addEventListener('message', (event) => {
                  const m = event.data;
                  if (m && m.type === 'height' && typeof m.height === 'number') {
                    frame.style.height = Math.min(20000, Math.max(120, Math.ceil(m.height))) + 'px';
                    window.__heights.push(m.height);
                  }
                });
                frame.srcdoc = built.srcdoc;
                return { scheme: built.scheme, frameTheme: built.frameTheme };
            }""", {**base, "theme": theme})
            try:
                tab.wait_for_function("window.__heights.length > 0", timeout=8000)
            except Exception:
                pass
            tab.wait_for_timeout(700)
            frame = next((f for f in tab.frames if f != tab.main_frame), None)
            audit = frame.evaluate(AUDIT_JS) if frame else {"error": "no frame"}
            tab.screenshot(path=str(out_dir / f"first-{theme}.png"))
            # The sealed frame paints in its own process, and Chromium leaves the
            # part of it outside the viewport blank in a full-page capture, so the
            # viewport grows to the whole page first.
            full_height = tab.evaluate("document.documentElement.scrollHeight")
            tab.set_viewport_size({"width": WIDTH, "height": min(16000, max(HEIGHT, full_height))})
            tab.wait_for_timeout(400)
            tab.screenshot(path=str(out_dir / f"full-{theme}.png"))
            report[theme] = {**built, "audit": audit,
                             "first": str(out_dir / f"first-{theme}.png"), "full": str(out_dir / f"full-{theme}.png")}
            context.close()
        browser.close()
    (out_dir / "shots.json").write_text(json.dumps(report, indent=1))
    return {theme: {"scheme": r["scheme"], "shown_as": r["frameTheme"],
                    "contrast_failures": r["audit"].get("failureCount"), "first": r["first"], "full": r["full"]}
            for theme, r in report.items()}


def shoot_cmd(args) -> None:
    page = Path(args.page).resolve()
    out = Path(args.out).resolve() if args.out else page.parent / f"{page.stem}-shots"
    files = [Path(f).resolve() for f in args.attach or []]
    print(json.dumps(shoot(page, out, files, {"question": args.question or page.stem}), indent=1))


# --- containment -----------------------------------------------------------------

# Runs work in place with no sandbox, so each run's own transcript is checked for
# what it wrote and whether it sent anything: every write must land in its case
# folder, and the Hub may only be reached through check-decision (always a dry
# run) or read-only `pages` and help commands.
SEND = re.compile(r"notification-hub\s+conversations\s+(?!\S*--help)(ask|answer|revise|withdraw|dismiss|respond)\b(?![^|;&]*--help)")
WRITE_TARGET = re.compile(r"(?:\d?>>?|\btee(?:\s+-a)?)\s*(/[^\s'\"|;&<>]+|~[^\s'\"|;&<>]*)")
WRITE_VERB = re.compile(r"\b(?:mkdir|cp|mv|rm|touch|sed\s+-i\S*|chmod|ln|install)\s+([^|;&\n]+)")
SCRIPT_PATH = re.compile(r"(?:Path|open)\(\s*['\"]((?:/|~)[^'\"]+)['\"]|^\s*\w+\s*=\s*['\"]((?:/|~)[^'\"]+)['\"]", re.M)
REPO_TOOLS = re.compile(r"(?:^|[;&|(]\s*|&&\s*)(git|make|pnpm|npm|go|gofumpt)\s")


def containment(out: Path, meta: dict) -> dict:
    events = sh(["agent-manager", "run", "events", meta["run_id"], "--json"], check=False, timeout=180)
    calls = [event.get("tool_call") or {} for event in ((as_json(events.stdout) or {}).get("events") or [])
             if event.get("event_type") == "RUN_EVENT_TYPE_TOOL_CALL"]
    (out / "tool-calls.jsonl").write_text("".join(json.dumps(call) + "\n" for call in calls))
    home, inside = str(Path.home()), str(out) + "/"
    outside, sends, tools = set(), set(), set()
    for call in calls:
        step = call.get("input") or {}
        if call.get("tool_name") == "file_change":
            for changed in step.get("files") or []:
                if not str(changed.get("path", "")).startswith(inside):
                    outside.add(f"{changed.get('kind')} {changed.get('path')}")
            continue
        if step.get("status") != "in_progress":
            continue
        command = step.get("command") or ""
        try:
            words = shlex.split(command)
            inner = words[2] if words[:2] in (["/bin/bash", "-lc"], ["bash", "-lc"]) and len(words) > 2 else command
        except ValueError:
            inner = command
        # The body of a heredoc is file content, and a quoted argument is text
        # (a question, a page): neither is a command or a write target.
        commands = re.sub(r"<<[\\'\"]*(\w+)[\\'\"]*.*?\n\1(?=\n|$)", "<<BODY", inner, flags=re.S)
        unquoted = re.sub(r"'[^']*'|\"(?:[^\"\\\\]|\\\\.)*\"", "''", commands)
        if SEND.search(commands):
            sends.add(commands[:240])
        tools.update(REPO_TOOLS.findall(unquoted))
        paths = WRITE_TARGET.findall(unquoted)
        for verb in WRITE_VERB.findall(unquoted):
            paths += re.findall(r"(?:^|(?<=\s))(/[^\s'\"]+|~[^\s'\"]*)", verb)
        if re.search(r"write_text|write_bytes|\.write\(|open\([^)]*['\"][wa]", command):
            # A script that writes: every path it names is treated as a target.
            paths += [a or b for a, b in SCRIPT_PATH.findall(command)]
        for path in paths:
            path = path.replace("~", home, 1)
            if path != "/dev/null" and not path.startswith(inside):
                outside.add(path)
    result = {"tool_calls": len(calls), "outside_writes": sorted(outside), "sends": sorted(sends), "repo_tools": sorted(tools)}
    if not calls:
        result["problem"] = f"no transcript: {(events.stderr or events.stdout).strip()[:200]}"
    (out / "containment.json").write_text(json.dumps(result, indent=1))
    return result


# --- summary --------------------------------------------------------------------


def summarize(args) -> None:
    rows = []
    for case in cases(args.cases):
        path = case_dir(args.round, case) / "result.json"
        if not path.is_file():
            rows.append(f"| {case} | not collected | | | | | | |")
            continue
        r = json.loads(path.read_text())
        contained_path = path.with_name("containment.json")
        contained = json.loads(contained_path.read_text()) if contained_path.is_file() else None
        stayed = ("not checked" if contained is None else "no transcript" if contained.get("problem")
                  else "yes" if not (contained["outside_writes"] or contained["sends"]) else "NO, see containment.json")
        shots = r.get("shots") or {}
        contrast = " / ".join(f"{t}:{(shots.get(t) or {}).get('contrast_failures')}" for t in ("light", "dark")) if shots else "no page"
        model = ", ".join(m.split("=", 1)[-1] for m in r.get("models", [])[:2]) or "?"
        if r.get("recheck_error"):
            rows.append(f"| {case} | {model} | {r.get('attempts', 0)} | recheck failed | recheck failed | "
                        f"{', '.join(r.get('kit_blocks', [])) or 'none'} | {contrast} | {stayed} |")
            continue
        rows.append(f"| {case} | {model} | {r.get('attempts', 0)} | {', '.join(r.get('blocking', [])) or 'none'} | "
                    f"{', '.join(r.get('advice', [])) or 'none'} | {', '.join(r.get('kit_blocks', [])) or 'none'} | {contrast} | {stayed} |")
    table = ["| Case | Model | Attempts | Blocking (final) | Advice (final) | Kit blocks | Contrast failures | Wrote only in its case folder, sent nothing |",
             "|---|---|---|---|---|---|---|---|", *rows]
    text = f"# Round {args.round}\n\nEvidence: {ROOT / args.round}\n\n" + "\n".join(table) + "\n"
    (ROOT / args.round / "SUMMARY.md").write_text(text)
    print(text)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)
    for name in ("launch", "collect", "all", "summary", "audit"):
        cmd = sub.add_parser(name)
        cmd.add_argument("--round", required=True, type=round_name,
                         help="baseline (the frozen skill copy), after, or after-<n> for a later iteration of the live skill")
        cmd.add_argument("--cases", help="comma-separated case names (default: every case)")
        if name in ("launch", "all"):
            cmd.add_argument("--profile", help="Agent Manager profile id (default: the code.cheap trials profile)")
            cmd.add_argument("--local-owner", action="store_true", help="authenticate run creation as the local operator")
        if name in ("collect", "all"):
            cmd.add_argument("--wait", default="45m", help="how long to wait for each run (default 45m)")
    shoot_parser = sub.add_parser("shoot")
    shoot_parser.add_argument("page")
    shoot_parser.add_argument("--out")
    shoot_parser.add_argument("--attach", action="append")
    shoot_parser.add_argument("--question")
    args = parser.parse_args()
    if args.command == "launch":
        launch(args)
    elif args.command == "collect":
        collect(args)
        summarize(args)
    elif args.command == "all":
        launch(args)
        collect(args)
        summarize(args)
    elif args.command == "summary":
        summarize(args)
    elif args.command == "audit":
        for case in cases(args.cases):
            out = case_dir(args.round, case)
            if (out / "run.json").is_file():
                print(case, json.dumps(containment(out, json.loads((out / "run.json").read_text()))))
        summarize(args)
    elif args.command == "shoot":
        shoot_cmd(args)


if __name__ == "__main__":
    main()
