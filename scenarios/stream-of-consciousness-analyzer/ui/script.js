// Stream of Consciousness Analyzer - Frontend Logic

(function initializeIframeBridge() {
    if (typeof window === 'undefined') {
        return;
    }
    if (window.__streamOfConsciousnessBridgeInitialized) {
        return;
    }
    if (window.parent !== window && typeof window.initIframeBridgeChild === 'function') {
        window.initIframeBridgeChild({ appId: 'stream-of-consciousness-analyzer-ui' });
        window.__streamOfConsciousnessBridgeInitialized = true;
    }
})();

const API_URL = window.API_URL || 'http://localhost:8700';

// State management
const state = {
    currentCampaign: 'general',
    isRecording: false,
    mediaRecorder: null,
    audioChunks: [],
    notes: [],
    insights: {
        actions: [],
        patterns: [],
        ideas: []
    }
};

// DOM elements
const elements = {
    streamTextarea: document.querySelector('.stream-textarea'),
    processBtn: document.querySelector('.process-btn'),
    campaignChips: document.querySelectorAll('.campaign-chip'),
    voiceIndicator: document.querySelector('.voice-indicator'),
    notesGrid: document.querySelector('.notes-grid'),
    searchInput: document.querySelector('.search-input'),
    loadingOverlay: document.querySelector('.loading-overlay'),
    modeBtns: document.querySelectorAll('.mode-btn'),
    viewToggles: document.querySelectorAll('.view-toggle'),
    insightCards: document.querySelectorAll('.insight-card')
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeCampaigns();
    initializeInputModes();
    initializeProcessing();
    initializeSearch();
    initializeViewToggles();
    loadNotes();
    loadInsights();
});

// Campaign management
function initializeCampaigns() {
    elements.campaignChips.forEach(chip => {
        chip.addEventListener('click', (e) => {
            const campaign = e.target.dataset.campaign;
            
            if (campaign === 'new') {
                createNewCampaign();
            } else {
                switchCampaign(campaign);
            }
        });
    });
}

function switchCampaign(campaign) {
    state.currentCampaign = campaign;
    
    // Update UI
    elements.campaignChips.forEach(chip => {
        chip.classList.toggle('active', chip.dataset.campaign === campaign);
    });
    
    // Update theme based on campaign
    updateThemeForCampaign(campaign);
    
    // Reload notes for this campaign
    loadNotes();
    loadInsights();
}

function updateThemeForCampaign(campaign) {
    const themes = {
        general: { primary: '#7C3AED', secondary: '#EC4899' },
        daily: { primary: '#EC4899', secondary: '#F59E0B' },
        work: { primary: '#3B82F6', secondary: '#10B981' }
    };
    
    const theme = themes[campaign] || themes.general;
    document.documentElement.style.setProperty('--accent-purple', theme.primary);
    document.documentElement.style.setProperty('--accent-pink', theme.secondary);
}

function createNewCampaign() {
    const name = prompt('Enter campaign name:');
    if (name) {
        const description = prompt('Enter campaign description (optional):');
        
        fetch(`${API_URL}/api/campaigns`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name, description })
        })
        .then(response => response.json())
        .then(campaign => {
            // Add new campaign chip
            const newChip = document.createElement('button');
            newChip.className = 'campaign-chip';
            newChip.dataset.campaign = campaign.id;
            newChip.textContent = campaign.name;
            
            const selector = document.querySelector('.campaign-selector');
            const newBtn = selector.querySelector('[data-campaign="new"]');
            selector.insertBefore(newChip, newBtn);
            
            // Switch to new campaign
            switchCampaign(campaign.id);
        })
        .catch(error => console.error('Error creating campaign:', error));
    }
}

// Input modes
function initializeInputModes() {
    const [textBtn, voiceBtn, uploadBtn] = elements.modeBtns;
    
    // Text mode (default)
    textBtn.addEventListener('click', () => {
        elements.streamTextarea.style.display = 'block';
        elements.voiceIndicator.classList.remove('active');
        if (state.isRecording) stopRecording();
    });
    
    // Voice mode
    voiceBtn.addEventListener('click', () => {
        if (state.isRecording) {
            stopRecording();
        } else {
            startRecording();
        }
    });
    
    // Upload mode
    uploadBtn.addEventListener('click', () => {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.txt,.md,.doc,.docx,.pdf';
        input.onchange = handleFileUpload;
        input.click();
    });
}

