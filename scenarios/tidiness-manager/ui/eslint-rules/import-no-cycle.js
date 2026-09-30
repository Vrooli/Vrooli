import fs from "node:fs";
import path from "node:path";
import tsParser from "@typescript-eslint/parser";

const sourceExtensions = [".ts", ".tsx", ".js", ".jsx"];

function stripExtension(filePath) {
  for (const extension of sourceExtensions) {
    if (filePath.endsWith(extension)) {
      return filePath.slice(0, -extension.length);
    }
  }
  return filePath;
}

function resolveImport(fromFile, specifier, rootDir) {
  if (!specifier.startsWith(".") && !specifier.startsWith("@/")) {
    return null;
  }
  const basePath = specifier.startsWith("@/")
    ? path.join(rootDir, "src", specifier.slice(2))
    : path.resolve(path.dirname(fromFile), specifier);
  const candidates = [
    ...sourceExtensions.map((extension) => `${basePath}${extension}`),
    ...sourceExtensions.map((extension) => path.join(basePath, `index${extension}`)),
  ];
  return candidates.find((candidate) => fs.existsSync(candidate)) ?? null;
}

function parseImports(filePath) {
  let ast;
  try {
    ast = tsParser.parse(fs.readFileSync(filePath, "utf8"), {
      ecmaVersion: 2020,
      sourceType: "module",
      ecmaFeatures: { jsx: true },
    });
  } catch {
    return [];
  }
  return ast.body
    .filter((node) => node.type === "ImportDeclaration")
    .map((node) => node.source?.value)
    .filter((value) => typeof value === "string");
}

export const noCycle = {
  meta: {
    type: "problem",
    docs: {
      description: "detect direct cycles between local TypeScript imports",
    },
    schema: [],
  },
  create(context) {
    const filename = context.filename ?? context.getFilename();
    const rootDir = process.cwd();
    return {
      ImportDeclaration(node) {
        const importedPath = resolveImport(filename, node.source.value, rootDir);
        if (!importedPath) {
          return;
        }
        const currentPath = stripExtension(path.resolve(filename));
        for (const nestedSpecifier of parseImports(importedPath)) {
          const nestedPath = resolveImport(importedPath, nestedSpecifier, rootDir);
          if (nestedPath && stripExtension(nestedPath) === currentPath) {
            context.report({
              node,
              message: "Direct import cycle detected between this file and {{file}}.",
              data: { file: path.relative(rootDir, importedPath) },
            });
          }
        }
      },
    };
  },
};
