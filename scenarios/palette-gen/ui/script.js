import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

if (typeof window !== 'undefined' && window.parent !== window && !window.__paletteGenBridgeInitialized) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[PaletteGen] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'palette-gen' });
    window.__paletteGenBridgeInitialized = true;
}

const API_BASE = resolveApiBase();

function buildApiUrl(path) {
    if (!path) {
        return API_BASE;
    }
    return `${API_BASE}${path.startsWith('/') ? path : `/${path}`}`;
}

function resolveApiBase() {
    const fallbackPort = '8780';

    if (typeof window === 'undefined') {
        return `http://127.0.0.1:${fallbackPort}/api`;
    }

    const proxyBase = resolveProxyBase();
    if (proxyBase) {
        return joinUrl(proxyBase, '/api');
    }

    const envApiUrl = typeof window.ENV?.API_URL === 'string' ? window.ENV.API_URL.trim() : '';
    if (envApiUrl) {
        return joinUrl(envApiUrl, '/api');
    }

    const { origin, protocol, hostname } = window.location || {};
    const scheme = protocol === 'https:' ? 'https' : 'http';
    const host = hostname || '127.0.0.1';
    const isRemoteHost = hostname ? !isLocalHostname(hostname) : false;

    const envApiPort = typeof window.ENV?.API_PORT === 'string' ? window.ENV.API_PORT.trim() : '';
    const defaultLocalPort = envApiPort || (isLocalHostname(host) ? fallbackPort : '');
    const localOrigin = `${scheme}://${host}${defaultLocalPort ? `:${defaultLocalPort}` : ''}`;

    if (origin && isRemoteHost) {
        return joinUrl(origin, '/api');
    }

    if (origin && isLocalHostname(host)) {
        return joinUrl(localOrigin, '/api');
    }

    if (origin) {
        return joinUrl(origin, '/api');
    }

    return joinUrl(localOrigin, '/api');
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

function isLocalHostname(hostname) {
    return /^(localhost|127\.0\.0\.1|\[::1\])$/i.test(hostname || '');
}



function safeReadFromStorage(key, fallback) {
    if (typeof window === 'undefined') {
        return fallback;
    }
    try {
        const raw = window.localStorage.getItem(key);
        if (!raw) {
            return fallback;
        }
        return JSON.parse(raw);
    } catch (error) {
        console.warn(`[PaletteGen] Unable to read ${key} from storage`, error);
        return fallback;
    }
}

function safeWriteToStorage(key, value) {
    if (typeof window === 'undefined') {
        return;
    }
    try {
        window.localStorage.setItem(key, JSON.stringify(value));
    } catch (error) {
        console.warn(`[PaletteGen] Unable to persist ${key} to storage`, error);
    }
}

// State management
let currentPalette = null;
let paletteHistory = safeReadFromStorage('paletteHistory', []);
const userPreferences = safeReadFromStorage('palettePreferences', {
    useBaseColor: false,
    includeAiDebug: false
});

// DOM elements
const themeInput = document.getElementById('theme');
const styleSelect = document.getElementById('style');
const numColorsInput = document.getElementById('numColors');
const numColorsValue = document.getElementById('numColorsValue');
const baseColorInput = document.getElementById('baseColor');
const baseColorHex = document.getElementById('baseColorHex');
const baseColorWrapper = document.querySelector('.color-input-wrapper');
const useBaseColorToggle = document.getElementById('useBaseColor');
const includeAiDebugToggle = document.getElementById('includeAIDebug');
const generateBtn = document.getElementById('generateBtn');
const paletteDisplay = document.getElementById('paletteDisplay');
const paletteInfo = document.getElementById('paletteInfo');
const paletteName = document.getElementById('paletteName');
const paletteDescription = document.getElementById('paletteDescription');
const exportOptions = document.getElementById('exportOptions');
const exportCode = document.getElementById('exportCode');
const historyContainer = document.getElementById('historyContainer');
const toast = document.getElementById('toast');
const debugPanel = document.getElementById('debugPanel');
const debugStrategy = document.getElementById('debugStrategy');
const debugContext = document.getElementById('debugContext');
const debugAiSection = document.getElementById('debugAiSection');
const debugAiStatus = document.getElementById('debugAiStatus');
const debugAiModel = document.getElementById('debugAiModel');
const debugAiPrompt = document.getElementById('debugAiPrompt');
const debugAiRaw = document.getElementById('debugAiRaw');
const debugAiSuggestions = document.getElementById('debugAiSuggestions');

if (useBaseColorToggle) {
    useBaseColorToggle.checked = Boolean(userPreferences.useBaseColor);
}
if (includeAiDebugToggle) {
    includeAiDebugToggle.checked = Boolean(userPreferences.includeAiDebug);
}

const persistPreferences = () => {
    userPreferences.useBaseColor = Boolean(useBaseColorToggle?.checked);
    userPreferences.includeAiDebug = Boolean(includeAiDebugToggle?.checked);
    safeWriteToStorage('palettePreferences', userPreferences);
};

const syncBaseColorState = () => {
    const active = Boolean(useBaseColorToggle?.checked);
    baseColorInput.disabled = !active;
    baseColorHex.disabled = !active;
    if (baseColorWrapper) {
        baseColorWrapper.classList.toggle('disabled', !active);
    }
};

useBaseColorToggle?.addEventListener('change', () => {
    syncBaseColorState();
    persistPreferences();
});

includeAiDebugToggle?.addEventListener('change', () => {
    persistPreferences();
});

syncBaseColorState();

const initialBaseColor = normalizeHexColor(baseColorHex.value || baseColorInput.value);
if (initialBaseColor) {
    baseColorHex.value = initialBaseColor;
    baseColorInput.value = initialBaseColor;
}

// Event listeners
numColorsInput.addEventListener('input', (e) => {
    numColorsValue.textContent = e.target.value;
});

baseColorInput.addEventListener('input', (e) => {
    const normalized = normalizeHexColor(e.target.value);
    if (normalized) {
        baseColorInput.value = normalized;
        baseColorHex.value = normalized;
    }
});

baseColorHex.addEventListener('input', (e) => {
    const normalized = normalizeHexColor(e.target.value);
    if (normalized) {
        baseColorHex.value = normalized;
        baseColorInput.value = normalized;
    }
});

generateBtn.addEventListener('click', generatePalette);

// Suggestion buttons
document.querySelectorAll('.suggestion-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        themeInput.value = btn.dataset.theme;
        styleSelect.value = btn.dataset.style || '';
        generatePalette();
    });
});

