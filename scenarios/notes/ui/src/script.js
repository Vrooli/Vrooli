// SmartNotes - Main JavaScript

const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const DEFAULT_API_SUFFIX = '/api';
const ICON_DEFAULTS = { size: 20, strokeWidth: 1.8 };

const API_BASE = resolveApiBase();
let currentNote = null;
let notes = [];
let folders = [];
let tags = [];
let templates = [];
let autoSaveTimeout = null;
let lucideReady = Boolean(window.lucide);

// Initialize
window.addEventListener('load', () => {
    if (window.lucide) {
        lucideReady = true;
        refreshStaticIcons();
    }
});

document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    refreshStaticIcons();
    initializeSidebarControls();
    loadNotes();
    loadFolders();
    loadTags();
    setupEventListeners();
    startAutoSave();
});

function initializeApp() {
    console.log('[SmartNotes] UI initialized');
    updateStats();
    updatePinButtonState(false);
    updateFavoriteButtonState(false);
    updateSidebarToggleIcon();
    toggleEditorPlaceholder(true);
}

function toggleEditorPlaceholder(show) {
    const editorSection = document.querySelector('.note-editor-section');
    const placeholder = document.getElementById('notePlaceholder');
    if (!editorSection || !placeholder) {
        return;
    }

    const isVisible = Boolean(show);
    editorSection.classList.toggle('show-placeholder', isVisible);
    placeholder.setAttribute('aria-hidden', isVisible ? 'false' : 'true');

    if (isVisible) {
        const titleInput = document.getElementById('noteTitle');
        const contentInput = document.getElementById('noteContent');
        if (titleInput) {
            titleInput.value = '';
        }
        if (contentInput) {
            contentInput.value = '';
        }
        const wordCount = document.getElementById('wordCount');
        const readingTime = document.getElementById('readingTime');
        const lastSaved = document.getElementById('lastSaved');
        if (wordCount) {
            wordCount.textContent = '0 words';
        }
        if (readingTime) {
            readingTime.textContent = '0 min read';
        }
        if (lastSaved) {
            lastSaved.textContent = 'Select a note to begin';
        }

        refreshStaticIcons(placeholder);
    }
}

function stripTrailingSlash(value = '') {
    return typeof value === 'string' ? value.replace(/\/+$/u, '') : '';
}

function ensureLeadingSlash(value = '') {
    if (typeof value !== 'string' || !value) {
        return '/';
    }
    return value.startsWith('/') ? value : `/${value}`;
}

function normalizePort(value) {
    if (typeof value === 'number' && Number.isFinite(value) && value > 0) {
        return String(value);
    }
    if (typeof value === 'string') {
        const trimmed = value.trim();
        return /^\d+$/.test(trimmed) ? trimmed : undefined;
    }
    return undefined;
}

function pickProxyEndpoint(info) {
    if (!info) {
        return undefined;
    }

    const candidates = [];
    const enqueue = (candidate) => {
        if (!candidate) {
            return;
        }
        if (typeof candidate === 'string') {
            const trimmed = candidate.trim();
            if (trimmed) {
                candidates.push(trimmed);
            }
            return;
        }
        if (typeof candidate === 'object') {
            enqueue(candidate.url || candidate.href || candidate.target || candidate.path);
            if (Array.isArray(candidate.endpoints)) {
                candidate.endpoints.forEach(enqueue);
            }
        }
    };

    enqueue(info);

    if (typeof info === 'object') {
        enqueue(info.apiBase || info.api_path || info.apiPath);
        enqueue(info.api);
        enqueue(info.baseUrl || info.baseURL);
        enqueue(info.apiUrl || info.apiURL);
        enqueue(info.url);
        enqueue(info.target);
        if (Array.isArray(info.endpoints)) {
            info.endpoints.forEach(enqueue);
        }
    }

    return candidates.find(Boolean);
}

