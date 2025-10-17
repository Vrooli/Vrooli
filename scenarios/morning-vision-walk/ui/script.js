import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__morningVisionWalkBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window[BRIDGE_FLAG]) {
        return;
    }

    if (window.parent !== window) {
        const options = { appId: 'morning-vision-walk' };
        try {
            if (document.referrer) {
                options.parentOrigin = new URL(document.referrer).origin;
            }
        } catch (error) {
            console.warn('[MorningVisionWalk] Unable to determine parent origin for iframe bridge', error);
        }

        initIframeBridgeChild(options);
        window[BRIDGE_FLAG] = true;
    }
}

bootstrapIframeBridge();

// Morning Vision Walk - Interactive JavaScript
const API_URL = window.API_URL || 'http://localhost:8900';

// State Management
const state = {
    sessionId: generateSessionId(),
    userId: localStorage.getItem('userId') || generateUserId(),
    isRecording: false,
    sessionStartTime: Date.now(),
    insights: [],
    actions: [],
    currentVision: '',
    energyLevel: 70,
    currentMood: 'focused',
    conversationHistory: []
};

// Save userId
localStorage.setItem('userId', state.userId);

// DOM Elements
const elements = {
    conversationMessages: document.getElementById('conversationMessages'),
    textInput: document.getElementById('textInput'),
    sendBtn: document.getElementById('sendBtn'),
    voiceBtn: document.getElementById('voiceBtn'),
    currentTime: document.getElementById('currentTime'),
    sessionDuration: document.getElementById('sessionDuration'),
    insightCount: document.getElementById('insightCount'),
    visionContent: document.getElementById('visionContent'),
    insightsList: document.getElementById('insightsList'),
    actionsList: document.getElementById('actionsList'),
    energyFill: document.getElementById('energyFill'),
    endSessionBtn: document.getElementById('endSessionBtn'),
    exportBtn: document.getElementById('exportBtn'),
    integrationBtn: document.getElementById('integrationBtn')
};

// Initialize
function escapeHtml(unsafe) {
    return unsafe
         .replace(/&/g, "&amp;")
         .replace(/</g, "&lt;")
         .replace(/>/g, "&gt;")
         .replace(/"/g, "&quot;")
         .replace(/'/g, "&#039;");
}

function init() {
    setupEventListeners();
    updateTime();
    setInterval(updateTime, 1000);
    autoResizeTextarea();
    loadPreviousSession();
}

// Event Listeners
function setupEventListeners() {
    // Text input
    elements.textInput.addEventListener('keydown', handleTextInput);
    elements.sendBtn.addEventListener('click', sendMessage);
    
    // Voice input
    elements.voiceBtn.addEventListener('mousedown', startRecording);
    elements.voiceBtn.addEventListener('mouseup', stopRecording);
    elements.voiceBtn.addEventListener('mouseleave', stopRecording);
    elements.voiceBtn.addEventListener('touchstart', startRecording);
    elements.voiceBtn.addEventListener('touchend', stopRecording);
    
    // Topic chips
    document.querySelectorAll('.topic-chip').forEach(chip => {
        chip.addEventListener('click', handleTopicClick);
    });
    
    // Mood buttons
    document.querySelectorAll('.mood-btn').forEach(btn => {
        btn.addEventListener('click', handleMoodChange);
    });
    
    // Quick actions
    elements.endSessionBtn.addEventListener('click', endSession);
    elements.exportBtn.addEventListener('click', exportSummary);
    elements.integrationBtn.addEventListener('click', syncWithScenarios);
    
    // Auto-resize textarea
    elements.textInput.addEventListener('input', autoResizeTextarea);
}

// Message Handling
async function sendMessage() {
    const message = elements.textInput.value.trim();
    if (!message) return;
    
    // Add user message to UI
    addMessage(message, 'user');
    
    // Clear input
    elements.textInput.value = '';
    autoResizeTextarea();
    
    // Process message
    await processMessage(message);
}

async function processMessage(message) {
    try {
        // Show typing indicator
        const typingId = showTypingIndicator();
        
        // Send to backend
        const response = await fetch(`${API_URL}/api/conversation`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message,
                session_id: state.sessionId,
                user_id: state.userId,
                context: {
                    time_of_day: getTimeOfDay(),
                    energy_level: state.energyLevel,
                    mood: state.currentMood,
                    conversation_history: state.conversationHistory.slice(-5)
                }
            })
        });
        
        const data = await response.json();
        
        // Remove typing indicator
        removeTypingIndicator(typingId);
        
        // Add AI response
        addMessage(data.response, 'ai');
        
        // Process insights and actions
        if (data.insights) {
            data.insights.forEach(addInsight);
        }
        
        if (data.actions) {
            data.actions.forEach(addAction);
        }
        
        if (data.vision) {
            updateVision(data.vision);
        }
        
        // Update conversation history
        state.conversationHistory.push({
            user: message,
            ai: data.response,
            timestamp: Date.now()
        });
        
    } catch (error) {
        console.error('Error processing message:', error);
        removeTypingIndicator();
        addMessage('I apologize, but I encountered an error. Please try again.', 'ai');
    }
}