// Export buttons
document.querySelectorAll('.export-buttons button').forEach(btn => {
    btn.addEventListener('click', () => {
        exportPalette(btn.dataset.format);
    });
});

// Generate palette function
async function generatePalette() {
    const theme = themeInput.value.trim();
    const style = styleSelect.value.trim();
    const numColors = parseInt(numColorsInput.value, 10) || 5;
    const includeAiDebug = Boolean(includeAiDebugToggle?.checked);
    const baseColorCandidate = useBaseColorToggle?.checked ? normalizeHexColor(baseColorHex.value) : '';

    const payload = {
        theme,
        style,
        num_colors: numColors
    };

    if (baseColorCandidate) {
        payload.base_color = baseColorCandidate;
    }
    if (includeAiDebug) {
        payload.include_ai_debug = true;
    }

    generateBtn.disabled = true;
    generateBtn.textContent = 'Generating...';

    try {
        const response = await fetch(buildApiUrl('/generate'), {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data?.error || `Request failed with status ${response.status}`);
        }

        const palettePayload = preparePalettePayload({
            apiResponse: data,
            fallbackTheme: theme,
            fallbackStyle: style,
            fallbackBaseColor: baseColorCandidate,
            requestedCount: numColors
        });

        currentPalette = palettePayload;
        displayPalette(palettePayload);
        saveToHistory(palettePayload);

        showToast(palettePayload.__source === 'fallback' ? 'Using fallback palette' : 'Palette generated successfully!');
    } catch (error) {
        console.error('Error generating palette:', error);
        const fallback = {
            success: true,
            palette: generateFallbackPalette(),
            name: theme || 'Procedural Palette',
            theme,
            style,
            description: 'Client-generated palette',
            debug: null,
            __source: 'client-fallback'
        };
        currentPalette = fallback;
        displayPalette(fallback);
        saveToHistory(fallback);
        showToast('Generated palette locally', 'error');
    } finally {
        generateBtn.disabled = false;
        generateBtn.textContent = 'Generate Palette';
    }
}