function resolveProxyEndpoint() {
    if (typeof window === 'undefined') {
        return undefined;
    }

    for (const key of PROXY_INFO_KEYS) {
        const info = window[key];
        if (!info) {
            continue;
        }

        const candidate = pickProxyEndpoint(info);
        if (candidate) {
            return candidate;
        }

        const ports = typeof info === 'object' ? info.ports : undefined;
        const rawPort = ports && (ports.api?.port ?? ports.api);
        const normalizedPort = normalizePort(rawPort);
        if (normalizedPort) {
            const protocol = window.location?.protocol === 'https:' ? 'https:' : 'http:';
            const hostname = window.location?.hostname || 'localhost';
            return `${protocol}//${hostname}:${normalizedPort}`;
        }
    }

    return undefined;
}

function ensureApiSuffix(value) {
    const trimmed = stripTrailingSlash((value || '').trim());
    if (!trimmed) {
        return DEFAULT_API_SUFFIX;
    }
    if (trimmed.endsWith(DEFAULT_API_SUFFIX)) {
        return trimmed;
    }
    if (/^https?:\/\//i.test(trimmed)) {
        return `${trimmed}${DEFAULT_API_SUFFIX}`;
    }
    return ensureLeadingSlash(trimmed === '/' ? DEFAULT_API_SUFFIX : `${trimmed}${DEFAULT_API_SUFFIX}`);
}

function resolveApiBase() {
    if (typeof window === 'undefined') {
        return DEFAULT_API_SUFFIX;
    }

    const proxyBase = resolveProxyEndpoint();
    if (proxyBase) {
        if (/^https?:\/\//i.test(proxyBase)) {
            return ensureApiSuffix(proxyBase);
        }
        const origin = window.location?.origin;
        if (origin) {
            return ensureApiSuffix(`${stripTrailingSlash(origin)}${ensureLeadingSlash(proxyBase)}`);
        }
        return ensureApiSuffix(proxyBase);
    }

    const envApiUrl = typeof window.API_URL === 'string' ? window.API_URL.trim() : '';
    if (envApiUrl) {
        return ensureApiSuffix(envApiUrl);
    }

    const apiPort = normalizePort(window.API_PORT);
    if (apiPort) {
        const protocol = window.location?.protocol === 'https:' ? 'https:' : 'http:';
        const host = window.location?.hostname || 'localhost';
        return ensureApiSuffix(`${protocol}//${host}:${apiPort}`);
    }

    const origin = window.location?.origin;
    if (origin) {
        return ensureApiSuffix(origin);
    }

    return DEFAULT_API_SUFFIX;
}

function joinApiUrl(path = '/') {
    const base = stripTrailingSlash(API_BASE);
    const suffix = ensureLeadingSlash(path);
    return `${base}${suffix}`;
}

function apiFetch(path, options = {}) {
    const target = joinApiUrl(path);
    return fetch(target, options);
}

async function apiFetchJson(path, options = {}) {
    const response = await apiFetch(path, options);
    const raw = await response.text();

    if (!response.ok) {
        const message = raw || `Request failed (${response.status})`;
        throw new Error(message);
    }

    if (!raw) {
        return null;
    }

    try {
        return JSON.parse(raw);
    } catch (error) {
        throw new Error(`Failed to parse response JSON: ${error instanceof Error ? error.message : String(error)}`);
    }
}

function buildJsonRequest(method, payload, headers = {}) {
    const options = {
        method,
        headers: {
            'Content-Type': 'application/json',
            ...headers,
        },
    };

    if (payload !== undefined) {
        options.body = JSON.stringify(payload);
    }

    return options;
}

function applyIcon(element, iconName, overrides = {}) {
    if (!element) {
        return;
    }

    const resolvedIcon = iconName || element.dataset.icon;
    if (!resolvedIcon) {
        element.textContent = overrides.fallback || '';
        return;
    }

    element.dataset.icon = resolvedIcon;

    const lucideApi = window.lucide;
    const iconNode = lucideApi?.icons?.[resolvedIcon];
    if (lucideReady && iconNode && typeof lucideApi?.createElement === 'function') {
        const size = Number(overrides.size ?? element.dataset.iconSize ?? ICON_DEFAULTS.size);
        const strokeWidth = Number(overrides.strokeWidth ?? element.dataset.iconStroke ?? ICON_DEFAULTS.strokeWidth);
        const svg = lucideApi.createElement(iconNode, { name: resolvedIcon, size, strokeWidth });
        svg.setAttribute('aria-hidden', 'true');
        element.replaceChildren(svg);
    } else if (!element.childElementCount && !element.textContent.trim()) {
        element.textContent = overrides.fallback || '';
    }
}

