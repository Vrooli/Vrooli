import { Fragment, type ReactNode } from "react";
import { Link } from "react-router-dom";

import { useResearchHome } from "../hooks/useResearchHome";
import { filePageHref } from "../lib/paths";

/**
 * A small, safe Markdown renderer for research files: headings, paragraphs,
 * lists, quotes, fenced code, pipe tables, rules, and inline code, emphasis
 * and links. It builds React elements, never HTML strings.
 *
 * Local because the library's markdown-renderer needs react-markdown,
 * remark-gfm, shiki and mermaid as peers, and dependency installs are refused
 * by disk admission today (docs/reference/component-library-gaps.md).
 *
 * Links: a path inside the research home opens the desk's file page; a path
 * relative to `baseDir` resolves against it; http(s) opens a new tab; any
 * other target is shown as text.
 */
export function Markdown({ content, baseDir, className }: { content: string; baseDir?: string; className?: string }) {
  const home = useResearchHome();
  const resolve = (href: string): { to: string; external: boolean } | null => {
    if (/^https?:\/\//.test(href)) return { to: href, external: true };
    let path = href.split("#")[0] ?? "";
    if (!path) return null;
    if (!path.startsWith("/") && baseDir) path = normalise(`${baseDir}/${path}`);
    const to = filePageHref(path, home);
    return to ? { to, external: false } : null;
  };
  return <div className={`rd-prose ${className ?? ""}`}>{blocks(content.replace(/\r\n/g, "\n"), resolve)}</div>;
}

type Resolve = (href: string) => { to: string; external: boolean } | null;

function normalise(path: string): string {
  const out: string[] = [];
  for (const part of path.split("/")) {
    if (part === "..") out.pop();
    else if (part !== "." && part !== "") out.push(part);
  }
  return `/${out.join("/")}`;
}

const LIST_ITEM = /^(\s*)([-*+]|\d+[.)])\s+(.*)$/;
const TABLE_RULE = /^\s*\|?\s*:?-{2,}:?\s*(\|\s*:?-{2,}:?\s*)*\|?\s*$/;

