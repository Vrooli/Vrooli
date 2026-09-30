import { readFileSync, existsSync, mkdirSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { resolve, dirname, join, extname, relative, isAbsolute } from 'node:path';
import { createRequire } from 'node:module';
import { pathToFileURL } from 'node:url';
import { createHash } from 'node:crypto';

// Same declared file contract as api-core/uimanifest.LoadAt. No ambient target
// fallback: an absent target manifest cannot accidentally select BAS's own UI.
export function selectorFiles(root) {
  root = resolve(root);
  let base = join(root, 'ui/manifest.json');
  if (!existsSync(base)) {
    const service = join(root, '.vrooli/service.json');
    const id = existsSync(service) ? JSON.parse(readFileSync(service, 'utf8')).generation?.template?.id : undefined;
    if (id) for (let dir = root; dirname(dir) !== dir; dir = dirname(dir)) {
      const candidate = join(dir, 'templates/scenarios', id, 'ui/manifest.json');
      if (existsSync(candidate)) { base = candidate; break; }
    }
  }
  const files = existsSync(base) ? (JSON.parse(readFileSync(base, 'utf8')).files ?? {}) : {};
  files.selectorRegistry ??= { path: existsSync(join(root, 'ui/src/constants/selectors.ts')) ? 'ui/src/constants/selectors.ts' : 'ui/src/consts/selectors.ts' };
  files.librarySelectors ??= { path: 'ui/src/consts/selectors.library.ts' };
  files.appEntry ??= { path: 'ui/src/main.tsx' };
  const overlay = join(root, '.vrooli/ui-manifest.json');
  if (existsSync(overlay)) for (const [key, value] of Object.entries(JSON.parse(readFileSync(overlay, 'utf8')).files ?? {})) {
    if (!Object.hasOwn(files, key)) throw new Error(`Undeclared UI file '${key}'`);
    files[key] = value;
  }
  for (const [key, file] of Object.entries(files)) {
    const rel = relative(root, resolve(root, file.path));
    if (!file.path || rel === '' || isAbsolute(file.path) || rel === '..' || rel.startsWith('../')) throw new Error(`UI file '${key}' escapes scenario root`);
  }
  const source = resolve(root, files.selectorRegistry.path);
  return { source, manifest: source.replace(/\.tsx?$/, '.manifest.json') };
}

export async function exportSelectorManifest({ root = resolve(process.cwd(), '..'), check = false } = {}) {
  const { source, manifest } = selectorFiles(root);
  const ui = resolve(root, 'ui');
  const require = createRequire(join(ui, 'package.json'));
  const ts = require('typescript');
  // Keep compiled modules within the consumer so bare package imports resolve
  // against its governed dependencies, including ESM-only shared packages.
  const cache = join(ui, 'node_modules/.cache');
  mkdirSync(cache, { recursive: true });
  const temp = mkdtempSync(join(cache, 'ui-selectors-'));
  const compiled = new Map();
  const sources = {};
  function compile(path) {
    if (compiled.has(path)) return compiled.get(path);
    const target = join(temp, `${compiled.size}.mjs`);
    compiled.set(path, target);
    const text = readFileSync(path, 'utf8');
    sources[relative(root, path).split('\\').join('/')] = createHash('sha256').update(text).digest('hex');
    const result = ts.transpileModule(text, {
      compilerOptions: { module: ts.ModuleKind.ES2022, target: ts.ScriptTarget.ES2022 },
      transformers: { before: [context => node => {
        const visit = child => {
          if ((ts.isImportDeclaration(child) && child.importClause?.isTypeOnly) || (ts.isExportDeclaration(child) && child.isTypeOnly)) return child;
          if ((ts.isImportDeclaration(child) || ts.isExportDeclaration(child)) && child.moduleSpecifier && ts.isStringLiteral(child.moduleSpecifier)) {
            const spec = child.moduleSpecifier.text;
            if (spec.startsWith('.')) {
              const base = resolve(dirname(path), spec);
              const candidates = [base, `${base}.ts`, `${base}.tsx`, join(base,'index.ts'), base.replace(/\.js$/,'.ts')];
              const local = candidates.find(p => existsSync(p) && /\.(ts|tsx|js|mjs)$/.test(p));
              if (!local) throw new Error(`Cannot resolve selector import '${spec}' from ${path}`);
              const literal = ts.factory.createStringLiteral(pathToFileURL(compile(local)).href);
              return ts.isImportDeclaration(child)
                ? ts.factory.updateImportDeclaration(child, child.modifiers, child.importClause, literal, child.attributes)
                : ts.factory.updateExportDeclaration(child, child.modifiers, child.isTypeOnly, child.exportClause, literal, child.attributes);
            }
          }
          return ts.visitEachChild(child, visit, context);
        };
        return ts.visitNode(node, visit);
      }] },
    });
    writeFileSync(target, result.outputText);
    return target;
  }
  try {
    const mod = await import(pathToFileURL(compile(source)).href);
    if (!mod.selectorsManifest?.selectors) throw new Error(`${source} does not export a selector manifest`);
    const output = `${JSON.stringify({ ...mod.selectorsManifest, sources: Object.fromEntries(Object.entries(sources).sort(([a],[b]) => a.localeCompare(b))) }, null, 2)}\n`;
    if (check) {
      if (!existsSync(manifest) || readFileSync(manifest,'utf8') !== output) throw new Error(`Stale selector manifest: ${manifest}; run selector:manifest`);
    } else writeFileSync(manifest, output);
    return manifest;
  } finally { rmSync(temp, { recursive: true, force: true }); }
}
