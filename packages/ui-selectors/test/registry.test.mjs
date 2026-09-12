import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createSelectorRegistry, defineDynamicSelector, resolveSelector, scopeSelector } from '../src/index.js';
const fixture = JSON.parse(readFileSync(new URL('../conformance.json',import.meta.url)));
for (const row of fixture.cases) test(row.name, () => {
  if (row.error) assert.throws(() => resolveSelector(fixture.manifest,row.key,row.params));
  else assert.equal(resolveSelector(fixture.manifest,row.key,row.params),row.want);
});
test('reject ambiguous definitions', () => {
  assert.throws(() => createSelectorRegistry({item:'item'},{item:defineDynamicSelector({description:'item',testIdPattern:'other'})}), /conflicting/);
  assert.throws(() => defineDynamicSelector({testIdPattern:'${missing}',params:{}}), /tokens/);
  assert.throws(() => defineDynamicSelector({testIdPattern:'a',selectorPattern:'#a'}), /exactly one/);
});
test('runtime IDs and browser locators agree for arbitrary IDs', () => {
  const registry=createSelectorRegistry({}, {item:defineDynamicSelector({description:'item',testIdPattern:'item-${name}',params:{name:{type:'string'}}})});
  assert.equal(registry.selectors.item({name:'a"b'}),'item-a"b');
  assert.equal(resolveSelector(registry.manifest,'item',{name:'a"b'}),'[data-testid="item-a\\22 b"]');
});
test('scope keeps comma branches inside their parent', () => {
  assert.equal(scopeSelector('#left, #right','.save'),':is(#left, #right) :is(.save)');
});
test('nested application and library definitions share one manifest', () => {
  const registry=createSelectorRegistry({app:{save:'save'}},{app:{row:defineDynamicSelector({description:'row',testIdPattern:'row-${id}',params:{id:{type:'number'}}})}},{dialog:{close:'close'}});
  assert.equal(registry.selectors.app.save,'save');
  assert.equal(registry.selectors.app.row({id:3}),'row-3');
  assert.equal(registry.manifest.selectors['library.dialog.close'].selector,'[data-testid="close"]');
  assert.deepEqual(registry.manifest.dynamicSelectors['app.row'].params,[{name:'id',type:'number',values:undefined}]);
});
test('library composition preserves application namespaces and rejects collisions', () => {
  const registry = createSelectorRegistry({library:{upload:'upload'}},{},{dialog:{close:'close'}});
  assert.equal(registry.selectors.library.upload,'upload');
  assert.equal(registry.selectors.library.dialog.close,'close');
  assert.throws(() => createSelectorRegistry({library:{upload:'upload'}},{},{upload:'other'}),/Conflicting/);
  assert.throws(() => createSelectorRegistry({'a.b':'one',a:{b:'two'}},{}),/Duplicate/);
});
