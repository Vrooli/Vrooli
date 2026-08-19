import { existsSync, readFileSync, readdirSync } from "node:fs";
import { basename, dirname, join, relative, resolve } from "node:path";
import ts from "typescript";

export const ASSET_ATTRIBUTE = "data-rcl-asset";
export const VERSION_ATTRIBUTE = "data-rcl-version";
export const STAMP_ATTRIBUTE = "data-rcl-stamp";
const SOURCE_EXTENSIONS = new Set([".ts", ".tsx"]);
const MARKER_ATTRIBUTES = new Set([
  ASSET_ATTRIBUTE,
  VERSION_ATTRIBUTE,
  STAMP_ATTRIBUTE,
]);

const defaultExemptionsURL = new URL("./asset-stamp-exemptions.json", import.meta.url);

function readExemptions(filePath = defaultExemptionsURL) {
  const path = filePath instanceof URL ? filePath : resolve(filePath);
  const raw = JSON.parse(readFileSync(path, "utf8"));
  if (!Array.isArray(raw)) {
    throw new Error("asset-stamp-exemptions.json must contain an array");
  }
  const result = new Map();
  for (const item of raw) {
    if (!item || typeof item.asset !== "string" || !item.asset.trim()) {
      throw new Error("every asset stamp exemption needs a non-empty asset");
    }
    if (typeof item.reason !== "string" || !item.reason.trim()) {
      throw new Error(`asset stamp exemption ${item.asset} needs a non-empty reason`);
    }
    if (result.has(item.asset)) {
      throw new Error(`duplicate asset stamp exemption: ${item.asset}`);
    }
    result.set(item.asset, { asset: item.asset, reason: item.reason.trim() });
  }
  return result;
}

function findExemption(exemptions, asset, libraryId) {
  if (asset && exemptions.has(asset)) return exemptions.get(asset);
  if (libraryId && exemptions.has(libraryId)) return exemptions.get(libraryId);
  for (const [pattern, exemption] of exemptions) {
    if (pattern.endsWith(".*") && asset?.startsWith(pattern.slice(0, -1))) {
      return exemption;
    }
  }
  return undefined;
}

function tagName(node) {
  if (!node || !node.tagName) return "";
  return node.tagName.getText();
}

function isMarkerAttribute(property) {
  return ts.isJsxAttribute(property) && MARKER_ATTRIBUTES.has(property.name.getText());
}

function markerAttributes(asset, version) {
  return [
    ts.factory.createJsxAttribute(
      ts.factory.createIdentifier(ASSET_ATTRIBUTE),
      ts.factory.createStringLiteral(asset),
    ),
    ts.factory.createJsxAttribute(
      ts.factory.createIdentifier(VERSION_ATTRIBUTE),
      ts.factory.createStringLiteral(version),
    ),
    ts.factory.createJsxAttribute(
      ts.factory.createIdentifier(STAMP_ATTRIBUTE),
      ts.factory.createStringLiteral("vite"),
    ),
  ];
}

function updateOpeningElement(node, asset, version, stamp) {
  const properties = node.attributes.properties.filter(
    (property) => !isMarkerAttribute(property),
  );
  if (stamp) properties.push(...markerAttributes(asset, version));
  return ts.isJsxSelfClosingElement(node)
    ? ts.factory.updateJsxSelfClosingElement(
        node,
        node.tagName,
        node.typeArguments,
        ts.factory.createJsxAttributes(properties),
      )
    : ts.factory.updateJsxOpeningElement(
        node,
        node.tagName,
        node.typeArguments,
        ts.factory.createJsxAttributes(properties),
      );
}

function isProviderOrFragment(node) {
  const name = tagName(node);
  return (
    !name ||
    name === "Fragment" ||
    name === "React.Fragment" ||
    name.endsWith(".Provider")
  );
}

function firstOwnedJSX(node) {
  if (ts.isJsxFragment(node)) {
    for (const child of node.children) {
      const candidate = firstOwnedJSX(child);
      if (candidate) return candidate;
    }
    return undefined;
  }
  if (ts.isJsxElement(node)) {
    if (tagName(node.openingElement) === "style") return undefined;
    if (isProviderOrFragment(node)) {
      for (const child of node.children) {
        const candidate = firstOwnedJSX(child);
        if (candidate) return candidate;
      }
    }
    return node;
  }
  if (ts.isJsxSelfClosingElement(node)) {
    return tagName(node) === "style" ? undefined : node;
  }
  return undefined;
}

