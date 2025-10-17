import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

if (typeof window !== 'undefined' && window.parent !== window && !window.__personalDigitalTwinBridgeInitialized) {
    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[PersonalDigitalTwin] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'personal-digital-twin' });
    window.__personalDigitalTwinBridgeInitialized = true;
}

// Matrix rain effect
const canvas = document.getElementById('matrix-rain');
const ctx = canvas.getContext('2d');

canvas.width = window.innerWidth;
canvas.height = window.innerHeight;

const matrix = "ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789@#$%^&*()*&^%+-/~{[|`]}";
const matrixArray = matrix.split("");

const fontSize = 10;
const columns = canvas.width / fontSize;

const drops = [];
for(let x = 0; x < columns; x++) {
    drops[x] = 1;
}

function drawMatrix() {
    ctx.fillStyle = 'rgba(0, 0, 0, 0.04)';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    
    ctx.fillStyle = '#00ffff';
    ctx.font = fontSize + 'px monospace';
    
    for(let i = 0; i < drops.length; i++) {
        const text = matrixArray[Math.floor(Math.random() * matrixArray.length)];
        ctx.fillText(text, i * fontSize, drops[i] * fontSize);
        
        if(drops[i] * fontSize > canvas.height && Math.random() > 0.975) {
            drops[i] = 0;
        }
        drops[i]++;
    }
}

setInterval(drawMatrix, 35);

// Resize canvas on window resize
window.addEventListener('resize', () => {
    canvas.width = window.innerWidth;
    canvas.height = window.innerHeight;
});

// API Configuration
const API_BASE_URL = window.location.port ? `http://localhost:${window.location.port}` : 'http://localhost:8080';

// UI Elements
const chatInput = document.getElementById('chatInput');
const sendBtn = document.getElementById('sendBtn');
const chatMessages = document.getElementById('chatMessages');
const trainBtn = document.getElementById('trainBtn');
const uploadBtn = document.getElementById('uploadBtn');
const uploadModal = document.getElementById('uploadModal');
const closeModalBtn = document.getElementById('closeModalBtn');
const uploadZone = document.getElementById('uploadZone');
const fileInput = document.getElementById('fileInput');
const memoryList = document.getElementById('memoryList');
const addMemoryBtn = document.getElementById('addMemoryBtn');

// State
let currentMode = 'mirror';
let isProcessing = false;
let twinInitialized = false;
let memories = [];
let conversationHistory = [];

// Initialize UI
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadTwinStatus();
    loadMemories();
    updateUIState();
});

// Event Listeners
function initializeEventListeners() {
    // Chat functionality
    sendBtn.addEventListener('click', sendMessage);
    chatInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    // Auto-resize textarea
    chatInput.addEventListener('input', () => {
        chatInput.style.height = 'auto';
        chatInput.style.height = Math.min(chatInput.scrollHeight, 120) + 'px';
    });

    // Mode selection
    document.querySelectorAll('.mode-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.mode-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            currentMode = btn.dataset.mode;
            updateChatPlaceholder();
        });
    });

    // Training and upload
    trainBtn.addEventListener('click', trainModel);
    uploadBtn.addEventListener('click', () => uploadModal.classList.add('active'));
    closeModalBtn.addEventListener('click', () => uploadModal.classList.remove('active'));
    addMemoryBtn.addEventListener('click', () => uploadModal.classList.add('active'));

    // Upload zone
    uploadZone.addEventListener('click', () => fileInput.click());
    uploadZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadZone.classList.add('dragover');
    });
    uploadZone.addEventListener('dragleave', () => {
        uploadZone.classList.remove('dragover');
    });
    uploadZone.addEventListener('drop', handleFileDrop);
    fileInput.addEventListener('change', handleFileSelect);

    // Category filters
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.category-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            filterMemories(btn.textContent);
        });
    });

    // Close modal on escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            uploadModal.classList.remove('active');
        }
    });
}

// Twin Status Management
async function loadTwinStatus() {
    try {
        const response = await fetch(`${API_BASE_URL}/api/twin/status`);
        if (response.ok) {
            const data = await response.json();
            updateTwinDisplay(data);
            twinInitialized = data.initialized || false;
        }
    } catch (error) {
        console.error('Failed to load twin status:', error);
        // Initialize with default values
        updateTwinDisplay({
            name: 'Not Initialized',
            personality_model: 'None',
            knowledge_size: '0 GB',
            conversations: 0,
            initialized: false
        });
    }
}

