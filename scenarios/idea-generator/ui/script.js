import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_STATE_KEY = '__ideaGeneratorBridgeInitialized';

if (typeof window !== 'undefined' && window.parent !== window && !window[BRIDGE_STATE_KEY]) {
    try {
        let parentOrigin;
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }

        initIframeBridgeChild({ parentOrigin, appId: 'idea-generator' });
        window[BRIDGE_STATE_KEY] = true;
    } catch (error) {
        console.warn('[idea-generator] iframe bridge bootstrap skipped', error);
    }
}

// Idea Generator UI JavaScript
class IdeaGeneratorApp {
    constructor() {
        this.currentCampaign = null; // Will be set when campaigns load
        this.ideas = [];
        this.campaigns = [];
        this.selectedColor = '#6366F1';
        this.currentIdeaId = null;
        
        this.init();
    }

    async init() {
        this.setupEventListeners();
        await this.loadCampaigns();
        this.updateCampaignTabs();
        this.loadIdeas();
    }

    setupEventListeners() {
        // Dice button animation and idea generation
        const diceButton = document.querySelector('.dice-button');
        if (diceButton) {
            diceButton.addEventListener('click', () => this.generateIdea());
        }

        // View toggle buttons
        document.querySelectorAll('.view-toggle').forEach(btn => {
            btn.addEventListener('click', (e) => this.switchView(e.target.dataset.view));
        });

        // Campaign form
        const campaignForm = document.getElementById('campaign-form');
        if (campaignForm) {
            campaignForm.addEventListener('submit', (e) => this.createCampaign(e));
        }

        // Color picker
        document.querySelectorAll('.color-option').forEach(option => {
            option.addEventListener('click', (e) => this.selectColor(e.target.dataset.color));
        });

        // File upload
        this.setupFileUpload();

        // Search functionality
        this.setupSearch();

        // Chat input
        const chatInput = document.getElementById('chat-input');
        if (chatInput) {
            chatInput.addEventListener('keypress', (e) => {
                if (e.key === 'Enter') this.sendChatMessage();
            });
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => this.handleKeyboard(e));
    }

