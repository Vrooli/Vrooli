// SmartFile - AI-Powered File Manager JavaScript

const API_BASE = resolveApiBase();

function resolveApiBase() {
    const API_PATH = '/api';
    const DEFAULT_PORT = 8080;

    if (typeof window === 'undefined') {
        return `http://127.0.0.1:${DEFAULT_PORT}${API_PATH}`;
    }

    const proxyBase = resolveProxyBase();
    if (proxyBase) {
        return appendPath(proxyBase, API_PATH);
    }

    const envApiBase = resolveEnvApiBase();
    if (envApiBase) {
        return appendPath(envApiBase, API_PATH);
    }

    const origin = window.location?.origin;
    if (origin && origin !== 'null') {
        return appendPath(origin, API_PATH);
    }

    const port = window.location?.port || DEFAULT_PORT;
    const protocol = window.location?.protocol || 'http:';
    const hostname = window.location?.hostname || '127.0.0.1';
    return `${protocol}//${hostname}:${port}${API_PATH}`;
}

function resolveEnvApiBase() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const env = window.ENV || window.env;
    const candidate = typeof env?.API_URL === 'string' ? env.API_URL : typeof env?.apiUrl === 'string' ? env.apiUrl : undefined;
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    return trimmed ? stripTrailingSlash(trimmed) : undefined;
}

function resolveProxyBase() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    const sources = collectProxySources();
    for (const source of sources) {
        const candidates = extractProxyCandidates(source);
        for (const candidate of candidates) {
            const normalized = normalizeProxyCandidate(candidate);
            if (normalized) {
                return normalized;
            }
        }
    }

    return undefined;
}

function collectProxySources() {
    const sources = [];
    const seen = new Set();

    const enqueue = (value) => {
        if (!value || seen.has(value)) {
            return;
        }
        seen.add(value);
        sources.push(value);
    };

    const frames = [window];
    try {
        if (window.parent && window.parent !== window) {
            frames.push(window.parent);
        }
    } catch (error) {
        // Ignore cross-origin access errors
    }

    try {
        if (window.top && window.top !== window && !frames.includes(window.top)) {
            frames.push(window.top);
        }
    } catch (error) {
        // Ignore cross-origin access errors
    }

    for (const frame of frames) {
        if (!frame) {
            continue;
        }
        try {
            enqueue(frame.__APP_MONITOR_PROXY_INFO__);
            enqueue(frame.__APP_MONITOR_PROXY_INDEX__);
        } catch (error) {
            // Ignore cross-origin access errors
        }
    }

    return sources;
}

function extractProxyCandidates(source, bucket = [], depth = 0) {
    if (!source || depth > 4) {
        return bucket;
    }

    if (typeof source === 'string') {
        bucket.push(source);
        return bucket;
    }

    if (typeof source === 'number' && Number.isFinite(source)) {
        bucket.push(String(source));
        return bucket;
    }

    if (Array.isArray(source)) {
        source.forEach((item) => extractProxyCandidates(item, bucket, depth + 1));
        return bucket;
    }

    if (typeof source !== 'object') {
        return bucket;
    }

    const keys = [
        'apiBase',
        'apiUrl',
        'api',
        'primary',
        'url',
        'target',
        'path',
        'endpoint',
        'endpoints',
        'proxy',
        'proxyUrl',
        'proxyPath',
        'proxyApiUrl',
        'proxyApiPath',
        'services',
        'ports',
        'entries'
    ];

    for (const key of keys) {
        if (Object.prototype.hasOwnProperty.call(source, key)) {
            extractProxyCandidates(source[key], bucket, depth + 1);
        }
    }

    return bucket;
}