function updateTwinDisplay(data) {
    const avatarInitial = document.querySelector('.avatar-initial');
    const avatarName = document.querySelector('.avatar-name');
    const stats = document.querySelectorAll('.avatar-stats .stat-value');
    
    if (data.initialized) {
        avatarInitial.textContent = (data.name || 'AI')[0].toUpperCase();
        avatarName.textContent = data.name || 'Digital Twin';
        stats[0].textContent = data.personality_model || 'Standard';
        stats[1].textContent = data.knowledge_size || '0 GB';
        stats[2].textContent = data.conversations || '0';
        
        // Update status indicators
        document.querySelector('.memory-usage .value').textContent = data.memory_count || '0';
        updateModelAccuracy(data.accuracy || 0);
    }
}

function updateModelAccuracy(accuracy) {
    const accuracyDisplay = document.querySelector('.status-bar .status-value:last-child');
    if (accuracyDisplay) {
        accuracyDisplay.textContent = accuracy > 0 ? `${accuracy}%` : '--';
    }
}

// Chat Functionality
async function sendMessage() {
    const message = chatInput.value.trim();
    if (!message || isProcessing) return;

    isProcessing = true;
    updateProcessingState(true);
    
    // Add user message to chat
    addMessageToChat('user', message);
    chatInput.value = '';
    chatInput.style.height = 'auto';

    try {
        const response = await fetch(`${API_BASE_URL}/api/chat`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                message: message,
                mode: currentMode,
                conversation_id: generateConversationId(),
                history: conversationHistory.slice(-10) // Send last 10 messages for context
            })
        });

        if (response.ok) {
            const data = await response.json();
            addMessageToChat('twin', data.response);
            conversationHistory.push({user: message, twin: data.response});
            
            // Update stats if provided
            if (data.stats) {
                updateTwinDisplay(data.stats);
            }
        } else {
            addMessageToChat('system', 'Failed to get response from twin. Please try again.');
        }
    } catch (error) {
        console.error('Chat error:', error);
        addMessageToChat('system', 'Connection error. Please check your connection and try again.');
    } finally {
        isProcessing = false;
        updateProcessingState(false);
    }
}

function addMessageToChat(type, content) {
    const messageDiv = document.createElement('div');
    messageDiv.className = `chat-message ${type}-message`;
    
    const timestamp = new Date().toLocaleTimeString('en-US', { 
        hour: '2-digit', 
        minute: '2-digit' 
    });
    
    messageDiv.innerHTML = `
        <div class="message-header">
            <span class="message-sender">${type === 'user' ? 'You' : type === 'twin' ? 'Digital Twin' : 'System'}</span>
            <span class="message-time">${timestamp}</span>
        </div>
        <div class="message-content">${escapeHtml(content)}</div>
    `;
    
    // Clear welcome message if it exists
    const welcomeMessage = chatMessages.querySelector('.welcome-message');
    if (welcomeMessage) {
        welcomeMessage.remove();
    }
    
    chatMessages.appendChild(messageDiv);
    chatMessages.scrollTop = chatMessages.scrollHeight;
    
    // Add typing animation for twin messages
    if (type === 'twin') {
        messageDiv.style.animation = 'messageSlideIn 0.3s ease-out';
    }
}

