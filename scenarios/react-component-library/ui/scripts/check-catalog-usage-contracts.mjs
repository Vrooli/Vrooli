#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../../..");
const args = process.argv.slice(2);
const arg = (name, fallback) => {
  const index = args.indexOf(name);
  return index < 0 ? fallback : path.resolve(args[index + 1]);
};
const libraryRoot = arg("--library-root", path.join(repo, "scenarios/react-component-library/library"));
const errors = [];
const findings = [];
const readJSON = (file) => JSON.parse(fs.readFileSync(file, "utf8"));
const string = (value) => typeof value === "string" && value.trim().length > 0;
const positive = (value) => Number.isInteger(value) && value >= 0;

function sourceFor(componentDir, metadata) {
  const version = metadata.latest;
  if (!string(version)) throw new Error("component.json must identify latest");
  const versionDir = path.join(componentDir, "versions", version);
  const candidates = fs.existsSync(versionDir) ? fs.readdirSync(versionDir).filter((name) => name.endsWith(".tsx") && !name.endsWith(".test.tsx")) : [];
  const preferred = candidates.find((name) => name === `${path.basename(componentDir)}.tsx`) || candidates.find((name) => name !== "story.tsx");
  if (!preferred) throw new Error(`latest ${version} has no component TSX source`);
  return { version, versionDir, file: path.join(versionDir, preferred) };
}

