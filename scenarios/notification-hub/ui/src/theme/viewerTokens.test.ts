import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

const read = (name: string) => readFileSync(new URL(name, import.meta.url), "utf8");
const DESIGN = read("../design-tokens.css");
const PALETTE = read("./tokens.css");
const VIEWER = read("./viewerTokens.css");

/** The declarations of the rule that opens with `header`, as `--name -> value`. */
function declarations(css: string, header: string): Map<string, string> {
  const open = css.indexOf(`${header} {`);
  if (open < 0) throw new Error(`no rule "${header}"`);
  const body = css.slice(css.indexOf("{", open) + 1, css.indexOf("}", open));
  return new Map([...body.matchAll(/(--[\w-]+):\s*([^;]+);/g)].map((match) => [match[1] ?? "", (match[2] ?? "").trim()]));
}

/** "#466dd5" as the "70 109 213" channel triplet the viewer composes alpha from. */
const channels = (hex: string): string => {
  const [, r = "", g = "", b = ""] = /^#(..)(..)(..)$/.exec(hex) ?? [];
  return [r, g, b].map((part) => parseInt(part, 16)).join(" ");
};

const LIGHT_HEADERS = [':root[data-resolved-theme="light"],\n:root[data-theme="light"]', ':root:not([data-resolved-theme="dark"]):not([data-theme="dark"])'];
const DARK_HEADER = ':root[data-resolved-theme="dark"],\n.dark';

const THEMES = [
  { name: "light", viewer: LIGHT_HEADERS, palette: LIGHT_HEADERS[0] ?? "", design: ":root", onAccent: "--color-primary-foreground" },
  // Dark is the default field: the base :root in tokens.css and viewerTokens.css.
  { name: "dark", viewer: [":root"], palette: ":root", design: DARK_HEADER, onAccent: "--color-background" },
] as const;

describe("viewer theme tokens", () => {
  it.each(THEMES.flatMap((theme) => theme.viewer.map((header) => ({ ...theme, header }))))("map the $name palette ($header) onto the viewer's channel triplets", ({ header, palette, design, onAccent }) => {
    const colors = new Map([...declarations(DESIGN, design), ...declarations(PALETTE, palette)]);
    const channelsOf = (token: string) => channels(colors.get(token) ?? "");
    const accent = channelsOf("--color-focus");
    const expected = new Map([
      ["--fv-surface-base", channelsOf("--color-background")],
      ["--fv-surface-raised", channelsOf("--color-surface-raised")],
      ["--fv-surface-input", channelsOf("--color-field")],
      ["--fv-text-primary", channelsOf("--color-foreground")],
      ["--fv-text-secondary", channelsOf("--color-foreground")],
      ["--fv-text-muted", channelsOf("--color-muted-foreground")],
      ["--fv-text-faint", channelsOf("--color-muted-foreground")],
      ["--fv-border-default", channelsOf("--color-border")],
      ["--fv-border-hover", channelsOf("--color-muted-foreground")],
      ["--fv-accent", accent],
      ["--fv-accent-fg", channelsOf(onAccent)],
      ["--fv-accent-border", `${accent} / 0.4`],
      ["--fv-error-text", channelsOf("--color-danger")],
      ["--fv-error-detail", channelsOf("--color-danger")],
      ["--fv-success", channelsOf("--color-success")],
    ]);

    expect(declarations(VIEWER, header)).toEqual(expected);
  });
});