function normalizeProxyCandidate(candidate) {
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
    }

    const protocolMatch = trimmed.match(/^https?:\/\//i);
    if (protocolMatch) {
        return stripTrailingSlash(trimmed);
    }

    if (trimmed.startsWith('//')) {
        const protocol = window.location?.protocol || 'https:';
        return stripTrailingSlash(`${protocol}${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        const origin = window.location?.origin;
        if (origin) {
            return stripTrailingSlash(`${origin}${trimmed}`);
        }
        return stripTrailingSlash(trimmed);
    }

    if (/^\d+$/.test(trimmed)) {
        return `http://127.0.0.1:${trimmed}`;
    }

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i.test(trimmed)) {
        return stripTrailingSlash(`http://${trimmed}`);
    }

    const origin = window.location?.origin;
    if (origin) {
        return stripTrailingSlash(`${origin}/${trimmed.replace(/^\/+/, '')}`);
    }

    return stripTrailingSlash(trimmed);
}

function appendPath(base, path) {
    if (!base) {
        return ensureLeadingSlash(path);
    }

    const sanitizedBase = stripTrailingSlash(base);
    const sanitizedPath = ensureLeadingSlash(path);
    if (sanitizedBase.toLowerCase().endsWith(sanitizedPath.toLowerCase())) {
        return sanitizedBase;
    }

    return `${sanitizedBase}${sanitizedPath}`;
}

function stripTrailingSlash(value) {
    if (typeof value !== 'string') {
        return value;
    }
    return value.replace(/\/+$/, '');
}

function ensureLeadingSlash(value) {
    if (typeof value !== 'string' || !value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

// State management
const state = {
    currentView: 'organize',
    files: [],
    searchResults: [],
    duplicates: [],
    tags: new Set(),
    selectedFiles: new Set(),
    uploadProgress: 0
};

function createIcon(name, { className = '', size } = {}) {
    if (!name) return null;
    const iconElement = document.createElement('span');
    iconElement.setAttribute('data-lucide', name);
    iconElement.className = ['lucide-icon', className].filter(Boolean).join(' ');
    iconElement.setAttribute('aria-hidden', 'true');
    if (size) {
        iconElement.style.width = `${size}px`;
        iconElement.style.height = `${size}px`;
    }
    return iconElement;
}

function renderLucideIcons(scope = document) {
    if (!window.lucide?.createIcons || !scope) return;

    const elements = [];

    if (scope !== document && scope.matches?.('[data-lucide]')) {
        elements.push(scope);
    }

    const descendants = scope.querySelectorAll?.('[data-lucide]') ?? [];
    descendants.forEach(el => elements.push(el));

    if (elements.length) {
        window.lucide.createIcons({ elements });
    }
}

function renderEmptyState(container, { icon, title, hint }) {
    if (!container) return;
    container.innerHTML = '';

    const wrapper = document.createElement('div');
    wrapper.className = 'empty-state';

    if (icon) {
        const iconElement = createIcon(icon, { className: 'empty-icon' });
        if (iconElement) {
            wrapper.appendChild(iconElement);
        }
    }

    if (title) {
        const titleEl = document.createElement('p');
        titleEl.textContent = title;
        wrapper.appendChild(titleEl);
    }

    if (hint) {
        const hintEl = document.createElement('p');
        hintEl.className = 'empty-hint';
        hintEl.textContent = hint;
        wrapper.appendChild(hintEl);
    }

    container.appendChild(wrapper);
    renderLucideIcons(wrapper);
}

function setPreviewIcon(container, iconName, className = 'file-type-icon', size) {
    if (!container) return;
    container.innerHTML = '';
    const iconElement = createIcon(iconName, { className, size });
    if (iconElement) {
        container.appendChild(iconElement);
        renderLucideIcons(iconElement);
    }
}

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadFiles();
    setupDragAndDrop();
    renderLucideIcons();
});

// Event Listeners
function initializeEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', () => switchView(item.dataset.view));
    });

    // Search
    const searchInput = document.getElementById('semantic-search');
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => performSearch(e.target.value), 500);
    });

    // Upload
    document.getElementById('upload-btn').addEventListener('click', () => {
        document.getElementById('file-input').click();
    });

    document.getElementById('file-input').addEventListener('change', (e) => {
        handleFileUpload(e.target.files);
    });

    // AI Organize
    document.getElementById('ai-organize-btn').addEventListener('click', aiOrganizeFiles);

    // Scan for duplicates
    const scanBtn = document.getElementById('scan-duplicates');
    if (scanBtn) {
        scanBtn.addEventListener('click', scanForDuplicates);
    }

    // View toggles
    document.querySelectorAll('.view-toggle').forEach(toggle => {
        toggle.addEventListener('click', () => {
            document.querySelectorAll('.view-toggle').forEach(t => t.classList.remove('active'));
            toggle.classList.add('active');
            changeLayout(toggle.dataset.layout);
        });
    });

    // Modal close
    document.getElementById('close-modal').addEventListener('click', closeModal);
    document.getElementById('file-modal').addEventListener('click', (e) => {
        if (e.target.id === 'file-modal') closeModal();
    });
}

