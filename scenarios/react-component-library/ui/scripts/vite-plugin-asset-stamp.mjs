import { existsSync, mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
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

// Exemption kinds are a policy distinction, not a formatting one. A permanent
// exemption is a structural fact about the asset — it renders no DOM root, so
// no stamp can ever exist. A backlog exemption is debt: the asset renders a
// root but has no catalog identity to stamp it with. Collapsing the two makes
// a growing suppression list look like a stable one.
const EXEMPTION_KINDS = new Set(["permanent", "backlog"]);

// Plain absolute paths, not `new URL("…", import.meta.url)`: Vite statically
// rewrites that expression into an "/@fs/…" asset URL, so the same module read
// its own sidecar JSON fine under Node and failed under any Vite-hosted runner.
const scriptDir = dirname(fileURLToPath(import.meta.url));
const defaultExemptionsPath = resolve(scriptDir, "asset-stamp-exemptions.json");
const defaultMapPath = resolve(scriptDir, "asset-stamp-map.json");

const SOURCE_MARKER = /@vrooliComponentSource\s+([^\s*]+)/;
// Only a re-export makes a file a shim for another module. A plain `import`
// means the file *composes* that asset, which is bespoke authorship and must
// not inherit the asset's identity — otherwise hand-written workbench code
// would be stamped as though it were the adopted library asset.
const REEXPORT_TARGET_ALL = /export\s+(?:\*|\{[^}]*\})\s*(?:as\s+\w+\s*)?from\s+["']([^"']+)["']/g;

function readExemptions(filePath = defaultExemptionsPath) {
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
    if (!EXEMPTION_KINDS.has(item.kind)) {
      throw new Error(
        `asset stamp exemption ${item.asset} needs kind "permanent" (renders no DOM root) or "backlog" (renders a root but has no catalog id yet)`,
      );
    }
    if (result.has(item.asset)) {
      throw new Error(`duplicate asset stamp exemption: ${item.asset}`);
    }
    result.set(item.asset, { asset: item.asset, kind: item.kind, reason: item.reason.trim() });
  }
  return result;
}