function blocks(text: string, resolve: Resolve): ReactNode[] {
  const lines = text.split("\n");
  const out: ReactNode[] = [];
  let i = 0;
  const key = () => out.length;
  while (i < lines.length) {
    const line = lines[i] ?? "";
    if (/^\s*$/.test(line)) {
      i += 1;
      continue;
    }
    const fence = /^\s*(```|~~~)/.exec(line);
    if (fence) {
      const body: string[] = [];
      i += 1;
      while (i < lines.length && !(lines[i] ?? "").trim().startsWith(fence[1] ?? "```")) body.push(lines[i++] ?? "");
      i += 1;
      out.push(<pre key={key()}><code>{body.join("\n")}</code></pre>);
      continue;
    }
    const heading = /^(#{1,6})\s+(.*)$/.exec(line);
    if (heading) {
      const level = Math.min(6, (heading[1] ?? "#").length + 1);
      const Tag = `h${level}` as "h2";
      out.push(<Tag key={key()}>{inline(heading[2] ?? "", resolve)}</Tag>);
      i += 1;
      continue;
    }
    if (/^\s*([-*_])(\s*\1){2,}\s*$/.test(line)) {
      out.push(<hr key={key()} />);
      i += 1;
      continue;
    }
    if (/^\s*>/.test(line)) {
      const body: string[] = [];
      while (i < lines.length && /^\s*>/.test(lines[i] ?? "")) body.push((lines[i++] ?? "").replace(/^\s*>\s?/, ""));
      out.push(<blockquote key={key()}>{blocks(body.join("\n"), resolve)}</blockquote>);
      continue;
    }
    if (line.includes("|") && TABLE_RULE.test(lines[i + 1] ?? "")) {
      const rows: string[][] = [];
      const cells = (row: string) => row.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map((c) => c.trim());
      const head = cells(line);
      i += 2;
      while (i < lines.length && (lines[i] ?? "").includes("|") && !/^\s*$/.test(lines[i] ?? "")) rows.push(cells(lines[i++] ?? ""));
      out.push(
        <div key={key()} className="rd-table-scroll">
          <table>
            <thead><tr>{head.map((c, n) => <th key={n}>{inline(c, resolve)}</th>)}</tr></thead>
            <tbody>{rows.map((r, n) => <tr key={n}>{r.map((c, m) => <td key={m}>{inline(c, resolve)}</td>)}</tr>)}</tbody>
          </table>
        </div>,
      );
      continue;
    }
    if (LIST_ITEM.test(line)) {
      const items: { indent: number; ordered: boolean; text: string }[] = [];
      while (i < lines.length) {
        const current = lines[i] ?? "";
        const match = LIST_ITEM.exec(current);
        if (match) {
          items.push({ indent: (match[1] ?? "").length, ordered: /\d/.test(match[2] ?? ""), text: match[3] ?? "" });
        } else if (/^\s{2,}\S/.test(current) && items.length > 0) {
          const last = items[items.length - 1];
          if (last) last.text += ` ${current.trim()}`;
        } else {
          break;
        }
        i += 1;
      }
      out.push(<Fragment key={key()}>{list(items, resolve)}</Fragment>);
      continue;
    }
    const para: string[] = [];
    while (i < lines.length && !/^\s*$/.test(lines[i] ?? "") && !/^(#{1,6}\s|\s*```|\s*>)/.test(lines[i] ?? "") && !(para.length > 0 && LIST_ITEM.test(lines[i] ?? ""))) {
      para.push(lines[i++] ?? "");
    }
    out.push(<p key={key()}>{inline(para.join(" "), resolve)}</p>);
  }
  return out;
}

function list(items: { indent: number; ordered: boolean; text: string }[], resolve: Resolve): ReactNode {
  const base = items[0]?.indent ?? 0;
  const groups: { text: string; children: typeof items }[] = [];
  for (const item of items) {
    const last = groups[groups.length - 1];
    if (item.indent > base && last) last.children.push(item);
    else groups.push({ text: item.text, children: [] });
  }
  const Tag = items[0]?.ordered ? "ol" : "ul";
  return (
    <Tag>
      {groups.map((group, n) => (
        <li key={n}>
          {inline(group.text, resolve)}
          {group.children.length > 0 ? list(group.children, resolve) : null}
        </li>
      ))}
    </Tag>
  );
}

const INLINE = /(`[^`]+`)|(\*\*[^*]+\*\*|__[^_]+__)|(\*[^*\s][^*]*\*|_[^_\s][^_]*_)|(\[[^\]]+\]\([^)\s]+\))|(https?:\/\/[^\s<>()]+)/g;

function inline(text: string, resolve: Resolve): ReactNode[] {
  const out: ReactNode[] = [];
  let last = 0;
  for (const match of text.matchAll(INLINE)) {
    const at = match.index;
    if (at > last) out.push(text.slice(last, at));
    const [token, code, strong, em, link, url] = match;
    const k = out.length;
    if (code) out.push(<code key={k}>{code.slice(1, -1)}</code>);
    else if (strong) out.push(<strong key={k}>{inline(strong.slice(2, -2), resolve)}</strong>);
    else if (em) out.push(<em key={k}>{inline(em.slice(1, -1), resolve)}</em>);
    else if (link) {
      const [, label = "", href = ""] = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(link) ?? [];
      out.push(<Anchor key={k} href={href} resolve={resolve}>{inline(label, resolve)}</Anchor>);
    } else if (url) out.push(<Anchor key={k} href={url} resolve={resolve}>{url}</Anchor>);
    else out.push(token);
    last = at + token.length;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

function Anchor({ href, resolve, children }: { href: string; resolve: Resolve; children: ReactNode }) {
  const target = resolve(href);
  if (!target) return <span className="rd-dead-link">{children}</span>;
  if (target.external) return <a href={target.to} target="_blank" rel="noreferrer noopener">{children}</a>;
  return <Link to={target.to}>{children}</Link>;
}
