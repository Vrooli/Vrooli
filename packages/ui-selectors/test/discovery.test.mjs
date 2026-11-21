import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { selectorFiles } from '../src/export.mjs';

test('discovery respects declared paths and rejects escaping overrides', () => {
 const root=mkdtempSync(join(tmpdir(),'selector-layout-'));
 const write=(path,data)=>{mkdirSync(join(root,path,'..'),{recursive:true});writeFileSync(join(root,path),JSON.stringify(data))};
 try {
  write('ui/manifest.json',{files:{selectorRegistry:{path:'ui/feature/selectors.tsx'}}});
  assert.equal(selectorFiles(root).manifest,join(root,'ui/feature/selectors.manifest.json'));
  write('.vrooli/ui-manifest.json',{files:{selectorRegistry:{path:'ui/moved/selectors.ts'}}});
  assert.equal(selectorFiles(root).source,join(root,'ui/moved/selectors.ts'));
  write('.vrooli/ui-manifest.json',{files:{selectorRegistry:{path:'../another-scenario/selectors.ts'}}});
  assert.throws(()=>selectorFiles(root),/escapes/);
 } finally {rmSync(root,{recursive:true,force:true})}
});