function refreshStaticIcons(root = document) {
    if (!root) {
        return;
    }
    const elements = root.querySelectorAll('[data-icon]');
    elements.forEach((node) => applyIcon(node, node.dataset.icon));
}

function updatePinButtonState(isPinned) {
    const iconTarget = document.querySelector('#pinBtn .icon');
    applyIcon(iconTarget, isPinned ? 'Pin' : 'PinOff');
}

function updateFavoriteButtonState(isFavorite) {
    const iconTarget = document.querySelector('#favoriteBtn .icon');
    applyIcon(iconTarget, isFavorite ? 'Star' : 'StarOff');
}

const sidebarState = {
    mediaQuery: typeof window !== 'undefined' ? window.matchMedia('(max-width: 768px)') : { matches: false, addEventListener: () => {} },
};

function initializeSidebarControls() {
    const appContainer = document.getElementById('appContainer');
    const overlay = document.getElementById('sidebarOverlay');
    const toggle = document.getElementById('sidebarToggle');

    if (!appContainer || !toggle) {
        return;
    }

    const handleViewportChange = () => {
        if (sidebarState.mediaQuery.matches) {
            setSidebarCollapsed(false, { skipIcon: true });
            setSidebarOpen(false, { skipIcon: true });
            overlay?.classList.add('hidden');
            document.body.classList.remove('sidebar-open');
        } else {
            setSidebarOpen(false, { skipIcon: true });
            overlay?.classList.add('hidden');
            document.body.classList.remove('sidebar-open');
        }
        updateSidebarToggleIcon();
    };

    toggle.addEventListener('click', () => {
        if (sidebarState.mediaQuery.matches) {
            const isOpen = appContainer.classList.contains('sidebar-open');
            setSidebarOpen(!isOpen);
        } else {
            const isCollapsed = appContainer.classList.contains('sidebar-collapsed');
            setSidebarCollapsed(!isCollapsed);
        }
        updateSidebarToggleIcon();
    });

    overlay?.addEventListener('click', () => setSidebarOpen(false));

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && sidebarState.mediaQuery.matches) {
            setSidebarOpen(false);
        }
    });

    if (typeof sidebarState.mediaQuery.addEventListener === 'function') {
        sidebarState.mediaQuery.addEventListener('change', handleViewportChange);
    }

    handleViewportChange();
}

function setSidebarOpen(shouldOpen, { skipIcon } = {}) {
    const appContainer = document.getElementById('appContainer');
    const overlay = document.getElementById('sidebarOverlay');
    if (!appContainer) {
        return;
    }

    if (shouldOpen) {
        appContainer.classList.add('sidebar-open');
        overlay?.classList.remove('hidden');
        document.body.classList.add('sidebar-open');
    } else {
        appContainer.classList.remove('sidebar-open');
        overlay?.classList.add('hidden');
        document.body.classList.remove('sidebar-open');
    }

    if (!skipIcon) {
        updateSidebarToggleIcon();
    }
}

function setSidebarCollapsed(collapsed, { skipIcon } = {}) {
    const appContainer = document.getElementById('appContainer');
    if (!appContainer) {
        return;
    }

    if (collapsed) {
        appContainer.classList.add('sidebar-collapsed');
    } else {
        appContainer.classList.remove('sidebar-collapsed');
    }

    if (!skipIcon) {
        updateSidebarToggleIcon();
    }
}

function updateSidebarToggleIcon() {
    const toggleIcon = document.querySelector('#sidebarToggle .icon');
    const appContainer = document.getElementById('appContainer');
    if (!toggleIcon || !appContainer) {
        return;
    }

    const iconName = sidebarState.mediaQuery.matches
        ? (appContainer.classList.contains('sidebar-open') ? 'PanelLeft' : 'Menu')
        : (appContainer.classList.contains('sidebar-collapsed') ? 'PanelLeftOpen' : 'PanelLeft');

    applyIcon(toggleIcon, iconName);
}


