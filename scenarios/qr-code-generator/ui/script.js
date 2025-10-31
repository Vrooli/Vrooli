import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const APP_ID = 'qr-code-generator';
const DEFAULT_API_PATH = '/api';

setupIframeBridge();

if (typeof window !== 'undefined' && typeof document !== 'undefined') {
    window.addEventListener('pageshow', setupIframeBridge);
    document.addEventListener('visibilitychange', () => {
        if (!document.hidden) {
            setupIframeBridge();
        }
    });
}

const state = {
    apiBase: '/api',
    currentQr: null,
    statusResetHandle: null
};

const elements = {};

let iframeBridgeController = null;

document.addEventListener('DOMContentLoaded', async () => {
    cacheElements();
    bindUIEvents();
    updateColorPreview();
    updateBatchCount();
    await hydrateApiBase();
});

function setupIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window) {
        return;
    }

    if (iframeBridgeController) {
        iframeBridgeController.notify();
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[QR UI] Unable to determine parent origin for iframe bridge', error);
    }

    const logLevels = ['log', 'info', 'warn', 'error', 'debug'];

    try {
        iframeBridgeController = initIframeBridgeChild({
            parentOrigin,
            appId: APP_ID,
            captureLogs: {
                enabled: true,
                streaming: true,
                bufferSize: 400,
                levels: logLevels
            },
            captureNetwork: {
                enabled: true,
                streaming: true,
                bufferSize: 200
            }
        });

        window.__qrCodeGeneratorBridgeInitialized = true;
    } catch (error) {
        console.error('[QR UI] Failed to initialize iframe bridge', error);
        iframeBridgeController = null;
    }
}

async function hydrateApiBase() {
    const config = await fetchConfig();
    state.apiBase = resolveApiBase(config);
}

async function fetchConfig() {
    try {
        const response = await fetch('/config', { cache: 'no-store' });
        if (!response.ok) {
            throw new Error(`Config request failed with status ${response.status}`);
        }
        return await response.json();
    } catch (error) {
        console.warn('[QR UI] Falling back to default API base', error);
        return {};
    }
}

function resolveApiBase(config = {}) {
    const hints = extractConfigApiHints(config);

    if (typeof window === 'undefined') {
        if (hints.absolute) {
            return hints.absolute;
        }
        if (hints.relative) {
            return hints.relative;
        }
        return DEFAULT_API_PATH;
    }

    const proxyBase = resolveProxyBase();
    if (proxyBase) {
        if (hints.absolute) {
            return hints.absolute;
        }
        const proxyPath = hints.relative || hints.path || DEFAULT_API_PATH;
        return joinUrl(proxyBase, proxyPath);
    }

    const pathBase = resolvePathDerivedProxyBase();

    if (hints.absolute) {
        return hints.absolute;
    }

    if (hints.relative) {
        if (pathBase) {
            return joinUrl(pathBase, hints.relative);
        }

        const originFromLocation = window.location?.origin;
        if (originFromLocation) {
            return joinUrl(originFromLocation, hints.relative);
        }

        return hints.relative;
    }

    const envApiUrl = typeof window.ENV?.API_URL === 'string' ? window.ENV.API_URL.trim() : '';
    if (envApiUrl) {
        return joinUrl(envApiUrl, DEFAULT_API_PATH);
    }

    if (pathBase) {
        return joinUrl(pathBase, DEFAULT_API_PATH);
    }

    const origin = window.location?.origin;
    if (origin) {
        return joinUrl(origin, DEFAULT_API_PATH);
    }

    return DEFAULT_API_PATH;
}

function extractConfigApiHints(config = {}) {
    const apiBaseRaw = typeof config.apiBase === 'string' ? config.apiBase.trim() : '';
    const apiUrlRaw = typeof config.apiUrl === 'string' ? config.apiUrl.trim() : '';
    const apiPathRaw = typeof config.apiPath === 'string' ? config.apiPath.trim() : '';

    const absoluteCandidate = [apiBaseRaw, apiUrlRaw].find((value) => value && isAbsoluteUrl(value));
    const absolute = absoluteCandidate ? stripTrailingSlash(absoluteCandidate) : '';

    const relativeCandidate = [apiPathRaw, apiBaseRaw].find((value) => value && !isAbsoluteUrl(value));
    const relative = relativeCandidate ? ensureLeadingSlash(stripTrailingSlash(relativeCandidate)) : '';

    const defaultPath = relative || (apiPathRaw ? ensureLeadingSlash(stripTrailingSlash(apiPathRaw)) : DEFAULT_API_PATH);

    return {
        absolute,
        relative,
        path: defaultPath
    };
}

