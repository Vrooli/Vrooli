import { describe, expect, it } from "vitest";
import { renderMarkdown, type InlineReferenceLink } from "./render-markdown";

describe("renderMarkdown reference linkification", () => {
  const refs: InlineReferenceLink[] = [
    { token: "initiative:ship-cockpit", href: "/initiatives/ship-cockpit" },
    { token: "backlog:execute/wire-snapshot", href: "/backlog/execute/wire-snapshot" },
  ];

  it("turns a resolved typed reference into a navigable anchor", () => {
    const html = renderMarkdown("Start `initiative:ship-cockpit` now.", refs);
    expect(html).toContain('data-entity-ref="true"');
    expect(html).toContain('href="/initiatives/ship-cockpit"');
    expect(html).toContain("initiative:ship-cockpit");
  });

  it("leaves an unresolved typed reference as a plain code span", () => {
    const html = renderMarkdown("Maybe `initiative:ghost` though.", refs);
    expect(html).not.toContain('href="/initiatives/ghost"');
    expect(html).not.toContain("data-entity-ref");
    expect(html).toContain("<code");
    expect(html).toContain("initiative:ghost");
  });

  it("does not linkify a code span that is a command", () => {
    const html = renderMarkdown("Run `initiatives list` to browse.", refs);
    expect(html).not.toContain("data-entity-ref");
    expect(html).toContain("<code");
  });

  it("renders backlog references with the kind/name path", () => {
    const html = renderMarkdown("See `backlog:execute/wire-snapshot`.", refs);
    expect(html).toContain('href="/backlog/execute/wire-snapshot"');
    expect(html).toContain('data-entity-ref="true"');
  });

  it("preserves XSS-safety: script content is escaped, not executed", () => {
    const html = renderMarkdown("<script>alert(1)</script> and `initiative:ship-cockpit`", refs);
    expect(html).not.toContain("<script>");
    expect(html).toContain("&lt;script&gt;");
    // The legitimate reference still linkifies.
    expect(html).toContain('href="/initiatives/ship-cockpit"');
  });

  it("ignores references with non-relative hrefs", () => {
    const html = renderMarkdown("`initiative:ship-cockpit`", [
      { token: "initiative:ship-cockpit", href: "https://evil.example.com" },
    ]);
    expect(html).not.toContain("data-entity-ref");
    expect(html).not.toContain("evil.example.com");
    expect(html).toContain("<code");
  });
});