// Training Functionality
async function trainModel() {
    if (isProcessing) return;
    
    isProcessing = true;
    trainBtn.disabled = true;
    trainBtn.querySelector('.btn-text').textContent = 'TRAINING...';
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/twin/train`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                data_sources: memories.map(m => m.id)
            })
        });

        if (response.ok) {
            const data = await response.json();
            addMessageToChat('system', 'Training completed successfully! Your digital twin has been updated.');
            await loadTwinStatus();
            twinInitialized = true;
            updateUIState();
        } else {
            addMessageToChat('system', 'Training failed. Please ensure you have uploaded sufficient data.');
        }
    } catch (error) {
        console.error('Training error:', error);
        addMessageToChat('system', 'Training error. Please try again.');
    } finally {
        isProcessing = false;
        trainBtn.disabled = false;
        trainBtn.querySelector('.btn-text').textContent = 'TRAIN MODEL';
    }
}

// File Upload Functionality
function handleFileDrop(e) {
    e.preventDefault();
    uploadZone.classList.remove('dragover');
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
        processFiles(files);
    }
}

function handleFileSelect(e) {
    const files = e.target.files;
    if (files.length > 0) {
        processFiles(files);
    }
}

async function processFiles(files) {
    const uploadQueue = document.getElementById('uploadQueue');
    uploadQueue.innerHTML = '';
    
    for (const file of files) {
        const queueItem = createUploadQueueItem(file);
        uploadQueue.appendChild(queueItem);
        
        try {
            await uploadFile(file, queueItem);
        } catch (error) {
            console.error('Upload error:', error);
            updateQueueItemStatus(queueItem, 'error');
        }
    }
}

function createUploadQueueItem(file) {
    const item = document.createElement('div');
    item.className = 'upload-item';
    item.innerHTML = `
        <div class="upload-item-info">
            <span class="upload-filename">${file.name}</span>
            <span class="upload-size">${formatFileSize(file.size)}</span>
        </div>
        <div class="upload-progress">
            <div class="upload-progress-bar"></div>
        </div>
        <span class="upload-status">Uploading...</span>
    `;
    return item;
}

async function uploadFile(file, queueItem) {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('type', getFileCategory(file));
    
    const progressBar = queueItem.querySelector('.upload-progress-bar');
    
    try {
        const response = await fetch(`${API_BASE_URL}/api/upload`, {
            method: 'POST',
            body: formData
        });

        if (response.ok) {
            const data = await response.json();
            progressBar.style.width = '100%';
            updateQueueItemStatus(queueItem, 'success');
            
            // Add to memories
            addMemory({
                id: data.id,
                title: file.name,
                type: getFileCategory(file),
                size: file.size,
                timestamp: new Date().toISOString()
            });
            
            // Close modal after successful upload
            setTimeout(() => {
                uploadModal.classList.remove('active');
                loadMemories();
            }, 1500);
        } else {
            throw new Error('Upload failed');
        }
    } catch (error) {
        updateQueueItemStatus(queueItem, 'error');
        throw error;
    }
}

function updateQueueItemStatus(item, status) {
    const statusElement = item.querySelector('.upload-status');
    if (status === 'success') {
        statusElement.textContent = '✓ Uploaded';
        statusElement.style.color = 'var(--accent-green)';
    } else if (status === 'error') {
        statusElement.textContent = '✗ Failed';
        statusElement.style.color = 'var(--accent-red)';
    }
}

// Memory Management
async function loadMemories() {
    try {
        const response = await fetch(`${API_BASE_URL}/api/memories`);
        if (response.ok) {
            memories = await response.json();
            displayMemories(memories);
            updateMemoryStats();
        }
    } catch (error) {
        console.error('Failed to load memories:', error);
        // Display placeholder
        memoryList.innerHTML = `
            <div class="memory-item">
                <div class="memory-icon">💭</div>
                <div class="memory-content">
                    <div class="memory-title">No memories yet</div>
                    <div class="memory-meta">Start uploading data to build your twin's memory</div>
                </div>
            </div>
        `;
    }
}

function displayMemories(memoriesToDisplay) {
    if (memoriesToDisplay.length === 0) {
        memoryList.innerHTML = `
            <div class="memory-item">
                <div class="memory-icon">💭</div>
                <div class="memory-content">
                    <div class="memory-title">No memories found</div>
                    <div class="memory-meta">Upload files to create memories</div>
                </div>
            </div>
        `;
        return;
    }
    
    memoryList.innerHTML = '';
    memoriesToDisplay.forEach(memory => {
        const memoryItem = createMemoryItem(memory);
        memoryList.appendChild(memoryItem);
    });
}

function createMemoryItem(memory) {
    const item = document.createElement('div');
    item.className = 'memory-item';
    item.innerHTML = `
        <div class="memory-icon">${getMemoryIcon(memory.type)}</div>
        <div class="memory-content">
            <div class="memory-title">${memory.title}</div>
            <div class="memory-meta">${memory.type} • ${formatDate(memory.timestamp)}</div>
        </div>
    `;
    return item;
}

function addMemory(memory) {
    memories.push(memory);
    displayMemories(memories);
    updateMemoryStats();
}

function filterMemories(category) {
    if (category === 'ALL') {
        displayMemories(memories);
    } else {
        const filtered = memories.filter(m => 
            m.type.toUpperCase() === category.toUpperCase()
        );
        displayMemories(filtered);
    }
}

function updateMemoryStats() {
    const totalSize = memories.reduce((sum, m) => sum + (m.size || 0), 0);
    const progressFill = document.querySelector('.memory-stats .progress-fill');
    const statValue = document.querySelector('.memory-stats .stat-value');
    
    const usedGB = (totalSize / (1024 * 1024 * 1024)).toFixed(2);
    const percentage = Math.min((totalSize / (10 * 1024 * 1024 * 1024)) * 100, 100);
    
    if (progressFill) progressFill.style.width = `${percentage}%`;
    if (statValue) statValue.textContent = `${usedGB} / 10 GB`;
    
    // Update header count
    document.querySelector('.memory-usage .value').textContent = memories.length;
}

// Utility Functions
function updateChatPlaceholder() {
    const placeholders = {
        mirror: 'Ask your digital twin anything...',
        advisor: 'Get advice from your twin...',
        creative: 'Explore creative ideas with your twin...'
    };
    chatInput.placeholder = placeholders[currentMode] || placeholders.mirror;
}

function updateProcessingState(processing) {
    const statusText = document.querySelector('.status-bar .status-item:nth-child(3) .status-value');
    if (statusText) {
        statusText.textContent = processing ? 'PROCESSING' : 'IDLE';
        statusText.style.color = processing ? 'var(--primary-cyan)' : 'var(--text-primary)';
    }
    
    // Update neural load animation
    const loadFill = document.querySelector('.neural-load .load-fill');
    if (loadFill) {
        loadFill.style.animationDuration = processing ? '1s' : '3s';
    }
}

function updateUIState() {
    if (twinInitialized) {
        document.querySelector('.avatar-name').textContent = 'Digital Twin Active';
        document.querySelector('.status-light').classList.add('active');
    }
}

function generateConversationId() {
    return `conv_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, m => map[m]);
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins} min ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)} hours ago`;
    return date.toLocaleDateString();
}

function getFileCategory(file) {
    const ext = file.name.split('.').pop().toLowerCase();
    const categories = {
        'txt,md,doc,docx,pdf': 'KNOWLEDGE',
        'jpg,jpeg,png,gif,bmp': 'EXPERIENCES',
        'mp3,wav,m4a,ogg': 'PERSONAL',
        'mp4,avi,mov,mkv': 'EXPERIENCES'
    };
    
    for (const [exts, category] of Object.entries(categories)) {
        if (exts.split(',').includes(ext)) return category;
    }
    return 'KNOWLEDGE';
}

function getMemoryIcon(type) {
    const icons = {
        'PERSONAL': '👤',
        'KNOWLEDGE': '📚',
        'EXPERIENCES': '🎯',
        'DEFAULT': '💭'
    };
    return icons[type] || icons.DEFAULT;
}

// Add message animation styles
const style = document.createElement('style');
style.textContent = `
    @keyframes messageSlideIn {
        from {
            opacity: 0;
            transform: translateY(10px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .chat-message {
        padding: 1rem;
        margin-bottom: 1rem;
        border-radius: 8px;
        background: rgba(0, 0, 0, 0.3);
        border-left: 3px solid transparent;
    }
    
    .user-message {
        border-left-color: var(--primary-cyan);
        background: rgba(0, 255, 255, 0.05);
    }
    
    .twin-message {
        border-left-color: var(--primary-purple);
        background: rgba(155, 89, 182, 0.05);
    }
    
    .system-message {
        border-left-color: var(--accent-green);
        background: rgba(0, 255, 136, 0.05);
        font-style: italic;
    }
    
    .message-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.5rem;
        font-size: 0.85rem;
    }
    
    .message-sender {
        color: var(--primary-cyan);
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 1px;
    }
    
    .message-time {
        color: var(--text-secondary);
    }
    
    .message-content {
        line-height: 1.6;
        color: var(--text-primary);
    }
    
    .upload-item {
        padding: 1rem;
        background: rgba(0, 0, 0, 0.3);
        border-radius: 8px;
        margin-bottom: 1rem;
    }
    
    .upload-item-info {
        display: flex;
        justify-content: space-between;
        margin-bottom: 0.5rem;
    }
    
    .upload-filename {
        font-weight: 600;
        color: var(--text-primary);
    }
    
    .upload-size {
        color: var(--text-secondary);
        font-size: 0.85rem;
    }
    
    .upload-progress {
        height: 4px;
        background: rgba(255, 255, 255, 0.1);
        border-radius: 2px;
        overflow: hidden;
        margin-bottom: 0.5rem;
    }
    
    .upload-progress-bar {
        height: 100%;
        width: 0;
        background: linear-gradient(90deg, var(--primary-cyan), var(--primary-purple));
        transition: width 0.3s ease;
    }
    
    .upload-status {
        font-size: 0.85rem;
        color: var(--text-secondary);
    }
`;
document.head.appendChild(style);