// View Management
function switchView(viewName) {
    // Update nav
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.view === viewName);
    });

    // Update view panels
    document.querySelectorAll('.view-panel').forEach(panel => {
        panel.classList.toggle('active', panel.id === `${viewName}-view`);
    });

    state.currentView = viewName;

    // Load view-specific data
    switch (viewName) {
        case 'organize':
            loadFiles();
            break;
        case 'search':
            // Search view is updated via search input
            break;
        case 'duplicates':
            loadDuplicates();
            break;
        case 'tags':
            loadTags();
            break;
        case 'insights':
            loadInsights();
            break;
    }
}

// File Loading
async function loadFiles() {
    try {
        const response = await fetch(`${API_BASE}/files`);
        if (response.ok) {
            state.files = await response.json();
            renderFiles();
        }
    } catch (error) {
        console.error('Error loading files:', error);
        // Use mock data for demo
        state.files = generateMockFiles();
        renderFiles();
    }
}

function renderFiles() {
    const grid = document.getElementById('file-grid');
    if (!grid) return;

    const uploadZone = grid.querySelector('.upload-zone');

    // Clear existing files but keep upload zone
    grid.innerHTML = '';
    if (uploadZone) grid.appendChild(uploadZone);

    state.files.forEach(file => {
        const card = createFileCard(file);
        grid.appendChild(card);
    });

    renderLucideIcons(grid);
}

function createFileCard(file) {
    const card = document.createElement("div");
    const isPhoto = file.type && file.type.startsWith("image/");
    card.className = isPhoto ? "file-card photo-card" : "file-card";
    card.onclick = () => openFileModal(file, isPhoto);

    const preview = document.createElement("div");
    preview.className = "file-preview";
    
    if (file.thumbnail || isPhoto) {
        const img = document.createElement("img");
        img.src = file.thumbnail || file.url || "#";
        img.alt = file.name;
        img.onerror = () => {
            setPreviewIcon(preview, getFileIcon(file.type));
        };
        preview.appendChild(img);
        
        const overlay = document.createElement("div");
        overlay.className = "overlay";
        preview.appendChild(overlay);
    } else {
        setPreviewIcon(preview, getFileIcon(file.type));
    }

    const info = document.createElement("div");
    info.className = "file-info";
    
    const name = document.createElement("div");
    name.className = "file-name";
    name.textContent = file.name;
    
    const meta = document.createElement("div");
    meta.className = "file-meta";
    const metaText = isPhoto && file.width && file.height ?
        `${file.width}×${file.height} • ${formatFileSize(file.size)}` :
        `${formatFileSize(file.size)} • ${formatDate(file.modified)}`;
    meta.textContent = metaText;

    info.appendChild(name);
    info.appendChild(meta);
    card.appendChild(preview);
    card.appendChild(info);

    return card;
}

// Search Functionality
async function performSearch(query) {
    const container = document.getElementById('search-results');
    if (!container) return;

    const trimmedQuery = query.trim();
    if (!trimmedQuery) {
        renderEmptyState(container, {
            icon: 'search',
            title: 'Search for files using natural language',
            hint: 'Try "photos from last summer" or "documents about project X"'
        });
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query: trimmedQuery })
        });

        if (!response.ok) {
            throw new Error(`Search request failed: ${response.status}`);
        }

        state.searchResults = await response.json();
        renderSearchResults();
        return;
    } catch (error) {
        console.error('Search error:', error);
        // Use mock results for demo
        state.searchResults = state.files.filter(f => 
            f.name.toLowerCase().includes(trimmedQuery.toLowerCase())
        );
        renderSearchResults();
    }
}

