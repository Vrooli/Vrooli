"""Build the private, blurred review page for sensitive image-tools output.

The operator rates generated and edited images that no agent is allowed to look at.
This script produces ONE self-contained HTML file from a manifest of machine facts
and the image files, **without ever rendering or describing an image**: it reads
only the file bytes (and, to shrink the page for a phone, decodes and re-encodes
them with Pillow, which is a byte transform, not a view).

  python3 scripts/build_nsfw_review.py <nsfw-private-dir> [--out REVIEW.html] [--max-px 768]

`<nsfw-private-dir>/manifest.json` is a JSON list; one object per result:

  {"id": "G1", "kind": "gen|edit", "file": "gen/g1-lustify.png",
   "model": "lustify-sdxl-v9", "prompt": "...", "seed": 123, "seconds": 41.2,
   "job_id": "...", "status": "succeeded",
   "nsfw_score": 0.97, "review": "passed|refused|unavailable", "review_detail": "...",
   "faces": 1, "source_id": "B1 (edits only: the base the edit started from)"}

`file` is relative to the directory. Items whose review is not "passed" are listed in
the table but their image is never embedded.

The page: ids, model/prompt/seed/timing, a machine-check table, every image blurred
behind a CSS-only tap-to-reveal (a hidden checkbox + label: no JavaScript, works from a
file on a phone), single-column phone layout. Ratings come back in chat as
"G1 4/5 note".
"""

from __future__ import annotations

import argparse
import base64
import html
import io
import json
import mimetypes
import os
import sys
from typing import Any, Dict, List, Optional, Sequence

DEFAULT_MAX_PX = 768
JPEG_QUALITY = 82

CSS = """
:root{color-scheme:dark light}
*{box-sizing:border-box}
body{margin:0;font:16px/1.45 system-ui,sans-serif;background:#14161a;color:#e8e8ea;padding:12px;max-width:760px;margin-inline:auto}
h1{font-size:1.25rem;margin:.2rem 0 .6rem}
h2{font-size:1.05rem;margin:1.6rem 0 .5rem;border-bottom:1px solid #333;padding-bottom:.25rem}
p.note{color:#a9adb5;font-size:.9rem;margin:.3rem 0}
table{width:100%;border-collapse:collapse;font-size:.8rem}
th,td{padding:.3rem .35rem;border-bottom:1px solid #2a2d33;text-align:left;vertical-align:top}
th{color:#a9adb5;font-weight:600}
.card{background:#1c1f25;border:1px solid #2a2d33;border-radius:10px;margin:.9rem 0;padding:.7rem}
.card h3{margin:.1rem 0 .4rem;font-size:1rem}
.meta{font-size:.82rem;color:#c3c6cc;margin:.15rem 0;overflow-wrap:anywhere}
.meta b{color:#e8e8ea}
.prompt{font-size:.82rem;color:#c3c6cc;background:#14161a;border-radius:6px;padding:.4rem .5rem;margin:.4rem 0;overflow-wrap:anywhere}
input.rv{position:absolute;opacity:0;pointer-events:none}
label.shot{display:block;position:relative;overflow:hidden;border-radius:8px;background:#000;cursor:pointer;-webkit-tap-highlight-color:transparent}
label.shot img{display:block;width:100%;height:auto;filter:blur(28px);transform:scale(1.12);transition:filter .15s}
label.shot .tap{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;font-weight:600;background:rgba(0,0,0,.35);color:#fff;text-align:center;padding:1rem}
input.rv:checked + label.shot img{filter:none;transform:none}
input.rv:checked + label.shot .tap{display:none}
input.rv:checked + label.shot::after{content:"tap to hide";position:absolute;right:8px;bottom:8px;background:rgba(0,0,0,.6);color:#fff;font-size:.75rem;padding:.2rem .5rem;border-radius:6px}
input.rv:focus-visible + label.shot{outline:2px solid #6ea8fe}
.ok{color:#7ddc9a}.bad{color:#ff8b8b}.warn{color:#ffd479}
"""


class ReviewBuildError(Exception):
    """The manifest or an image file cannot be turned into a page."""