function preparePalettePayload({ apiResponse, fallbackTheme, fallbackStyle, fallbackBaseColor, requestedCount }) {
    const response = apiResponse || {};
    const sanitizedTheme = fallbackTheme || response.theme || '';
    const sanitizedStyle = (response.style || fallbackStyle || '').trim();
    const palette = Array.isArray(response.palette) ? response.palette : [];
    const paletteCount = palette.length || requestedCount || parseInt(numColorsInput.value, 10) || 5;

    if (response.success) {
        return {
            ...response,
            palette,
            theme: response.theme ?? sanitizedTheme,
            style: response.style ?? sanitizedStyle,
            description: response.description || buildClientDescription({
                theme: response.theme ?? sanitizedTheme,
                style: response.style ?? sanitizedStyle,
                baseColor: fallbackBaseColor,
                count: paletteCount
            }),
            debug: response.debug || null,
            __source: 'api'
        };
    }

    const fallbackPalette = Array.isArray(response.fallback_palette) ? response.fallback_palette : [];
    const resolvedPalette = fallbackPalette.length ? fallbackPalette : generateFallbackPalette();
    return {
        success: true,
        palette: resolvedPalette,
        name: response.name || sanitizedTheme || 'Custom Palette',
        theme: response.theme ?? sanitizedTheme,
        style: response.style ?? sanitizedStyle,
        description: response.description || buildClientDescription({
            theme: response.theme ?? sanitizedTheme,
            style: response.style ?? sanitizedStyle,
            baseColor: fallbackBaseColor,
            count: resolvedPalette.length || paletteCount
        }),
        debug: response.debug || null,
        __source: 'fallback'
    };
}

function buildClientDescription({ theme, style, baseColor, count }) {
    const fragments = [];
    if (theme) {
        fragments.push(`Inspired by ${theme}`);
    }
    if (style) {
        fragments.push(`using a ${formatLabel(style)} style`);
    }
    if (baseColor) {
        fragments.push(`anchored by ${baseColor.toUpperCase()}`);
    }

    const context = fragments.join(' ');
    const countText = `${count || 0} ${count === 1 ? 'color' : 'colors'}`;
    if (!context) {
        return `Procedurally generated palette with ${countText}`;
    }
    return `${context} with ${countText}`;
}

function normalizeHexColor(value) {
    if (typeof value !== 'string') {
        return '';
    }
    const trimmed = value.trim();
    if (!trimmed) {
        return '';
    }
    const prefixed = trimmed.startsWith('#') ? trimmed : `#${trimmed}`;
    if (/^#[0-9A-F]{6}$/i.test(prefixed)) {
        return prefixed.toUpperCase();
    }
    return '';
}

function formatLabel(value) {
    if (!value) {
        return '';
    }
    return value
        .split(/[\s_-]+/)
        .filter(Boolean)
        .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
        .join(' ');
}