function resolveProxyBase() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const info = window.__APP_MONITOR_PROXY_INFO__ || window.__APP_MONITOR_PROXY_INDEX__;
    if (!info) {
        return undefined;
    }

    const candidate = pickProxyEndpoint(info);
    if (!candidate) {
        return undefined;
    }

    if (/^https?:\/\//i.test(candidate)) {
        return stripTrailingSlash(candidate);
    }

    const origin = window.location?.origin;
    if (!origin) {
        return undefined;
    }

    return stripTrailingSlash(joinUrl(origin, ensureLeadingSlash(candidate)));
}

function isAbsoluteUrl(value) {
    return typeof value === 'string' && /^https?:\/\//i.test(value);
}

function pickProxyEndpoint(info) {
    const candidates = [];

    const append = (value) => {
        if (!value) {
            return;
        }
        if (typeof value === 'string') {
            const trimmed = value.trim();
            if (trimmed) {
                candidates.push(trimmed);
            }
            return;
        }
        if (typeof value === 'object') {
            append(value.url);
            append(value.path);
            append(value.target);
        }
    };

    append(info);

    if (info && typeof info === 'object') {
        append(info.primary);
        if (Array.isArray(info.endpoints)) {
            info.endpoints.forEach(append);
        }
    }

    return candidates.find(Boolean);
}

function cacheElements() {
    elements.textInput = document.getElementById('qr-text');
    elements.sizeSelect = document.getElementById('qr-size');
    elements.correctionSelect = document.getElementById('qr-correction');
    elements.foregroundInput = document.getElementById('qr-color');
    elements.backgroundInput = document.getElementById('qr-bg');
    elements.foregroundLabel = document.getElementById('color-hex');
    elements.backgroundLabel = document.getElementById('bg-hex');
    elements.generateButton = document.getElementById('generate-btn');
    elements.downloadButton = document.getElementById('download-btn');
    elements.copyButton = document.getElementById('copy-btn');
    elements.qrDisplay = document.getElementById('qr-display');
    elements.actionButtons = document.getElementById('action-buttons');
    elements.status = document.getElementById('status');
    elements.batchItems = document.getElementById('batch-items');
    elements.addBatchButton = document.getElementById('add-batch-item');
    elements.processBatchButton = document.getElementById('process-batch');
    elements.batchCount = document.getElementById('batch-count');
}

function bindUIEvents() {
    elements.generateButton?.addEventListener('click', (event) => {
        event.preventDefault();
        generateQRCode();
    });

    elements.downloadButton?.addEventListener('click', (event) => {
        event.preventDefault();
        downloadQR();
    });

    elements.copyButton?.addEventListener('click', (event) => {
        event.preventDefault();
        copyText();
    });

    elements.foregroundInput?.addEventListener('change', updateColorPreview);
    elements.backgroundInput?.addEventListener('change', updateColorPreview);

    elements.addBatchButton?.addEventListener('click', (event) => {
        event.preventDefault();
        addBatchItem();
    });

    elements.processBatchButton?.addEventListener('click', (event) => {
        event.preventDefault();
        processBatch();
    });
}

function buildApiUrl(path = '/') {
    const base = stripTrailingSlash(state.apiBase || '/api');
    const normalizedPath = ensureLeadingSlash(path || '/');
    if (!base || base === '.') {
        return normalizedPath;
    }
    if (base.startsWith('http')) {
        return `${base}${normalizedPath}`;
    }
    return `${ensureLeadingSlash(base)}${normalizedPath}`;
}