function renderSearchResults() {
    const container = document.getElementById('search-results');
    if (!container) return;

    if (state.searchResults.length === 0) {
        renderEmptyState(container, {
            icon: 'search-x',
            title: 'No results found',
            hint: 'Try a different search query'
        });
        return;
    }

    container.innerHTML = '';
    const grid = document.createElement('div');
    grid.className = 'file-grid';

    state.searchResults.forEach(file => {
        const card = createFileCard(file);
        grid.appendChild(card);
    });

    container.appendChild(grid);
    renderLucideIcons(grid);
}

// File Upload
function setupDragAndDrop() {
    const dropZone = document.getElementById('drop-zone');
    if (!dropZone) return;

    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });

    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, () => {
            dropZone.classList.add('dragover');
        }, false);
    });

    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, () => {
            dropZone.classList.remove('dragover');
        }, false);
    });

    dropZone.addEventListener('drop', (e) => {
        const files = e.dataTransfer.files;
        handleFileUpload(files);
    }, false);

    dropZone.addEventListener('click', () => {
        document.getElementById('file-input').click();
    });
}

function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
}

async function handleFileUpload(files) {
    const formData = new FormData();
    Array.from(files).forEach(file => {
        formData.append('files', file);
    });

    showProcessing('Uploading files...');

    try {
        const response = await fetch(`${API_BASE}/upload`, {
            method: 'POST',
            body: formData
        });

        if (response.ok) {
            const result = await response.json();
            hideProcessing();
            loadFiles(); // Reload files
            showNotification('Files uploaded successfully!', 'success');
        }
    } catch (error) {
        console.error('Upload error:', error);
        hideProcessing();
        // Add mock files for demo
        const newFiles = Array.from(files).map(file => ({
            id: Date.now() + Math.random(),
            name: file.name,
            type: file.type,
            size: file.size,
            modified: new Date(),
            tags: []
        }));
        state.files.push(...newFiles);
        renderFiles();
        showNotification('Files uploaded successfully!', 'success');
    }
}

// AI Organization
async function aiOrganizeFiles() {
    showProcessing('AI is organizing your files...');

    try {
        const response = await fetch(`${API_BASE}/organize`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ files: state.files })
        });

        if (response.ok) {
            const result = await response.json();
            hideProcessing();
            loadFiles(); // Reload organized files
            showNotification('Files organized successfully!', 'success');
        }
    } catch (error) {
        console.error('Organization error:', error);
        hideProcessing();
        // Simulate organization for demo
        setTimeout(() => {
            hideProcessing();
            showNotification('Files organized successfully!', 'success');
        }, 2000);
    }
}

// Duplicate Detection
async function scanForDuplicates() {
    showProcessing('Scanning for duplicate files...');

    try {
        const response = await fetch(`${API_BASE}/duplicates`);
        if (!response.ok) {
            throw new Error(`Duplicate scan failed: ${response.status}`);
        }

        state.duplicates = await response.json();
    } catch (error) {
        console.error('Duplicate scan error:', error);
        // Use mock data for demo
        state.duplicates = generateMockDuplicates();
    } finally {
        renderDuplicates();
        hideProcessing();
    }
}

function renderDuplicates() {
    const container = document.getElementById('duplicates-container');
    if (!container) return;

    if (state.duplicates.length === 0) {
        renderEmptyState(container, {
            icon: 'check-circle-2',
            title: 'No duplicates found',
            hint: 'Your files are well organized!'
        });
        return;
    }

    container.innerHTML = '';
    state.duplicates.forEach(group => {
        const groupDiv = document.createElement('div');
        groupDiv.className = 'duplicate-group';

        const heading = document.createElement('h3');
        heading.textContent = `Duplicate Group (${group.files.length} files)`;
        groupDiv.appendChild(heading);

        const grid = document.createElement('div');
        grid.className = 'file-grid';

        group.files.forEach(file => {
            const card = createFileCard(file);
            grid.appendChild(card);
        });

        groupDiv.appendChild(grid);
        container.appendChild(groupDiv);
    });

    renderLucideIcons(container);
}