function updateDebugPanel(debug) {
    if (!debugPanel) {
        return;
    }

    if (!debug) {
        debugPanel.classList.add('hidden');
        if (debugStrategy) debugStrategy.textContent = '—';
        if (debugContext) debugContext.textContent = 'Defaults applied';
        if (debugAiSection) debugAiSection.classList.add('hidden');
        if (debugAiStatus) debugAiStatus.textContent = 'AI not requested';
        if (debugAiModel) debugAiModel.textContent = '—';
        if (debugAiPrompt) debugAiPrompt.textContent = '';
        if (debugAiRaw) debugAiRaw.textContent = '';
        if (debugAiSuggestions) debugAiSuggestions.textContent = '';
        return;
    }

    debugPanel.classList.remove('hidden');

    if (debugStrategy) {
        debugStrategy.textContent = formatLabel(debug.strategy) || 'Procedural Default';
    }

    if (debugContext) {
        const contextParts = [];
        if (debug.requested_theme) {
            contextParts.push(`Theme: ${debug.requested_theme}`);
        }
        const styleLabel = debug.resolved_style || debug.requested_style;
        if (styleLabel) {
            contextParts.push(`Style: ${formatLabel(styleLabel)}`);
        }
        if (debug.requested_base_color) {
            contextParts.push(`Base: ${debug.requested_base_color}`);
        }
        if (Number.isFinite(debug.base_hue)) {
            contextParts.push(`Base Hue: ${Number(debug.base_hue).toFixed(2)}°`);
        }
        if (debug.requested_colors) {
            contextParts.push(`Count: ${debug.requested_colors}`);
        }
        debugContext.textContent = contextParts.join(' • ') || 'Defaults applied';
    }

    if (!debug.ai_requested) {
        if (debugAiSection) debugAiSection.classList.add('hidden');
        if (debugAiStatus) debugAiStatus.textContent = 'AI not requested';
        if (debugAiModel) debugAiModel.textContent = '—';
        if (debugAiPrompt) debugAiPrompt.textContent = '';
        if (debugAiRaw) debugAiRaw.textContent = '';
        if (debugAiSuggestions) debugAiSuggestions.textContent = '';
        return;
    }

    if (debugAiSection) debugAiSection.classList.remove('hidden');

    if (debugAiStatus) {
        if (debug.ai_available) {
            const duration = debug.ai_duration_ms ? ` in ${debug.ai_duration_ms} ms` : '';
            debugAiStatus.textContent = `AI suggestions loaded${duration}`;
        } else if (debug.ai_error) {
            debugAiStatus.textContent = `AI error: ${debug.ai_error}`;
        } else {
            debugAiStatus.textContent = 'AI service unavailable';
        }
    }

    if (debugAiModel) {
        debugAiModel.textContent = debug.ai_model || 'Unknown';
    }
    if (debugAiPrompt) {
        debugAiPrompt.textContent = debug.ai_prompt || 'Prompt not available';
    }
    if (debugAiRaw) {
        debugAiRaw.textContent = debug.ai_raw_output || '—';
    }
    if (debugAiSuggestions) {
        debugAiSuggestions.textContent = debug.ai_suggestions
            ? JSON.stringify(debug.ai_suggestions, null, 2)
            : '—';
    }
}


// Display palette function
function displayPalette(data) {
    const colors = data.palette || [];
    
    // Create palette display
    const paletteHTML = `
        <div class="palette-colors">
            ${colors.map((color, index) => `
                <div class="color-card" data-color="${color}">
                    <div class="color-preview" style="background: ${color}">
                        <div class="copy-indicator">Copied!</div>
                    </div>
                    <div class="color-info">
                        <div class="color-hex">${color}</div>
                        <div class="color-name">Color ${index + 1}</div>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
    
    paletteDisplay.innerHTML = paletteHTML;
    
    // Update palette info
    const fallbackName = formatLabel(data.style) ? `${formatLabel(data.style)} Palette` : 'Generated Palette';
    paletteName.textContent = data.name || fallbackName;

    const fallbackDescription = buildClientDescription({
        theme: data.theme,
        style: data.style,
        baseColor: data.debug?.requested_base_color || '',
        count: colors.length
    });
    paletteDescription.textContent = data.description || fallbackDescription;
    
    // Show info and export options
    paletteInfo.classList.remove('hidden');
    exportOptions.classList.remove('hidden');
    
    // Add click to copy functionality
    document.querySelectorAll('.color-card').forEach(card => {
        card.addEventListener('click', () => {
            const color = card.dataset.color;
            copyToClipboard(color);
            
            const indicator = card.querySelector('.copy-indicator');
            indicator.classList.add('show');
            setTimeout(() => indicator.classList.remove('show'), 1500);
        });
    });

    updateDebugPanel(data.debug || null);
}

// Export palette function
async function exportPalette(format) {
    if (!currentPalette) {
        showToast('No palette to export', 'error');
        return;
    }

    try {
        const response = await fetch(buildApiUrl('/export'), {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                format: format,
                palette: currentPalette.palette
            })
        });

        const data = await response.json();
        
        if (data.success) {
            if (data.export) {
                exportCode.textContent = data.export;
                exportCode.classList.remove('hidden');
                copyToClipboard(data.export);
                showToast(`${format.toUpperCase()} code copied to clipboard!`);
            } else {
                showToast('Export failed', 'error');
            }
        }
    } catch (error) {
        // Generate export locally
        const exportData = generateExport(currentPalette.palette, format);
        exportCode.textContent = exportData;
        exportCode.classList.remove('hidden');
        copyToClipboard(exportData);
        showToast(`${format.toUpperCase()} code copied to clipboard!`);
    }
}