// Event Listeners
function setupEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-item').forEach(item => {
        item.addEventListener('click', handleNavigation);
    });
    
    // New Note
    document.getElementById('newNoteBtn').addEventListener('click', createNewNote);
    
    // Search
    document.getElementById('searchBtn').addEventListener('click', performSearch);
    document.getElementById('searchInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
    
    // Editor
    document.getElementById('noteTitle').addEventListener('input', handleTitleChange);
    document.getElementById('noteContent').addEventListener('input', handleContentChange);
    
    // Editor Actions
    document.getElementById('pinBtn').addEventListener('click', togglePin);
    document.getElementById('favoriteBtn').addEventListener('click', toggleFavorite);
    document.getElementById('deleteBtn').addEventListener('click', deleteCurrentNote);
    
    // Editor Modes
    document.getElementById('editModeBtn').addEventListener('click', () => setEditorMode('edit'));
    document.getElementById('previewModeBtn').addEventListener('click', () => setEditorMode('preview'));
    document.getElementById('splitModeBtn').addEventListener('click', () => setEditorMode('split'));
    
    // Toolbar
    document.querySelectorAll('.toolbar-btn').forEach(btn => {
        btn.addEventListener('click', handleFormat);
    });
    
    // Folders
    document.querySelector('.add-folder-btn').addEventListener('click', showFolderModal);
    
    // AI Panel
    document.getElementById('chatSend').addEventListener('click', sendChatMessage);
    document.getElementById('chatInput').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') sendChatMessage();
    });
}

// Navigation
function handleNavigation(e) {
    const view = e.currentTarget.dataset.view;
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });
    e.currentTarget.classList.add('active');
    
    loadNotesByView(view);
    document.getElementById('sectionTitle').textContent = 
        e.currentTarget.textContent.trim();
}

// Notes Management
async function loadNotes() {
    try {
        const data = await apiFetchJson('/notes');
        notes = Array.isArray(data) ? data : [];
        displayNotes(notes);
        updateStats();
    } catch (error) {
        console.error('Failed to load notes:', error);
        showNotification('Failed to load notes', 'error');
    }
}

function displayNotes(notesToDisplay) {
    const notesList = document.getElementById('notesList');
    if (!notesList) {
        return;
    }

    notesList.innerHTML = '';

    if (!notesToDisplay || notesToDisplay.length === 0) {
        notesList.innerHTML = `
            <div class="empty-state">
                <p>No notes yet</p>
                <button onclick="createNewNote()" class="btn btn-primary">
                    Create your first note
                </button>
            </div>
        `;
        refreshStaticIcons(notesList);
        return;
    }

    notesToDisplay.forEach((note) => {
        const noteCard = createNoteCard(note);
        notesList.appendChild(noteCard);
    });
}

function createNoteCard(note) {
    const card = document.createElement('div');
    card.className = 'note-card';
    card.dataset.noteId = note.id;

    const titleRow = document.createElement('div');
    titleRow.className = 'note-card-title';

    const flags = document.createElement('div');
    flags.className = 'note-card-flags';

    if (note.is_pinned) {
        const pinIcon = document.createElement('span');
        pinIcon.className = 'icon flag-icon';
        applyIcon(pinIcon, 'Pin');
        flags.appendChild(pinIcon);
    }

    if (note.is_favorite) {
        const favoriteIcon = document.createElement('span');
        favoriteIcon.className = 'icon flag-icon';
        applyIcon(favoriteIcon, 'Star');
        flags.appendChild(favoriteIcon);
    }

    const title = document.createElement('span');
    title.className = 'note-card-title-text';
    title.textContent = note.title || 'Untitled';

    titleRow.appendChild(flags);
    titleRow.appendChild(title);

    const preview = document.createElement('div');
    preview.className = 'note-card-preview';
    const previewText = note.content ? note.content.replace(/[#*`]/g, '').slice(0, 150) : 'Empty note';
    preview.textContent = previewText.length === 150 ? `${previewText}...` : previewText;

    const meta = document.createElement('div');
    meta.className = 'note-card-meta';

    const tagsWrapper = document.createElement('div');
    tagsWrapper.className = 'note-card-tags';
    (note.tags || []).slice(0, 3).forEach((tag) => {
        const tagElement = document.createElement('span');
        tagElement.className = 'note-card-tag';
        tagElement.textContent = tag;
        tagsWrapper.appendChild(tagElement);
    });

    const dateLabel = document.createElement('span');
    const updatedAt = note.updated_at ? new Date(note.updated_at) : new Date();
    dateLabel.textContent = updatedAt.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
    });

    meta.appendChild(tagsWrapper);
    meta.appendChild(dateLabel);

    card.appendChild(titleRow);
    card.appendChild(preview);
    card.appendChild(meta);

    card.addEventListener('click', () => loadNote(note.id));

    refreshStaticIcons(card);
    return card;
}