async function generateQRCode() {
    const text = elements.textInput?.value.trim();
    if (!text) {
        showStatus('Error: Enter text before generating.', 'error');
        return;
    }

    const button = elements.generateButton;
    const originalLabel = button.innerHTML;
    button.innerHTML = '<span class="button-text">⏳ Processing…</span>';
    button.disabled = true;

    showStatus('Generating QR code…', 'processing');
    setPreviewBusy(true);

    const payload = {
        text,
        size: parseInt(elements.sizeSelect?.value, 10) || 256,
        color: elements.foregroundInput?.value || '#000000',
        background: elements.backgroundInput?.value || '#ffffff',
        errorCorrection: elements.correctionSelect?.value || 'M',
        format: 'png'
    };

    try {
        const response = await fetch(buildApiUrl('/generate'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const data = await response.json();

        if (!response.ok || !data?.success) {
            throw new Error(data?.error || data?.message || `Request failed with status ${response.status}`);
        }

        displayQRCode(data, text);
        showStatus('QR code generated', 'success');
    } catch (error) {
        console.error('[QR UI] Generate error', error);
        showStatus(`Error: ${error.message}`, 'error');
    } finally {
        button.innerHTML = originalLabel;
        button.disabled = false;
        setPreviewBusy(false);
    }
}

function displayQRCode(data, sourceText) {
    const base64 = extractBase64Payload(data);
    if (!base64) {
        throw new Error('Invalid QR payload');
    }

    state.currentQr = {
        base64,
        format: data?.format || 'base64',
        text: sourceText
    };

    const display = elements.qrDisplay;
    if (!display) {
        return;
    }
    display.innerHTML = '';
    display.setAttribute('aria-label', sourceText ? `QR code preview for ${sourceText}` : 'QR code preview');
    setPreviewBusy(false);

    const img = document.createElement('img');
    img.src = `data:image/png;base64,${base64}`;
    img.alt = sourceText ? `Generated QR code for ${sourceText}` : 'Generated QR code';
    img.style.maxWidth = '100%';
    img.style.imageRendering = 'pixelated';

    display.appendChild(img);
    if (elements.actionButtons) {
        elements.actionButtons.style.display = 'flex';
    }
}

function extractBase64Payload(data) {
    if (!data) {
        return '';
    }
    if (typeof data.data === 'string' && data.data.trim()) {
        return data.data.trim();
    }
    if (typeof data.base64 === 'string' && data.base64.trim()) {
        return data.base64.trim();
    }
    return '';
}

function downloadQR() {
    if (!state.currentQr?.base64) {
        showStatus('Generate a code first', 'error');
        return;
    }

    const link = document.createElement('a');
    link.href = `data:image/png;base64,${state.currentQr.base64}`;
    link.download = buildDownloadFilename(state.currentQr.text);
    document.body.appendChild(link);
    link.click();
    link.remove();

    showStatus('Download started', 'success');
}

function buildDownloadFilename(text) {
    const sanitized = (text || '')
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '');
    const timestamp = Date.now();
    return sanitized ? `qr-code-${sanitized}-${timestamp}.png` : `qr-code-${timestamp}.png`;
}

async function copyText() {
    const text = elements.textInput?.value.trim();
    if (!text) {
        showStatus('Nothing to copy', 'error');
        return;
    }

    try {
        await navigator.clipboard.writeText(text);
        showStatus('Copied to clipboard', 'success');
    } catch (error) {
        console.warn('[QR UI] Clipboard error', error);
        showStatus('Copy failed', 'error');
    }
}

function addBatchItem() {
    if (!elements.batchItems) {
        return;
    }

    const itemDiv = document.createElement('div');
    itemDiv.className = 'batch-item';

    const textInput = document.createElement('input');
    textInput.type = 'text';
    textInput.placeholder = 'Enter text/URL';
    textInput.className = 'batch-input';

    const labelInput = document.createElement('input');
    labelInput.type = 'text';
    labelInput.placeholder = 'Label (optional)';
    labelInput.className = 'batch-label';

    const removeButton = document.createElement('button');
    removeButton.type = 'button';
    removeButton.className = 'batch-remove';
    removeButton.textContent = '✖';
    removeButton.setAttribute('aria-label', 'Remove batch item');
    removeButton.addEventListener('click', () => {
        itemDiv.remove();
        updateBatchCount();
    });

    itemDiv.appendChild(textInput);
    itemDiv.appendChild(labelInput);
    itemDiv.appendChild(removeButton);

    elements.batchItems.appendChild(itemDiv);
    updateBatchCount();
    textInput.focus();
}

function updateBatchCount() {
    if (!elements.batchItems || !elements.batchCount || !elements.processBatchButton) {
        return;
    }

    const items = elements.batchItems.querySelectorAll('.batch-item');
    const count = items.length;
    elements.batchCount.textContent = String(count);
    elements.processBatchButton.style.display = count > 0 ? 'block' : 'none';
    elements.processBatchButton.disabled = count === 0;
}

async function processBatch() {
    const items = collectBatchItems();
    if (items.length === 0) {
        showStatus('Error: No valid items in batch', 'error');
        return;
    }

    const button = elements.processBatchButton;
    const originalLabel = button.innerHTML;
    button.innerHTML = '⏳ Processing batch…';
    button.disabled = true;

    showStatus(`Processing ${items.length} ${items.length === 1 ? 'item' : 'items'}…`, 'processing');

    const payload = {
        items,
        options: {
            size: parseInt(elements.sizeSelect?.value, 10) || 256,
            color: elements.foregroundInput?.value || '#000000',
            background: elements.backgroundInput?.value || '#ffffff',
            errorCorrection: elements.correctionSelect?.value || 'M',
            format: 'png'
        }
    };

    try {
        const response = await fetch(buildApiUrl('/batch'), {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });

        const data = await response.json();

        if (!response.ok || !data?.success) {
            throw new Error(data?.error || data?.message || `Request failed with status ${response.status}`);
        }

        const results = Array.isArray(data.results) ? data.results : [];
        const successCount = results.filter((item) => item?.success).length;
        const total = results.length || items.length;
        const failureCount = total - successCount;

        const message = failureCount > 0
            ? `Batch complete with warnings: ${successCount}/${total} generated`
            : `Batch complete: ${successCount}/${total} generated`;

        showStatus(message, failureCount > 0 ? 'error' : 'success');
    } catch (error) {
        console.error('[QR UI] Batch error', error);
        showStatus(`Error: ${error.message}`, 'error');
    } finally {
        button.innerHTML = originalLabel;
        button.disabled = items.length === 0;
        updateBatchCount();
    }
}

function collectBatchItems() {
    if (!elements.batchItems) {
        return [];
    }

    return Array.from(elements.batchItems.querySelectorAll('.batch-item'))
        .map((item, index) => {
            const inputs = item.querySelectorAll('input');
            const text = inputs[0]?.value.trim();
            const label = inputs[1]?.value.trim();
            if (!text) {
                return null;
            }
            return {
                text,
                label: label || `QR_${index + 1}`
            };
        })
        .filter(Boolean);
}

function updateColorPreview() {
    updateColorDisplay();
    updateBgDisplay();
}

function updateColorDisplay() {
    if (!elements.foregroundInput || !elements.foregroundLabel) {
        return;
    }
    const color = elements.foregroundInput.value;
    elements.foregroundLabel.textContent = color.toUpperCase();
}

function updateBgDisplay() {
    if (!elements.backgroundInput || !elements.backgroundLabel) {
        return;
    }
    const bg = elements.backgroundInput.value;
    elements.backgroundLabel.textContent = bg.toUpperCase();
}

function setPreviewBusy(isBusy) {
    if (!elements.qrDisplay) {
        return;
    }
    elements.qrDisplay.setAttribute('aria-busy', isBusy ? 'true' : 'false');
}

function showStatus(message, type = 'normal') {
    const status = elements.status;
    if (!status) {
        return;
    }

    status.textContent = message;
    status.classList.remove('processing', 'is-error', 'is-success');

    if (state.statusResetHandle) {
        clearTimeout(state.statusResetHandle);
        state.statusResetHandle = null;
    }

    if (type === 'processing') {
        status.classList.add('processing');
        return;
    }

    if (type === 'error') {
        status.classList.add('is-error');
    } else if (type === 'success') {
        status.classList.add('is-success');
    }

    state.statusResetHandle = window.setTimeout(() => {
        status.textContent = 'System ready';
        status.classList.remove('is-error', 'is-success', 'processing');
        state.statusResetHandle = null;
    }, 4000);
}

function resolvePathDerivedProxyBase() {
    if (typeof window === 'undefined') {
        return '';
    }

    const { origin, pathname } = window.location || {};
    if (!origin || typeof pathname !== 'string') {
        return '';
    }

    const proxyIndex = pathname.indexOf('/proxy');
    if (proxyIndex === -1) {
        return '';
    }

    const basePath = pathname.slice(0, proxyIndex + '/proxy'.length);
    return stripTrailingSlash(`${origin}${basePath}`);
}

function joinUrl(base, next) {
    const normalizedBase = stripTrailingSlash(base || '');
    const normalizedNext = ensureLeadingSlash(next || '/');
    if (!normalizedBase) {
        return normalizedNext;
    }
    return `${normalizedBase}${normalizedNext}`;
}

function ensureLeadingSlash(value) {
    if (!value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

function stripTrailingSlash(value) {
    if (!value) {
        return '';
    }
    return value.replace(/\/+$/, '');
}