// Generate export locally
function generateExport(palette, format) {
    switch (format) {
        case 'css':
            return `:root {\n${palette.map((color, i) => `  --color-${i + 1}: ${color};`).join('\n')}\n}`;
        case 'scss':
            return palette.map((color, i) => `$color-${i + 1}: ${color};`).join('\n');
        case 'json':
            return JSON.stringify(palette, null, 2);
        case 'svg':
            const width = 100 * palette.length;
            return `<svg width="${width}" height="100" xmlns="http://www.w3.org/2000/svg">
${palette.map((color, i) => `  <rect x="${i * 100}" y="0" width="100" height="100" fill="${color}"/>`).join('\n')}
</svg>`;
        default:
            return '';
    }
}

// Save to history
function saveToHistory(palette) {
    if (!palette) {
        return;
    }

    const { debug, __source, ...rest } = palette;

    const historyItem = {
        ...rest,
        timestamp: Date.now()
    };
    
    paletteHistory.unshift(historyItem);
    paletteHistory = paletteHistory.slice(0, 20); // Keep last 20
    safeWriteToStorage('paletteHistory', paletteHistory);
    
    renderHistory();
}

// Render history
function renderHistory() {
    if (paletteHistory.length === 0) {
        historyContainer.innerHTML = '<p class="empty-state">No palettes generated yet</p>';
        return;
    }
    
    historyContainer.innerHTML = paletteHistory.map(item => {
        const encoded = encodeURIComponent(JSON.stringify(item));
        const historyLabel = item.name || item.theme || 'Untitled';
        const historySubtitle = item.style ? formatLabel(item.style) : '';
        const historyDate = item.timestamp ? new Date(item.timestamp).toLocaleDateString() : '—';
        return `
        <div class="history-item" data-palette="${encoded}">
            <div class="history-palette">
                ${(item.palette || []).map(color => `
                    <div class="history-color" style="background: ${color}"></div>
                `).join('')}
            </div>
            <div class="history-info">
                <div class="history-theme">${historyLabel}</div>
                <div class="history-subtitle">${historySubtitle}</div>
                <div class="history-time">${historyDate}</div>
            </div>
        </div>
    `;
    }).join('');
    
    // Add click handlers
    document.querySelectorAll('.history-item').forEach(item => {
        item.addEventListener('click', () => {
            try {
                const raw = item.dataset.palette;
                const palette = JSON.parse(decodeURIComponent(raw));
                currentPalette = palette;
                displayPalette(palette);
            } catch (error) {
                console.warn('[PaletteGen] Unable to restore palette from history', error);
                showToast('Unable to restore palette', 'error');
            }
        });
    });
}

// Generate fallback palette
function generateFallbackPalette() {
    const hue = Math.random() * 360;
    const palette = [];
    
    for (let i = 0; i < parseInt(numColorsInput.value); i++) {
        const h = (hue + (i * 30)) % 360;
        const s = 70 - (i * 5);
        const l = 50 + (i * 5);
        palette.push(hslToHex(h, s, l));
    }
    
    return palette;
}

// HSL to HEX converter
function hslToHex(h, s, l) {
    l /= 100;
    const a = s * Math.min(l, 1 - l) / 100;
    const f = n => {
        const k = (n + h / 30) % 12;
        const color = l - a * Math.max(Math.min(k - 3, 9 - k, 1), -1);
        return Math.round(255 * color).toString(16).padStart(2, '0');
    };
    return `#${f(0)}${f(8)}${f(4)}`.toUpperCase();
}

// Copy to clipboard
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).catch(() => {
        // Fallback for older browsers
        const textarea = document.createElement('textarea');
        textarea.value = text;
        document.body.appendChild(textarea);
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
    });
}

// Show toast notification
function showToast(message, type = 'success') {
    toast.textContent = message;
    toast.className = `toast ${type}`;
    toast.classList.add('show');
    
    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

// Initialize
renderHistory();
