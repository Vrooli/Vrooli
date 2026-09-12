// Uses the consumer's resolver so major aliases and exact pins cannot drift.
import { readFile, readdir } from 'node:fs/promises';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
const repo = process.argv[2];
const root = join(repo, 'scenarios/react-component-library/library');
const { resolveLibrarySpecifier } = await import(pathToFileURL(join(repo, 'packages/react-component-library/tooling/resolve-specifier.mjs')));
let input = '';
for await (const chunk of process.stdin) input += chunk;
const requests = JSON.parse(input);
const assets = new Map();
for (const kind of await readdir(root, { withFileTypes: true })) {
  if (!kind.isDirectory()) continue;
  for (const entry of await readdir(join(root, kind.name), { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    let manifest;
    try { manifest = JSON.parse(await readFile(join(root, kind.name, entry.name, 'component.json'), 'utf8')); }
    catch (error) { if (error.code === 'ENOENT') continue; throw error; }
    for (const id of [manifest.libraryId].filter(Boolean)) {
      const previous = assets.get(id);
      assets.set(id, previous && previous !== entry.name ? null : entry.name);
    }
  }
}
const results = {};
for (const { asset, version } of requests) {
  const key = `${asset}@${version}`;
  try {
    const name = assets.get(asset);
    if (!name) throw new Error('stamp has no unique catalog identity');
    if (!/^\d+(?:\.\d+\.\d+)?$/.test(version)) throw new Error('stamp needs an exact release or major alias');
    const resolution = await resolveLibrarySpecifier(`@vrooli/react-component-library/${name}/${version}`, { libraryRoot: root });
    results[key] = { status: 'resolved', stamped_asset: asset, stamped_version: version,
      library_id: resolution.libraryId, resolved_version: resolution.version, source_path: resolution.sourcePath,
      resolution_rule: version.includes('.') ? 'exact-release' : 'consumer-major-alias' };
  } catch (error) {
    results[key] = { status: 'unresolved', stamped_asset: asset, stamped_version: version, reason: error.message };
  }
}
process.stdout.write(JSON.stringify(results));
