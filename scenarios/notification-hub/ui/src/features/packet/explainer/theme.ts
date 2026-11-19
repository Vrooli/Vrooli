/**
 * Which colour schemes an explainer page works in.
 *
 * The Hub shows a page in its own light or dark theme, and the starter kit
 * colours the page from tokens that follow that theme. A page that paints
 * with literal colours (white cards, a light page background, dark text) and
 * has no dark styles of its own only works in one scheme: shown in the other,
 * the kit's text colour lands on the page's own background (the unreadable
 * page of 15 Nov 2026). Such a page is shown in the scheme it was written for,
 * whatever the Hub's theme.
 *
 * The rules mirror the packet check's (packages/api-core/pagescheme,
 * Kit.Scheme), and both read the same fixtures
 * (api/internal/hub/testdata/page-schemes), so the check judges the page in
 * the scheme the frame shows it in:
 *
 *  1. A color-scheme the page declares (meta name="color-scheme", or the
 *     color-scheme property on :root/html/body) decides.
 *  2. A page with dark styles of its own (a prefers-color-scheme media block, a
 *     [data-theme="dark"] selector, or a script that follows hubthemechange)
 *     follows the Hub's theme.
 *  3. A literal background or text colour on :root, html or body decides by
 *     its lightness.
 *  4. Otherwise unpaired literal colours vote: a background with no text colour
 *     in the same rule votes by its lightness, a text colour with no
 *     background votes the other way (dark text means a light page).
 *  5. A page with neither follows the Hub's theme.
 */
export type PageScheme = "both" | "light" | "dark";

interface Rule {
  selector: string;
  decls: Map<string, string>;
  scheme: "" | "light" | "dark";
  inline: boolean;
}

interface Styles {
  rules: Rule[];
  declared: PageScheme | "";
  darkStyles: boolean;
}

