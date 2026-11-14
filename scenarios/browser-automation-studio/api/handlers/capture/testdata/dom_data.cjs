// Execute the production serializer with browser-shaped nodes, without a driver.
const assert = require('node:assert/strict');
const vm = require('node:vm');
const expression = require('node:fs').readFileSync(0, 'utf8');
const styles = {display: 'block', color: 'rgb(1, 2, 3)', backgroundColor: 'rgb(4, 5, 6)'};
const rect = {x: 1, y: 2, width: 30, height: 40};
function element(dataset = {}) {
  return {
    nodeType: 1, tagName: 'DIV', dataset, children: [],
    id: 'example', className: 'surface', innerText: 'Example',
    clientWidth: 30, clientHeight: 40, scrollWidth: 30, scrollHeight: 40,
    getAttribute: () => null, hasAttribute: () => false,
    getBoundingClientRect: () => rect,
  };
}
function read(el) {
  return JSON.parse(JSON.stringify(vm.runInNewContext(expression, {
    document: {body: el}, Node: {ELEMENT_NODE: 1},
    getComputedStyle: () => styles, window: {innerHeight: 800},
  })));
}
const plain = read(element());
assert.equal(Object.hasOwn(plain, 'data'), false);
const values = {rclAsset: 'react-component-library:Button', rclVersion: '2.2.10', testid: 'save', empty: '', '__proto__': 'ignored literal'};
Object.defineProperty(values, '__proto__', {value: 'safe data', enumerable: true});
const stamped = read(element(values));
assert.deepEqual(stamped.data, values);
const {data, ...existing} = stamped;
assert.deepEqual(existing, plain, 'Adding data preserves every existing field');
assert.deepEqual(stamped.rect, rect);
assert.equal(stamped.computed.color, styles.color);
const excessive = Object.fromEntries(Array.from({length: 50}, (_, i) => ['key' + i, 'x'.repeat(2000)]));
const bounded = read(element(excessive));
assert.equal(Object.keys(bounded.data).length, 32);
assert.equal(bounded.data.key0.length, 1024);
assert.equal(bounded.id, 'example', 'Oversized attributes do not drop the node');
const root = element();
root.children = [element(values)];
root.children[0].parentElement = root;
assert.deepEqual(read(root).children[0].data, values);
root.children = Array.from({length: 24}, () => element(values));
for (const child of root.children) child.parentElement = root;
assert.equal(read(root).children.length, 24, 'Wide regions retain every sibling within the total budget');
root.children = Array.from({length: 4100}, () => element(values));
const capped = read(root);
assert.equal(capped.children.length, 3999, 'The total budget includes the root');
assert.equal(capped.truncated, true);
console.log('DOM data: presence, absence, nesting, prototype keys, caps, and existing fields passed');