function propShapes(sourceFile) {
  const source = ts.createSourceFile(sourceFile.file, fs.readFileSync(sourceFile.file, "utf8"), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const declarations = [];
  const aliases = new Map();
  const visit = (node) => {
    if ((ts.isInterfaceDeclaration(node) || ts.isTypeAliasDeclaration(node)) && /Props$/.test(node.name.text)) declarations.push(node);
    if (ts.isTypeAliasDeclaration(node)) aliases.set(node.name.text, node.type);
    ts.forEachChild(node, visit);
  };
  visit(source);
  const isCollection = (type, seen = new Set()) => {
    if (ts.isArrayTypeNode(type) || ts.isTupleTypeNode(type)) return true;
    if (ts.isTypeOperatorNode(type)) return isCollection(type.type, seen);
    if (ts.isTypeReferenceNode(type)) {
      const name = type.typeName.getText(source);
      if (/^(ReadonlyArray|Array)$/.test(name)) return true;
      if (seen.has(name) || !aliases.has(name)) return false;
      seen.add(name);
      return isCollection(aliases.get(name), seen);
    }
    return false;
  };
  const elementIsPrimitive = (type, seen = new Set()) => {
    if (ts.isTypeOperatorNode(type)) return elementIsPrimitive(type.type, seen);
    if (ts.isArrayTypeNode(type)) return elementIsPrimitive(type.elementType, seen);
    if (ts.isTupleTypeNode(type)) return type.elements.length > 0 && type.elements.every((element) => elementIsPrimitive(element, seen));
    if (ts.isUnionTypeNode(type)) return type.types.every((member) => elementIsPrimitive(member, seen));
    if (ts.isLiteralTypeNode(type)) return true;
    if (type.kind === ts.SyntaxKind.StringKeyword || type.kind === ts.SyntaxKind.NumberKeyword || type.kind === ts.SyntaxKind.BooleanKeyword || type.kind === ts.SyntaxKind.BigIntKeyword) return true;
    if (type.kind === ts.SyntaxKind.UnknownKeyword || type.kind === ts.SyntaxKind.AnyKeyword || type.kind === ts.SyntaxKind.ObjectKeyword) return false;
    if (ts.isTypeReferenceNode(type)) {
      const name = type.typeName.getText(source);
      if (/^(ReadonlyArray|Array)$/.test(name)) return type.typeArguments?.length ? elementIsPrimitive(type.typeArguments[0], seen) : false;
      if (seen.has(name) || !aliases.has(name)) return false;
      seen.add(name);
      return elementIsPrimitive(aliases.get(name), seen);
    }
    return false;
  };
  const result = [];
  for (const declaration of declarations) {
    const members = ts.isInterfaceDeclaration(declaration) ? declaration.members : ts.isTypeLiteralNode(declaration.type) ? declaration.type.members : [];
    for (const member of members) {
      if (!ts.isPropertySignature(member) || !member.name || !member.type) continue;
      const prop = member.name.getText(source).replace(/^['"]|['"]$/g, "");
      const type = member.type;
      const isArray = isCollection(type);
      // Props often contain arrays which describe configuration rather than
      // renderable entries (searchFields, defaultExpandedIds, disabledValues).
      // Those arrays do not have item identity or a visible label. Keep the
      // inventory focused on child/entry/action collections by requiring an
      // entry-oriented prop name instead of treating every `T[]` as items.
      const isEntryCollection = /^(items?|actions?|.*actions?|options?|tabs?|steps?|nodes?|rows?|entries|records|choices|messages|attachments|commands|links|sections|columns|filters|children)$/i.test(prop);
      if (prop === "children") result.push({ prop, kind: "children" });
      else if (isArray && (!elementIsPrimitive(type) || isEntryCollection)) result.push({ prop, kind: /action/i.test(prop) ? "actions" : "items" });
    }
  }
  return [...new Map(result.map((shape) => [shape.prop, shape])).values()];
}

function validateContract(asset, contract) {
  if (contract.version !== asset.version) errors.push(`${asset.name}: contract version must equal latest ${asset.version}`);
  if (contract.componentId !== `react-component-library:${asset.name}`) errors.push(`${asset.name}: componentId must be react-component-library:${asset.name}`);
  for (const shape of asset.shapes) {
    const collections = [...(contract.items ? [contract.items] : []), ...(Array.isArray(contract.collections) ? contract.collections : [])];
    const entry = shape.kind === "children" ? contract.children : collections.find((item) => item.property === shape.prop);
    if (!entry || typeof entry !== "object") {
      errors.push(`${asset.name}.${shape.prop}: missing contract`);
      continue;
    }
    const maxKey = entry.maximumCount === undefined ? "maxCount" : "maximumCount";
    const max = entry[maxKey];
    for (const [key, value] of [["typicalCount", entry.typicalCount], [maxKey, max]]) if (!positive(value)) errors.push(`${asset.name}.${shape.prop}: ${key} must be a non-negative integer`);
    if (positive(entry.typicalCount) && positive(max) && entry.typicalCount > max) errors.push(`${asset.name}.${shape.prop}: typicalCount exceeds maximumCount`);
    if (shape.kind === "children") {
      const allowed = entry.allowedChildComponents ?? entry.allowedChildren;
      if (!Array.isArray(allowed) || allowed.some((child) => !string(child))) errors.push(`${asset.name}.${shape.prop}: allowedChildComponents must list supported component names (use [] when no child components are supported)`);
      continue;
    }
    const fields = entry.requiredFields ?? [];
    if (!string(entry.stableIdField) && !fields.includes("id")) errors.push(`${asset.name}.${shape.prop}: stableIdField or requiredFields containing id is required`);
    if (!string(entry.labelField) && !fields.includes("label")) errors.push(`${asset.name}.${shape.prop}: labelField or requiredFields containing label is required`);
    const labelBudget = entry.maximumLabelCharacters ?? entry.labelMaxChars;
    if (!positive(labelBudget)) errors.push(`${asset.name}.${shape.prop}: maximumLabelCharacters must be a non-negative integer`);
    if (shape.kind === "actions" && !(string(entry.primaryField) && positive(entry.primaryCount) || ["required", "optional", "not-applicable"].includes(entry.primaryAction))) errors.push(`${asset.name}.${shape.prop}: primaryField/primaryCount or primaryAction is required`);
    if (shape.kind === "items" && entry.primaryAction !== "not-applicable" && entry.primaryField !== "not-applicable") errors.push(`${asset.name}.${shape.prop}: explicitly mark primary action not-applicable`);
  }
  for (const collection of [...(contract.items ? [contract.items] : []), ...(Array.isArray(contract.collections) ? contract.collections : [])]) {
    if (!asset.shapes.some((shape) => shape.kind !== "children" && shape.prop === collection.property)) errors.push(`${asset.name}.${collection.property}: contract does not correspond to a detected eligible collection prop`);
  }
}

for (const entry of fs.readdirSync(path.join(libraryRoot, "components"), { withFileTypes: true }).filter((item) => item.isDirectory())) {
  const componentDir = path.join(libraryRoot, "components", entry.name);
  const metadataPath = path.join(componentDir, "component.json");
  if (!fs.existsSync(metadataPath)) continue;
  try {
    const metadata = readJSON(metadataPath);
    const source = sourceFor(componentDir, metadata);
    const shapes = propShapes(source);
    if (!shapes.length) continue;
    const asset = { name: entry.name, version: source.version, source: path.relative(libraryRoot, source.file), shapes };
    findings.push(asset);
    const contractPath = path.join(source.versionDir, "usage-contract.json");
    if (!fs.existsSync(contractPath)) errors.push(`${entry.name}: missing usage-contract.json for latest ${source.version}`);
    else validateContract(asset, readJSON(contractPath));
  } catch (error) {
    errors.push(`${entry.name}: cannot inspect latest asset: ${error.message}`);
  }
}

const report = { eligible: findings, coverage: { eligible: findings.length, contracted: findings.length - new Set(errors.map((error) => error.split(":")[0])).size }, errors };
if (args.includes("--json")) console.log(JSON.stringify(report, null, 2));
else {
  for (const asset of findings) console.log(`${asset.name}@${asset.version}: ${asset.shapes.map(({ prop, kind }) => `${prop} (${kind})`).join(", ")}`);
  for (const error of errors) console.error(`ERROR ${error}`);
  console.log(`Usage-contract catalog coverage: ${report.coverage.contracted}/${report.coverage.eligible} eligible latest assets contracted.`);
}
if (errors.length) process.exitCode = 1;
