#!/usr/bin/env node

import { createHash } from "node:crypto";
import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { dirname, extname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const SOURCE_EXTENSIONS = new Set([".ts", ".tsx", ".js", ".jsx"]);
const LIBRARY_KINDS = ["components", "primitives", "hooks", "services", "foundations"];
const UTILITY_EXACT = new Set([
  "absolute", "block", "border-collapse", "fixed", "flex", "grid", "hidden",
  "inline", "inline-block", "inline-flex", "relative", "sr-only", "table-fixed",
  "touch-target", "truncate", "uppercase", "whitespace-nowrap",
]);
const UTILITY_PREFIXES = [
  "bg-", "border-", "bottom-", "divide-", "end-", "flex-", "font-", "from-",
  "gap-", "grid-", "h-", "inset-", "items-", "justify-", "leading-", "left-",
  "m-", "max-", "mb-", "min-", "ml-", "mr-", "mt-", "mx-", "my-", "opacity-",
  "outline-", "overflow-", "p-", "pb-", "pl-", "pointer-events-", "pr-", "pt-",
  "px-", "py-", "right-", "ring-", "rounded-", "shadow-", "shrink-", "size-",
  "space-", "start-", "text-", "to-", "top-", "tracking-", "transition", "via-",
  "w-", "z-", "-translate-", "translate-",
];

function parseArgs(argv) {
  const options = { root: resolve(fileURLToPath(new URL("../../..", import.meta.url))) };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--root") options.root = resolve(argv[++index]);
    else if (argument === "--output") options.output = resolve(argv[++index]);
    else if (argument === "--generated-at") options.generatedAt = argv[++index];
    else if (argument === "--help") options.help = true;
    else throw new Error(`unknown argument: ${argument}`);
  }
  return options;
}

function walk(root, predicate = () => true) {
  if (!existsSync(root)) return [];
  const files = [];
  const visit = (path) => {
    const entries = readdirSync(path, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name));
    for (const entry of entries) {
      const child = join(path, entry.name);
      if (entry.isDirectory()) visit(child);
      else if (entry.isFile() && predicate(child)) files.push(child);
    }
  };
  visit(root);
  return files;
}

function read(path) {
  return readFileSync(path, "utf8");
}

function readJSON(path) {
  return JSON.parse(read(path));
}

function repoPath(repoRoot, path) {
  return relative(repoRoot, path).split("\\").join("/");
}

function lineAt(source, offset) {
  return source.slice(0, offset).split("\n").length;
}