function firstOwnedJSXInTree(node) {
  let result;
  function visit(candidate) {
    if (result) return;
    if (ts.isJsxElement(candidate) || ts.isJsxSelfClosingElement(candidate) || ts.isJsxFragment(candidate)) {
      result = firstOwnedJSX(candidate);
      if (result) return;
    }
    ts.forEachChild(candidate, visit);
  }
  visit(node);
  return result;
}

function declarationName(node) {
  if (ts.isFunctionDeclaration(node) || ts.isClassDeclaration(node)) {
    return node.name?.getText() || "";
  }
  if (ts.isVariableDeclaration(node)) return node.name.getText();
  return "";
}

function findNamedDeclaration(sourceFile, name) {
  let result;
  function visit(node) {
    if (result) return;
    if (
      (ts.isFunctionDeclaration(node) || ts.isClassDeclaration(node) || ts.isVariableDeclaration(node)) &&
      declarationName(node) === name
    ) {
      result = node;
      return;
    }
    ts.forEachChild(node, visit);
  }
  visit(sourceFile);
  return result;
}

function findCreateElementCall(node) {
  let result;
  function visit(candidate) {
    if (result) return;
    if (ts.isCallExpression(candidate) && candidate.expression.getText() === "createElement") {
      const first = candidate.arguments[0];
      if (first && !ts.isStringLiteral(first)) result = candidate;
    }
    ts.forEachChild(candidate, visit);
  }
  visit(node);
  return result;
}

function updateCreateElementCall(node, asset, version) {
  const args = [...node.arguments];
  const propsIndex = 1;
  const existing = args[propsIndex];
  const properties = [];
  if (existing && ts.isObjectLiteralExpression(existing)) {
    for (const property of existing.properties) {
      if (
        ts.isPropertyAssignment(property) &&
        (property.name.getText().replaceAll('"', "") === ASSET_ATTRIBUTE ||
          property.name.getText().replaceAll('"', "") === VERSION_ATTRIBUTE ||
          property.name.getText().replaceAll('"', "") === STAMP_ATTRIBUTE)
      ) {
        continue;
      }
      properties.push(property);
    }
  } else if (existing) {
    properties.push(ts.factory.createSpreadAssignment(existing));
    args.splice(propsIndex, 1);
  }
  properties.push(
    ts.factory.createPropertyAssignment(ASSET_ATTRIBUTE, ts.factory.createStringLiteral(asset)),
    ts.factory.createPropertyAssignment(VERSION_ATTRIBUTE, ts.factory.createStringLiteral(version)),
    ts.factory.createPropertyAssignment(STAMP_ATTRIBUTE, ts.factory.createStringLiteral("vite")),
  );
  if (!args[propsIndex] || !ts.isObjectLiteralExpression(args[propsIndex])) {
    args.splice(propsIndex, 0, ts.factory.createObjectLiteralExpression(properties, true));
  } else {
    args[propsIndex] = ts.factory.createObjectLiteralExpression(properties, true);
  }
  return ts.factory.updateCallExpression(
    node,
    node.expression,
    node.typeArguments,
    args,
  );
}

function renderTarget(sourceFile, componentName) {
  const declaration = findNamedDeclaration(sourceFile, componentName);
  const call = findCreateElementCall(declaration || sourceFile);
  if (call) return { kind: "createElement", node: call };
  const root = firstOwnedJSXInTree(declaration || sourceFile);
  return root ? { kind: "jsx", node: root } : undefined;
}