// Voice recording
async function startRecording() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        state.mediaRecorder = new MediaRecorder(stream);
        state.audioChunks = [];
        
        state.mediaRecorder.ondataavailable = (event) => {
            state.audioChunks.push(event.data);
        };
        
        state.mediaRecorder.onstop = async () => {
            const audioBlob = new Blob(state.audioChunks, { type: 'audio/wav' });
            await processAudio(audioBlob);
        };
        
        state.mediaRecorder.start();
        state.isRecording = true;
        
        // Update UI
        elements.voiceIndicator.classList.add('active');
        elements.modeBtns[1].style.color = 'var(--accent-pink)';
    } catch (error) {
        console.error('Error starting recording:', error);
        alert('Could not access microphone. Please check permissions.');
    }
}

function stopRecording() {
    if (state.mediaRecorder && state.isRecording) {
        state.mediaRecorder.stop();
        state.isRecording = false;
        
        // Update UI
        elements.voiceIndicator.classList.remove('active');
        elements.modeBtns[1].style.color = '';
        
        // Stop all tracks
        state.mediaRecorder.stream.getTracks().forEach(track => track.stop());
    }
}

async function processAudio(audioBlob) {
    showLoading(true);
    
    const formData = new FormData();
    formData.append('audio', audioBlob);
    formData.append('campaign_id', state.currentCampaign);
    
    try {
        const response = await fetch(`${API_URL}/api/transcribe`, {
            method: 'POST',
            body: formData
        });
        
        const result = await response.json();
        elements.streamTextarea.value = result.transcript;
        
        // Auto-process if transcription successful
        if (result.transcript) {
            await processStream();
        }
    } catch (error) {
        console.error('Error processing audio:', error);
        alert('Error processing audio. Please try again.');
    } finally {
        showLoading(false);
    }
}

// File upload
function handleFileUpload(event) {
    const file = event.target.files[0];
    if (file) {
        const reader = new FileReader();
        reader.onload = (e) => {
            elements.streamTextarea.value = e.target.result;
        };
        reader.readAsText(file);
    }
}

// Processing
function initializeProcessing() {
    elements.processBtn.addEventListener('click', processStream);
    
    // Auto-save draft
    let saveTimeout;
    elements.streamTextarea.addEventListener('input', () => {
        clearTimeout(saveTimeout);
        saveTimeout = setTimeout(saveDraft, 2000);
    });
}

async function processStream() {
    const content = elements.streamTextarea.value.trim();
    
    if (!content) {
        alert('Please enter some thoughts to process');
        return;
    }
    
    showLoading(true);
    
    try {
        const response = await fetch(`${API_URL}/api/process`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                campaign_id: state.currentCampaign,
                content: content,
                type: 'text'
            })
        });
        
        const result = await response.json();
        
        // Clear input
        elements.streamTextarea.value = '';
        
        // Add new notes
        if (result.notes && result.notes.length > 0) {
            result.notes.forEach(note => addNoteToGrid(note));
        }
        
        // Update insights
        if (result.insights) {
            updateInsights(result.insights);
        }
        
        // Show success feedback
        showSuccessFeedback();
        
    } catch (error) {
        console.error('Error processing stream:', error);
        alert('Error processing your thoughts. Please try again.');
    } finally {
        showLoading(false);
    }
}

function saveDraft() {
    const content = elements.streamTextarea.value;
    if (content) {
        localStorage.setItem(`soc_draft_${state.currentCampaign}`, content);
    }
}

// Notes management
async function loadNotes() {
    try {
        const response = await fetch(`${API_URL}/api/notes?campaign_id=${state.currentCampaign}`);
        const notes = await response.json();
        
        state.notes = notes;
        displayNotes(notes);
    } catch (error) {
        console.error('Error loading notes:', error);
    }
}

function displayNotes(notes) {
    elements.notesGrid.innerHTML = '';
    
    notes.forEach(note => addNoteToGrid(note));
}

function addNoteToGrid(note) {
    const noteCard = document.createElement('div');
    noteCard.className = 'note-card';
    noteCard.dataset.noteId = note.id;
    
    const timestamp = new Date(note.created_at).toLocaleString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });
    
    const tags = note.tags ? note.tags.map(tag => 
        `<span class="note-tag">${tag}</span>`
    ).join('') : '';
    
    noteCard.innerHTML = `
        <div class="note-timestamp">${timestamp}</div>
        <h3 class="note-title">${note.title || 'Untitled Note'}</h3>
        <div class="note-content">${note.summary || note.content}</div>
        ${tags ? `<div class="note-tags">${tags}</div>` : ''}
    `;
    
    noteCard.addEventListener('click', () => openNoteDetail(note));
    
    // Add with animation
    noteCard.style.opacity = '0';
    noteCard.style.transform = 'translateY(20px)';
    elements.notesGrid.prepend(noteCard);
    
    setTimeout(() => {
        noteCard.style.transition = 'all 0.3s ease';
        noteCard.style.opacity = '1';
        noteCard.style.transform = 'translateY(0)';
    }, 10);
}