const DARK_SELECTOR = /\[data-theme\s*=\s*["']?dark["']?\]|\.dark\b|\.theme-dark\b/i;
const LIGHT_SELECTOR = /\[data-theme\s*=\s*["']?light["']?\]/i;
const NOT_PATTERN = /:not\([^()]*\)/g;
const ROOT_SELECTOR = /^(?:html|:root|body)$/i;
const VAR_PATTERN = /var\(\s*(--[A-Za-z0-9_-]+)\s*(?:,\s*([^()]*(?:\([^()]*\))?[^()]*))?\)/g;

const NAMED: Record<string, string> = {
  white: "#ffffff", black: "#000000", snow: "#fffafa", ivory: "#fffff0", whitesmoke: "#f5f5f5",
  ghostwhite: "#f8f8ff", aliceblue: "#f0f8ff", azure: "#f0ffff", mintcream: "#f5fffa", honeydew: "#f0fff0",
  seashell: "#fff5ee", floralwhite: "#fffaf0", oldlace: "#fdf5e6", linen: "#faf0e6", beige: "#f5f5dc",
  lavender: "#e6e6fa", lightyellow: "#ffffe0", cornsilk: "#fff8dc", papayawhip: "#ffefd5", gainsboro: "#dcdcdc",
  lightgray: "#d3d3d3", lightgrey: "#d3d3d3", silver: "#c0c0c0", darkgray: "#a9a9a9", darkgrey: "#a9a9a9",
  gray: "#808080", grey: "#808080", dimgray: "#696969", dimgrey: "#696969", slategray: "#708090",
  darkslategray: "#2f4f4f", red: "#ff0000", darkred: "#8b0000", maroon: "#800000", green: "#008000",
  darkgreen: "#006400", blue: "#0000ff", navy: "#000080", darkblue: "#00008b", midnightblue: "#191970",
  yellow: "#ffff00", orange: "#ffa500", purple: "#800080", teal: "#008080", olive: "#808000",
  lime: "#00ff00", aqua: "#00ffff", cyan: "#00ffff", fuchsia: "#ff00ff", magenta: "#ff00ff",
  lightblue: "#add8e6", lightgreen: "#90ee90", pink: "#ffc0cb", lightpink: "#ffb6c1", wheat: "#f5deb3",
};

interface Color {
  r: number;
  g: number;
  b: number;
  alpha: number;
}

function schemeWords(value: string): PageScheme | "" {
  const lower = value.toLowerCase();
  const light = lower.includes("light");
  const dark = lower.includes("dark");
  if (light && dark) return "both";
  if (light) return "light";
  if (dark) return "dark";
  return "";
}

/** Split on `sep` outside parentheses, brackets and quotes. */
function splitTop(text: string, sep: string): string[] {
  const parts: string[] = [];
  let depth = 0;
  let start = 0;
  let quote = "";
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (quote) {
      if (c === quote) quote = "";
    } else if (c === '"' || c === "'") quote = c;
    else if (c === "(" || c === "[") depth++;
    else if (c === ")" || c === "]") depth--;
    else if (c === sep && depth === 0) {
      parts.push(text.slice(start, i));
      start = i + 1;
    }
  }
  parts.push(text.slice(start));
  return parts;
}

function blockBody(css: string, open: number): [string, number] {
  let depth = 0;
  for (let j = open; j < css.length; j++) {
    if (css[j] === "{") depth++;
    else if (css[j] === "}") {
      depth--;
      if (depth === 0) return [css.slice(open + 1, j), j + 1];
    }
  }
  return [css.slice(open + 1), css.length];
}

function parenBody(text: string, open: number): string {
  let depth = 0;
  for (let j = open; j < text.length; j++) {
    if (text[j] === "(") depth++;
    else if (text[j] === ")") {
      depth--;
      if (depth === 0) return text.slice(open + 1, j);
    }
  }
  return text.slice(open + 1);
}

function parseDeclarations(body: string): Map<string, string> {
  const decls = new Map<string, string>();
  for (const decl of splitTop(body, ";")) {
    const colon = decl.indexOf(":");
    if (colon < 0) continue;
    let name = decl.slice(0, colon).trim();
    if (!name.startsWith("--")) name = name.toLowerCase();
    const value = decl.slice(colon + 1).trim().replace(/!important$/, "").trim();
    if (name && value) decls.set(name, value);
  }
  return decls;
}

function selectorScheme(selector: string): Rule["scheme"] {
  const bare = selector.replace(NOT_PATTERN, "");
  if (DARK_SELECTOR.test(bare)) return "dark";
  if (LIGHT_SELECTOR.test(bare)) return "light";
  return "";
}

function parseCSS(css: string, scheme: Rule["scheme"]): Rule[] {
  const rules: Rule[] = [];
  let i = 0;
  while (i < css.length) {
    const open = css.indexOf("{", i);
    if (open < 0) break;
    let prelude = css.slice(i, open).trim();
    const semi = prelude.lastIndexOf(";");
    if (semi >= 0) prelude = prelude.slice(semi + 1).trim();
    const [body, end] = blockBody(css, open);
    i = end;
    const lower = prelude.toLowerCase();
    if (lower.startsWith("@media")) {
      let inner = scheme;
      if (lower.includes("prefers-color-scheme")) inner = lower.includes("dark") ? "dark" : "light";
      rules.push(...parseCSS(body, inner));
    } else if (lower.startsWith("@supports") || lower.startsWith("@layer") || lower.startsWith("@container")) {
      rules.push(...parseCSS(body, scheme));
    } else if (!lower.startsWith("@")) {
      for (const part of splitTop(prelude, ",")) {
        const selector = part.split(/\s+/).filter(Boolean).join(" ");
        if (!selector) continue;
        rules.push({ selector, decls: parseDeclarations(body), scheme: scheme || selectorScheme(selector), inline: false });
      }
    }
  }
  return rules;
}

function isRootRule(selector: string): boolean {
  const bare = selector.replace(NOT_PATTERN, "").replace(DARK_SELECTOR, "").replace(LIGHT_SELECTOR, "").trim();
  return ROOT_SELECTOR.test(bare);
}

function stripComments(css: string): string {
  return css.replace(/\/\*[\s\S]*?\*\//g, "");
}

function readStyles(doc: Document): Styles {
  const styles: Styles = { rules: [], declared: "", darkStyles: false };
  for (const node of doc.querySelectorAll("style")) {
    if (node.hasAttribute("data-hub")) continue;
    styles.rules.push(...parseCSS(stripComments(node.textContent), ""));
  }
  // Each style attribute is its own rule: two elements' attributes never pair up.
  let inline = 0;
  for (const node of doc.querySelectorAll("[style]")) {
    const style = node.getAttribute("style") ?? "";
    if (style.trim()) styles.rules.push({ selector: `${node.tagName.toLowerCase()}[style="#${inline++}"]`, decls: parseDeclarations(style), scheme: "", inline: true });
  }
  const meta = doc.querySelector('meta[name="color-scheme" i]');
  if (meta) styles.declared = schemeWords(meta.getAttribute("content") ?? "");
  for (const rule of styles.rules) {
    if (rule.scheme === "dark") styles.darkStyles = true;
    const value = rule.decls.get("color-scheme");
    if (value !== undefined && rule.scheme === "" && styles.declared === "" && isRootRule(rule.selector)) styles.declared = schemeWords(value);
  }
  for (const node of doc.querySelectorAll("script")) {
    if (node.hasAttribute("data-hub")) continue;
    if (/hubthemechange|prefers-color-scheme/.test(node.textContent)) styles.darkStyles = true;
  }
  return styles;
}

// --- colours ---------------------------------------------------------------

function parseHex(text: string): Color | null {
  let hex = text.replace(/^#/, "");
  if (hex.length === 3 || hex.length === 4) hex = hex.replace(/./g, "$&$&");
  if (hex.length !== 6 && hex.length !== 8) return null;
  const value = Number.parseInt(hex, 16);
  if (Number.isNaN(value)) return null;
  if (hex.length === 8) return { r: (value >>> 24) & 255, g: (value >>> 16) & 255, b: (value >>> 8) & 255, alpha: (value & 255) / 255 };
  return { r: (value >> 16) & 255, g: (value >> 8) & 255, b: value & 255, alpha: 1 };
}

function hslToRgb(h: number, s: number, l: number): [number, number, number] {
  h = (((h % 360) + 360) % 360) / 360;
  if (s === 0) return [l * 255, l * 255, l * 255];
  const q = l < 0.5 ? l * (1 + s) : l + s - l * s;
  const p = 2 * l - q;
  const hue = (t: number) => {
    t = (t + 1) % 1;
    if (t < 1 / 6) return p + (q - p) * 6 * t;
    if (t < 0.5) return q;
    if (t < 2 / 3) return p + (q - p) * (2 / 3 - t) * 6;
    return p;
  };
  return [hue(h + 1 / 3) * 255, hue(h) * 255, hue(h - 1 / 3) * 255];
}

function parseFuncColor(name: string, args: string): Color | null {
  const [first, second, third, fourth] = args.split(/[\s,/]+/).filter(Boolean);
  if (first === undefined || second === undefined || third === undefined) return null;
  const num = (field: string, scale: number) => (field.endsWith("%") ? (Number.parseFloat(field) / 100) * scale : Number.parseFloat(field));
  const alpha = fourth !== undefined ? num(fourth, 1) : 1;
  if (name.startsWith("rgb")) return { r: num(first, 255), g: num(second, 255), b: num(third, 255), alpha };
  const s = second.endsWith("%") ? num(second, 1) : Number.parseFloat(second) / 100;
  const l = third.endsWith("%") ? num(third, 1) : Number.parseFloat(third) / 100;
  const [r, g, b] = hslToRgb(Number.parseFloat(first), s, l);
  return { r, g, b, alpha };
}

function mixColor(body: string): Color | null {
  const parts = splitTop(body, ",");
  if (parts.length !== 3) return null;
  const read = (part: string): [Color | null, number] => {
    let text = part.trim();
    let weight = -1;
    const fields = text.split(/\s+/);
    const last = fields[fields.length - 1] ?? "";
    if (fields.length > 1 && last.endsWith("%")) {
      weight = Number.parseFloat(last) / 100;
      text = text.slice(0, text.length - last.length).trim();
    }
    return [firstColor(text), weight];
  };
  const [a, readA] = read(parts[1] ?? "");
  const [b, readB] = read(parts[2] ?? "");
  if (!a || !b) return null;
  let wa = readA;
  let wb = readB;
  if (wa < 0 && wb < 0) [wa, wb] = [0.5, 0.5];
  else if (wa < 0) wa = 1 - wb;
  else if (wb < 0) wb = 1 - wa;
  const total = wa + wb;
  if (total > 0) [wa, wb] = [wa / total, wb / total];
  const alpha = a.alpha * wa + b.alpha * wb;
  if (alpha <= 0) return { r: 0, g: 0, b: 0, alpha: 0 };
  const mix = (x: number, y: number) => (x * a.alpha * wa + y * b.alpha * wb) / alpha;
  return { r: mix(a.r, b.r), g: mix(a.g, b.g), b: mix(a.b, b.b), alpha };
}

/** The first literal colour in a value (a shorthand or a gradient's first stop counts). */
function firstColor(value: string): Color | null {
  const mixAt = value.toLowerCase().indexOf("color-mix(");
  if (mixAt >= 0) {
    const before = firstColor(value.slice(0, mixAt));
    if (before) return before;
    return mixColor(parenBody(value, mixAt + "color-mix".length));
  }
  let best: Color | null = null;
  let at = value.length + 1;
  const hex = /#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})\b/i.exec(value);
  if (hex && hex.index < at) {
    const color = parseHex(hex[0]);
    if (color) [best, at] = [color, hex.index];
  }
  const fn = /\b(rgba?|hsla?)\(([^()]*)\)/i.exec(value);
  if (fn && fn.index < at) {
    const color = parseFuncColor((fn[1] ?? "").toLowerCase(), fn[2] ?? "");
    if (color) [best, at] = [color, fn.index];
  }
  const blank = (s: string) => " ".repeat(s.length);
  const stripped = value.replace(/\b(rgba?|hsla?)\(([^()]*)\)/gi, blank).replace(/url\([^)]*\)/g, blank);
  for (const match of stripped.matchAll(/\b[a-z]+\b/gi)) {
    if (match.index >= at) break;
    const word = match[0].toLowerCase();
    if (word === "transparent") return { r: 0, g: 0, b: 0, alpha: 0 };
    if (NAMED[word]) return parseHex(NAMED[word]);
  }
  return best;
}

function isLight(color: Color): boolean {
  const channel = (v: number) => {
    v /= 255;
    return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(color.r) + 0.7152 * channel(color.g) + 0.0722 * channel(color.b) > 0.18;
}

function themeVars(kit: Rule[], styles: Styles, scheme: "light" | "dark"): Map<string, string> {
  const vars = new Map<string, string>();
  for (const rules of [kit, styles.rules]) {
    for (const pass of ["", scheme]) {
      for (const rule of rules) {
        if (rule.scheme !== pass || !isRootRule(rule.selector)) continue;
        for (const [name, value] of rule.decls) if (name.startsWith("--")) vars.set(name, value);
      }
    }
  }
  return vars;
}

function resolve(value: string, vars: Map<string, string>): string | null {
  for (let depth = 0; depth < 4 && value.includes("var("); depth++) {
    const unresolved = [...value.matchAll(VAR_PATTERN)].some((match) => {
      const name = match[1] ?? "";
      return vars.get(name) === undefined && !match[2];
    });
    if (unresolved) return null;
    value = value.replace(VAR_PATTERN, (_match, name: string, fallback?: string) => {
      const found = vars.get(name);
      if (found !== undefined) return found;
      return fallback ?? "";
    });
  }
  return value.includes("var(") ? null : value;
}

function usesThemeVar(value: string, light: Map<string, string>, dark: Map<string, string>): boolean {
  for (const match of value.matchAll(VAR_PATTERN)) {
    const name = match[1] ?? "";
    if (light.get(name) !== dark.get(name)) return true;
  }
  return false;
}

function backgroundOf(decls: Map<string, string>): string | undefined {
  return decls.get("background-color") ?? decls.get("background");
}

function mergeRules(rules: Rule[]): Rule[] {
  const index = new Map<string, Rule>();
  for (const rule of rules) {
    if (rule.scheme !== "") continue;
    let merged = index.get(rule.selector);
    if (!merged) {
      merged = { selector: rule.selector, decls: new Map(), scheme: "", inline: rule.inline };
      index.set(rule.selector, merged);
    }
    for (const [name, value] of rule.decls) merged.decls.set(name, value);
  }
  return [...index.values()];
}

const kitCache = new Map<string, Rule[]>();

function kitRules(kitCss: string): Rule[] {
  let rules = kitCache.get(kitCss);
  if (!rules) {
    rules = parseCSS(stripComments(kitCss), "");
    kitCache.set(kitCss, rules);
  }
  return rules;
}

/**
 * The schemes a page works in: "both" (it follows the Hub's theme), or the
 * one scheme its own colours were written for. `kitCss` is the starter kit as
 * the Hub injects it; its tokens tell a theme colour from a literal one.
 */
export function pageColorScheme(html: string, kitCss: string): PageScheme {
  const doc = new DOMParser().parseFromString(html, "text/html");
  const styles = readStyles(doc);
  if (styles.declared) return styles.declared;
  if (styles.darkStyles) return "both";
  const kit = kitRules(kitCss);
  const light = themeVars(kit, styles, "light");
  const dark = themeVars(kit, styles, "dark");
  const literal = (value: string): Color | null => {
    if (usesThemeVar(value, light, dark)) return null;
    const resolved = resolve(value, light);
    if (resolved === null) return null;
    const color = firstColor(resolved);
    return color && color.alpha >= 0.5 ? color : null;
  };
  const roots = styles.rules.filter((rule) => rule.scheme === "" && isRootRule(rule.selector));
  for (const rule of roots) {
    const bg = backgroundOf(rule.decls);
    const color = bg === undefined ? null : literal(bg);
    if (color) return isLight(color) ? "light" : "dark";
  }
  for (const rule of roots) {
    const fg = rule.decls.get("color");
    const color = fg === undefined ? null : literal(fg);
    if (color) return isLight(color) ? "dark" : "light";
  }
  let votes = 0;
  for (const rule of mergeRules(styles.rules)) {
    const bg = backgroundOf(rule.decls);
    const fg = rule.decls.get("color");
    if (bg !== undefined && fg === undefined) {
      const color = literal(bg);
      if (color) votes += isLight(color) ? 1 : -1;
    } else if (fg !== undefined && bg === undefined) {
      const color = literal(fg);
      if (color) votes += isLight(color) ? -1 : 1;
    }
  }
  if (votes > 0) return "light";
  if (votes < 0) return "dark";
  return "both";
}

/** The theme a page is shown in, given the Hub's theme. */
export function pageTheme(scheme: PageScheme, hubTheme: "light" | "dark"): "light" | "dark" {
  return scheme === "both" ? hubTheme : scheme;
}