export function stampSource(source, { asset, version, componentName }) {
  const sourceFile = ts.createSourceFile(
    "asset.tsx",
    source,
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX,
  );
  const target = renderTarget(sourceFile, componentName);
  if (!target?.node) return { changed: false, code: source, reason: "no-owned-root" };
  const edits = [];
  function collectAuthoredMarkers(node) {
    const opening = ts.isJsxElement(node) ? node.openingElement : node;
    if (ts.isJsxElement(node) || ts.isJsxSelfClosingElement(node)) {
      for (const property of opening.attributes.properties) {
        if (isMarkerAttribute(property)) {
          edits.push({ start: property.getStart(sourceFile), end: property.end, text: "" });
        }
      }
    }
    ts.forEachChild(node, collectAuthoredMarkers);
  }
  collectAuthoredMarkers(sourceFile);

  if (target.kind === "createElement") {
    const existing = target.node.arguments[1];
    if (existing && ts.isObjectLiteralExpression(existing)) {
      const prefix = existing.properties.length ? ", " : "";
      edits.push({
        start: existing.end - 1,
        end: existing.end - 1,
        text:
          `${prefix}${ASSET_ATTRIBUTE}: ${JSON.stringify(asset)}, ` +
          `${VERSION_ATTRIBUTE}: ${JSON.stringify(version)}, ` +
          `${STAMP_ATTRIBUTE}: "vite"`,
      });
    }
  } else {
    const owned = target.node;
    const opening = ts.isJsxElement(owned) ? owned.openingElement : owned;
    const openingStart = opening.getStart(sourceFile);
    const openingText = source.slice(openingStart, opening.end);
    const closingOffset = openingText.lastIndexOf(">");
    const closingPosition = openingStart + closingOffset;
    const insertionPosition = openingText[closingOffset - 1] === "/"
      ? closingPosition - 1
      : closingPosition;
    edits.push({
      start: insertionPosition,
      end: insertionPosition,
      text:
        ` ${ASSET_ATTRIBUTE}=${JSON.stringify(asset)}` +
        ` ${VERSION_ATTRIBUTE}=${JSON.stringify(version)}` +
        ` ${STAMP_ATTRIBUTE}="vite"`,
    });
  }

  edits.sort((a, b) => b.start - a.start || b.end - a.end);
  let code = source;
  for (const edit of edits) {
    code = code.slice(0, edit.start) + edit.text + code.slice(edit.end);
  }
  return { changed: code !== source, code, reason: target.kind };
}

function sourceMetadata(id, scenarioRoot, exemptions) {
  const normalized = id.replaceAll("\\", "/");
  const marker = "/library/";
  const markerIndex = normalized.indexOf(marker);
  if (markerIndex < 0) return undefined;
  const relativePath = normalized.slice(markerIndex + marker.length);
  const parts = relativePath.split("/");
  if (parts.length < 5 || parts[2] !== "versions") return undefined;
  const [kind, slug, , version, file] = parts;
  if (!kind || !slug || !version || !file || !SOURCE_EXTENSIONS.has(file.slice(file.lastIndexOf(".")))) return undefined;
  if (file === "story.tsx") return undefined;
  const versionDir = dirname(id);
  const assetDir = resolve(versionDir, "..", "..");
  const manifestPath = join(assetDir, "component.json");
  if (!existsSync(manifestPath)) return undefined;
  const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
  const sourceFiles = readdirSync(versionDir).filter((entry) => {
    const extension = entry.slice(entry.lastIndexOf("."));
    return SOURCE_EXTENSIONS.has(extension) && entry !== "story.tsx";
  });
  const tsxEntries = sourceFiles.filter((entry) => entry.endsWith(".tsx"));
  const tsEntries = sourceFiles.filter((entry) => entry.endsWith(".ts"));
  const entry = manifest.entry ||
    (tsxEntries.length === 1 ? tsxEntries[0] : tsEntries.length === 1 ? tsEntries[0] : "");
  if (!entry || basename(id) !== entry) return undefined;
  const asset = String(manifest.catalogId || "").trim();
  const libraryId = String(manifest.libraryId || "").trim();
  const exemption = findExemption(exemptions, asset, libraryId);
  if (!asset) {
    if (!exemption) {
      throw new Error(`catalog entry ${libraryId || relativePath} has no catalogId or stamp exemption`);
    }
    return { exempt: true, reason: exemption.reason, relativePath };
  }
  return {
    asset,
    version,
    componentName: basename(entry, entry.slice(entry.lastIndexOf("."))),
    exempt: Boolean(exemption),
    reason: exemption?.reason,
    relativePath,
    scenarioRoot,
  };
}

export function assetStampMetadata(id, scenarioRoot, exemptionFile) {
  return sourceMetadata(id, scenarioRoot, readExemptions(exemptionFile));
}

export default function assetStampPlugin(options = {}) {
  let scenarioRoot = options.scenarioRoot;
  const exemptions = readExemptions(options.exemptionFile || defaultExemptionsURL);
  return {
    name: "rcl-build-asset-stamp",
    enforce: "pre",
    configResolved(config) {
      scenarioRoot ||= dirname(config.root);
    },
    transform(source, id) {
      const metadata = sourceMetadata(id, scenarioRoot, exemptions);
      if (!metadata || metadata.exempt) return null;
      const result = stampSource(source, metadata);
      if (!result.changed && result.reason === "no-owned-root") {
        throw new Error(
          `catalog asset ${metadata.asset}@${metadata.version} entry ${metadata.relativePath} has no owned root; add a reasoned exemption`,
        );
      }
      return { code: result.code, map: null };
    },
  };
}
