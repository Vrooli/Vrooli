const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');

const sourceDir = path.join(__dirname, '..', 'src');
const runtimeSource = fs.readFileSync(path.join(sourceDir, 'api-runtime.js'), 'utf8');

function loadRuntime(overrides = {}) {
    const listeners = new Map();
    const window = {
        location: { protocol: 'http:', hostname: 'localhost', origin: 'http://localhost' },
        addEventListener: (name, listener) => listeners.set(name, listener),
        ...overrides,
    };
    const context = {
        window,
        document: { querySelectorAll: () => [] },
        fetch: async () => ({ ok: true, status: 200, text: async () => '' }),
    };
    vm.runInNewContext(runtimeSource, context, { filename: 'api-runtime.js' });
    return { runtime: window.NotesRuntime, window, listeners, context };
}

test('API base preserves proxy, URL, port, origin, and fallback precedence', () => {
    assert.equal(loadRuntime({
        __APP_MONITOR_PROXY_INFO__: { apiBase: '/proxy/notes/' },
        API_URL: 'https://env.example/v2',
        API_PORT: '9100',
    }).runtime.API_BASE, 'http://localhost/proxy/notes/api');

    assert.equal(loadRuntime({ API_URL: 'https://env.example/v2/' }).runtime.API_BASE, 'https://env.example/v2/api');
    assert.equal(loadRuntime({ API_PORT: 9100 }).runtime.API_BASE, 'http://localhost:9100/api');
    assert.equal(loadRuntime().runtime.API_BASE, 'http://localhost/api');
    assert.equal(loadRuntime({ location: undefined, API_PORT: 'bad' }).runtime.API_BASE, '/api');
});

test('API request builder preserves JSON headers, overrides, and optional body', () => {
    const runtime = loadRuntime().runtime;
    assert.deepEqual(JSON.parse(JSON.stringify(runtime.buildJsonRequest('POST', { title: 'a' }, { 'X-Trace': '1' }))), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Trace': '1' },
        body: '{"title":"a"}',
    });
    assert.deepEqual(JSON.parse(JSON.stringify(runtime.buildJsonRequest('DELETE'))), {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
    });
});

test('API JSON client joins the base and preserves success, empty, and error behavior', async () => {
    const seen = [];
    const { runtime, context } = loadRuntime({ API_URL: 'https://api.example/root' });
    context.fetch = async (url, options) => {
        seen.push([url, options]);
        if (url.endsWith('/empty')) return { ok: true, status: 204, text: async () => '' };
        if (url.endsWith('/failure')) return { ok: false, status: 503, text: async () => 'offline' };
        if (url.endsWith('/invalid')) return { ok: true, status: 200, text: async () => '{bad' };
        return { ok: true, status: 200, text: async () => '{"ok":true}' };
    };

    assert.equal((await runtime.apiFetchJson('notes')).ok, true);
    assert.equal(await runtime.apiFetchJson('/empty'), null);
    await assert.rejects(runtime.apiFetchJson('/failure'), /offline/);
    await assert.rejects(runtime.apiFetchJson('/invalid'), /Failed to parse response JSON/);
    assert.equal(seen[0][0], 'https://api.example/root/api/notes');
    assert.deepEqual(JSON.parse(JSON.stringify(seen[0][1])), {});
});

test('both standalone pages load the shared runtime before their own script', () => {
    for (const [htmlName, appScript] of [['index.html', 'script.js'], ['zen-index.html', 'zen-script.js']]) {
        const html = fs.readFileSync(path.join(sourceDir, htmlName), 'utf8');
        assert.ok(html.indexOf('api-runtime.js') >= 0, `${htmlName} loads the shared runtime`);
        assert.ok(html.indexOf('api-runtime.js') < html.indexOf(appScript), `${htmlName} loads runtime first`);
    }
});

test('shared icon renderer retains fallback and loaded-icon behavior', () => {
    const calls = [];
    const { runtime, window, listeners } = loadRuntime();
    window.lucide = {
        icons: { Check: {} },
        createElement: (_icon, options) => ({ setAttribute: (...args) => calls.push(['attr', ...args]), options }),
    };
    const element = {
        dataset: { icon: 'Check' },
        textContent: '',
        childElementCount: 0,
        replaceChildren: (node) => calls.push(['child', node]),
    };
    runtime.applyIcon(element, 'Check');
    assert.equal(element.textContent, '');
    listeners.get('load')();
    runtime.applyIcon(element, 'Check');
    assert.equal(calls[0][0], 'attr');
    assert.deepEqual(calls[0].slice(1), ['aria-hidden', 'true']);
    assert.deepEqual(JSON.parse(JSON.stringify(calls[1][1].options)), { name: 'Check', size: 20, strokeWidth: 1.8 });
});