def _load_manifest(directory: str) -> List[Dict[str, Any]]:
    path = os.path.join(directory, "manifest.json")
    try:
        with open(path, "r", encoding="utf-8") as fh:
            items = json.load(fh)
    except (OSError, ValueError) as exc:
        raise ReviewBuildError(f"cannot read {path}: {exc}") from exc
    if not isinstance(items, list) or not items:
        raise ReviewBuildError(f"{path} must be a non-empty JSON list")
    seen = set()
    for item in items:
        ident = str(item.get("id", "")).strip()
        if not ident:
            raise ReviewBuildError("every manifest item needs an id (G1..., E1...)")
        if ident in seen:
            raise ReviewBuildError(f"duplicate id {ident}")
        seen.add(ident)
    return items


def embed_image(path: str, max_px: int) -> Dict[str, Any]:
    """Return {data_uri, width, height, bytes} for a file without displaying it.

    With `max_px` the long side is shrunk (JPEG re-encode) to keep the page small on a
    phone; 0 embeds the original bytes unchanged.
    """
    from PIL import Image  # imported lazily: only the embed step needs it

    try:
        with open(path, "rb") as fh:
            raw = fh.read()
    except OSError as exc:
        raise ReviewBuildError(f"cannot read {path}: {exc}") from exc
    try:
        with Image.open(io.BytesIO(raw)) as img:
            width, height = img.size
            if max_px and max(width, height) > max_px:
                scale = max_px / max(width, height)
                small = img.convert("RGB").resize((max(1, round(width * scale)), max(1, round(height * scale))), Image.Resampling.LANCZOS)
                out = io.BytesIO()
                small.save(out, format="JPEG", quality=JPEG_QUALITY, optimize=True)
                payload, mime = out.getvalue(), "image/jpeg"
            else:
                payload = raw
                mime = mimetypes.guess_type(path)[0] or Image.MIME.get(img.format or "", "image/png")
    except Exception as exc:  # noqa: BLE001 - a corrupt file must name itself, not crash the build
        raise ReviewBuildError(f"{path} is not a decodable image: {exc}") from exc
    return {
        "data_uri": f"data:{mime};base64,{base64.b64encode(payload).decode('ascii')}",
        "width": width,
        "height": height,
        "bytes": len(raw),
    }


def _esc(value: Any) -> str:
    return html.escape("" if value is None else str(value), quote=True)


def _fmt_score(value: Any) -> str:
    try:
        return f"{float(value):.3f}"
    except (TypeError, ValueError):
        return "n/a"


def _verdict_cell(review: str) -> str:
    cls = {"passed": "ok", "refused": "bad"}.get(review, "warn")
    return f'<span class="{cls}">{_esc(review or "unknown")}</span>'


def _table(items: Sequence[Dict[str, Any]], facts: Dict[str, Dict[str, Any]]) -> str:
    rows = []
    for item in items:
        f = facts.get(item["id"], {})
        dims = f"{f['width']}x{f['height']}" if f else "-"
        size = f"{f['bytes'] / 1024:.0f} KB" if f else "-"
        rows.append(
            "<tr>"
            f"<td><b>{_esc(item['id'])}</b></td>"
            f"<td>{_esc(item.get('model'))}</td>"
            f"<td>{_esc(item.get('status', ''))}</td>"
            f"<td>{dims}<br>{size}</td>"
            f"<td>{_fmt_score(item.get('nsfw_score'))}</td>"
            f"<td>{_verdict_cell(str(item.get('review', '')))}</td>"
            f"<td>{_esc(item.get('faces', '-'))}</td>"
            "</tr>"
        )
    head = "<tr><th>id</th><th>model</th><th>job</th><th>size</th><th>nsfw</th><th>reviewer</th><th>faces</th></tr>"
    return f"<table>{head}{''.join(rows)}</table>"