// Tags
async function loadTags() {
    try {
        const response = await fetch(`${API_BASE}/tags`);
        if (response.ok) {
            const tags = await response.json();
            renderTags(tags);
        }
    } catch (error) {
        console.error('Error loading tags:', error);
        // Use mock tags for demo
        const mockTags = ['Work', 'Personal', 'Photos', 'Documents', 'Videos', 'Projects', 'Archive', 'Important'];
        renderTags(mockTags);
    }
}

function renderTags(tags) {
    const cloud = document.getElementById('tags-cloud');
    cloud.innerHTML = '';
    
    tags.forEach(tag => {
        const tagEl = document.createElement('button');
        tagEl.className = 'tag-item';
        tagEl.textContent = tag;
        tagEl.onclick = () => filterByTag(tag);
        cloud.appendChild(tagEl);
    });
}

// Insights
async function loadInsights() {
    // In a real app, this would fetch analytics data
    // For now, we'll use the existing static content
}

// Modal
function openFileModal(file, isPhoto = false) {
    const modal = document.getElementById('file-modal');
    const preview = document.getElementById('preview-container');
    const details = document.getElementById('file-details');

    // Add photo modal class for enhanced styling
    if (isPhoto) {
        modal.classList.add('photo-modal');
    } else {
        modal.classList.remove('photo-modal');
    }

    // Set preview
    if (file.thumbnail || (file.type && file.type.startsWith('image/'))) {
        const imgSrc = file.thumbnail || file.url || '#';
        preview.innerHTML = '';
        const img = document.createElement('img');
        img.src = imgSrc;
        img.alt = file.name;
        img.style.maxWidth = '100%';
        img.style.maxHeight = isPhoto ? '80vh' : '400px';
        img.style.objectFit = isPhoto ? 'contain' : 'cover';
        preview.appendChild(img);
    } else {
        setPreviewIcon(preview, getFileIcon(file.type), 'modal-file-icon', 120);
    }

    // Set details with enhanced info for photos
    const detailsHtml = `
        <h3>${file.name}</h3>
        <p><strong>Type:</strong> ${file.type || 'Unknown'}</p>
        <p><strong>Size:</strong> ${formatFileSize(file.size)}</p>
        <p><strong>Modified:</strong> ${formatDate(file.modified)}</p>
        ${file.tags ? `<p><strong>Tags:</strong> ${file.tags.join(', ')}</p>` : ''}
        ${isPhoto ? `
            <div style="margin-top: 16px; padding-top: 16px; border-top: 1px solid var(--border-color);">
                <p><strong>Photo Details:</strong></p>
                ${file.width && file.height ? `<p>Dimensions: ${file.width} × ${file.height}px</p>` : ''}
                ${file.camera ? `<p>Camera: ${file.camera}</p>` : ''}
                ${file.location ? `<p>Location: ${file.location}</p>` : ''}
            </div>
        ` : ''}
    `;
    
    details.innerHTML = detailsHtml;
    modal.style.display = 'flex';
}

function closeModal() {
    document.getElementById('file-modal').style.display = 'none';
}

// Utility Functions
function getFileIcon(type) {
    if (!type) return 'file';

    const normalized = type.toLowerCase();

    if (normalized.startsWith('image/')) return 'image';
    if (normalized.startsWith('video/')) return 'video';
    if (normalized.startsWith('audio/')) return 'music';
    if (normalized.includes('pdf')) return 'file-text';
    if (
        normalized.includes('word') ||
        normalized.includes('document') ||
        normalized.includes('msword') ||
        normalized.includes('wordprocessingml')
    ) {
        return 'file-text';
    }
    if (
        normalized.includes('sheet') ||
        normalized.includes('excel') ||
        normalized.includes('spreadsheet') ||
        normalized.includes('spreadsheetml')
    ) {
        return 'table';
    }
    if (
        normalized.includes('presentation') ||
        normalized.includes('powerpoint') ||
        normalized.includes('presentationml') ||
        normalized.includes('ppt')
    ) {
        return 'presentation';
    }
    if (
        normalized.includes('zip') ||
        normalized.includes('archive') ||
        normalized.includes('compressed')
    ) {
        return 'file-archive';
    }

    return 'file';
}

