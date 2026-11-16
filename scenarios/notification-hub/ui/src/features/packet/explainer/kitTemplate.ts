import type { JsonObject, JsonValue } from "@bufbuild/protobuf";

const templates = import.meta.glob<string>("../../../../../skills/notification-hub/kit/templates/*.html", { eager: true, query: "?raw", import: "default" });
const schemas = import.meta.glob<string>("../../../../../skills/notification-hub/kit/schemas/*.schema.json", { eager: true, query: "?raw", import: "default" });

type Schema = Record<string, unknown>;

function typeOf(value: JsonValue): string {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  if (typeof value === "number" && Number.isInteger(value)) return "integer";
  return typeof value;
}

function matchesType(value: JsonValue, expected: string): boolean {
  return expected === "number" ? typeof value === "number" && Number.isFinite(value) : expected === "integer" ? Number.isInteger(value) : typeOf(value) === expected;
}

function sameJSON(left: JsonValue, right: JsonValue): boolean {
  return JSON.stringify(left) === JSON.stringify(right);
}

function validateSchema(schema: Schema, data: JsonValue): string[] {
  const errors: string[] = [];
  const root = schema;
  const resolveRef = (reference: string): Schema | null => {
    if (!reference.startsWith("#/")) return null;
    let current: unknown = root;
    for (const key of reference.slice(2).split("/")) current = current && typeof current === "object" ? (current as Record<string, unknown>)[key] : undefined;
    return current && typeof current === "object" ? current as Schema : null;
  };
  const check = (rule: Schema, value: JsonValue, path: string, out: string[]): void => {
    if (typeof rule.$ref === "string") {
      const target = resolveRef(rule.$ref);
      if (target) check(target, value, path, out);
      else out.push(`${path}: schema reference is unavailable`);
      return;
    }
    const expected = Array.isArray(rule.type) ? rule.type : typeof rule.type === "string" ? [rule.type] : [];
    if (expected.length && !expected.some((kind) => typeof kind === "string" && matchesType(value, kind))) {
      out.push(`${path}: expected ${expected.join(" or ")}, got ${typeOf(value)}`);
      return;
    }
    if (Array.isArray(rule.enum) && !rule.enum.some((item) => sameJSON(item as JsonValue, value))) out.push(`${path}: value is not in the allowed set`);
    if (Object.prototype.hasOwnProperty.call(rule, "const") && !sameJSON(rule.const as JsonValue, value)) out.push(`${path}: value does not match the required constant`);
    if (typeof value === "number") {
      if (typeof rule.minimum === "number" && value < rule.minimum) out.push(`${path}: must be at least ${rule.minimum}`);
      if (typeof rule.maximum === "number" && value > rule.maximum) out.push(`${path}: must be at most ${rule.maximum}`);
    }
    if (typeof value === "string") {
      if (typeof rule.minLength === "number" && value.length < rule.minLength) out.push(`${path}: must contain at least ${rule.minLength} characters`);
      if (typeof rule.pattern === "string" && !new RegExp(rule.pattern).test(value)) out.push(`${path}: format is invalid`);
      if (rule.format === "date-time" && (!/^\d{4}-\d\d-\d\dT\d\d:\d\d/.test(value) || Number.isNaN(Date.parse(value)))) out.push(`${path}: must be an ISO date and time`);
    }
    if (Array.isArray(value)) {
      if (typeof rule.minItems === "number" && value.length < rule.minItems) out.push(`${path}: needs at least ${rule.minItems} items`);
      if (typeof rule.maxItems === "number" && value.length > rule.maxItems) out.push(`${path}: allows at most ${rule.maxItems} items`);
      if (rule.items && typeof rule.items === "object") value.forEach((item, index) => check(rule.items as Schema, item, `${path}[${index}]`, out));
    }
    if (value !== null && typeof value === "object" && !Array.isArray(value)) {
      const object = value as JsonObject;
      const properties = rule.properties && typeof rule.properties === "object" ? rule.properties as Record<string, Schema> : {};
      if (Array.isArray(rule.required)) for (const key of rule.required) if (typeof key === "string" && !(key in object)) out.push(`${path}: missing field "${key}"`);
      for (const [key, item] of Object.entries(object)) {
        if (properties[key]) check(properties[key], item, `${path}.${key}`, out);
        else if (rule.additionalProperties === false) out.push(`${path}: unknown field "${key}"`);
      }
    }
    for (const variant of ["anyOf", "oneOf"] as const) {
      const choices = rule[variant];
      if (!Array.isArray(choices)) continue;
      const results = choices.map((choice) => {
        const found: string[] = [];
        if (choice && typeof choice === "object") check(choice as Schema, value, path, found);
        return found;
      });
      const passing = results.filter((result) => result.length === 0).length;
      if ((variant === "anyOf" && passing === 0) || (variant === "oneOf" && passing !== 1)) out.push(`${path}: does not match an allowed shape`);
    }
  };
  check(schema, data, "$", errors);
  return errors;
}

function escapeScriptJSON(data: JsonObject): string {
  return JSON.stringify(data)
    .replace(/</g, "\\u003c").replace(/>/g, "\\u003e").replace(/&/g, "\\u0026")
    .replace(/\u2028/g, "\\u2028").replace(/\u2029/g, "\\u2029");
}

/** Render one published Hub kit template with its stored data object. */
export function renderKitTemplate(name: string, data: JsonObject | undefined): { html: string; error: string } {
  if (!data) return { html: "", error: "This template has no data object." };
  const path = Object.keys(templates).find((candidate) => candidate.endsWith(`/templates/${name}.html`));
  const schemaPath = Object.keys(schemas).find((candidate) => candidate.endsWith(`/schemas/${name}.schema.json`));
  const template = path ? templates[path] : undefined;
  const schemaText = schemaPath ? schemas[schemaPath] : undefined;
  if (!template || !schemaText) return { html: "", error: `The ${name} template is not available in this Hub build.` };
  let schema: Schema;
  try { schema = JSON.parse(schemaText) as Schema; } catch { return { html: "", error: `The ${name} template schema could not be read.` }; }
  const errors = validateSchema(schema, data);
  if (errors.length) return { html: "", error: `The ${name} template data is invalid: ${errors.slice(0, 3).join("; ")}` };
  const slots = [...template.matchAll(/<script type="application\/json" id="hub-data">[\s\S]*?<\/script>/g)];
  if (slots.length !== 1) return { html: "", error: `The ${name} template must contain exactly one data slot.` };
  const html = template.replace(/(<script type="application\/json" id="hub-data">)[\s\S]*?(<\/script>)/, (_match, open: string, close: string) => `${open}${escapeScriptJSON(data)}${close}`);
  if (new TextEncoder().encode(html).length > 2 * 1024 * 1024) return { html: "", error: `The rendered ${name} page exceeds the Hub page size limit.` };
  return { html, error: "" };
}