async function loadNote(noteId) {
    try {
        const note = await apiFetchJson(`/notes/${noteId}`);

        if (!note) {
            throw new Error('Note not found');
        }

        currentNote = note;
        displayNote(note);

        // Highlight active card
        document.querySelectorAll('.note-card').forEach(card => {
            card.classList.remove('active');
        });
        document.querySelector(`[data-note-id="${noteId}"]`)?.classList.add('active');

        // Get AI suggestions
        getAISuggestions(note.content);

    } catch (error) {
        console.error('Failed to load note:', error);
        showNotification('Failed to load note', 'error');
    }
}

function displayNote(note) {
    toggleEditorPlaceholder(false);
    document.getElementById('noteTitle').value = note.title || '';
    document.getElementById('noteContent').value = note.content || '';

    updatePinButtonState(Boolean(note.is_pinned));
    updateFavoriteButtonState(Boolean(note.is_favorite));

    updateWordCount();

    const lastSaved = document.getElementById('lastSaved');
    if (lastSaved) {
        lastSaved.textContent = 'All changes saved';
    }
}

async function createNewNote() {
    const note = {
        title: 'Untitled Note',
        content: '',
        user_id: 'default-user',
        content_type: 'markdown'
    };
    
    try {
        const newNote = await apiFetchJson('/notes', buildJsonRequest('POST', note));
        if (!newNote) {
            throw new Error('API did not return a note');
        }
        currentNote = newNote;
        
        await loadNotes();
        await loadNote(newNote.id);
        
        showNotification('New note created', 'success');
        
    } catch (error) {
        console.error('Failed to create note:', error);
        showNotification('Failed to create note', 'error');
    }
}

async function saveNote(options = {}) {
    if (!currentNote) {
        return;
    }

    const title = document.getElementById('noteTitle').value;
    const content = document.getElementById('noteContent').value;

    try {
        const payload = {
            ...currentNote,
            title,
            content
        };

        const savedNote = await apiFetchJson(`/notes/${currentNote.id}`, buildJsonRequest('PUT', payload));
        if (!savedNote) {
            throw new Error('API did not return saved note');
        }

        currentNote = savedNote;
        document.getElementById('lastSaved').textContent = 'Saved';
        updatePinButtonState(Boolean(currentNote.is_pinned));
        updateFavoriteButtonState(Boolean(currentNote.is_favorite));

        if (!options.skipRefresh) {
            await loadNotes();
        }

    } catch (error) {
        console.error('Failed to save note:', error);
        document.getElementById('lastSaved').textContent = 'Failed to save';
    }
}

async function deleteCurrentNote() {
    if (!currentNote) return;
    
    if (!confirm('Are you sure you want to delete this note?')) return;
    
    try {
        const response = await apiFetch(`/notes/${currentNote.id}`, { method: 'DELETE' });
        if (!response.ok) {
            const message = await response.text();
            throw new Error(message || 'Failed to delete');
        }
        
        currentNote = null;
        document.getElementById('noteTitle').value = '';
        document.getElementById('noteContent').value = '';
        toggleEditorPlaceholder(true);
        
        await loadNotes();
        showNotification('Note deleted', 'success');
        
    } catch (error) {
        console.error('Failed to delete note:', error);
        showNotification('Failed to delete note', 'error');
    }
}