    async generateIdea() {
        const diceButton = document.querySelector('.dice-button');
        const loadingOverlay = document.getElementById('loading-overlay');
        const contextInput = document.getElementById('context-input');
        const creativitySlider = document.getElementById('creativity-level');

        // Show loading state
        loadingOverlay.classList.add('active');
        diceButton.classList.add('generating');
        
        // Animate dice
        const dice = document.querySelector('.dice');
        dice.style.animation = 'spin 2s linear infinite';

        try {
            const response = await fetch('/api/ideas/generate', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    campaign_id: this.currentCampaign,
                    context: contextInput.value || '',
                    creativity_level: creativitySlider.value / 100
                })
            });

            if (!response.ok) {
                throw new Error('Failed to generate idea');
            }

            const newIdea = await response.json();
            this.addIdeaToBoard(newIdea);
            
            // Update campaign count
            const campaign = this.campaigns.find(c => c.id === this.currentCampaign);
            if (campaign) {
                campaign.ideas++;
                this.updateCampaignTabs();
            }

            // Show success animation
            this.showSuccessMessage('New idea generated!');

        } catch (error) {
            console.error('Error generating idea:', error);
            this.showErrorMessage('Failed to generate idea. Please try again.');
        } finally {
            // Hide loading state
            loadingOverlay.classList.remove('active');
            diceButton.classList.remove('generating');
            dice.style.animation = 'bounce 1s ease-in-out infinite';
        }
    }

    addIdeaToBoard(idea) {
        const ideasContainer = document.getElementById('ideas-container');
        const emptyState = ideasContainer.querySelector('.empty-state');
        
        // Remove empty state if it exists
        if (emptyState) {
            emptyState.remove();
        }

        // Create idea card
        const ideaCard = document.createElement('div');
        ideaCard.className = 'idea-card fade-in';
        ideaCard.style.setProperty('--campaign-color', this.getCampaignColor(this.currentCampaign));
        
        ideaCard.innerHTML = `
            <div class="idea-header">
                <h3 class="idea-title">${idea.title}</h3>
                <div class="idea-actions">
                    <button class="idea-action" onclick="app.refineIdea('${idea.id}')" title="Refine with AI">
                        <i class="fas fa-brain"></i>
                    </button>
                    <button class="idea-action" onclick="app.editIdea('${idea.id}')" title="Edit">
                        <i class="fas fa-edit"></i>
                    </button>
                    <button class="idea-action" onclick="app.deleteIdea('${idea.id}')" title="Delete">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
            <div class="idea-content">${idea.content}</div>
            <div class="idea-meta">
                <span class="idea-timestamp">${new Date().toLocaleDateString()}</span>
                <span class="idea-status ${idea.status || 'draft'}">${idea.status || 'draft'}</span>
            </div>
        `;

        // Add to ideas array
        this.ideas.push({
            ...idea,
            element: ideaCard,
            campaign: this.currentCampaign
        });

        ideasContainer.appendChild(ideaCard);
    }

    refineIdea(ideaId) {
        this.currentIdeaId = ideaId;
        const chatPanel = document.getElementById('chat-panel');
        chatPanel.classList.add('active');
        
        // Clear previous messages
        const chatMessages = document.getElementById('chat-messages');
        chatMessages.innerHTML = '';
        
        // Add welcome message
        this.addChatMessage('system', 'Select an agent type and start refining your idea!');
    }

    async sendChatMessage() {
        const chatInput = document.getElementById('chat-input');
        const message = chatInput.value.trim();
        const agentType = document.getElementById('agent-type').value;

        if (!message || !this.currentIdeaId) return;

        // Add user message
        this.addChatMessage('user', message);
        chatInput.value = '';

        // Add typing indicator
        this.addTypingIndicator();

        try {
            const response = await fetch('/api/chat/refine', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    idea_id: this.currentIdeaId,
                    agent_type: agentType,
                    message: message
                })
            });

            if (!response.ok) {
                throw new Error('Failed to get agent response');
            }

            const result = await response.json();
            
            // Remove typing indicator
            this.removeTypingIndicator();
            
            // Add agent response
            this.addChatMessage('agent', result.agent_response, agentType);

            // Add suggestions if provided
            if (result.suggestions && result.suggestions.length > 0) {
                this.addSuggestions(result.suggestions);
            }

        } catch (error) {
            this.removeTypingIndicator();
            this.addChatMessage('system', 'Sorry, I encountered an error. Please try again.');
            console.error('Chat error:', error);
        }
    }

    addChatMessage(type, message, agentType = null) {
        const chatMessages = document.getElementById('chat-messages');
        const messageDiv = document.createElement('div');
        messageDiv.className = `chat-message ${type}`;
        
        let avatar = '';
        if (type === 'agent') {
            const agentEmojis = {
                'revise': '✏️',
                'research': '🔍',
                'critique': '🎯',
                'expand': '🚀',
                'synthesize': '🧠',
                'validate': '✅',
                'auto': '🤖'
            };
            avatar = `<div class="message-avatar">${agentEmojis[agentType] || '🤖'}</div>`;
        } else if (type === 'user') {
            avatar = '<div class="message-avatar">👤</div>';
        }

        messageDiv.innerHTML = `
            ${avatar}
            <div class="message-content">
                <div class="message-text">${message}</div>
                <div class="message-time">${new Date().toLocaleTimeString()}</div>
            </div>
        `;

        chatMessages.appendChild(messageDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    addTypingIndicator() {
        const chatMessages = document.getElementById('chat-messages');
        const typingDiv = document.createElement('div');
        typingDiv.className = 'typing-indicator';
        typingDiv.innerHTML = `
            <div class="message-avatar">🤖</div>
            <div class="typing-dots">
                <span></span><span></span><span></span>
            </div>
        `;
        chatMessages.appendChild(typingDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    removeTypingIndicator() {
        const typingIndicator = document.querySelector('.typing-indicator');
        if (typingIndicator) {
            typingIndicator.remove();
        }
    }

    addSuggestions(suggestions) {
        const chatMessages = document.getElementById('chat-messages');
        const suggestionsDiv = document.createElement('div');
        suggestionsDiv.className = 'suggestions';
        
        const suggestionsList = suggestions.map(s => `<button class="suggestion-btn" onclick="app.applySuggestion('${s}')">${s}</button>`).join('');
        
        suggestionsDiv.innerHTML = `
            <div class="suggestions-label">Quick suggestions:</div>
            <div class="suggestions-list">${suggestionsList}</div>
        `;
        
        chatMessages.appendChild(suggestionsDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    applySuggestion(suggestion) {
        const chatInput = document.getElementById('chat-input');
        chatInput.value = suggestion;
        chatInput.focus();
    }

    async loadCampaigns() {
        try {
            const response = await fetch('/campaigns');
            if (response.ok) {
                const campaigns = await response.json();
                this.campaigns = campaigns.map(c => ({
                    id: c.id,
                    name: c.name,
                    description: c.description,
                    color: c.color || '#6366F1',
                    ideas: 0 // Will be updated when ideas load
                }));
                
                // Set the first campaign as current if we have campaigns
                if (this.campaigns.length > 0) {
                    this.currentCampaign = this.campaigns[0].id;
                }
            }
        } catch (error) {
            console.error('Failed to load campaigns:', error);
            // Create a default campaign if loading fails
            this.campaigns = [{
                id: 'default',
                name: 'Default Campaign',
                color: '#6366F1',
                ideas: 0
            }];
            this.currentCampaign = 'default';
        }
    }

    switchView(viewType) {
        document.querySelectorAll('.view-toggle').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.view === viewType);
        });
        
        const ideasContainer = document.getElementById('ideas-container');
        ideasContainer.className = `ideas-container view-${viewType}`;
    }

    async createCampaign(e) {
        e.preventDefault();
        
        const name = document.getElementById('campaign-name').value;
        const description = document.getElementById('campaign-description').value;
        
        try {
            const response = await fetch('/campaigns', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    name: name,
                    description: description,
                    color: this.selectedColor
                })
            });
            
            if (response.ok) {
                const newCampaign = await response.json();
                this.campaigns.push({
                    id: newCampaign.id,
                    name: newCampaign.name,
                    description: newCampaign.description,
                    color: newCampaign.color || this.selectedColor,
                    ideas: 0
                });
                
                this.updateCampaignTabs();
                this.closeCampaignModal();
                
                // Switch to new campaign
                this.switchCampaign(newCampaign.id);
                
                this.showSuccessMessage(`Campaign "${name}" created!`);
            }
        } catch (error) {
            console.error('Failed to create campaign:', error);
            this.showErrorMessage('Failed to create campaign');
        }
    }

    updateCampaignTabs() {
        const tabsContainer = document.querySelector('.tabs-container');
        const addBtn = tabsContainer.querySelector('.add-campaign-btn');
        
        // Remove existing tabs (except add button)
        document.querySelectorAll('.campaign-tab').forEach(tab => tab.remove());
        
        // Add campaign tabs
        this.campaigns.forEach(campaign => {
            const tab = document.createElement('div');
            tab.className = 'campaign-tab';
            tab.dataset.campaign = campaign.id;
            tab.style.setProperty('--campaign-color', campaign.color);
            
            if (campaign.id === this.currentCampaign) {
                tab.classList.add('active');
            }
            
            tab.innerHTML = `
                <span class="tab-name">${campaign.name}</span>
                <span class="tab-count">${campaign.ideas}</span>
            `;
            
            tab.addEventListener('click', () => this.switchCampaign(campaign.id));
            tabsContainer.insertBefore(tab, addBtn);
        });
    }

    switchCampaign(campaignId) {
        this.currentCampaign = campaignId;
        this.updateCampaignTabs();
        this.loadIdeas();
    }

    loadIdeas() {
        const ideasContainer = document.getElementById('ideas-container');
        ideasContainer.innerHTML = '';
        
        const campaignIdeas = this.ideas.filter(idea => idea.campaign === this.currentCampaign);
        
        if (campaignIdeas.length === 0) {
            ideasContainer.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">
                        <i class="fas fa-lightbulb"></i>
                    </div>
                    <h3>Ready to Brainstorm?</h3>
                    <p>Click the magic dice to generate your first AI-powered idea!</p>
                </div>
            `;
        } else {
            campaignIdeas.forEach(idea => {
                ideasContainer.appendChild(idea.element);
            });
        }
    }

    getCampaignColor(campaignId) {
        const campaign = this.campaigns.find(c => c.id === campaignId);
        return campaign ? campaign.color : '#6366F1';
    }

    selectColor(color) {
        this.selectedColor = color;
        document.querySelectorAll('.color-option').forEach(option => {
            option.classList.toggle('active', option.dataset.color === color);
        });
    }

    setupFileUpload() {
        const uploadArea = document.getElementById('upload-area');
        const fileInput = document.getElementById('file-input');
        
        if (!uploadArea || !fileInput) return;

        uploadArea.addEventListener('click', () => fileInput.click());
        
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('drag-over');
        });
        
        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('drag-over');
        });
        
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('drag-over');
            this.handleFiles(e.dataTransfer.files);
        });
        
        fileInput.addEventListener('change', (e) => {
            this.handleFiles(e.target.files);
        });
    }

    async handleFiles(files) {
        const uploadedFilesContainer = document.getElementById('uploaded-files');
        
        for (const file of files) {
            const fileDiv = document.createElement('div');
            fileDiv.className = 'uploaded-file';
            fileDiv.innerHTML = `
                <div class="file-info">
                    <i class="fas fa-file"></i>
                    <span class="file-name">${file.name}</span>
                    <span class="file-size">${this.formatFileSize(file.size)}</span>
                </div>
                <div class="file-status">Uploading...</div>
            `;
            
            uploadedFilesContainer.appendChild(fileDiv);
            
            try {
                await this.uploadFile(file, fileDiv);
            } catch (error) {
                console.error('Upload error:', error);
                fileDiv.querySelector('.file-status').textContent = 'Failed';
                fileDiv.classList.add('error');
            }
        }
    }

    async uploadFile(file, fileDiv) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('campaign_id', this.currentCampaign);
        formData.append('extract_context', 'true');

        const response = await fetch('/api/documents/upload', {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error('Upload failed');
        }

        const result = await response.json();
        fileDiv.querySelector('.file-status').textContent = 'Processed';
        fileDiv.classList.add('success');
        
        this.showSuccessMessage(`Document "${file.name}" uploaded successfully!`);
    }

    setupSearch() {
        const searchInput = document.getElementById('search-input');
        if (!searchInput) return;

        let searchTimeout;
        searchInput.addEventListener('input', (e) => {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => this.performSearch(e.target.value), 300);
        });
    }

    async performSearch(query) {
        if (!query.trim()) {
            document.getElementById('search-results').innerHTML = '';
            return;
        }

        try {
            const response = await fetch(`/api/search/semantic?query=${encodeURIComponent(query)}&campaign_id=${this.currentCampaign}`);
            if (!response.ok) throw new Error('Search failed');

            const results = await response.json();
            this.displaySearchResults(results.results);
        } catch (error) {
            console.error('Search error:', error);
            document.getElementById('search-results').innerHTML = '<p>Search failed. Please try again.</p>';
        }
    }

    displaySearchResults(results) {
        const searchResults = document.getElementById('search-results');
        
        if (results.length === 0) {
            searchResults.innerHTML = '<p>No results found.</p>';
            return;
        }

        const resultsHtml = results.map(result => `
            <div class="search-result" onclick="app.selectSearchResult('${result.id}')">
                <div class="result-title">${result.title}</div>
                <div class="result-content">${result.content.substring(0, 120)}...</div>
                <div class="result-meta">
                    <span class="result-type">${result.type}</span>
                    <span class="result-similarity">${Math.round(result.similarity * 100)}% match</span>
                </div>
            </div>
        `).join('');

        searchResults.innerHTML = resultsHtml;
    }

    handleKeyboard(e) {
        // Esc to close modals/panels
        if (e.key === 'Escape') {
            this.closeAllModals();
        }
        
        // Cmd/Ctrl + K to open search
        if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
            e.preventDefault();
            this.openSearch();
        }
        
        // Space to generate idea (when not in input)
        if (e.key === ' ' && e.target.tagName !== 'INPUT' && e.target.tagName !== 'TEXTAREA') {
            e.preventDefault();
            this.generateIdea();
        }
    }

    // Utility functions
    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    showSuccessMessage(message) {
        this.showToast(message, 'success');
    }

    showErrorMessage(message) {
        this.showToast(message, 'error');
    }

    showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type} fade-in`;
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.classList.add('fade-out');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    closeAllModals() {
        document.querySelectorAll('.modal, .chat-panel, .floating-search').forEach(el => {
            el.classList.remove('active');
        });
    }
}

// Global functions for HTML onclick handlers
function createNewCampaign() {
    document.getElementById('campaign-modal').classList.add('active');
}

function closeCampaignModal() {
    document.getElementById('campaign-modal').classList.remove('active');
    // Reset form
    document.getElementById('campaign-form').reset();
    app.selectColor('#6366F1');
}

function openDocumentUpload() {
    document.getElementById('document-modal').classList.add('active');
}

function closeDocumentUpload() {
    document.getElementById('document-modal').classList.remove('active');
}

function openSearch() {
    document.getElementById('floating-search').classList.add('active');
    document.getElementById('search-input').focus();
}

function closeSearch() {
    document.getElementById('floating-search').classList.remove('active');
}

function closeChat() {
    document.getElementById('chat-panel').classList.remove('active');
}

// Initialize app
const app = new IdeaGeneratorApp();

// Add toast styles
const toastStyles = `
.toast {
    position: fixed;
    top: 2rem;
    right: 2rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 0.75rem;
    padding: 1rem 1.5rem;
    box-shadow: 0 8px 24px var(--shadow-lg);
    z-index: 2000;
    font-weight: 500;
    min-width: 200px;
}

.toast-success {
    border-left: 4px solid var(--tertiary);
    color: var(--tertiary);
}

.toast-error {
    border-left: 4px solid #EF4444;
    color: #EF4444;
}

.fade-out {
    opacity: 0;
    transform: translateX(100px);
}

.chat-message {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 1rem;
}

.message-avatar {
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    background: var(--background);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1rem;
    flex-shrink: 0;
}

.message-content {
    flex: 1;
}

.message-text {
    background: var(--background);
    padding: 0.75rem;
    border-radius: 0.75rem;
    line-height: 1.5;
}

.message-time {
    font-size: 0.75rem;
    color: var(--text-light);
    margin-top: 0.25rem;
}

.chat-message.user .message-text {
    background: var(--primary);
    color: white;
}

.typing-indicator {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 1rem;
}

.typing-dots {
    display: flex;
    gap: 0.25rem;
    align-items: center;
    padding: 0.75rem;
    background: var(--background);
    border-radius: 0.75rem;
}

.typing-dots span {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--text-light);
    animation: typing 1.5s infinite;
}

.typing-dots span:nth-child(2) { animation-delay: 0.5s; }
.typing-dots span:nth-child(3) { animation-delay: 1s; }

@keyframes typing {
    0%, 60%, 100% { opacity: 0.3; }
    30% { opacity: 1; }
}

.suggestions {
    margin: 1rem 0;
}

.suggestions-label {
    font-size: 0.875rem;
    color: var(--text-light);
    margin-bottom: 0.5rem;
}

.suggestions-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
}

.suggestion-btn {
    padding: 0.5rem 1rem;
    background: var(--background);
    border: 1px solid var(--border);
    border-radius: 1rem;
    font-size: 0.75rem;
    cursor: pointer;
    transition: var(--transition);
}

.suggestion-btn:hover {
    background: var(--primary);
    color: white;
    border-color: var(--primary);
}

.search-result {
    padding: 1rem;
    border: 1px solid var(--border);
    border-radius: 0.75rem;
    margin-bottom: 0.75rem;
    cursor: pointer;
    transition: var(--transition);
}

.search-result:hover {
    background: var(--background);
}

.result-title {
    font-weight: 500;
    margin-bottom: 0.5rem;
}

.result-content {
    color: var(--text-light);
    font-size: 0.875rem;
    line-height: 1.4;
    margin-bottom: 0.5rem;
}

.result-meta {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--text-light);
}

.uploaded-file {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem;
    background: var(--background);
    border-radius: 0.5rem;
    margin-bottom: 0.5rem;
}

.file-info {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

.file-name {
    font-weight: 500;
}

.file-size {
    color: var(--text-light);
    font-size: 0.75rem;
}

.uploaded-file.success .file-status {
    color: var(--tertiary);
}

.uploaded-file.error .file-status {
    color: #EF4444;
}
`;

// Inject toast styles
const style = document.createElement('style');
style.textContent = toastStyles;
document.head.appendChild(style);
