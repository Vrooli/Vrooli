/* Actual input-hook logic with synthetic React hooks and WebSocket sink.
 * No renderer, browser, network connection or product write is started.
 * Run with node from any directory. A false result is a behavioral mismatch.
 */
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { createRequire } = require('node:module');
const scenario = path.resolve(__dirname, '../..');
const req = createRequire(path.join(scenario, 'playwright-driver/package.json'));
const ts = req('typescript');
const messages = [];
function evaluate(relativePath, mocks = {}) {
  const filename = path.join(scenario, relativePath);
  const code = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true }, fileName: filename,
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(code, {
    module, exports: module.exports,
    require(name) { if (!Object.hasOwn(mocks, name)) throw new Error(`Unexpected dependency: ${name}`); return mocks[name]; },
    console, performance,
  }, { filename });
  return module.exports;
}
const coordinates = evaluate('ui/src/domains/recording/utils/coordinateMapping.ts');
const { useInputForwarding } = evaluate('ui/src/domains/recording/capture/useInputForwarding.ts', {
  react: { useCallback: fn => fn, useRef: initial => ({ current: initial }) },
  '@/config': { getConfig: async () => { throw new Error('Unexpected HTTP fallback'); } },
  '@/contexts/WebSocketContext': { useWebSocket: () => ({ isConnected: true, send: message => messages.push(message) }) },
  '../utils/coordinateMapping': coordinates,
});
const hook = useInputForwarding({ sessionId: 'synthetic-session', pageId: 'synthetic-page', viewport: { width: 800, height: 600 }, frameDimensions: { width: 800, height: 600 } });
hook.setWsSubscribed(true);
const observations = [];
const event = overrides => ({ key: 'a', altKey: false, ctrlKey: false, metaKey: false, shiftKey: false, preventDefault() {}, stopPropagation() {}, ...overrides });
for (const [name, input, modifier] of [
  ['control-a', { key: 'a', ctrlKey: true }, 'Control'],
  ['command-c', { key: 'c', metaKey: true }, 'Meta'],
  ['alt-f', { key: 'f', altKey: true }, 'Alt'],
]) {
  hook.handleKey(event(input), true);
  const actual = messages.at(-1);
  observations.push({ id: name, expected: 'Printable shortcut retains key and modifier', actual, expected_behavior_met: actual.input.key === input.key && actual.input.modifiers?.includes(modifier) === true });
}
hook.handleKey(event({ key: 'x' }), true);
observations.push({ id: 'plain-text-control', expected: 'Ordinary printable text remains x', actual: messages.at(-1), expected_behavior_met: messages.at(-1).input.text === 'x' });
hook.handleKey(event({ key: 'Tab', shiftKey: true }), true);
observations.push({ id: 'special-shortcut-control', expected: 'Shift-Tab retains its modifier', actual: messages.at(-1), expected_behavior_met: messages.at(-1).input.modifiers.includes('Shift') });
hook.handlePointer('down', event({ clientX: 100, clientY: 200, button: 0, shiftKey: true }), { left: 0, top: 0, width: 800, height: 600 }, true);
observations.push({ id: 'shift-pointer-down', expected: 'Modified pointer input retains modifier state', actual: messages.at(-1), expected_behavior_met: messages.at(-1).input.modifiers?.includes('Shift') === true });
const beforeComposition = messages.length;
hook.handleKey(event({ key: 'Process', isComposing: true }), true);
observations.push({ id: 'composition-keydown', expected: 'Composition bookkeeping is not forwarded as a standalone browser key command', actual: messages.slice(beforeComposition), expected_behavior_met: messages.length === beforeComposition });
console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(), scope: 'actual hook/coordinate logic; synthetic React hooks and WebSocket sink; OS/browser effects not tested', results: observations }, null, 2));