function readStampMap(filePath = defaultMapPath) {
  const path = filePath instanceof URL ? filePath : resolve(filePath);
  if (!existsSync(path)) return new Map();
  const raw = JSON.parse(readFileSync(path, "utf8"));
  if (!Array.isArray(raw)) {
    throw new Error("asset-stamp-map.json must contain an array");
  }
  const result = new Map();
  for (const item of raw) {
    for (const field of ["path", "asset", "version"]) {
      if (!item || typeof item[field] !== "string" || !item[field].trim()) {
        throw new Error(`every asset stamp map entry needs a non-empty ${field}`);
      }
    }
    if (result.has(item.path)) {
      throw new Error(`duplicate asset stamp map entry: ${item.path}`);
    }
    result.set(item.path, { asset: item.asset.trim(), version: item.version.trim() });
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
      const existingText = source.slice(existing.getStart(sourceFile), existing.end - 1).trimEnd();
      const prefix = existing.properties.length && !existingText.endsWith(",") ? ", " : "";
      // The marker names contain hyphens, so they are only valid as quoted
      // object keys. Emitting them bare produces a syntax error that no unit
      // test catches until a real asset happens to use a dynamic root.
      edits.push({
        start: existing.end - 1,
        end: existing.end - 1,
        text:
          `${prefix}${JSON.stringify(ASSET_ATTRIBUTE)}: ${JSON.stringify(asset)}, ` +
          `${JSON.stringify(VERSION_ATTRIBUTE)}: ${JSON.stringify(version)}, ` +
          `${JSON.stringify(STAMP_ATTRIBUTE)}: "vite"`,
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

// ---------------------------------------------------------------------------
// Resolver chain
//
// The plugin's only real job is source file -> (asset id, version). There is
// more than one legitimate way to know that, so resolution is an ordered list
// of strategies rather than a single hardcoded path shape. Keying solely off
// "/library/" is what left every adopted copy — which is what a scenario
// actually renders — permanently unstamped and therefore unmeasurable.
// ---------------------------------------------------------------------------

function versionedParts(normalizedPath, marker) {
  const markerIndex = normalizedPath.indexOf(marker);
  if (markerIndex < 0) return undefined;
  const relativePath = normalizedPath.slice(markerIndex + marker.length);
  const parts = relativePath.split("/");
  return { relativePath, parts };
}

// Strategy 1: the library tree, where component.json is the identity oracle.
function libraryMetadata(id, exemptions) {
  const normalized = id.replaceAll("\\", "/");
  const found = versionedParts(normalized, "/library/");
  if (!found) return undefined;
  const { relativePath, parts } = found;
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
    return SOURCE_EXTENSIONS.has(extension) &&
      entry !== "story.tsx" &&
      !entry.includes(".test.") &&
      !entry.includes(".spec.");
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
    return { exempt: true, kind: exemption.kind, reason: exemption.reason, relativePath, strategy: "library" };
  }
  return {
    asset,
    version,
    componentName: basename(entry, entry.slice(entry.lastIndexOf("."))),
    exempt: Boolean(exemption),
    kind: exemption?.kind,
    reason: exemption?.reason,
    relativePath,
    strategy: "library",
  };
}

// Strategy 2: an adopted copy. Adopted trees carry no component.json, so the
// identity oracle is the @vrooliComponentSource marker on the re-export shim
// that points at the implementation. The index is built from shims outward so
// the strategy does not depend on how deeply a consumer nests its vendored
// files.
function buildAdoptedIndex(componentsRoot) {
  const index = new Map();
  if (!componentsRoot || !existsSync(componentsRoot)) return index;

  function walkShims(directory) {
    let entries;
    try {
      entries = readdirSync(directory, { withFileTypes: true });
    } catch {
      return;
    }
    for (const entry of entries) {
      const full = join(directory, entry.name);
      if (entry.isDirectory()) {
        // Implementations live under versions/; only shims are scanned here.
        if (entry.name === "versions" || entry.name === "node_modules") continue;
        walkShims(full);
        continue;
      }
      if (!entry.name.endsWith(".tsx") && !entry.name.endsWith(".ts")) continue;
      if (entry.name.includes(".test.") || entry.name.includes(".spec.")) continue;
      let body;
      try {
        body = readFileSync(full, "utf8");
      } catch {
        continue;
      }
      const markerMatch = SOURCE_MARKER.exec(body);
      if (!markerMatch) continue;
      const asset = markerMatch[1].trim();
      const targets = [];
      for (const match of body.matchAll(REEXPORT_TARGET_ALL)) {
        const candidate = match[1];
        if (!candidate.startsWith(".")) continue;
        const parts = candidate.split("/");
        const at = parts.lastIndexOf("versions");
        if (at < 0 || at + 1 >= parts.length) continue;
        if (!targets.includes(candidate)) targets.push(candidate);
      }
      if (targets.length === 0) continue;
      // A shim that re-exports more than one versioned implementation is a
      // composite, not an identity. Stamping the first target would attribute
      // one asset's rendered nodes to another, so the ambiguity is refused.
      if (targets.length > 1) {
        throw new Error(
          `shim ${full} declares @vrooliComponentSource ${asset} but re-exports ${targets.length} versioned implementations (${targets.join(", ")}); split it into one shim per asset or drop the marker`,
        );
      }
      const target = targets[0];
      const targetParts = target.split("/");
      const version = targetParts[targetParts.lastIndexOf("versions") + 1];
      const resolvedBase = resolve(dirname(full), target);
      for (const extension of [".tsx", ".ts"]) {
        const candidate = resolvedBase + extension;
        if (!existsSync(candidate)) continue;
        index.set(candidate.replaceAll("\\", "/"), {
          asset,
          version,
          componentName: basename(candidate, extension),
          shim: full,
        });
        break;
      }
    }
  }

  walkShims(componentsRoot);
  return index;
}

function adoptedMetadata(id, exemptions, adoptedIndex) {
  const normalized = id.replaceAll("\\", "/");
  const entry = adoptedIndex.get(normalized);
  if (!entry) return undefined;
  // Exemptions are consulted before the identity check: an asset with no
  // catalog id yet is legitimately still carrying a library-id marker, and
  // that is tracked as backlog rather than treated as a build error.
  const exemption = findExemption(exemptions, entry.asset, entry.asset);
  // A library-id marker cannot be joined against catalog evidence. The
  // resolver refuses it rather than stamping an identity the gates cannot
  // reconcile, which would look like coverage while measuring nothing.
  if (entry.asset.includes(":") && !exemption) {
    throw new Error(
      `adopted shim ${entry.shim} declares @vrooliComponentSource ${entry.asset}; the marker must be a catalog asset id (for example "primitives.card") so rendered evidence can join the catalog`,
    );
  }
  return {
    asset: entry.asset,
    version: entry.version,
    componentName: entry.componentName,
    exempt: Boolean(exemption),
    kind: exemption?.kind,
    reason: exemption?.reason,
    relativePath: normalized,
    strategy: "adopted",
  };
}

// Strategy 3: an explicit override for anything the first two cannot see.
function mappedMetadata(id, exemptions, stampMap, scenarioRoot) {
  const normalized = id.replaceAll("\\", "/");
  const root = (scenarioRoot || "").replaceAll("\\", "/").replace(/\/$/, "");
  const relative = root && normalized.startsWith(root + "/")
    ? normalized.slice(root.length + 1)
    : normalized;
  const entry = stampMap.get(relative) || stampMap.get(normalized);
  if (!entry) return undefined;
  const exemption = findExemption(exemptions, entry.asset, undefined);
  return {
    asset: entry.asset,
    version: entry.version,
    componentName: basename(normalized, normalized.slice(normalized.lastIndexOf("."))),
    exempt: Boolean(exemption),
    kind: exemption?.kind,
    reason: exemption?.reason,
    relativePath: relative,
    strategy: "map",
  };
}

function sourceMetadata(id, scenarioRoot, exemptions, context = {}) {
  const adoptedIndex = context.adoptedIndex || new Map();
  const stampMap = context.stampMap || new Map();
  return (
    libraryMetadata(id, exemptions) ||
    adoptedMetadata(id, exemptions, adoptedIndex) ||
    mappedMetadata(id, exemptions, stampMap, scenarioRoot)
  );
}

export function assetStampMetadata(id, scenarioRoot, exemptionFile, context) {
  return sourceMetadata(id, scenarioRoot, readExemptions(exemptionFile), context);
}

export function adoptedAssetIndex(componentsRoot) {
  return buildAdoptedIndex(componentsRoot);
}

// ---------------------------------------------------------------------------
// Declared-asset census
//
// transform() only ever sees files the bundle actually imports, so a report
// built from transform alone cannot tell "no stamp" from "not in this build".
// The census enumerates every asset that *should* be stampable up front, and
// the transform pass marks off what it reached.
// ---------------------------------------------------------------------------

function censusLibraryAssets(scenarioRoot, exemptions) {
  const declared = new Map();
  const libraryRoot = join(scenarioRoot, "library");
  if (!existsSync(libraryRoot)) return declared;
  for (const kind of readdirSync(libraryRoot)) {
    const kindDir = join(libraryRoot, kind);
    let stat;
    try {
      stat = statSync(kindDir);
    } catch {
      continue;
    }
    if (!stat.isDirectory()) continue;
    for (const slug of readdirSync(kindDir)) {
      const manifestPath = join(kindDir, slug, "component.json");
      if (!existsSync(manifestPath)) continue;
      let manifest;
      try {
        manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
      } catch {
        continue;
      }
      const asset = String(manifest.catalogId || "").trim();
      const libraryId = String(manifest.libraryId || "").trim();
      const exemption = findExemption(exemptions, asset, libraryId);
      const identity = asset || libraryId;
      if (!identity) continue;
      declared.set(identity, {
        asset: asset || null,
        libraryId,
        version: String(manifest.latest || "").trim(),
        exempt: Boolean(exemption),
        exemptKind: exemption?.kind || null,
        reason: exemption?.reason || null,
        strategy: "library",
      });
    }
  }
  return declared;
}

function censusAdoptedAssets(adoptedIndex, exemptions) {
  const declared = new Map();
  for (const entry of adoptedIndex.values()) {
    const exemption = findExemption(exemptions, entry.asset, undefined);
    declared.set(entry.asset, {
      asset: entry.asset,
      libraryId: null,
      version: entry.version,
      exempt: Boolean(exemption),
      exemptKind: exemption?.kind || null,
      reason: exemption?.reason || null,
      strategy: "adopted",
    });
  }
  return declared;
}

export function buildStampReport({ declared, stamped, generatedAt }) {
  const assets = [];
  for (const [identity, entry] of declared) {
    const hit = stamped.get(identity);
    let state;
    if (hit) state = "stamped";
    else if (entry.exempt) state = entry.exemptKind === "backlog" ? "exempt-backlog" : "exempt-permanent";
    else state = "unbundled";
    assets.push({
      asset: entry.asset,
      libraryId: entry.libraryId,
      identity,
      version: hit?.version || entry.version,
      state,
      strategy: hit?.strategy || entry.strategy,
      reason: entry.reason,
    });
  }
  for (const [identity, hit] of stamped) {
    if (declared.has(identity)) continue;
    assets.push({
      asset: hit.asset,
      libraryId: null,
      identity,
      version: hit.version,
      state: "stamped",
      strategy: hit.strategy,
      reason: null,
    });
  }
  assets.sort((a, b) => a.identity.localeCompare(b.identity));
  const totals = { stamped: 0, "exempt-permanent": 0, "exempt-backlog": 0, unbundled: 0 };
  for (const asset of assets) totals[asset.state] += 1;
  return { generatedAt, totals, assets };
}

export default function assetStampPlugin(options = {}) {
  let scenarioRoot = options.scenarioRoot;
  let reportFile = options.reportFile;
  const exemptions = readExemptions(options.exemptionFile || defaultExemptionsPath);
  const stampMap = readStampMap(options.mapFile || defaultMapPath);
  let adoptedIndex = new Map();
  let declared = new Map();
  const stamped = new Map();

  return {
    name: "rcl-build-asset-stamp",
    enforce: "pre",
    configResolved(config) {
      scenarioRoot ||= dirname(config.root);
      reportFile ||= join(config.build.outDir, "asset-stamp-report.json");
      const componentsRoot = options.componentsRoot || join(config.root, "src", "components");
      adoptedIndex = buildAdoptedIndex(componentsRoot);
      declared = new Map([
        ...censusLibraryAssets(scenarioRoot, exemptions),
        ...censusAdoptedAssets(adoptedIndex, exemptions),
      ]);
    },
    transform(source, id) {
      const metadata = sourceMetadata(id, scenarioRoot, exemptions, { adoptedIndex, stampMap });
      if (!metadata || metadata.exempt) return null;
      const result = stampSource(source, metadata);
      if (!result.changed && result.reason === "no-owned-root") {
        throw new Error(
          `catalog asset ${metadata.asset}@${metadata.version} entry ${metadata.relativePath} has no owned root; add a reasoned exemption`,
        );
      }
      stamped.set(metadata.asset, {
        asset: metadata.asset,
        version: metadata.version,
        strategy: metadata.strategy,
      });
      return { code: result.code, map: null };
    },
    buildEnd() {
      if (!scenarioRoot) return;
      const report = buildStampReport({
        declared,
        stamped,
        generatedAt: new Date().toISOString(),
      });
      const directory = dirname(reportFile);
      try {
        mkdirSync(directory, { recursive: true });
        writeFileSync(
          reportFile,
          `${JSON.stringify(report, null, 2)}\n`,
        );
      } catch {
        // The report is diagnostic evidence, not a build input. A read-only
        // or sandboxed output directory must not fail an otherwise valid build.
      }
    },
  };
}