// Editor Functions
function handleTitleChange(e) {
    if (currentNote) {
        currentNote.title = e.target.value;
        scheduleAutoSave();
    }
}

function handleContentChange(e) {
    if (currentNote) {
        currentNote.content = e.target.value;
        scheduleAutoSave();
        updateWordCount();
    }
}

function updateWordCount() {
    const content = document.getElementById('noteContent').value;
    const words = content.trim().split(/\s+/).filter(w => w.length > 0).length;
    const readingTime = Math.max(1, Math.round(words / 200));
    
    document.getElementById('wordCount').textContent = `${words} words`;
    document.getElementById('readingTime').textContent = `${readingTime} min read`;
}

function scheduleAutoSave() {
    document.getElementById('lastSaved').textContent = 'Unsaved changes';
    
    if (autoSaveTimeout) {
        clearTimeout(autoSaveTimeout);
    }
    
    autoSaveTimeout = setTimeout(() => {
        saveNote({ skipRefresh: true });
    }, 2000);
}

function startAutoSave() {
    setInterval(() => {
        if (currentNote && document.getElementById('lastSaved').textContent.includes('Unsaved')) {
            saveNote({ skipRefresh: true });
        }
    }, 30000); // Auto-save every 30 seconds
}

function togglePin() {
    if (!currentNote) return;
    currentNote.is_pinned = !currentNote.is_pinned;
    updatePinButtonState(currentNote.is_pinned);
    saveNote();
}

function toggleFavorite() {
    if (!currentNote) return;
    currentNote.is_favorite = !currentNote.is_favorite;
    updateFavoriteButtonState(currentNote.is_favorite);
    saveNote();
}