def _card(item: Dict[str, Any], image: Optional[Dict[str, Any]]) -> str:
    ident = item["id"]
    meta = [
        ("model", item.get("model")),
        ("seed", item.get("seed")),
        ("time", f"{float(item['seconds']):.1f} s" if item.get("seconds") not in (None, "") else None),
        ("job", item.get("job_id")),
        ("from", item.get("source_id")),
    ]
    meta_html = "".join(f'<div class="meta"><b>{k}</b> {_esc(v)}</div>' for k, v in meta if v not in (None, ""))
    if image:
        shot = (
            f'<input class="rv" type="checkbox" id="r-{_esc(ident)}">'
            f'<label class="shot" for="r-{_esc(ident)}">'
            f'<img alt="{_esc(ident)}" loading="lazy" src="{image["data_uri"]}">'
            '<span class="tap">Tap to reveal</span></label>'
        )
    else:
        shot = '<p class="note">No image: not embedded.</p>'
    return (
        f'<section class="card" id="{_esc(ident)}"><h3>{_esc(ident)}</h3>{meta_html}'
        f'<div class="prompt">{_esc(item.get("prompt"))}</div>{shot}</section>'
    )


def build_html(items: Sequence[Dict[str, Any]], directory: str, max_px: int = DEFAULT_MAX_PX) -> str:
    """Assemble the page. Only items whose reviewer verdict is 'passed' get an image."""
    facts: Dict[str, Dict[str, Any]] = {}
    images: Dict[str, Optional[Dict[str, Any]]] = {}
    for item in items:
        ident = item["id"]
        images[ident] = None
        rel = item.get("file")
        if not rel or item.get("review") != "passed":
            continue
        embedded = embed_image(os.path.join(directory, rel), max_px)
        facts[ident] = embedded
        images[ident] = embedded

    gens = [i for i in items if str(i["id"]).upper().startswith("G")]
    edits = [i for i in items if str(i["id"]).upper().startswith("E")]
    other = [i for i in items if i not in gens and i not in edits]
    passed = [i for i in items if images[i["id"]]]
    not_passed = [i for i in items if not images[i["id"]]]

    def section(title: str, group: Sequence[Dict[str, Any]]) -> str:
        shown = [i for i in group if images[i["id"]]]
        if not shown:
            return ""
        return f"<h2>{_esc(title)}</h2>" + "".join(_card(i, images[i["id"]]) for i in shown)

    body = [
        "<h1>Private review</h1>",
        '<p class="note">Fictional adults only. Every image is blurred until you tap it. '
        'Reply in chat with <b>G1 4/5 note</b> (id, score out of 5, optional note).</p>',
        f'<p class="note">{len(passed)} of {len(items)} results passed the minor-safety reviewer and are embedded.</p>',
        "<h2>Machine checks</h2>",
        _table(items, facts),
        section("Generated (G)", gens),
        section("Edited (E)", edits),
        section("Other", other),
    ]
    if not_passed:
        body.append("<h2>Not embedded</h2>")
        body.append(
            '<p class="note">These did not pass the reviewer (refused, unavailable or failed) so no image is shown: '
            + ", ".join(f"{_esc(i['id'])} ({_esc(i.get('review') or i.get('status') or 'no result')})" for i in not_passed)
            + ".</p>"
        )
    return (
        '<!doctype html><html lang="en"><head><meta charset="utf-8">'
        '<meta name="viewport" content="width=device-width,initial-scale=1">'
        '<meta name="robots" content="noindex,nofollow"><title>Private review</title>'
        f"<style>{CSS}</style></head><body>{''.join(body)}</body></html>"
    )


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("directory", help="the nsfw-private directory holding manifest.json and the images")
    parser.add_argument("--out", default="", help="output file (default: <directory>/REVIEW.html)")
    parser.add_argument("--max-px", type=int, default=DEFAULT_MAX_PX, help="shrink the long side to this many pixels; 0 keeps the original bytes")
    args = parser.parse_args(argv)
    try:
        items = _load_manifest(args.directory)
        page = build_html(items, args.directory, args.max_px)
    except ReviewBuildError as exc:
        print(f"build_nsfw_review: {exc}", file=sys.stderr)
        return 1
    out = args.out or os.path.join(args.directory, "REVIEW.html")
    with open(out, "w", encoding="utf-8") as fh:
        fh.write(page)
    print(f"wrote {out} ({os.path.getsize(out) / 1e6:.1f} MB, {len(items)} items)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
