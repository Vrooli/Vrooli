// ZenNotes - Mindful Note Taking App

const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__'];
const DEFAULT_API_SUFFIX = '/api';
const ICON_DEFAULTS = { size: 20, strokeWidth: 1.8 };

let lucideReady = Boolean(window.lucide);
const API_BASE = resolveApiBase();

window.addEventListener('load', () => {
    if (window.lucide) {
        lucideReady = true;
        refreshStaticIcons();
    }
});


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
    return fetch(joinApiUrl(path), options);
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

class ZenNotes {
    constructor() {
        this.currentNote = null;
        this.notes = [];
        this.autoSaveTimer = null;
        this.analysisTimer = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadNotes();
        this.startAutoSave();
        this.applyZenAnimations();
        refreshStaticIcons();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.zen-nav-item').forEach(btn => {
            btn.addEventListener('click', (e) => this.handleNavigation(e));
        });

        // Editor
        const noteContent = document.getElementById('noteContent');
        const noteTitle = document.getElementById('noteTitle');
        
        noteContent.addEventListener('input', () => {
            this.updateWordCount();
            this.scheduleAnalysis();
        });
        
        noteTitle.addEventListener('input', () => {
            this.scheduleAutoSave();
        });

        // Toolbar
        document.querySelectorAll('.zen-tool[data-format]').forEach(btn => {
            btn.addEventListener('click', (e) => this.applyFormat(e.target.dataset.format));
        });

        // Actions
        document.getElementById('saveBtn').addEventListener('click', () => this.saveNote());
        document.getElementById('shareBtn').addEventListener('click', () => this.shareNote());
        document.getElementById('zenModeBtn').addEventListener('click', () => this.toggleZenMode());
        document.getElementById('newNoteBtn').addEventListener('click', () => this.createNewNote());
        
        // Search
        document.getElementById('searchInput').addEventListener('input', (e) => {
            this.searchNotes(e.target.value);
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch(e.key) {
                    case 's':
                        e.preventDefault();
                        this.saveNote();
                        break;
                    case 'n':
                        e.preventDefault();
                        this.createNewNote();
                        break;
                    case 'f':
                        e.preventDefault();
                        document.getElementById('searchInput').focus();
                        break;
                    case 'z':
                        if (e.shiftKey) {
                            e.preventDefault();
                            this.toggleZenMode();
                        }
                        break;
                }
            }
        });
    }

    handleNavigation(e) {
        const btn = e.currentTarget;
        const view = btn.dataset.view;
        
        // Update active state
        document.querySelectorAll('.zen-nav-item').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        
        // Load appropriate view
        switch(view) {
            case 'all':
                this.showAllNotes();
                break;
            case 'today':
                this.showTodayNotes();
                break;
            case 'insights':
                this.showInsights();
                break;
            case 'connections':
                this.showConnections();
                break;
            case 'archive':
                this.showArchive();
                break;
        }
    }

    async loadNotes() {
        try {
            const data = await apiFetchJson('/notes');
            this.notes = Array.isArray(data) ? data : [];
            this.renderNotesList();
        } catch (error) {
            console.error('Failed to load notes:', error);
        }
    }

    async saveNote(options = {}) {
        const title = document.getElementById('noteTitle').value || 'Untitled';
        const content = document.getElementById('noteContent').value;
        
        if (!content.trim()) return;
        
        const noteData = {
            id: this.currentNote?.id,
            title,
            content,
            user_id: 'default-user',
            updated_at: new Date().toISOString()
        };
        
        try {
            const endpoint = noteData.id ? `/notes/${noteData.id}` : '/notes';
            const method = noteData.id ? 'PUT' : 'POST';

            const savedNote = await apiFetchJson(endpoint, buildJsonRequest(method, noteData));

            if (!savedNote) {
                throw new Error('API did not return saved note');
            }

            this.currentNote = savedNote;
            this.updateSaveStatus('All changes saved');
            if (!options.silent) {
                this.showZenNotification('Note saved mindfully');
            }

            this.analyzeNote();
            if (!options.skipReload) {
                this.loadNotes();
            }
        } catch (error) {
            console.error('Failed to save note:', error);
            this.updateSaveStatus('Failed to save');
        }
    }

    async analyzeNote() {
        if (!this.currentNote) return;
        
        const title = document.getElementById('noteTitle').value;
        const content = document.getElementById('noteContent').value;
        
        if (!content || content.length < 50) return;
        
        // Show loading state
        this.showAnalysisLoading();
        
        try {
            const analysis = await apiFetchJson('/notes/analyze', buildJsonRequest('POST', {
                note_id: this.currentNote.id,
                title,
                content,
                user_id: 'default-user',
                analysis_depth: 'comprehensive'
            }));

            if (analysis) {
                this.renderAnalysis(analysis);
            }
        } catch (error) {
            console.error('Failed to analyze note:', error);
            // Fallback to local analysis
            this.performLocalAnalysis(content);
        }
    }

    renderAnalysis(analysis) {
        // Summary
        const summaryEl = document.getElementById('aiSummary');
        if (analysis.summary) {
            summaryEl.textContent = analysis.summary;
            this.fadeIn(summaryEl.parentElement);
        }
        
        // Key Topics
        const topicsEl = document.getElementById('keyTopics');
        if (analysis.topics && analysis.topics.length) {
            topicsEl.innerHTML = analysis.topics.map(topic => 
                `<span class="zen-tag">${topic}</span>`
            ).join('');
            this.fadeIn(topicsEl.parentElement);
        }
        
        // Action Items
        const actionEl = document.getElementById('actionItems');
        if (analysis.action_items && analysis.action_items.length) {
            actionEl.innerHTML = analysis.action_items.map(item => `
                <div class="action-item-zen">
                    <div class="action-checkbox" onclick="this.classList.toggle('checked')"></div>
                    <div>
                        ${item.task}
                        ${item.assignee ? `<span style="color: var(--text-secondary); font-size: 0.85rem;"> - ${item.assignee}</span>` : ''}
                        ${item.deadline ? `<span style="color: var(--zen-gold); font-size: 0.85rem;"> (${item.deadline})</span>` : ''}
                    </div>
                </div>
            `).join('');
            this.fadeIn(actionEl.parentElement);
        }
        
        // Sentiment
        const sentimentEl = document.getElementById('sentiment');
        if (analysis.sentiment) {
            const iconName = this.getSentimentIcon(analysis.sentiment);
            const moodText = this.getSentimentText(analysis.sentiment);
            sentimentEl.innerHTML = `
                <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <span class="icon" data-icon="${iconName}"></span>
                    <span>${moodText}</span>
                </div>
            `;
            refreshStaticIcons(sentimentEl);
            this.fadeIn(sentimentEl.parentElement);
        }
        
        // Related Notes
        const relatedEl = document.getElementById('relatedNotes');
        if (analysis.related_notes && analysis.related_notes.length) {
            relatedEl.innerHTML = analysis.related_notes.map(note => `
                <div style="padding: 0.5rem 0; border-bottom: 1px solid var(--zen-stone); cursor: pointer;" 
                     onclick="zenNotes.loadNote('${note.id}')">
                    <div style="font-weight: 500;">${note.title}</div>
                    <div style="font-size: 0.85rem; color: var(--text-secondary);">
                        Score: ${(note.score * 100).toFixed(0)}%
                    </div>
                </div>
            `).join('');
            this.fadeIn(relatedEl.parentElement);
        }
        
        // AI Suggestions
        const suggestionsEl = document.getElementById('aiSuggestions');
        if (analysis.deep_insights) {
            const suggestions = [];
            
            if (analysis.deep_insights.opportunities?.length) {
                suggestions.push(...analysis.deep_insights.opportunities.map(o => 
                    `<div style="padding: 0.5rem; background: rgba(125, 163, 161, 0.1); border-radius: 8px; margin-bottom: 0.5rem;">
                        <span class="icon" data-icon="Lightbulb"></span> ${o}
                    </div>`
                ));
            }
            
            if (analysis.deep_insights.risks?.length) {
                suggestions.push(...analysis.deep_insights.risks.map(r => 
                    `<div style="padding: 0.5rem; background: rgba(212, 165, 116, 0.1); border-radius: 8px; margin-bottom: 0.5rem;">
                        <span class="icon" data-icon="AlertTriangle"></span> ${r}
                    </div>`
                ));
            }
            
            if (analysis.deep_insights.creative_suggestions?.length) {
                suggestions.push(...analysis.deep_insights.creative_suggestions.map(s => 
                    `<div style="padding: 0.5rem; background: rgba(107, 142, 111, 0.1); border-radius: 8px; margin-bottom: 0.5rem;">
                        <span class="icon" data-icon="Leaf"></span> ${s}
                    </div>`
                ));
            }
            
            if (suggestions.length) {
                suggestionsEl.innerHTML = suggestions.join('');
                refreshStaticIcons(suggestionsEl);
                this.fadeIn(suggestionsEl.parentElement);
            }
        }
    }

    performLocalAnalysis(content) {
        // Simple local analysis as fallback
        const words = content.split(/\s+/).filter(w => w.length > 0);
        const sentences = content.split(/[.!?]+/).filter(s => s.trim().length > 0);
        
        // Extract potential action items (lines starting with -, *, TODO, etc.)
        const actionPattern = /^[-*]\s+(.+)|^TODO:?\s+(.+)|^TASK:?\s+(.+)/gmi;
        const actions = [...content.matchAll(actionPattern)].map(m => m[1] || m[2] || m[3]);
        
        // Extract topics (common nouns, capitalized words)
        const topics = [...new Set(
            words.filter(w => w.length > 4 && /^[A-Z]/.test(w))
        )].slice(0, 5);
        
        // Simple sentiment (based on positive/negative word counts)
        const positiveWords = ['good', 'great', 'excellent', 'happy', 'success', 'achieved'];
        const negativeWords = ['bad', 'problem', 'issue', 'failed', 'difficult', 'challenge'];
        
        const positiveCount = words.filter(w => 
            positiveWords.some(p => w.toLowerCase().includes(p))
        ).length;
        
        const negativeCount = words.filter(w => 
            negativeWords.some(n => w.toLowerCase().includes(n))
        ).length;
        
        const sentiment = positiveCount > negativeCount ? 'positive' : 
                         negativeCount > positiveCount ? 'negative' : 'neutral';
        
        // Render local analysis
        this.renderAnalysis({
            summary: `${sentences.length} sentences, ${words.length} words`,
            topics: topics,
            action_items: actions.map(a => ({ task: a })),
            sentiment: sentiment
        });
    }

    getSentimentIcon(sentiment = '') {
        const icons = {
            positive: 'Smile',
            negative: 'Frown',
            neutral: 'Meh',
            focused: 'Target',
            creative: 'Palette',
            analytical: 'Search'
        };
        return icons[sentiment.toLowerCase()] || 'Smile';
    }

    getSentimentText(sentiment) {
        const texts = {
            positive: 'Positive & Uplifting',
            negative: 'Challenging & Reflective',
            neutral: 'Balanced & Thoughtful',
            focused: 'Focused & Productive',
            creative: 'Creative & Inspired',
            analytical: 'Analytical & Structured'
        };
        return texts[sentiment.toLowerCase()] || 'Mindful & Present';
    }

    applyFormat(format) {
        const textarea = document.getElementById('noteContent');
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        const selectedText = textarea.value.substring(start, end);
        
        let replacement = '';
        switch(format) {
            case 'heading':
                replacement = `## ${selectedText}`;
                break;
            case 'bold':
                replacement = `**${selectedText}**`;
                break;
            case 'italic':
                replacement = `*${selectedText}*`;
                break;
            case 'list':
                replacement = `- ${selectedText}`;
                break;
            case 'checkbox':
                replacement = `- [ ] ${selectedText}`;
                break;
            case 'quote':
                replacement = `> ${selectedText}`;
                break;
            case 'code':
                replacement = `\`${selectedText}\``;
                break;
            case 'link':
                const url = prompt('Enter URL:');
                if (url) replacement = `[${selectedText}](${url})`;
                else return;
                break;
        }
        
        textarea.value = textarea.value.substring(0, start) + replacement + textarea.value.substring(end);
        textarea.focus();
        textarea.setSelectionRange(start, start + replacement.length);
    }

    toggleZenMode() {
        document.body.classList.toggle('zen-mode');
        const isZenMode = document.body.classList.contains('zen-mode');
        
        if (isZenMode) {
            // Hide sidebars
            document.querySelector('.sidebar-zen').style.display = 'none';
            document.querySelector('.insights-zen').style.display = 'none';
            document.querySelector('.app-zen').style.gridTemplateColumns = '1fr';
            
            // Focus on editor
            document.getElementById('noteContent').focus();
            
            this.showZenNotification('Zen mode activated - Focus on your thoughts');
        } else {
            // Restore layout
            document.querySelector('.sidebar-zen').style.display = 'flex';
            document.querySelector('.insights-zen').style.display = 'flex';
            document.querySelector('.app-zen').style.gridTemplateColumns = '280px 1fr 320px';
            
            this.showZenNotification('Returning to normal view');
        }
    }

    createNewNote() {
        this.currentNote = null;
        document.getElementById('noteTitle').value = '';
        document.getElementById('noteContent').value = '';
        this.updateWordCount();
        document.getElementById('noteTitle').focus();
        this.showZenNotification('New canvas ready');
    }

    async searchNotes(query) {
        if (!query || query.length < 2) {
            this.renderNotesList();
            return;
        }
        
        try {
            const results = await apiFetchJson(`/notes/search?q=${encodeURIComponent(query)}`);
            if (Array.isArray(results)) {
                this.renderNotesList(results);
            } else if (Array.isArray(results?.results)) {
                this.renderNotesList(results.results);
            }
        } catch (error) {
            console.error('Search failed:', error);
        }
    }

    renderNotesList(notes = this.notes) {
        const modal = document.getElementById('notesListModal');
        const list = document.getElementById('notesList');
        
        if (!notes || notes.length === 0) {
            list.innerHTML = '<p style="color: var(--text-secondary);">No notes found. Start writing!</p>';
            return;
        }
        
        list.innerHTML = notes.map(note => `
            <div class="note-item" style="padding: 1rem; margin-bottom: 0.5rem; background: var(--zen-mist); border-radius: 8px; cursor: pointer;"
                 onclick="zenNotes.loadNote('${note.id}')">
                <h4 style="margin-bottom: 0.25rem;">${note.title || 'Untitled'}</h4>
                <p style="font-size: 0.9rem; color: var(--text-secondary);">
                    ${note.content ? note.content.substring(0, 100) + '...' : 'Empty note'}
                </p>
                <span style="font-size: 0.8rem; color: var(--text-secondary);">
                    ${new Date(note.updated_at).toLocaleDateString()}
                </span>
            </div>
        `).join('');
    }

    async loadNote(noteId) {
        try {
            const note = await apiFetchJson(`/notes/${noteId}`);
            if (!note) {
                throw new Error('Note not found');
            }

            this.currentNote = note;
            document.getElementById('noteTitle').value = note.title || '';
            document.getElementById('noteContent').value = note.content || '';
            this.updateWordCount();

            document.getElementById('notesListModal').style.display = 'none';
            this.analyzeNote();
        } catch (error) {
            console.error('Failed to load note:', error);
        }
    }

    showAllNotes() {
        document.getElementById('notesListModal').style.display = 'block';
        this.renderNotesList();
    }

    showTodayNotes() {
        const today = new Date().toDateString();
        const todayNotes = this.notes.filter(note => 
            new Date(note.updated_at).toDateString() === today
        );
        document.getElementById('notesListModal').style.display = 'block';
        this.renderNotesList(todayNotes);
    }

    showInsights() {
        // Focus on insights panel
        document.querySelector('.insights-zen').scrollIntoView({ behavior: 'smooth' });
        this.showZenNotification('AI insights panel focused');
    }

    showConnections() {
        // Show related notes
        const relatedSection = document.getElementById('relatedNotes');
        relatedSection.parentElement.scrollIntoView({ behavior: 'smooth' });
    }

    showArchive() {
        const archivedNotes = this.notes.filter(note => note.archived);
        document.getElementById('notesListModal').style.display = 'block';
        this.renderNotesList(archivedNotes);
    }

    shareNote() {
        if (!this.currentNote) {
            this.showZenNotification('Save your note first');
            return;
        }
        
        const shareUrl = `${window.location.origin}/note/${this.currentNote.id}`;
        navigator.clipboard.writeText(shareUrl);
        this.showZenNotification('Link copied to clipboard');
    }

    updateWordCount() {
        const content = document.getElementById('noteContent').value;
        const words = content.split(/\s+/).filter(w => w.length > 0).length;
        const readingTime = Math.ceil(words / 200);
        
        document.getElementById('wordCount').textContent = `${words} words`;
        document.getElementById('readingTime').textContent = `~${readingTime} min read`;
    }

    updateSaveStatus(status) {
        document.getElementById('lastSaved').textContent = status;
    }

    startAutoSave() {
        setInterval(() => {
            if (this.currentNote && document.getElementById('noteContent').value) {
                this.saveNote({ skipReload: true, silent: true });
            }
        }, 30000); // Auto-save every 30 seconds
    }

    scheduleAutoSave() {
        clearTimeout(this.autoSaveTimer);
        this.autoSaveTimer = setTimeout(() => this.saveNote({ skipReload: true, silent: true }), 2000);
    }

    scheduleAnalysis() {
        clearTimeout(this.analysisTimer);
        this.analysisTimer = setTimeout(() => this.analyzeNote(), 3000);
    }

    showAnalysisLoading() {
        document.getElementById('aiSummary').innerHTML = '<div class="zen-loading"></div> Analyzing...';
    }

    showZenNotification(message) {
        const notification = document.createElement('div');
        notification.className = 'zen-notification';
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            bottom: 2rem;
            left: 50%;
            transform: translateX(-50%);
            background: var(--zen-ink);
            color: white;
            padding: 1rem 2rem;
            border-radius: 24px;
            font-size: 0.9rem;
            z-index: 1000;
            animation: slideUp 0.3s ease, fadeOut 0.3s ease 2s forwards;
        `;
        
        document.body.appendChild(notification);
        setTimeout(() => notification.remove(), 3000);
    }

    fadeIn(element) {
        element.classList.add('fade-in');
    }

    applyZenAnimations() {
        // Add subtle animations to cards
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('fade-in');
                }
            });
        });
        
        document.querySelectorAll('.insight-card').forEach(card => {
            observer.observe(card);
        });
    }
}

// Initialize app
const zenNotes = new ZenNotes();

// Add required CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideUp {
        from { transform: translate(-50%, 20px); opacity: 0; }
        to { transform: translate(-50%, 0); opacity: 1; }
    }
    
    @keyframes fadeOut {
        to { opacity: 0; }
    }
`;
document.head.appendChild(style);
