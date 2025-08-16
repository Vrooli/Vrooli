// ZenNotes - Mindful Note Taking App
const API_URL = window.API_URL || `http://localhost:${window.SERVICE_PORT || 8950}`;

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
            const response = await fetch(`${API_URL}/api/notes`);
            if (response.ok) {
                this.notes = await response.json();
                this.renderNotesList();
            }
        } catch (error) {
            console.error('Failed to load notes:', error);
        }
    }

    async saveNote() {
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
            const endpoint = noteData.id ? `/api/notes/${noteData.id}` : '/api/notes';
            const method = noteData.id ? 'PUT' : 'POST';
            
            const response = await fetch(`${API_URL}${endpoint}`, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(noteData)
            });
            
            if (response.ok) {
                const savedNote = await response.json();
                this.currentNote = savedNote;
                this.updateSaveStatus('All changes saved');
                this.showZenNotification('Note saved mindfully');
                
                // Trigger AI analysis
                this.analyzeNote();
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
            const response = await fetch(`${API_URL}/api/notes/analyze`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    note_id: this.currentNote.id,
                    title,
                    content,
                    user_id: 'default-user',
                    analysis_depth: 'comprehensive'
                })
            });
            
            if (response.ok) {
                const analysis = await response.json();
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
            const moodEmoji = this.getSentimentEmoji(analysis.sentiment);
            const moodText = this.getSentimentText(analysis.sentiment);
            sentimentEl.innerHTML = `
                <div style="display: flex; align-items: center; gap: 0.5rem;">
                    <span style="font-size: 1.5rem;">${moodEmoji}</span>
                    <span>${moodText}</span>
                </div>
            `;
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
                        💡 ${o}
                    </div>`
                ));
            }
            
            if (analysis.deep_insights.risks?.length) {
                suggestions.push(...analysis.deep_insights.risks.map(r => 
                    `<div style="padding: 0.5rem; background: rgba(212, 165, 116, 0.1); border-radius: 8px; margin-bottom: 0.5rem;">
                        ⚠️ ${r}
                    </div>`
                ));
            }
            
            if (analysis.deep_insights.creative_suggestions?.length) {
                suggestions.push(...analysis.deep_insights.creative_suggestions.map(s => 
                    `<div style="padding: 0.5rem; background: rgba(107, 142, 111, 0.1); border-radius: 8px; margin-bottom: 0.5rem;">
                        🌱 ${s}
                    </div>`
                ));
            }
            
            if (suggestions.length) {
                suggestionsEl.innerHTML = suggestions.join('');
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

    getSentimentEmoji(sentiment) {
        const emojis = {
            positive: '😊',
            negative: '😔',
            neutral: '😌',
            focused: '🎯',
            creative: '🎨',
            analytical: '🔍'
        };
        return emojis[sentiment.toLowerCase()] || '😌';
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
            const response = await fetch(`${API_URL}/api/notes/search?q=${encodeURIComponent(query)}`);
            if (response.ok) {
                const results = await response.json();
                this.renderNotesList(results);
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
            const response = await fetch(`${API_URL}/api/notes/${noteId}`);
            if (response.ok) {
                const note = await response.json();
                this.currentNote = note;
                document.getElementById('noteTitle').value = note.title || '';
                document.getElementById('noteContent').value = note.content || '';
                this.updateWordCount();
                
                // Close modal if open
                document.getElementById('notesListModal').style.display = 'none';
                
                // Analyze the loaded note
                this.analyzeNote();
            }
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
                this.saveNote();
            }
        }, 30000); // Auto-save every 30 seconds
    }

    scheduleAutoSave() {
        clearTimeout(this.autoSaveTimer);
        this.autoSaveTimer = setTimeout(() => this.saveNote(), 2000);
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