function openNoteDetail(note) {
    // This would open a modal or detail view
    console.log('Opening note:', note);
}

// Insights
async function loadInsights() {
    try {
        const response = await fetch(`${API_URL}/api/insights?campaign_id=${state.currentCampaign}`);
        const insights = await response.json();
        
        updateInsights(insights);
    } catch (error) {
        console.error('Error loading insights:', error);
    }
}

function updateInsights(insights) {
    // Update action items
    const actionsCard = elements.insightCards[0];
    if (insights.actions && insights.actions.length > 0) {
        const actionsHtml = insights.actions
            .slice(0, 5)
            .map(action => `<p>• ${action.content}</p>`)
            .join('');
        actionsCard.querySelector('.insight-content').innerHTML = actionsHtml;
    }
    
    // Update patterns
    const patternsCard = elements.insightCards[1];
    if (insights.patterns && insights.patterns.length > 0) {
        patternsCard.querySelector('.insight-content').textContent = 
            insights.patterns[0].content;
    }
    
    // Update ideas
    const ideasCard = elements.insightCards[2];
    if (insights.ideas && insights.ideas.length > 0) {
        ideasCard.querySelector('.insight-content').textContent = 
            insights.ideas[0].content;
    }
}

// Search
function initializeSearch() {
    let searchTimeout;
    
    elements.searchInput.addEventListener('input', (e) => {
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => {
            searchNotes(e.target.value);
        }, 300);
    });
}

async function searchNotes(query) {
    if (!query) {
        displayNotes(state.notes);
        return;
    }
    
    try {
        const response = await fetch(
            `${API_URL}/api/search?q=${encodeURIComponent(query)}&campaign_id=${state.currentCampaign}`
        );
        const results = await response.json();
        
        displayNotes(results);
    } catch (error) {
        console.error('Error searching notes:', error);
    }
}

// View toggles
function initializeViewToggles() {
    elements.viewToggles.forEach(toggle => {
        toggle.addEventListener('click', (e) => {
            elements.viewToggles.forEach(t => t.classList.remove('active'));
            e.target.classList.add('active');
            
            const view = e.target.textContent.toLowerCase();
            if (view.includes('grid')) {
                elements.notesGrid.className = 'notes-grid';
            } else if (view.includes('list')) {
                elements.notesGrid.className = 'notes-list';
            } else if (view.includes('timeline')) {
                elements.notesGrid.className = 'notes-timeline';
            }
        });
    });
}

// UI helpers
function showLoading(show) {
    elements.loadingOverlay.classList.toggle('active', show);
}

function showSuccessFeedback() {
    // Create success message
    const success = document.createElement('div');
    success.style.cssText = `
        position: fixed;
        bottom: 2rem;
        right: 2rem;
        padding: 1rem 2rem;
        background: linear-gradient(135deg, var(--accent-green), #34D399);
        color: white;
        border-radius: 12px;
        box-shadow: var(--shadow-lg);
        font-weight: 600;
        z-index: 1000;
        animation: slideIn 0.3s ease;
    `;
    success.textContent = '✓ Thoughts organized successfully!';
    
    document.body.appendChild(success);
    
    setTimeout(() => {
        success.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => success.remove(), 300);
    }, 3000);
}

// Add slide animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
    
    .notes-list {
        display: flex;
        flex-direction: column;
        gap: 1rem;
    }
    
    .notes-list .note-card {
        max-width: 100%;
    }
    
    .notes-timeline {
        display: flex;
        flex-direction: column;
        gap: 2rem;
        position: relative;
        padding-left: 3rem;
    }
    
    .notes-timeline::before {
        content: '';
        position: absolute;
        left: 1rem;
        top: 0;
        bottom: 0;
        width: 2px;
        background: var(--border-light);
    }
    
    .notes-timeline .note-card::before {
        content: '';
        position: absolute;
        left: -2.5rem;
        top: 1.5rem;
        width: 12px;
        height: 12px;
        background: var(--accent-purple);
        border-radius: 50%;
    }
`;
document.head.appendChild(style);
