// Shared helpers for the Notification Hub chart kit tools: copy the kit into
// the templates, render a template with data, and validate data against a
// template's schema. Node 20+, no dependencies.
import { readFileSync, readdirSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const KIT_DIR = resolve(dirname(fileURLToPath(import.meta.url)), "..");
export const TEMPLATES_DIR = join(KIT_DIR, "templates");
export const SCHEMAS_DIR = join(KIT_DIR, "schemas");
export const SAMPLES_DIR = join(KIT_DIR, "samples");
export const MAX_PAGE_BYTES = 2 * 1024 * 1024;

const STYLE_SLOT = /(<style data-kit="hub-charts">)([\s\S]*?)(<\/style>)/;
const SCRIPT_SLOT = /(<script data-kit="hub-charts">)([\s\S]*?)(<\/script>)/;
const DATA_SLOT = /(<script type="application\/json" id="hub-data">)([\s\S]*?)(<\/script>)/g;

export function templateNames() {
  return readdirSync(TEMPLATES_DIR).filter((f) => f.endsWith(".html")).map((f) => f.slice(0, -5)).sort();
}

export function kitSources() {
  const css = readFileSync(join(KIT_DIR, "hub-charts.css"), "utf8");
  const js = readFileSync(join(KIT_DIR, "hub-charts.js"), "utf8");
  if (/<\/style/i.test(css)) throw new Error("hub-charts.css must not contain </style");
  if (/<\/script/i.test(js)) throw new Error("hub-charts.js must not contain </script");
  return { css, js };
}

/** The template with the current kit copied into its two kit slots. */
export function syncTemplate(html, { css, js }) {
  if (!STYLE_SLOT.test(html) || !SCRIPT_SLOT.test(html)) throw new Error('template lacks <style data-kit="hub-charts"> or <script data-kit="hub-charts">');
  return html.replace(STYLE_SLOT, (_, a, _b, c) => a + "\n" + css + c).replace(SCRIPT_SLOT, (_, a, _b, c) => a + "\n" + js + c);
}

/**
 * JSON for the hub-data element, escaped the way Go's json.Marshal escapes
 * by default, so the text can never close the script element.
 */
export function embedJSON(data) {
  return JSON.stringify(data)
    .replace(/</g, "\\u003c").replace(/>/g, "\\u003e").replace(/&/g, "\\u0026")
    .replace(/\u2028/g, "\\u2028").replace(/\u2029/g, "\\u2029");
}

/** The render contract: put the data into the template's one hub-data element. */
export function render(templateHtml, data) {
  const slots = templateHtml.match(DATA_SLOT) || [];
  if (slots.length !== 1) throw new Error(`template must hold exactly one hub-data element, found ${slots.length}`);
  const page = templateHtml.replace(DATA_SLOT, (_, a, _b, c) => a + embedJSON(data) + c);
  if (Buffer.byteLength(page) > MAX_PAGE_BYTES) throw new Error(`rendered page is ${Buffer.byteLength(page)} bytes, over the Hub's 2 MiB page limit`);
  return page;
}

export function loadSchema(name) { return JSON.parse(readFileSync(join(SCHEMAS_DIR, `${name}.schema.json`), "utf8")); }
export function loadSample(name) { return JSON.parse(readFileSync(join(SAMPLES_DIR, `${name}.json`), "utf8")); }
export function loadTemplate(name) { return readFileSync(join(TEMPLATES_DIR, `${name}.html`), "utf8"); }

/*
 * A JSON Schema validator for the subset the kit's schemas use: type, enum,
 * const, properties, required, additionalProperties, items, minItems,
 * maxItems, minimum, maximum, minLength, pattern, format date-time, $ref to
 * #/$defs, dependentRequired, anyOf and oneOf. Returns a list of "path: problem" strings.
 */
export function validate(schema, value) {
  const errors = [];
  const root = schema;
  const typeOf = (v) => (v === null ? "null" : Array.isArray(v) ? "array" : Number.isInteger(v) ? "integer" : typeof v);
  const isType = (v, t) => t === "number" ? typeof v === "number" && Number.isFinite(v) : t === "integer" ? Number.isInteger(v) : typeOf(v) === t;
  function resolveRef(ref, base) {
    let source = base || root;
    if (ref.startsWith("./")) {
      const [file, fragment] = ref.split("#");
      source = JSON.parse(readFileSync(join(SCHEMAS_DIR, file), "utf8"));
      ref = fragment || "";
    }
    if (ref && !ref.startsWith("#/")) throw new Error(`unsupported $ref ${ref}`);
    return { schema: ref ? ref.slice(2).split("/").reduce((node, key) => node[key], source) : source, base: source };
  }
  function check(s, v, path, out, base = root) {
    if (s.$ref) { const target = resolveRef(s.$ref, base); return check(target.schema, v, path, out, target.base); }
    if (s.type) {
      const types = Array.isArray(s.type) ? s.type : [s.type];
      if (!types.some((t) => isType(v, t))) { out.push(`${path}: expected ${types.join(" or ")}, got ${typeOf(v)}`); return; }
    }
    if (s.enum && !s.enum.some((e) => JSON.stringify(e) === JSON.stringify(v))) out.push(`${path}: must be one of ${s.enum.map((e) => JSON.stringify(e)).join(", ")}`);
    if ("const" in s && JSON.stringify(s.const) !== JSON.stringify(v)) out.push(`${path}: must be ${JSON.stringify(s.const)}`);
    if (typeof v === "number") {
      if (typeof s.minimum === "number" && v < s.minimum) out.push(`${path}: must be >= ${s.minimum}`);
      if (typeof s.maximum === "number" && v > s.maximum) out.push(`${path}: must be <= ${s.maximum}`);
    }
    if (typeof v === "string") {
      if (typeof s.minLength === "number" && v.length < s.minLength) out.push(`${path}: must have at least ${s.minLength} characters`);
      if (typeof s.maxLength === "number" && v.length > s.maxLength) out.push(`${path}: must have at most ${s.maxLength} characters`);
      if (s.pattern && !new RegExp(s.pattern).test(v)) out.push(`${path}: must match ${s.pattern}`);
      if (s.format === "date-time" && (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/.test(v) || Number.isNaN(Date.parse(v)))) out.push(`${path}: must be an ISO 8601 date-time with zone, like 2026-11-16T08:00:00Z`);
    }
    if (Array.isArray(v)) {
      if (typeof s.minItems === "number" && v.length < s.minItems) out.push(`${path}: needs at least ${s.minItems} items`);
      if (typeof s.maxItems === "number" && v.length > s.maxItems) out.push(`${path}: allows at most ${s.maxItems} items`);
      if (s.items) v.forEach((item, i) => check(s.items, item, `${path}[${i}]`, out, base));
    }
    if (typeOf(v) === "object") {
      for (const key of s.required || []) if (!(key in v)) out.push(`${path}: missing required field "${key}"`);
      for (const [key, required] of Object.entries(s.dependentRequired || {})) {
        if (key in v) for (const dependency of required) if (!(dependency in v)) out.push(`${path}: field "${key}" requires "${dependency}"`);
      }
      const props = s.properties || {};
      for (const [key, item] of Object.entries(v)) {
        if (props[key]) check(props[key], item, `${path}.${key}`, out, base);
        else if (s.additionalProperties === false) out.push(`${path}: unknown field "${key}"`);
        else if (s.additionalProperties && typeof s.additionalProperties === "object") check(s.additionalProperties, item, `${path}.${key}`, out, base);
      }
    }
    for (const kw of ["anyOf", "oneOf"]) {
      if (!s[kw]) continue;
      const results = s[kw].map((sub) => { const o = []; check(sub, v, path, o, base); return o; });
      const passing = results.filter((r) => r.length === 0).length;
      if (kw === "anyOf" && passing === 0) out.push(...results.sort((a, b) => a.length - b.length)[0]);
      if (kw === "oneOf" && passing !== 1) out.push(passing === 0 ? results.sort((a, b) => a.length - b.length)[0].join("; ") || `${path}: matches no allowed shape` : `${path}: matches ${passing} shapes, expected exactly one`);
    }
  }
  check(schema, value, "$", errors);
  return errors;
}
