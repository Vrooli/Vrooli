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

const API_URL = window.location.hostname === 'localhost' 
    ? 'http://localhost:8780' 
    : '/api';

// State management
let currentPalette = null;
let paletteHistory = JSON.parse(localStorage.getItem('paletteHistory') || '[]');

// DOM elements
const themeInput = document.getElementById('theme');
const styleSelect = document.getElementById('style');
const numColorsInput = document.getElementById('numColors');
const numColorsValue = document.getElementById('numColorsValue');
const baseColorInput = document.getElementById('baseColor');
const baseColorHex = document.getElementById('baseColorHex');
const generateBtn = document.getElementById('generateBtn');
const paletteDisplay = document.getElementById('paletteDisplay');
const paletteInfo = document.getElementById('paletteInfo');
const paletteName = document.getElementById('paletteName');
const paletteDescription = document.getElementById('paletteDescription');
const exportOptions = document.getElementById('exportOptions');
const exportCode = document.getElementById('exportCode');
const historyContainer = document.getElementById('historyContainer');
const toast = document.getElementById('toast');

// Event listeners
numColorsInput.addEventListener('input', (e) => {
    numColorsValue.textContent = e.target.value;
});

baseColorInput.addEventListener('input', (e) => {
    baseColorHex.value = e.target.value;
});

baseColorHex.addEventListener('input', (e) => {
    if (/^#[0-9A-F]{6}$/i.test(e.target.value)) {
        baseColorInput.value = e.target.value;
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
    if (!theme) {
        showToast('Please enter a theme or concept', 'error');
        return;
    }

    generateBtn.disabled = true;
    generateBtn.textContent = 'Generating...';

    try {
        const response = await fetch(`${API_URL}/generate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                theme: theme,
                style: styleSelect.value,
                num_colors: parseInt(numColorsInput.value),
                base_color: baseColorHex.value
            })
        });

        const data = await response.json();
        
        if (data.success) {
            currentPalette = data;
            displayPalette(data);
            saveTHistory(data);
            showToast('Palette generated successfully!');
        } else {
            // Use fallback palette if generation fails
            const fallback = {
                success: true,
                palette: data.fallback_palette || generateFallbackPalette(),
                name: theme,
                theme: theme,
                description: 'Generated palette'
            };
            currentPalette = fallback;
            displayPalette(fallback);
            showToast('Using fallback palette', 'error');
        }
    } catch (error) {
        console.error('Error generating palette:', error);
        // Generate client-side fallback
        const fallback = {
            success: true,
            palette: generateFallbackPalette(),
            name: theme,
            theme: theme,
            description: 'Client-generated palette'
        };
        currentPalette = fallback;
        displayPalette(fallback);
        showToast('Generated palette locally', 'error');
    } finally {
        generateBtn.disabled = false;
        generateBtn.textContent = 'Generate Palette';
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
    paletteName.textContent = data.name || 'Generated Palette';
    paletteDescription.textContent = data.description || `A ${data.style || 'custom'} palette based on "${data.theme}"`;
    
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
}

// Export palette function
async function exportPalette(format) {
    if (!currentPalette) {
        showToast('No palette to export', 'error');
        return;
    }

    try {
        const response = await fetch(`${API_URL}/export`, {
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
            exportCode.textContent = data.export;
            exportCode.classList.remove('hidden');
            copyToClipboard(data.export);
            showToast(`${format.toUpperCase()} code copied to clipboard!`);
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
function saveTHistory(palette) {
    const historyItem = {
        ...palette,
        timestamp: Date.now()
    };
    
    paletteHistory.unshift(historyItem);
    paletteHistory = paletteHistory.slice(0, 20); // Keep last 20
    localStorage.setItem('paletteHistory', JSON.stringify(paletteHistory));
    
    renderHistory();
}

// Render history
function renderHistory() {
    if (paletteHistory.length === 0) {
        historyContainer.innerHTML = '<p class="empty-state">No palettes generated yet</p>';
        return;
    }
    
    historyContainer.innerHTML = paletteHistory.map(item => `
        <div class="history-item" data-palette='${JSON.stringify(item)}'>
            <div class="history-palette">
                ${(item.palette || []).map(color => `
                    <div class="history-color" style="background: ${color}"></div>
                `).join('')}
            </div>
            <div class="history-info">
                <div class="history-theme">${item.theme || 'Untitled'}</div>
                <div class="history-time">${new Date(item.timestamp).toLocaleDateString()}</div>
            </div>
        </div>
    `).join('');
    
    // Add click handlers
    document.querySelectorAll('.history-item').forEach(item => {
        item.addEventListener('click', () => {
            const palette = JSON.parse(item.dataset.palette);
            currentPalette = palette;
            displayPalette(palette);
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
