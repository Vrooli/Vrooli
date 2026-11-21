import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const sourcePath = join(root, "src/i18n/locales/en.json");
const targetPath = join(root, "src/consts/strings.generated.ts");

const header = `// AUTO-GENERATED - do not edit by hand.
//
// Source : src/i18n/locales/en.json
// Codegen: scripts/gen-strings.mjs
`;

const safeKey = /^[A-Za-z_$][\w$]*$/;

function renderValue(value, depth = 1) {
  const pad = "  ".repeat(depth);
  const close = "  ".repeat(depth - 1);
  if (typeof value === "string") {
    return JSON.stringify(value);
  }
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error("String catalog leaves must be strings or nested objects");
  }
  const entries = Object.entries(value).map(([key, child]) => {
    const renderedKey = safeKey.test(key) ? key : JSON.stringify(key);
    return `${pad}${renderedKey}: ${renderValue(child, depth + 1)},`;
  });
  return `{\n${entries.join("\n")}\n${close}}`;
}

function generateContents() {
  const catalog = JSON.parse(readFileSync(sourcePath, "utf-8"));
  return `${header}
export const strings = ${renderValue(catalog)} as const;

export type Strings = typeof strings;
`;
}

function isOutOfDate() {
  const current = existsSync(targetPath) ? readFileSync(targetPath, "utf-8") : "";
  return current !== generateContents();
}

const check = process.argv.includes("--check");
const targetRel = relative(root, targetPath);

if (check) {
  if (isOutOfDate()) {
    console.error(`${targetRel} is out of date. Run pnpm strings:gen.`);
    process.exit(1);
  }
  console.log(`${targetRel} is in sync`);
} else {
  writeFileSync(targetPath, generateContents());
  console.log(`wrote ${targetRel}`);
}