// UI Updates
function addMessage(text, sender) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${sender}-message`;
    
    const avatar = document.createElement('div');
    avatar.className = 'message-avatar';
    avatar.innerHTML = sender === 'ai' ? 
        `<svg viewBox="0 0 32 32" fill="none">
            <circle cx="16" cy="16" r="14" fill="white"/>
            <path d="M16 10L16 22M10 16L22 16" stroke="#667eea" stroke-width="2" stroke-linecap="round"/>
        </svg>` :
        `<svg viewBox="0 0 32 32" fill="none">
            <circle cx="16" cy="16" r="14" fill="white"/>
            <circle cx="16" cy="16" r="8" fill="#fdcb6e"/>
        </svg>`;
    
    const content = document.createElement('div');
    content.className = 'message-content';
    content.innerHTML = `<p>${escapeHtml(text)}</p>`;
    
    messageDiv.appendChild(avatar);
    messageDiv.appendChild(content);
    
    elements.conversationMessages.appendChild(messageDiv);
    elements.conversationMessages.scrollTop = elements.conversationMessages.scrollHeight;
}

function showTypingIndicator() {
    const id = 'typing-' + Date.now();
    const indicator = document.createElement('div');
    indicator.id = id;
    indicator.className = 'message ai-message typing-indicator';
    indicator.innerHTML = `
        <div class="message-avatar">
            <svg viewBox="0 0 32 32" fill="none">
                <circle cx="16" cy="16" r="14" fill="white"/>
                <path d="M16 10L16 22M10 16L22 16" stroke="#667eea" stroke-width="2" stroke-linecap="round"/>
            </svg>
        </div>
        <div class="message-content">
            <div class="typing-dots">
                <span></span><span></span><span></span>
            </div>
        </div>
    `;
    elements.conversationMessages.appendChild(indicator);
    elements.conversationMessages.scrollTop = elements.conversationMessages.scrollHeight;
    return id;
}

function removeTypingIndicator(id) {
    const indicator = document.getElementById(id);
    if (indicator) {
        indicator.remove();
    }
}

function addInsight(insight) {
    const insightDiv = document.createElement('div');
    insightDiv.className = 'insight-item';
    insightDiv.textContent = insight;
    elements.insightsList.appendChild(insightDiv);
    
    state.insights.push(insight);
    elements.insightCount.textContent = state.insights.length;
}

function addAction(action) {
    const actionDiv = document.createElement('div');
    actionDiv.className = 'action-item';
    actionDiv.innerHTML = `
        <div class="action-checkbox"></div>
        <span class="action-text">${escapeHtml(action)}</span>
    `;
    
    actionDiv.addEventListener('click', function() {
        this.classList.toggle('completed');
    });
    
    elements.actionsList.appendChild(actionDiv);
    state.actions.push(action);
}

function updateVision(vision) {
    state.currentVision = vision;
    elements.visionContent.innerHTML = `<p>${escapeHtml(vision)}</p>`;
}

// Voice Recording
let mediaRecorder;
let audioChunks = [];

async function startRecording(e) {
    e.preventDefault();
    
    try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
        mediaRecorder = new MediaRecorder(stream);
        audioChunks = [];
        
        mediaRecorder.ondataavailable = event => {
            audioChunks.push(event.data);
        };
        
        mediaRecorder.onstop = async () => {
            const audioBlob = new Blob(audioChunks, { type: 'audio/wav' });
            await processAudio(audioBlob);
        };
        
        mediaRecorder.start();
        state.isRecording = true;
        elements.voiceBtn.classList.add('recording');
        elements.voiceBtn.querySelector('.voice-status').textContent = 'Recording...';
        
    } catch (error) {
        console.error('Error accessing microphone:', error);
        alert('Please allow microphone access to use voice input.');
    }
}

function stopRecording(e) {
    e.preventDefault();
    
    if (mediaRecorder && state.isRecording) {
        mediaRecorder.stop();
        state.isRecording = false;
        elements.voiceBtn.classList.remove('recording');
        elements.voiceBtn.querySelector('.voice-status').textContent = 'Hold to speak';
        
        // Stop all tracks
        mediaRecorder.stream.getTracks().forEach(track => track.stop());
    }
}

async function processAudio(audioBlob) {
    // In a real implementation, this would send to a speech-to-text service
    // For now, simulate with a placeholder
    const simulatedTranscript = "I'm thinking about the scenarios we need to improve today...";
    addMessage(simulatedTranscript, 'user');
    await processMessage(simulatedTranscript);
}

// Topic Handling
function handleTopicClick(e) {
    const topic = e.target.dataset.topic;
    const topicMessages = {
        'daily-priorities': "What should be our top priorities for Vrooli today?",
        'scenario-review': "Let's review the current state of our scenarios and identify which ones need attention.",
        'creative-ideas': "I have some creative ideas for new scenarios. Can we brainstorm together?",
        'challenges': "What are the main challenges we're facing with Vrooli right now?"
    };
    
    const message = topicMessages[topic];
    if (message) {
        elements.textInput.value = message;
        sendMessage();
    }
}

// Mood Handling
function handleMoodChange(e) {
    document.querySelectorAll('.mood-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    e.target.classList.add('active');
    state.currentMood = e.target.dataset.mood;
    
    // Adjust energy based on mood
    const energyMap = {
        'energized': 90,
        'focused': 70,
        'creative': 80,
        'reflective': 50
    };
    
    state.energyLevel = energyMap[state.currentMood];
    elements.energyFill.style.width = `${state.energyLevel}%`;
}

// Session Management
async function endSession() {
    if (confirm('End this vision walk session?')) {
        await saveSession();
        window.location.reload();
    }
}

async function exportSummary() {
    const summary = {
        session_id: state.sessionId,
        date: new Date().toISOString(),
        duration: formatDuration(Date.now() - state.sessionStartTime),
        vision: state.currentVision,
        insights: state.insights,
        actions: state.actions,
        conversation_history: state.conversationHistory
    };
    
    // Download as JSON
    const blob = new Blob([JSON.stringify(summary, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `vision-walk-${new Date().toISOString().split('T')[0]}.json`;
    a.click();
}

async function syncWithScenarios() {
    try {
        const response = await fetch(`${API_URL}/api/sync-scenarios`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                session_id: state.sessionId,
                insights: state.insights,
                actions: state.actions,
                vision: state.currentVision
            })
        });
        
        if (response.ok) {
            alert('Successfully synced with connected scenarios!');
        }
    } catch (error) {
        console.error('Error syncing with scenarios:', error);
        alert('Failed to sync with scenarios. Please try again.');
    }
}

async function saveSession() {
    try {
        await fetch(`${API_URL}/api/session/save`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                session_id: state.sessionId,
                user_id: state.userId,
                insights: state.insights,
                actions: state.actions,
                vision: state.currentVision,
                conversation_history: state.conversationHistory
            })
        });
    } catch (error) {
        console.error('Error saving session:', error);
    }
}

async function loadPreviousSession() {
    try {
        const response = await fetch(`${API_URL}/api/session/latest?user_id=${state.userId}`);
        if (response.ok) {
            const data = await response.json();
            if (data.vision) {
                updateVision(data.vision);
            }
        }
    } catch (error) {
        console.error('Error loading previous session:', error);
    }
}

// Utility Functions
function handleTextInput(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
    }
}

function autoResizeTextarea() {
    elements.textInput.style.height = 'auto';
    elements.textInput.style.height = Math.min(elements.textInput.scrollHeight, 120) + 'px';
}

function updateTime() {
    // Update current time
    const now = new Date();
    elements.currentTime.textContent = now.toLocaleTimeString('en-US', { 
        hour: 'numeric', 
        minute: '2-digit', 
        hour12: true 
    });
    
    // Update session duration
    const duration = Date.now() - state.sessionStartTime;
    elements.sessionDuration.textContent = formatDuration(duration);
}

function formatDuration(ms) {
    const minutes = Math.floor(ms / 60000);
    const seconds = Math.floor((ms % 60000) / 1000);
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

function getTimeOfDay() {
    const hour = new Date().getHours();
    if (hour < 6) return 'early_morning';
    if (hour < 12) return 'morning';
    if (hour < 17) return 'afternoon';
    if (hour < 21) return 'evening';
    return 'night';
}

function generateSessionId() {
    return 'session-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
}

function generateUserId() {
    return 'user-' + Math.random().toString(36).substr(2, 9);
}

// Add CSS for typing indicator
const style = document.createElement('style');
style.textContent = `
    .typing-indicator .typing-dots {
        display: flex;
        gap: 4px;
    }
    
    .typing-dots span {
        width: 8px;
        height: 8px;
        background: #667eea;
        border-radius: 50%;
        animation: typing 1.4s infinite;
    }
    
    .typing-dots span:nth-child(2) {
        animation-delay: 0.2s;
    }
    
    .typing-dots span:nth-child(3) {
        animation-delay: 0.4s;
    }
    
    @keyframes typing {
        0%, 60%, 100% {
            transform: translateY(0);
            opacity: 0.7;
        }
        30% {
            transform: translateY(-10px);
            opacity: 1;
        }
    }
`;
document.head.appendChild(style);

// Initialize app
document.addEventListener('DOMContentLoaded', init);