function isUtilityToken(token) {
  const normalized = token.trim().replace(/^!/, "");
  if (
    !normalized
    || normalized.startsWith("data-")
    || normalized.startsWith("--")
    || normalized.includes("${")
    || /[{};'"`]/.test(normalized)
  ) return false;
  if (/^(?:[A-Za-z0-9_-]+:)+!?[-A-Za-z0-9_[\]/().,%]+$/.test(normalized)) return true;
  if (normalized.includes("[") && normalized.includes("]")) return true;
  return UTILITY_EXACT.has(normalized) || UTILITY_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}

function quotedStrings(source) {
  const strings = [];
  const pattern = /(["'`])((?:\\.|(?!\1)[\s\S])*?)\1/g;
  for (const match of source.matchAll(pattern)) strings.push({ value: match[2], offset: match.index ?? 0 });
  return strings;
}

function balancedExpression(source, start) {
  let depth = 0;
  let quote = "";
  let escaped = false;
  for (let index = start; index < source.length; index += 1) {
    const character = source[index];
    if (quote) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === quote) quote = "";
      continue;
    }
    if (character === "\"" || character === "'" || character === "`") quote = character;
    else if (character === "{") depth += 1;
    else if (character === "}" && --depth === 0) return source.slice(start + 1, index);
  }
  return source.slice(start + 1);
}

export function extractUtilityClasses(source) {
  const candidateStrings = [];
  const referencedVariables = new Set();
  const attributePattern = /\b(?:className|class)\s*=\s*/g;
  for (const match of source.matchAll(attributePattern)) {
    let cursor = (match.index ?? 0) + match[0].length;
    const next = source[cursor];
    if (next === "\"" || next === "'" || next === "`") {
      const literal = quotedStrings(source.slice(cursor))[0];
      if (literal) candidateStrings.push({ ...literal, offset: cursor + literal.offset });
    } else if (next === "{") {
      const expression = balancedExpression(source, cursor);
      candidateStrings.push(...quotedStrings(expression).map((item) => ({ ...item, offset: cursor + item.offset })));
      for (const identifier of expression.matchAll(/\b([A-Za-z_$][\w$]*)\b/g)) referencedVariables.add(identifier[1]);
    }
  }
  const inspectedVariables = new Set();
  const pendingVariables = [...referencedVariables];
  while (pendingVariables.length) {
    const identifier = pendingVariables.shift();
    if (!identifier || inspectedVariables.has(identifier)) continue;
    inspectedVariables.add(identifier);
    const escaped = identifier.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const initializer = new RegExp(`\\b(?:const|let)\\s+${escaped}\\s*=([\\s\\S]*?);`, "m").exec(source)?.[1];
    if (!initializer) continue;
    candidateStrings.push(...quotedStrings(initializer));
    for (const nested of initializer.matchAll(/\b([A-Za-z_$][\w$]*)\b/g)) pendingVariables.push(nested[1]);
  }
  const hits = new Map();
  for (const literal of candidateStrings) {
    for (const token of literal.value.split(/\s+/)) {
      if (!isUtilityToken(token)) continue;
      const clean = token.trim();
      if (!hits.has(clean)) hits.set(clean, { class: clean, line: lineAt(source, literal.offset) });
    }
  }
  return [...hits.values()].sort((a, b) => a.class.localeCompare(b.class));
}

function scanImports(repoRoot) {
  const scenariosRoot = join(repoRoot, "scenarios");
  const imports = [];
  if (!existsSync(scenariosRoot)) return imports;
  for (const scenarioEntry of readdirSync(scenariosRoot, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
    if (!scenarioEntry.isDirectory()) continue;
    const sourceRoot = join(scenariosRoot, scenarioEntry.name, "ui", "src");
    for (const path of walk(sourceRoot, (file) => SOURCE_EXTENSIONS.has(extname(file)))) {
      const source = read(path);
      const pattern = /@vrooli\/react-component-library\/([A-Za-z0-9._-]+)(?:\/([0-9]+(?:\.[0-9]+){0,2}))?/g;
      for (const match of source.matchAll(pattern)) {
        imports.push({
          scenario: scenarioEntry.name,
          asset: match[1],
          version: match[2] ?? "bare",
          specifier: match[0],
          file: repoPath(repoRoot, path),
          line: lineAt(source, match.index ?? 0),
        });
      }
    }
  }
  return imports.sort((a, b) => a.scenario.localeCompare(b.scenario) || a.asset.localeCompare(b.asset) || a.file.localeCompare(b.file) || a.line - b.line);
}

function scanAssets(repoRoot) {
  const libraryRoot = join(repoRoot, "scenarios", "react-component-library", "library");
  const assets = [];
  const versions = [];
  for (const kind of LIBRARY_KINDS) {
    const kindRoot = join(libraryRoot, kind);
    if (!existsSync(kindRoot)) continue;
    for (const entry of readdirSync(kindRoot, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
      if (!entry.isDirectory()) continue;
      const manifestPath = join(kindRoot, entry.name, "component.json");
      if (!existsSync(manifestPath)) continue;
      const manifest = readJSON(manifestPath);
      assets.push({
        name: entry.name,
        kind,
        manifest: repoPath(repoRoot, manifestPath),
        latest: manifest.latest ?? "",
        draft: manifest.draft ?? "",
        deprecated_versions: [...(manifest.deprecatedVersions ?? [])].sort(),
        catalog_id: manifest.catalogId ?? "",
        description: manifest.description ?? "",
        supplemental_justification: manifest["x-supplementalJustification"] ?? "",
      });
      const versionsRoot = join(kindRoot, entry.name, "versions");
      if (!existsSync(versionsRoot)) continue;
      for (const versionEntry of readdirSync(versionsRoot, { withFileTypes: true }).sort((a, b) => a.name.localeCompare(b.name))) {
        if (!versionEntry.isDirectory()) continue;
        const versionRoot = join(versionsRoot, versionEntry.name);
        const sourceFiles = walk(versionRoot, (file) => SOURCE_EXTENSIONS.has(extname(file)) && !/(?:^|\/)(?:story|.*\.test|.*\.spec|styles)\.[^.]+$/.test(file));
        const fileReports = [];
        for (const path of sourceFiles) {
          const hits = extractUtilityClasses(read(path));
          if (hits.length) fileReports.push({ file: repoPath(repoRoot, path), hits });
        }
        const allFiles = walk(versionRoot, (file) => SOURCE_EXTENSIONS.has(extname(file)));
        let generation = "none";
        if (fileReports.length) generation = "tailwind";
        else if (allFiles.some((file) => /styles\.(?:ts|tsx|js|jsx)$/.test(file)) || allFiles.some((file) => read(file).includes("StyleSheet"))) generation = "stylesheet";
        else if (allFiles.some((file) => /style\s*=\s*\{/.test(read(file)))) generation = "inline";
        versions.push({ asset: entry.name, kind, version: versionEntry.name, generation, files: fileReports });
      }
    }
  }
  return {
    assets: assets.sort((a, b) => a.kind.localeCompare(b.kind) || a.name.localeCompare(b.name)),
    versions: versions.sort((a, b) => a.asset.localeCompare(b.asset) || a.version.localeCompare(b.version, undefined, { numeric: true })),
  };
}

function scanCatalog(repoRoot) {
  const catalogRoot = join(repoRoot, "scenarios", "react-component-library", "catalog", "assets");
  return walk(catalogRoot, (file) => extname(file) === ".json").map((path) => {
    const document = readJSON(path);
    const asset = document.asset ?? {};
    const target = asset.target ?? {};
    return {
      id: asset.id ?? "",
      name: asset.name ?? "",
      domain: asset.domain ?? "",
      priority: target.priority ?? "",
      maturity: target.maturity ?? "",
      required_capabilities: [...(document.requiredCapabilities ?? [])].sort(),
      file: repoPath(repoRoot, path),
    };
  }).sort((a, b) => a.id.localeCompare(b.id));
}

function cssEscapeClass(token) {
  return token.replace(/([^A-Za-z0-9_-])/g, "\\$1");
}

function cssContainsClass(css, token) {
  const escapedClass = cssEscapeClass(token).replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`\\.${escapedClass}(?=[\\s,{:]|$)`).test(css);
}

const DRAWER_SHELL_LAYOUT_CLASSES = [
  "md:inset-x-8", "md:top-8", "md:bottom-8", "md:inset-x-auto", "md:left-1/2",
  "md:top-1/2", "md:-translate-x-1/2", "md:-translate-y-1/2", "md:w-full",
  "md:max-w-md", "md:max-h-[80vh]", "md:bottom-auto", "md:rounded-2xl", "md:border",
  "rounded-t-[20px]", "top-[max(1rem,var(--wc-safe-top,0px))]",
  "bottom-[var(--wc-kb-height,0px)]", "touch-target", "z-wc-drawer", "bg-wc-backdrop",
  "shadow-2xl",
];

function customProperties(cssFiles) {
  const declarations = new Map();
  for (const path of cssFiles) {
    const source = read(path);
    const pattern = /(--[A-Za-z0-9_-]+)\s*:\s*([^;}]+)/g;
    for (const match of source.matchAll(pattern)) declarations.set(match[1], match[2].trim());
  }
  return declarations;
}

function tokenRefs(source) {
  // A custom property with a fallback is optional. Nested fallback references
  // are still discovered independently when they have no fallback of their own.
  return [...new Set([...source.matchAll(/var\(\s*(--[A-Za-z0-9_-]+)\s*\)/g)].map((match) => match[1]))].sort();
}

function runtimeWrittenProperties(source) {
  return new Set([...source.matchAll(/\.setProperty\s*\(\s*["'](--[A-Za-z0-9_-]+)["']/g)].map((match) => match[1]));
}

function resolveModule(path) {
  const candidates = [path, `${path}.js`, `${path}.mjs`, join(path, "index.js")];
  return candidates.find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
}

function exportEntry(packageRoot, asset, version) {
  const suffix = version === "bare" ? join("exports", asset) : join("exports", asset, version);
  return resolveModule(join(packageRoot, "dist", suffix));
}

function runtimeClosure(packageRoot, roots) {
  const queue = roots.filter(Boolean);
  const visited = new Set();
  const importPattern = /(?:from\s*|import\s*\()(["'])([^"']+)\1/g;
  while (queue.length) {
    const current = queue.shift();
    if (!current || visited.has(current) || !existsSync(current)) continue;
    visited.add(current);
    const source = read(current);
    for (const match of source.matchAll(importPattern)) {
      const specifier = match[2];
      let next;
      if (specifier.startsWith(".")) next = resolveModule(resolve(dirname(current), specifier));
      else {
        const packageMatch = specifier.match(/^@vrooli\/react-component-library\/([^/]+)(?:\/(.+))?$/);
        if (packageMatch) next = exportEntry(packageRoot, packageMatch[1], packageMatch[2] ?? "bare");
      }
      if (next && next.startsWith(packageRoot)) queue.push(next);
    }
  }
  return [...visited].sort();
}

function hashTree(root) {
  if (!existsSync(root)) return "";
  const hash = createHash("sha256");
  for (const path of walk(root, (file) => [".js", ".json"].includes(extname(file)))) {
    hash.update(repoPath(root, path));
    hash.update(readFileSync(path));
  }
  return hash.digest("hex");
}

function scanConsumers(repoRoot, imports, assetsByName, versionsByKey) {
  const byScenario = new Map();
  for (const item of imports) {
    const items = byScenario.get(item.scenario) ?? [];
    items.push(item);
    byScenario.set(item.scenario, items);
  }
  const consumers = [];
  for (const [scenario, scenarioImports] of [...byScenario.entries()].sort(([a], [b]) => a.localeCompare(b))) {
    const uiRoot = join(repoRoot, "scenarios", scenario, "ui");
    const cssFiles = walk(join(uiRoot, "src"), (file) => extname(file) === ".css");
    const builtCssFiles = walk(join(uiRoot, "dist", "assets"), (file) => extname(file) === ".css");
    const builtCss = builtCssFiles.map(read).join("\n");
    const declarations = customProperties(cssFiles);
	const consumerSourceFiles = walk(join(uiRoot, "src"), (file) => SOURCE_EXTENSIONS.has(extname(file)));
	const runtimeWrittenTokens = runtimeWrittenProperties(consumerSourceFiles.map(read).join("\n"));
    const transitiveGaps = [];
    for (const [property, value] of declarations) {
      for (const dependency of tokenRefs(value)) {
        if (!declarations.has(dependency)) transitiveGaps.push({ property, missing: dependency });
      }
    }
    const packageRoot = join(uiRoot, "node_modules", "@vrooli", "react-component-library");
    const closure = runtimeClosure(packageRoot, scenarioImports.map((item) => exportEntry(packageRoot, item.asset, item.version)));
    const requiredTokens = [...new Set(closure.flatMap((path) => tokenRefs(read(path))))].sort();
    // Runtime modules can own scoped or foundation-level custom properties in
    // their emitted stylesheet strings. Those declarations travel with the
    // component and therefore do not belong in the consumer token ramp.
    const runtimeDeclarations = customProperties(closure);
    const unsatisfiedTokens = requiredTokens.filter((token) => !declarations.has(token) && !runtimeDeclarations.has(token) && !runtimeWrittenTokens.has(token));
    const purge = [];
    if (builtCssFiles.length) {
      const distinctPins = new Map();
      for (const item of scenarioImports) {
        const manifest = assetsByName.get(item.asset);
        const version = item.version === "bare" ? manifest?.latest : item.version;
        if (version) distinctPins.set(`${item.asset}@${version}`, { asset: item.asset, version });
      }
      for (const pin of [...distinctPins.values()].sort((a, b) => a.asset.localeCompare(b.asset) || a.version.localeCompare(b.version))) {
        const report = versionsByKey.get(`${pin.asset}@${pin.version}`);
        if (report?.generation !== "tailwind") continue;
        const classes = [...new Set(report.files.flatMap((file) => file.hits.map((hit) => hit.class)))].sort();
        const purged = classes.filter((className) => !cssContainsClass(builtCss, className));
        purge.push({ ...pin, class_count: classes.length, present_count: classes.length - purged.length, purged_count: purged.length, purged_classes: purged });
      }
    }
    const tailwindConfig = ["tailwind.config.ts", "tailwind.config.js", "tailwind.config.cjs", "tailwind.config.mjs"].map((name) => join(uiRoot, name)).find(existsSync);
    const tailwindCovered = Boolean(tailwindConfig && /node_modules\/@vrooli\/react-component-library\/(?:dist|library)/.test(read(tailwindConfig)));
	const sourceText = consumerSourceFiles.map(read).join("\n");
    const closureMountsBaseStyles = closure.some((path) => path.includes("/BaseStyles/"));
    const newestBuildMtime = builtCssFiles.length ? Math.max(...builtCssFiles.map((path) => statSync(path).mtimeMs)) : 0;
    const newestSourceMtime = cssFiles.length ? Math.max(...cssFiles.map((path) => statSync(path).mtimeMs)) : 0;
    consumers.push({
      scenario,
      import_sites: scenarioImports.length,
      imported_assets: [...new Set(scenarioImports.map((item) => item.asset))].sort(),
      tailwind_config: tailwindConfig ? repoPath(repoRoot, tailwindConfig) : "",
      tailwind_covers_package: tailwindCovered,
      mounts_base_styles: /react-component-library\/BaseStyles(?:\/|["'])/.test(sourceText) || closureMountsBaseStyles,
      built_css_files: builtCssFiles.map((path) => repoPath(repoRoot, path)),
      build_mtime: newestBuildMtime ? new Date(newestBuildMtime).toISOString() : "",
      source_mtime: newestSourceMtime ? new Date(newestSourceMtime).toISOString() : "",
      excluded_reason: builtCssFiles.length ? "" : "compiled CSS bundle not found",
      installed_package_hash: hashTree(join(packageRoot, "dist")),
      runtime_modules: closure.length,
      required_tokens: requiredTokens,
      unsatisfied_tokens: unsatisfiedTokens,
      transitive_token_gaps: transitiveGaps.sort((a, b) => a.property.localeCompare(b.property) || a.missing.localeCompare(b.missing)),
      purge,
    });
  }
  return consumers;
}

function countTailwindConfigs(repoRoot) {
  const scenariosRoot = join(repoRoot, "scenarios");
  if (!existsSync(scenariosRoot)) return 0;
  let count = 0;
  for (const entry of readdirSync(scenariosRoot, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const uiRoot = join(scenariosRoot, entry.name, "ui");
    count += ["tailwind.config.ts", "tailwind.config.js", "tailwind.config.cjs", "tailwind.config.mjs"]
      .filter((name) => existsSync(join(uiRoot, name))).length;
  }
  return count;
}

export function buildCensus(repoRoot, generatedAt = new Date().toISOString()) {
  const imports = scanImports(repoRoot);
  const { assets, versions } = scanAssets(repoRoot);
  const catalog = scanCatalog(repoRoot);
  const assetsByName = new Map(assets.map((asset) => [asset.name, asset]));
  const versionsByKey = new Map(versions.map((version) => [`${version.asset}@${version.version}`, version]));
  const consumers = scanConsumers(repoRoot, imports, assetsByName, versionsByKey);
  const catalogImplementations = new Set(assets.map((asset) => asset.catalog_id).filter(Boolean));
  const missingCatalog = catalog.filter((asset) => !catalogImplementations.has(asset.id));
  const overlayMissingCatalog = missingCatalog.filter((asset) => asset.domain === "overlays");
  const selfReferential = assets.filter((asset) => asset.catalog_id === `react-component-library:${asset.name}`);
  const supplemental = assets.filter((asset) => asset.supplemental_justification);
  const emptyOverlayDescriptions = assets.filter((asset) => asset.kind === "components" && (asset.catalog_id.startsWith("overlays.") || asset.catalog_id === `react-component-library:${asset.name}`) && !asset.description.trim());
  let deprecatedPins = 0;
  let staleMajorPins = 0;
  for (const item of imports) {
    const manifest = assetsByName.get(item.asset);
    if (!manifest || item.version === "bare") continue;
    if (manifest.deprecated_versions.includes(item.version)) deprecatedPins += 1;
    if (Number.parseInt(item.version, 10) < Number.parseInt(manifest.latest, 10)) staleMajorPins += 1;
  }
  const tailwindVersions = versions.filter((version) => version.generation === "tailwind");
  const libraryFiles = new Set(tailwindVersions.flatMap((version) => version.files.map((file) => file.file)));
  const purgedClasses = consumers.reduce((total, consumer) => total + consumer.purge.reduce((sum, pin) => sum + pin.purged_count, 0), 0);
  const webConsole = consumers.find((consumer) => consumer.scenario === "web-console");
  const webConsoleCss = webConsole?.built_css_files.map((path) => read(join(repoRoot, path))).join("\n") ?? "";
  const webConsoleDrawerPurged = DRAWER_SHELL_LAYOUT_CLASSES.filter((className) => !cssContainsClass(webConsoleCss, className));
  const webConsoleTransitiveMissing = new Set(webConsole?.transitive_token_gaps.map((gap) => gap.missing) ?? []);
  return {
    generated_at: generatedAt,
    schema_version: 1,
    detector: "local",
    summary: {
      total_tailwind_configs: countTailwindConfigs(repoRoot),
      scenarios_importing_rcl: consumers.length,
      import_sites: imports.length,
      tailwind_configs_covering_package: consumers.filter((consumer) => consumer.tailwind_covers_package).length,
      scenarios_mounting_basestyles: consumers.filter((consumer) => consumer.mounts_base_styles).length,
      library_files_emitting_utility_classes: libraryFiles.size,
      library_versions_emitting_utility_classes: tailwindVersions.length,
      purged_classes_total: purgedClasses,
      catalog_assets_without_implementation: missingCatalog.length,
      overlay_catalog_assets_without_implementation: overlayMissingCatalog.length,
      assets_with_supplemental_justification: supplemental.length,
      self_referential_catalog_ids: selfReferential.length,
      overlay_manifests_with_empty_description: emptyOverlayDescriptions.length,
      deprecated_pins: deprecatedPins,
      stale_major_pins: staleMajorPins,
      transitive_token_gaps: consumers.reduce((total, consumer) => total + consumer.transitive_token_gaps.length, 0),
      unsatisfied_tokens: consumers.reduce((total, consumer) => total + consumer.unsatisfied_tokens.length, 0),
      web_console_drawer_shell_layout_classes: DRAWER_SHELL_LAYOUT_CLASSES.length,
      web_console_drawer_shell_layout_classes_purged: webConsoleDrawerPurged.length,
      web_console_drawer_shell_layout_purged_classes: webConsoleDrawerPurged,
      web_console_transitive_token_gaps: webConsoleTransitiveMissing.size,
      web_console_transitive_token_gap_properties: [...webConsoleTransitiveMissing].sort(),
    },
    imports,
    assets,
    versions,
    catalog,
    catalog_assets_without_implementation: missingCatalog.map((asset) => asset.id),
    overlay_catalog_assets_without_implementation: overlayMissingCatalog.map((asset) => asset.id),
    consumers,
  };
}

function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    process.stdout.write("Usage: node overlay-census.mjs [--root REPO] [--output FILE] [--generated-at ISO]\n");
    return;
  }
  const census = buildCensus(options.root, options.generatedAt);
  const payload = `${JSON.stringify(census, null, 2)}\n`;
  if (options.output) writeFileSync(options.output, payload);
  else process.stdout.write(payload);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) main();
