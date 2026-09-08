import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, writeFile, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { migrateStylesheetKeys } from './codemod-stylesheet-keys.mjs';

test('migration preserves releases and refuses to collapse multiple sheets into one key', async () => {
 const root=await mkdtemp(path.join(tmpdir(),'rcl-sheet-migration-'));
 try {
  for(const [name,body] of [['Single','useLibraryStyleSheet("old", styles);'],['Multiple','useLibraryStyleSheet("base", base); useLibraryStyleSheet("tabs", tabs);']]) {
   const asset=path.join(root,'components',name);
   await mkdir(path.join(asset,'versions','1.0.0'),{recursive:true});
   await mkdir(path.join(asset,'versions','1.0.1-draft.1'),{recursive:true});
   await writeFile(path.join(asset,'component.json'),JSON.stringify({libraryId:`react-component-library:${name}`,draft:'1.0.1-draft.1'}));
   for(const version of ['1.0.0','1.0.1-draft.1'])await writeFile(path.join(asset,'versions',version,`${name}.tsx`),body);
  }
  const result=await migrateStylesheetKeys({libraryRoot:root,apply:true,log:()=>{}});
  assert.equal(result.changedFiles,1);assert.equal(result.blockedFiles,1);
  assert.equal(await readFile(path.join(root,'components/Single/versions/1.0.0/Single.tsx'),'utf8'),'useLibraryStyleSheet("old", styles);');
  assert.match(await readFile(path.join(root,'components/Single/versions/1.0.1-draft.1/Single.tsx'),'utf8'),/"react-component-library:Single", "1.0.1-draft.1"/);
  assert.equal(await readFile(path.join(root,'components/Multiple/versions/1.0.1-draft.1/Multiple.tsx'),'utf8'),'useLibraryStyleSheet("base", base); useLibraryStyleSheet("tabs", tabs);');
  const again=await migrateStylesheetKeys({libraryRoot:root,apply:true,log:()=>{}});assert.equal(again.changedFiles,0);
 } finally { await rm(root,{recursive:true,force:true}); }
});