function setEditorMode(mode) {
    const content = document.getElementById('noteContent');
    const preview = document.getElementById('notePreview');
    
    document.querySelectorAll('.mode-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    
    switch(mode) {
        case 'edit':
            content.classList.remove('hidden');
            preview.classList.add('hidden');
            document.getElementById('editModeBtn').classList.add('active');
            break;
        case 'preview':
            content.classList.add('hidden');
            preview.classList.remove('hidden');
            preview.innerHTML = renderMarkdown(content.value);
            document.getElementById('previewModeBtn').classList.add('active');
            break;
        case 'split':
            content.classList.remove('hidden');
            preview.classList.remove('hidden');
            content.style.width = '50%';
            preview.style.width = '50%';
            preview.innerHTML = renderMarkdown(content.value);
            document.getElementById('splitModeBtn').classList.add('active');
            break;
    }
}

function renderMarkdown(text) {
    // Simple markdown rendering
    return text
        .replace(/^### (.*$)/gim, '<h3>$1</h3>')
        .replace(/^## (.*$)/gim, '<h2>$1</h2>')
        .replace(/^# (.*$)/gim, '<h1>$1</h1>')
        .replace(/^\* (.+)/gim, '<li>$1</li>')
        .replace(/\*\*(.+)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.+)\*/g, '<em>$1</em>')
        .replace(/\n\n/g, '</p><p>')
        .replace(/^/, '<p>')
        .replace(/$/, '</p>')
        .replace(/<li>(.+)<\/li>/g, '<ul><li>$1</li></ul>')
        .replace(/`([^`]+)`/g, '<code>$1</code>')
        .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
}

function handleFormat(e) {
    const format = e.currentTarget.dataset.format;
    const textarea = document.getElementById('noteContent');
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const text = textarea.value;
    const selectedText = text.substring(start, end);
    
    let formattedText = '';
    let cursorOffset = 0;
    
    switch(format) {
        case 'bold':
            formattedText = `**${selectedText || 'text'}**`;
            cursorOffset = selectedText ? 4 : 2;
            break;
        case 'italic':
            formattedText = `*${selectedText || 'text'}*`;
            cursorOffset = selectedText ? 2 : 1;
            break;
        case 'h1':
            formattedText = `# ${selectedText || 'Heading'}`;
            cursorOffset = 2;
            break;
        case 'h2':
            formattedText = `## ${selectedText || 'Heading'}`;
            cursorOffset = 3;
            break;
        case 'h3':
            formattedText = `### ${selectedText || 'Heading'}`;
            cursorOffset = 4;
            break;
        case 'ul':
            formattedText = `\n* ${selectedText || 'List item'}`;
            cursorOffset = 3;
            break;
        case 'ol':
            formattedText = `\n1. ${selectedText || 'List item'}`;
            cursorOffset = 4;
            break;
        case 'checkbox':
            formattedText = `\n- [ ] ${selectedText || 'Task'}`;
            cursorOffset = 6;
            break;
        case 'code':
            formattedText = `\`${selectedText || 'code'}\``;
            cursorOffset = selectedText ? 2 : 1;
            break;
        case 'quote':
            formattedText = `\n> ${selectedText || 'Quote'}`;
            cursorOffset = 3;
            break;
        case 'link':
            formattedText = `[${selectedText || 'text'}](url)`;
            cursorOffset = selectedText ? selectedText.length + 3 : 1;
            break;
    }
    
    textarea.value = text.substring(0, start) + formattedText + text.substring(end);
    textarea.selectionStart = start + cursorOffset;
    textarea.selectionEnd = start + cursorOffset;
    textarea.focus();
    
    handleContentChange({ target: textarea });
}

// Search
async function performSearch() {
    const query = document.getElementById('searchInput').value;
    if (!query) return;
    
    try {
        const results = await apiFetchJson('/search', buildJsonRequest('POST', {
            query,
            user_id: 'default-user',
            limit: 20
        }));
        const matches = Array.isArray(results?.results) ? results.results : [];
        displayNotes(matches);
        
        document.getElementById('sectionTitle').textContent = 
            `Search Results (${results?.count ?? matches.length})`;
        
    } catch (error) {
        console.error('Search failed:', error);
        showNotification('Search failed', 'error');
    }
}

// Folders
async function loadFolders() {
    try {
        const data = await apiFetchJson('/folders');
        folders = Array.isArray(data) ? data : [];
        displayFolders(folders);
    } catch (error) {
        console.error('Failed to load folders:', error);
    }
}

function displayFolders(folders) {
    const foldersList = document.getElementById('foldersList');
    foldersList.innerHTML = '';
    
    folders.forEach(folder => {
        const item = document.createElement('div');
        item.className = 'folder-item';
        item.innerHTML = `
            <span class="folder-icon" style="background: ${folder.color}"></span>
            <span>${folder.name}</span>
        `;
        item.addEventListener('click', () => loadNotesByFolder(folder.id));
        foldersList.appendChild(item);
    });
}

function showFolderModal() {
    document.getElementById('folderModal').classList.remove('hidden');
    document.getElementById('modalOverlay').classList.remove('hidden');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.add('hidden');
    document.getElementById('modalOverlay').classList.add('hidden');
}

async function createFolder() {
    const name = document.getElementById('folderName').value;
    const color = document.querySelector('.color-option.selected')?.dataset.color || '#6366f1';

    if (!name) return;

    try {
        const response = await apiFetch('/folders', buildJsonRequest('POST', {
            name,
            color,
            user_id: 'default-user'
        }));
        if (!response.ok) {
            const message = await response.text();
            throw new Error(message || 'Failed to create folder');
        }

        document.getElementById('folderName').value = '';
        await loadFolders();
        closeModal('folderModal');
        showNotification('Folder created', 'success');

    } catch (error) {
        console.error('Failed to create folder:', error);
        showNotification('Failed to create folder', 'error');
    }
}

// Tags
async function loadTags() {
    try {
        const data = await apiFetchJson('/tags');
        tags = Array.isArray(data) ? data : [];
        displayTags(tags);
    } catch (error) {
        console.error('Failed to load tags:', error);
    }
}

function displayTags(tags) {
    const tagsList = document.getElementById('tagsList');
    tagsList.innerHTML = '';
    
    tags.forEach(tag => {
        const item = document.createElement('span');
        item.className = 'tag-item';
        item.style.borderLeft = `3px solid ${tag.color}`;
        item.textContent = tag.name;
        item.addEventListener('click', () => loadNotesByTag(tag.name));
        tagsList.appendChild(item);
    });
}

// AI Features
async function getAISuggestions(content) {
    if (!content) return;
    
    // This would call the smart-suggestions n8n workflow
    // For now, show placeholder suggestions
    const suggestions = [
        'Consider adding a summary section',
        'Link to related note: "Machine Learning Basics"',
        'Add tags: #productivity #notes',
        'Template suggestion: Meeting Notes'
    ];
    
    displaySuggestions(suggestions);
}

function displaySuggestions(suggestions) {
    const suggestionsList = document.getElementById('suggestions');
    suggestionsList.innerHTML = '';
    
    suggestions.forEach(suggestion => {
        const item = document.createElement('div');
        item.className = 'suggestion-item';
        item.textContent = suggestion;
        item.addEventListener('click', () => applySuggestion(suggestion));
        suggestionsList.appendChild(item);
    });
}

function applySuggestion(suggestion) {
    showNotification(`Applied: ${suggestion}`, 'success');
}

async function sendChatMessage() {
    const input = document.getElementById('chatInput');
    const message = input.value.trim();
    if (!message) return;
    
    // Add user message
    addChatMessage(message, 'user');
    input.value = '';
    
    // Simulate AI response
    setTimeout(() => {
        const response = 'I can help you organize your notes better. Try using tags to categorize your content.';
        addChatMessage(response, 'ai');
    }, 1000);
}

function addChatMessage(message, sender) {
    const messagesContainer = document.getElementById('chatMessages');
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${sender}`;
    messageDiv.innerHTML = `
        <div class="chat-message-bubble">${message}</div>
    `;
    messagesContainer.appendChild(messageDiv);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// View Management
function loadNotesByView(view) {
    switch(view) {
        case 'notes':
            loadNotes();
            break;
        case 'favorites':
            displayNotes(notes.filter(n => n.is_favorite));
            break;
        case 'archived':
            displayNotes(notes.filter(n => n.is_archived));
            break;
        case 'templates':
            loadTemplates();
            break;
    }
}

function loadNotesByFolder(folderId) {
    displayNotes(notes.filter(n => n.folder_id === folderId));
}

function loadNotesByTag(tagName) {
    displayNotes(notes.filter(n => n.tags && n.tags.includes(tagName)));
}

async function loadTemplates() {
    try {
        const data = await apiFetchJson('/templates');
        templates = Array.isArray(data) ? data : [];
        displayTemplates(templates);
    } catch (error) {
        console.error('Failed to load templates:', error);
    }
}

function displayTemplates(templates) {
    const notesList = document.getElementById('notesList');
    notesList.innerHTML = '<h3>Available Templates</h3>';
    
    templates.forEach(template => {
        const card = document.createElement('div');
        card.className = 'note-card';
        card.innerHTML = `
            <div class="note-card-title">${template.name}</div>
            <div class="note-card-preview">${template.description}</div>
            <div class="note-card-meta">
                <span>${template.category}</span>
                <button onclick="useTemplate('${template.id}')">Use Template</button>
            </div>
        `;
        notesList.appendChild(card);
    });
}

// Stats
function updateStats() {
    const totalNotes = notes.length;
    const totalWords = notes.reduce((sum, note) => 
        sum + (note.word_count || 0), 0);
    
    document.getElementById('totalNotes').textContent = totalNotes;
    document.getElementById('totalWords').textContent = 
        totalWords > 1000 ? `${(totalWords/1000).toFixed(1)}k` : totalWords;
}

// Utilities
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        bottom: 20px;
        right: 20px;
        padding: 12px 20px;
        background: ${type === 'error' ? '#ef4444' : '#10b981'};
        color: white;
        border-radius: 8px;
        box-shadow: 0 4px 6px rgba(0,0,0,0.1);
        z-index: 1000;
        animation: slideIn 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    setTimeout(() => {
        notification.remove();
    }, 3000);
}

// Color picker for folders
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('color-option')) {
        document.querySelectorAll('.color-option').forEach(opt => {
            opt.classList.remove('selected');
        });
        e.target.classList.add('selected');
    }
});
