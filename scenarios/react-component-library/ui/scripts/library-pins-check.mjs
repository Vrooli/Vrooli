import { readFile, readdir } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const ui = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const library = resolve(ui, "../library");
const prefix = "@vrooli/react-component-library/";
const latest = new Map();
for (const kind of await readdir(library, { withFileTypes: true })) {
  if (!kind.isDirectory()) continue;
  for (const asset of await readdir(join(library, kind.name), { withFileTypes: true })) {
    if (!asset.isDirectory()) continue;
    try {
      const manifest = JSON.parse(await readFile(join(library, kind.name, asset.name, "component.json"), "utf8"));
      latest.set(asset.name, manifest.latest);
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
    }
  }
}
const failures = [];
let imports = 0;
async function visitDirectory(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) { await visitDirectory(path); continue; }
    if (!entry.isFile() || !/\.[cm]?[jt]sx?$/.test(entry.name)) continue;
    const source = ts.createSourceFile(path, await readFile(path, "utf8"), ts.ScriptTarget.Latest, true);
    function check(node) {
      let specifier;
      if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) specifier = node.moduleSpecifier;
      else if (ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument)) specifier = node.argument.literal;
      else if (ts.isCallExpression(node) && (node.expression.kind === ts.SyntaxKind.ImportKeyword || node.expression.getText(source) === "require")) specifier = node.arguments[0];
      if (specifier && ts.isStringLiteralLike(specifier) && specifier.text.startsWith(prefix)) {
        imports++;
        const [asset, pin] = specifier.text.slice(prefix.length).split("/");
        const current = latest.get(asset);
        const major = /^\d+(?:\.\d+\.\d+)?$/.test(pin || "") ? Number(pin.split(".")[0]) : NaN;
        if (!current || !Number.isFinite(major) || major < Number(current.split(".")[0])) {
          const line = source.getLineAndCharacterOfPosition(specifier.getStart(source)).line + 1;
          failures.push(`${path.slice(ui.length + 1)}:${line}: ${specifier.text}; latest ${current || "unknown"}`);
        }
      }
      ts.forEachChild(node, check);
    }
    check(source);
  }
}
await visitDirectory(join(ui, "src"));
if (failures.length) {
  console.error(`Library pins need updating:\n${failures.join("\n")}`);
  process.exitCode = 1;
} else console.log(`Checked ${imports} library imports: no superseded major pins.`);
