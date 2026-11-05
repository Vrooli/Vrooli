import { readFile, readdir } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const ui = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const experience = resolve(ui, "../experience");
const parse = async path => ts.createSourceFile(path, await readFile(path, "utf8"), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
const source = await parse(join(ui, "src/routes.ts"));
const routes = new Map();
function visit(node) {
  if (ts.isVariableDeclaration(node) && node.name.getText(source) === "appRoutes") {
    let value = node.initializer;
    while (value && ts.isAsExpression(value)) value = value.expression;
    if (!value || !ts.isObjectLiteralExpression(value)) throw new Error("appRoutes must be a literal route registry");
    for (const property of value.properties) {
      if (!ts.isPropertyAssignment(property) || !ts.isStringLiteral(property.initializer)) throw new Error("every appRoutes entry must have a literal URL");
      routes.set(property.name.getText(source), property.initializer.text);
    }
  }
  ts.forEachChild(node, visit);
}
visit(source);
if (!routes.size) throw new Error("appRoutes registry is empty");
const app = await parse(join(ui, "src/App.tsx"));
const registered = new Set();
function visitApp(node) {
  if (ts.isPropertyAccessExpression(node) && node.expression.getText(app) === "appRoutes") registered.add(node.name.text);
  if (ts.isJsxAttribute(node) && node.name.text === "path" && node.initializer && ts.isStringLiteral(node.initializer) && node.initializer.text !== "*") throw new Error("App route paths must come from appRoutes");
  ts.forEachChild(node, visitApp);
}
visitApp(app);
const index = JSON.parse(await readFile(join(experience, "index.json"), "utf8"));
const pages = [];
const failures = [];
for (const name of await readdir(join(experience, "pages"))) {
  if (!name.endsWith(".json")) continue;
  const doc = JSON.parse(await readFile(join(experience, "pages", name), "utf8"));
  if (doc.kind !== "experience-page") continue;
  const id = doc.page?.id;
  const entries = index.pages.filter(entry => entry.id === id && entry.path === `pages/${name}`);
  if (entries.length !== 1) failures.push(`${name}: must appear once in the experience index`);
  if (!doc.regions?.length) failures.push(`${name}: needs a meaningful region`);
  for (const region of doc.regions ?? []) {
    const binding = doc.bindings?.regions?.[region.id];
    if (!binding?.testid && !binding?.selector) failures.push(`${name}/${region.id}: missing source binding`);
  }
  pages.push(doc);
}
for (const [name, route] of routes) {
  if (!registered.has(name)) failures.push(`${name}: declared route is not referenced by App`);
  const matches = pages.filter(doc => doc.page.routes?.includes(route));
  if (matches.length !== 1) failures.push(`${route}: expected one page contract, found ${matches.length}`);
}
for (const entry of index.pages) if (!pages.some(doc => doc.page.id === entry.id)) failures.push(`${entry.id}: index references a missing page`);
// Every served URL pattern also needs an executable source-adjacent page story.
const storyRoutes = [];
async function visitStories(dir) {
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) await visitStories(path);
    else if (entry.name.endsWith(".story.json")) {
      const contract = JSON.parse(await readFile(path, "utf8"));
      if (contract.kind !== "page") continue;
      await readFile(path.replace(/\.story\.json$/, ".tsx"), "utf8");
      for (const story of contract.stories ?? []) storyRoutes.push(story.route ?? contract.route);
    }
  }
}
await visitStories(join(ui, "src"));
for (const route of routes.values()) {
  const parts = route.split("/");
  const matches = concrete => {
    const actual = concrete.split("?")[0].split("/");
    return parts.length === actual.length && parts.every((part, index) => part.startsWith(":") ? Boolean(actual[index]) : part === actual[index]);
  };
  if (!storyRoutes.some(matches)) failures.push(`${route}: missing executable page story`);
}
if (failures.length) throw new Error(failures.join("\n"));
console.log(`Every served route has one registered page contract (${routes.size} routes; aliases share page identity) and an executable page story.`);
