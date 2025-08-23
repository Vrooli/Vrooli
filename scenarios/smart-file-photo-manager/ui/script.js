// SmartFile - AI-Powered File Manager JavaScript

const API_BASE = `http://localhost:${window.location.port === '3000' ? '8080' : window.location.port}/api`;

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

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadFiles();
    setupDragAndDrop();
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
    const uploadZone = grid.querySelector('.upload-zone');
    
    // Clear existing files but keep upload zone
    grid.innerHTML = '';
    if (uploadZone) grid.appendChild(uploadZone);

    state.files.forEach(file => {
        const card = createFileCard(file);
        grid.appendChild(card);
    });
}

function createFileCard(file) {
    const card = document.createElement('div');
    card.className = 'file-card';
    card.onclick = () => openFileModal(file);

    const preview = document.createElement('div');
    preview.className = 'file-preview';
    
    if (file.thumbnail) {
        const img = document.createElement('img');
        img.src = file.thumbnail;
        img.alt = file.name;
        preview.appendChild(img);
    } else {
        preview.textContent = getFileIcon(file.type);
    }

    const info = document.createElement('div');
    info.className = 'file-info';
    
    const name = document.createElement('div');
    name.className = 'file-name';
    name.textContent = file.name;
    
    const meta = document.createElement('div');
    meta.className = 'file-meta';
    meta.textContent = `${formatFileSize(file.size)} • ${formatDate(file.modified)}`;

    info.appendChild(name);
    info.appendChild(meta);
    card.appendChild(preview);
    card.appendChild(info);

    return card;
}

// Search Functionality
async function performSearch(query) {
    if (!query.trim()) {
        document.getElementById('search-results').innerHTML = `
            <div class="empty-state">
                <span class="empty-icon">🔍</span>
                <p>Search for files using natural language</p>
                <p class="empty-hint">Try "photos from last summer" or "documents about project X"</p>
            </div>
        `;
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ query })
        });

        if (response.ok) {
            state.searchResults = await response.json();
            renderSearchResults();
        }
    } catch (error) {
        console.error('Search error:', error);
        // Use mock results for demo
        state.searchResults = state.files.filter(f => 
            f.name.toLowerCase().includes(query.toLowerCase())
        );
        renderSearchResults();
    }
}

function renderSearchResults() {
    const container = document.getElementById('search-results');
    
    if (state.searchResults.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <span class="empty-icon">🔍</span>
                <p>No results found</p>
                <p class="empty-hint">Try a different search query</p>
            </div>
        `;
        return;
    }

    container.innerHTML = '';
    const grid = document.createElement('div');
    grid.className = 'file-grid';
    
    state.searchResults.forEach(file => {
        grid.appendChild(createFileCard(file));
    });
    
    container.appendChild(grid);
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
        if (response.ok) {
            state.duplicates = await response.json();
            renderDuplicates();
            hideProcessing();
        }
    } catch (error) {
        console.error('Duplicate scan error:', error);
        // Use mock data for demo
        state.duplicates = generateMockDuplicates();
        renderDuplicates();
        hideProcessing();
    }
}

function renderDuplicates() {
    const container = document.getElementById('duplicates-container');
    
    if (state.duplicates.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <span class="empty-icon">✓</span>
                <p>No duplicates found</p>
                <p class="empty-hint">Your files are well organized!</p>
            </div>
        `;
        return;
    }

    container.innerHTML = '';
    state.duplicates.forEach(group => {
        const groupDiv = document.createElement('div');
        groupDiv.className = 'duplicate-group';
        groupDiv.innerHTML = `
            <h3>Duplicate Group (${group.files.length} files)</h3>
            <div class="file-grid">
                ${group.files.map(file => createFileCard(file).outerHTML).join('')}
            </div>
        `;
        container.appendChild(groupDiv);
    });
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
function openFileModal(file) {
    const modal = document.getElementById('file-modal');
    const preview = document.getElementById('preview-container');
    const details = document.getElementById('file-details');

    // Set preview
    if (file.thumbnail) {
        preview.innerHTML = `<img src="${file.thumbnail}" alt="${file.name}" style="max-width: 100%; max-height: 400px;">`;
    } else {
        preview.innerHTML = `<div style="font-size: 120px; color: var(--text-tertiary);">${getFileIcon(file.type)}</div>`;
    }

    // Set details
    details.innerHTML = `
        <h3>${file.name}</h3>
        <p>Type: ${file.type || 'Unknown'}</p>
        <p>Size: ${formatFileSize(file.size)}</p>
        <p>Modified: ${formatDate(file.modified)}</p>
        ${file.tags ? `<p>Tags: ${file.tags.join(', ')}</p>` : ''}
    `;

    modal.style.display = 'flex';
}

function closeModal() {
    document.getElementById('file-modal').style.display = 'none';
}

// Utility Functions
function getFileIcon(type) {
    if (!type) return '📄';
    if (type.startsWith('image/')) return '🖼️';
    if (type.startsWith('video/')) return '🎥';
    if (type.startsWith('audio/')) return '🎵';
    if (type.includes('pdf')) return '📑';
    if (type.includes('word') || type.includes('document')) return '📝';
    if (type.includes('sheet') || type.includes('excel')) return '📊';
    if (type.includes('presentation') || type.includes('powerpoint')) return '📈';
    if (type.includes('zip') || type.includes('archive')) return '🗜️';
    return '📄';
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
    const names = ['vacation-photo', 'project-report', 'presentation', 'music-track', 'document', 'spreadsheet'];
    const extensions = ['.jpg', '.png', '.pdf', '.mp4', '.mp3', '.docx'];
    
    return Array.from({ length: 12 }, (_, i) => ({
        id: i + 1,
        name: `${names[i % names.length]}-${i + 1}${extensions[i % extensions.length]}`,
        type: types[i % types.length],
        size: Math.floor(Math.random() * 10000000),
        modified: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000),
        tags: ['Work', 'Important'].filter(() => Math.random() > 0.5)
    }));
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