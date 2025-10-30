// SmartNotes - Main JavaScript
// Get API port from the backend (injected dynamically if needed)
const API_URL = window.location.hostname === 'localhost' 
    ? `http://localhost:${window.API_PORT || '17001'}` 
    : '/api';

let currentNote = null;
let notes = [];
let folders = [];
let tags = [];
let templates = [];
let autoSaveTimeout = null;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    loadNotes();
    loadFolders();
    loadTags();
    setupEventListeners();
    startAutoSave();
});

function initializeApp() {
    console.log('🚀 SmartNotes initialized');
    updateStats();
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
        const response = await fetch(`${API_URL}/api/notes`);
        notes = await response.json();
        displayNotes(notes);
        updateStats();
    } catch (error) {
        console.error('Failed to load notes:', error);
        showNotification('Failed to load notes', 'error');
    }
}

function displayNotes(notesToDisplay) {
    const notesList = document.getElementById('notesList');
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
        return;
    }
    
    notesToDisplay.forEach(note => {
        const noteCard = createNoteCard(note);
        notesList.appendChild(noteCard);
    });
}

function createNoteCard(note) {
    const card = document.createElement('div');
    card.className = 'note-card';
    card.dataset.noteId = note.id;
    
    const preview = note.content ? 
        note.content.substring(0, 150).replace(/[#*`]/g, '') : 
        'Empty note';
    
    const date = new Date(note.updated_at);
    const dateStr = date.toLocaleDateString('en-US', { 
        month: 'short', 
        day: 'numeric' 
    });
    
    card.innerHTML = `
        <div class="note-card-title">
            ${note.is_pinned ? '📌 ' : ''}
            ${note.is_favorite ? '⭐ ' : ''}
            ${note.title || 'Untitled'}
        </div>
        <div class="note-card-preview">${preview}...</div>
        <div class="note-card-meta">
            <div class="note-card-tags">
                ${(note.tags || []).slice(0, 3).map(tag => 
                    `<span class="note-card-tag">${tag}</span>`
                ).join('')}
            </div>
            <span>${dateStr}</span>
        </div>
    `;
    
    card.addEventListener('click', () => loadNote(note.id));
    
    return card;
}

async function loadNote(noteId) {
    try {
        const response = await fetch(`${API_URL}/api/notes/${noteId}`);
        const note = await response.json();
        
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
    document.getElementById('noteTitle').value = note.title || '';
    document.getElementById('noteContent').value = note.content || '';
    
    // Update pin/favorite buttons
    document.getElementById('pinBtn').innerHTML = 
        note.is_pinned ? '📌' : '📍';
    document.getElementById('favoriteBtn').innerHTML = 
        note.is_favorite ? '⭐' : '☆';
    
    updateWordCount();
}

async function createNewNote() {
    const note = {
        title: 'Untitled Note',
        content: '',
        user_id: 'default-user',
        content_type: 'markdown'
    };
    
    try {
        const response = await fetch(`${API_URL}/api/notes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(note)
        });
        
        const newNote = await response.json();
        currentNote = newNote;
        
        await loadNotes();
        loadNote(newNote.id);
        
        showNotification('New note created', 'success');
        
    } catch (error) {
        console.error('Failed to create note:', error);
        showNotification('Failed to create note', 'error');
    }
}

async function saveNote() {
    if (!currentNote) return;
    
    const title = document.getElementById('noteTitle').value;
    const content = document.getElementById('noteContent').value;
    
    try {
        const response = await fetch(`${API_URL}/api/notes/${currentNote.id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                ...currentNote,
                title,
                content
            })
        });
        
        currentNote = await response.json();
        document.getElementById('lastSaved').textContent = 'Saved';
        
        // Refresh notes list
        await loadNotes();
        
    } catch (error) {
        console.error('Failed to save note:', error);
        document.getElementById('lastSaved').textContent = 'Failed to save';
    }
}

async function deleteCurrentNote() {
    if (!currentNote) return;
    
    if (!confirm('Are you sure you want to delete this note?')) return;
    
    try {
        await fetch(`${API_URL}/api/notes/${currentNote.id}`, {
            method: 'DELETE'
        });
        
        currentNote = null;
        document.getElementById('noteTitle').value = '';
        document.getElementById('noteContent').value = '';
        
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
        saveNote();
    }, 2000);
}

function startAutoSave() {
    setInterval(() => {
        if (currentNote && document.getElementById('lastSaved').textContent.includes('Unsaved')) {
            saveNote();
        }
    }, 30000); // Auto-save every 30 seconds
}

function togglePin() {
    if (!currentNote) return;
    currentNote.is_pinned = !currentNote.is_pinned;
    document.getElementById('pinBtn').innerHTML = 
        currentNote.is_pinned ? '📌' : '📍';
    saveNote();
}

function toggleFavorite() {
    if (!currentNote) return;
    currentNote.is_favorite = !currentNote.is_favorite;
    document.getElementById('favoriteBtn').innerHTML = 
        currentNote.is_favorite ? '⭐' : '☆';
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
        const response = await fetch(`${API_URL}/api/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                query,
                user_id: 'default-user',
                limit: 20
            })
        });
        
        const results = await response.json();
        displayNotes(results.results);
        
        document.getElementById('sectionTitle').textContent = 
            `Search Results (${results.count})`;
        
    } catch (error) {
        console.error('Search failed:', error);
        showNotification('Search failed', 'error');
    }
}

// Folders
async function loadFolders() {
    try {
        const response = await fetch(`${API_URL}/api/folders`);
        folders = await response.json();
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
        await fetch(`${API_URL}/api/folders`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                name,
                color,
                user_id: 'default-user'
            })
        });
        
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
        const response = await fetch(`${API_URL}/api/tags`);
        tags = await response.json();
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
        const response = await fetch(`${API_URL}/api/templates`);
        templates = await response.json();
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