#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../library");

function walk(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) return walk(file);
    return entry.isFile() && /\.tsx$/.test(entry.name) && entry.name !== "story.tsx" ? [file] : [];
  });
}

function matchingBrace(source, open) {
  let depth = 0;
  let quote = "";
  for (let index = open; index < source.length; index += 1) {
    const char = source[index];
    const next = source[index + 1];
    if (quote) {
      if (char === "\\") index += 1;
      else if (char === quote) quote = "";
      continue;
    }
    if (char === "\"" || char === "'" || char === "`") {
      quote = char;
      continue;
    }
    if (char === "/" && next === "/") {
      index = source.indexOf("\n", index + 2);
      if (index < 0) return source.length;
      continue;
    }
    if (char === "/" && next === "*") {
      index = source.indexOf("*/", index + 2);
      if (index < 0) return source.length;
      index += 1;
      continue;
    }
    if (char === "{") depth += 1;
    if (char === "}" && --depth === 0) return index;
  }
  return source.length;
}

function componentBodies(source) {
  const bodies = [];
  const functions = /function\s+([A-Z][A-Za-z0-9_]*)\s*\(/g;
  for (const match of source.matchAll(functions)) {
    const openParen = source.indexOf("(", match.index);
    let parens = 0;
    let closeParen = openParen;
    for (; closeParen < source.length; closeParen += 1) {
      if (source[closeParen] === "(") parens += 1;
      if (source[closeParen] === ")" && --parens === 0) break;
    }
    const openBrace = source.indexOf("{", closeParen);
    if (openBrace < 0) continue;
    bodies.push({ open: openBrace, close: matchingBrace(source, openBrace) });
  }
  return bodies;
}

let changed = 0;
for (const file of walk(root)) {
  const original = fs.readFileSync(file, "utf8");
  if (!original.includes("resolveStrings(")) {
    let cleaned = original.replace(
      'import { resolveStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
      'import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
    );
    if (!/\bstrings\(/.test(cleaned)) {
      cleaned = cleaned.replace(/\n\s*const strings = useStrings\(\);/, "");
    }
    if (cleaned !== original) {
      fs.writeFileSync(file, cleaned);
      changed += 1;
    }
    continue;
  }
  const bodies = componentBodies(original).filter(({ open, close }) =>
    original.slice(open, close).includes("resolveStrings("),
  );
  if (bodies.length === 0) continue;
  const replacements = [];
  for (const body of bodies) {
    replacements.push({ start: body.open + 1, end: body.open + 1, text: "\n  const strings = useStrings();" });
    const pattern = /resolveStrings\(/g;
    for (const match of original.slice(body.open, body.close).matchAll(pattern)) {
      replacements.push({ start: body.open + match.index, end: body.open + match.index + "resolveStrings".length, text: "strings" });
    }
  }
  let updated = original;
  for (const replacement of replacements.sort((a, b) => b.start - a.start)) {
    updated = updated.slice(0, replacement.start) + replacement.text + updated.slice(replacement.end);
  }
  updated = updated.replace(
    'import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
    'import { resolveStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
  );
  if (!/\bresolveStrings\(/.test(updated)) {
    updated = updated.replace(
      'import { resolveStrings, useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
      'import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";',
    );
  }
  if (!/\bstrings\(/.test(updated)) {
    updated = updated.replace(/\n\s*const strings = useStrings\(\);/, "");
  }
  if (updated !== original) {
    fs.writeFileSync(file, updated);
    changed += 1;
  }
}

console.log(JSON.stringify({ filesChanged: changed }));