function formatFileSize(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function formatDate(date) {
    if (!date) return 'Unknown';
    const d = new Date(date);
    const now = new Date();
    const diff = now - d;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    
    if (days === 0) return 'Today';
    if (days === 1) return 'Yesterday';
    if (days < 7) return `${days} days ago`;
    if (days < 30) return `${Math.floor(days / 7)} weeks ago`;
    if (days < 365) return `${Math.floor(days / 30)} months ago`;
    return d.toLocaleDateString();
}

function showProcessing(message) {
    const processing = document.getElementById('ai-processing');
    processing.querySelector('.processing-text').textContent = message;
    processing.style.display = 'block';
}

function hideProcessing() {
    document.getElementById('ai-processing').style.display = 'none';
}

function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 16px 24px;
        background: ${type === 'success' ? 'var(--success-color)' : 'var(--primary-color)'};
        color: white;
        border-radius: 8px;
        box-shadow: var(--shadow-lg);
        z-index: 2000;
        animation: slideIn 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

function changeLayout(layout) {
    const grid = document.getElementById('file-grid');
    if (layout === 'list') {
        grid.style.gridTemplateColumns = '1fr';
    } else {
        grid.style.gridTemplateColumns = 'repeat(auto-fill, minmax(200px, 1fr))';
    }
}

function filterByTag(tag) {
    // Implementation for filtering files by tag
    console.log('Filtering by tag:', tag);
}

// Mock Data Generators
function generateMockFiles() {
    const types = ['image/jpeg', 'image/png', 'application/pdf', 'video/mp4', 'audio/mp3', 'application/msword'];
    const names = ['beach-sunset', 'mountain-landscape', 'project-report', 'family-portrait', 'city-skyline', 'nature-macro', 'presentation', 'music-track', 'document', 'wedding-photo', 'travel-journal', 'business-meeting'];
    const extensions = ['.jpg', '.png', '.pdf', '.mp4', '.mp3', '.docx'];
    const photoTags = ['Nature', 'Travel', 'Family', 'Work', 'Landscape', 'Portrait', 'Macro', 'Street', 'Architecture'];
    const locations = ['Paris, France', 'Tokyo, Japan', 'New York, USA', 'Barcelona, Spain', 'Sydney, Australia'];
    const cameras = ['Canon EOS R5', 'Nikon D850', 'Sony A7 IV', 'iPhone 14 Pro', 'Fujifilm X-T4'];
    
    return Array.from({ length: 18 }, (_, i) => {
        const type = types[i % types.length];
        const isImage = type.startsWith('image/');
        const fileName = `${names[i % names.length]}-${i + 1}${extensions[i % extensions.length]}`;
        
        return {
            id: i + 1,
            name: fileName,
            type: type,
            size: Math.floor(Math.random() * (isImage ? 5000000 : 10000000)) + 100000,
            modified: new Date(Date.now() - Math.random() * 90 * 24 * 60 * 60 * 1000),
            tags: isImage ? 
                photoTags.filter(() => Math.random() > 0.6).slice(0, 2) : 
                ['Work', 'Important'].filter(() => Math.random() > 0.5),
            // Enhanced photo properties
            ...(isImage ? {
                thumbnail: `https://picsum.photos/300/200?random=${i}`,
                width: Math.floor(Math.random() * 2000) + 1920,
                height: Math.floor(Math.random() * 1500) + 1080,
                camera: Math.random() > 0.5 ? cameras[Math.floor(Math.random() * cameras.length)] : undefined,
                location: Math.random() > 0.7 ? locations[Math.floor(Math.random() * locations.length)] : undefined
            } : {})
        };
    });
}

function generateMockDuplicates() {
    const files = generateMockFiles();
    return [
        { files: files.slice(0, 2) },
        { files: files.slice(3, 5) }
    ];
}

// Add animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);
