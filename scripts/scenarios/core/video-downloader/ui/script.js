// Video Downloader - Client-side JavaScript

const API_URL = window.location.hostname === 'localhost' 
    ? 'http://localhost:8850/api' 
    : '/api';

// State management
let currentQueue = [];
let downloadHistory = [];
let currentVideoInfo = null;

// DOM Elements
const elements = {
    urlInput: document.getElementById('urlInput'),
    analyzeBtn: document.getElementById('analyzeBtn'),
    downloadBtn: document.getElementById('downloadBtn'),
    urlPreview: document.getElementById('urlPreview'),
    previewThumbnail: document.getElementById('previewThumbnail'),
    previewTitle: document.getElementById('previewTitle'),
    previewDescription: document.getElementById('previewDescription'),
    previewDuration: document.getElementById('previewDuration'),
    previewPlatform: document.getElementById('previewPlatform'),
    qualitySelect: document.getElementById('qualitySelect'),
    formatSelect: document.getElementById('formatSelect'),
    audioOnly: document.getElementById('audioOnly'),
    queueList: document.getElementById('queueList'),
    historyList: document.getElementById('historyList'),
    searchInput: document.getElementById('searchInput'),
    batchUrls: document.getElementById('batchUrls'),
    batchDownloadBtn: document.getElementById('batchDownloadBtn'),
    clearBatchBtn: document.getElementById('clearBatchBtn'),
    processQueueBtn: document.getElementById('processQueueBtn'),
    queueCount: document.getElementById('queueCount'),
    downloadCount: document.getElementById('downloadCount'),
    toastContainer: document.getElementById('toastContainer')
};

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeTabs();
    initializeEventListeners();
    loadQueue();
    loadHistory();
    setupDragAndDrop();
});

// Tab System
function initializeTabs() {
    const tabs = document.querySelectorAll('.tab');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetTab = tab.getAttribute('data-tab');
            
            // Update active states
            tabs.forEach(t => t.classList.remove('active'));
            tabContents.forEach(tc => tc.classList.remove('active'));
            
            tab.classList.add('active');
            document.getElementById(`${targetTab}Tab`).classList.add('active');
        });
    });
}

// Event Listeners
function initializeEventListeners() {
    elements.analyzeBtn.addEventListener('click', analyzeURL);
    elements.downloadBtn.addEventListener('click', startDownload);
    elements.batchDownloadBtn.addEventListener('click', addBatchToQueue);
    elements.clearBatchBtn.addEventListener('click', () => {
        elements.batchUrls.value = '';
        showToast('Batch URLs cleared', 'success');
    });
    elements.processQueueBtn.addEventListener('click', processQueue);
    elements.searchInput.addEventListener('input', filterHistory);
    elements.audioOnly.addEventListener('change', handleAudioOnlyToggle);
    
    // Enter key on URL input
    elements.urlInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            analyzeURL();
        }
    });
}

// Drag and Drop
function setupDragAndDrop() {
    const dropZone = elements.urlInput;
    
    ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, preventDefaults, false);
        document.body.addEventListener(eventName, preventDefaults, false);
    });
    
    ['dragenter', 'dragover'].forEach(eventName => {
        dropZone.addEventListener(eventName, () => {
            dropZone.style.borderColor = 'var(--accent-primary)';
            dropZone.style.boxShadow = '0 0 20px var(--accent-glow)';
        });
    });
    
    ['dragleave', 'drop'].forEach(eventName => {
        dropZone.addEventListener(eventName, () => {
            dropZone.style.borderColor = '';
            dropZone.style.boxShadow = '';
        });
    });
    
    dropZone.addEventListener('drop', handleDrop);
}

function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
}

function handleDrop(e) {
    const dt = e.dataTransfer;
    const text = dt.getData('text/plain');
    
    if (text && isValidURL(text)) {
        elements.urlInput.value = text;
        analyzeURL();
    }
}

// URL Analysis
async function analyzeURL() {
    const url = elements.urlInput.value.trim();
    
    if (!url || !isValidURL(url)) {
        showToast('Please enter a valid URL', 'error');
        return;
    }
    
    elements.analyzeBtn.disabled = true;
    elements.analyzeBtn.textContent = 'Analyzing...';
    
    try {
        const response = await fetch(`${API_URL}/analyze`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ url })
        });
        
        if (!response.ok) throw new Error('Failed to analyze URL');
        
        const data = await response.json();
        currentVideoInfo = data;
        displayVideoInfo(data);
        showToast('URL analyzed successfully', 'success');
    } catch (error) {
        console.error('Analysis error:', error);
        showToast('Failed to analyze URL', 'error');
    } finally {
        elements.analyzeBtn.disabled = false;
        elements.analyzeBtn.textContent = 'Analyze';
    }
}

// Display Video Information
function displayVideoInfo(info) {
    elements.urlPreview.classList.remove('hidden');
    elements.previewThumbnail.src = info.thumbnail || '/placeholder.jpg';
    elements.previewTitle.textContent = info.title || 'Unknown Title';
    elements.previewDescription.textContent = info.description || '';
    elements.previewDuration.textContent = formatDuration(info.duration);
    elements.previewPlatform.textContent = info.platform || 'Unknown';
    
    // Update quality options based on available formats
    if (info.availableQualities) {
        updateQualityOptions(info.availableQualities);
    }
}

// Start Download
async function startDownload() {
    if (!currentVideoInfo) {
        showToast('Please analyze a URL first', 'error');
        return;
    }
    
    const downloadData = {
        url: elements.urlInput.value.trim(),
        title: currentVideoInfo.title,
        quality: elements.qualitySelect.value,
        format: elements.audioOnly.checked ? 'mp3' : elements.formatSelect.value,
        user_id: getUserId()
    };
    
    try {
        const response = await fetch(`${API_URL}/download`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(downloadData)
        });
        
        if (!response.ok) throw new Error('Failed to start download');
        
        const result = await response.json();
        addToQueue(result);
        showToast('Download added to queue', 'success');
        
        // Clear input
        elements.urlInput.value = '';
        elements.urlPreview.classList.add('hidden');
        currentVideoInfo = null;
    } catch (error) {
        console.error('Download error:', error);
        showToast('Failed to start download', 'error');
    }
}

// Queue Management
function addToQueue(item) {
    currentQueue.push(item);
    updateQueueDisplay();
    updateStats();
}

function updateQueueDisplay() {
    if (currentQueue.length === 0) {
        elements.queueList.innerHTML = `
            <div class="empty-state">
                <svg width="60" height="60" viewBox="0 0 60 60" fill="none" opacity="0.3">
                    <rect x="10" y="20" width="40" height="30" rx="2" stroke="currentColor" stroke-width="2"/>
                    <path d="M20 35H40M20 40H35" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                </svg>
                <p>No downloads in queue</p>
            </div>
        `;
        return;
    }
    
    elements.queueList.innerHTML = currentQueue.map(item => `
        <div class="download-item" data-id="${item.id}">
            <div class="download-info">
                <div class="download-title">${item.title}</div>
                <div class="download-meta">
                    ${item.quality} • ${item.format} • ${item.status || 'Pending'}
                </div>
            </div>
            <div class="download-progress">
                <div class="progress-bar" style="width: ${item.progress || 0}%"></div>
            </div>
            <button class="btn-secondary" onclick="cancelDownload('${item.id}')">Cancel</button>
        </div>
    `).join('');
}

// Load Queue from API
async function loadQueue() {
    try {
        const response = await fetch(`${API_URL}/queue`);
        if (!response.ok) throw new Error('Failed to load queue');
        
        currentQueue = await response.json();
        updateQueueDisplay();
        updateStats();
    } catch (error) {
        console.error('Load queue error:', error);
    }
}

// Process Queue
async function processQueue() {
    if (currentQueue.length === 0) {
        showToast('Queue is empty', 'warning');
        return;
    }
    
    elements.processQueueBtn.disabled = true;
    elements.processQueueBtn.textContent = 'Processing...';
    
    try {
        const response = await fetch(`${API_URL}/queue/process`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' }
        });
        
        if (!response.ok) throw new Error('Failed to process queue');
        
        showToast('Queue processing started', 'success');
        startQueuePolling();
    } catch (error) {
        console.error('Process queue error:', error);
        showToast('Failed to process queue', 'error');
    } finally {
        elements.processQueueBtn.disabled = false;
        elements.processQueueBtn.textContent = 'Process Queue';
    }
}

// Cancel Download
async function cancelDownload(id) {
    try {
        const response = await fetch(`${API_URL}/download/${id}`, {
            method: 'DELETE'
        });
        
        if (!response.ok) throw new Error('Failed to cancel download');
        
        currentQueue = currentQueue.filter(item => item.id !== id);
        updateQueueDisplay();
        updateStats();
        showToast('Download cancelled', 'success');
    } catch (error) {
        console.error('Cancel error:', error);
        showToast('Failed to cancel download', 'error');
    }
}

// History Management
async function loadHistory() {
    try {
        const response = await fetch(`${API_URL}/history`);
        if (!response.ok) throw new Error('Failed to load history');
        
        downloadHistory = await response.json();
        updateHistoryDisplay();
        updateStats();
    } catch (error) {
        console.error('Load history error:', error);
    }
}

function updateHistoryDisplay(items = downloadHistory) {
    if (items.length === 0) {
        elements.historyList.innerHTML = `
            <div class="empty-state">
                <svg width="60" height="60" viewBox="0 0 60 60" fill="none" opacity="0.3">
                    <circle cx="30" cy="30" r="20" stroke="currentColor" stroke-width="2"/>
                    <path d="M30 20V30L36 36" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                </svg>
                <p>No download history</p>
            </div>
        `;
        return;
    }
    
    elements.historyList.innerHTML = items.map(item => `
        <div class="download-item">
            <div class="download-info">
                <div class="download-title">${item.title}</div>
                <div class="download-meta">
                    ${item.quality} • ${item.format} • ${new Date(item.downloaded_at).toLocaleDateString()}
                </div>
            </div>
            <button class="btn-secondary" onclick="redownload('${item.url}')">Re-download</button>
        </div>
    `).join('');
}

function filterHistory() {
    const searchTerm = elements.searchInput.value.toLowerCase();
    const filtered = downloadHistory.filter(item => 
        item.title.toLowerCase().includes(searchTerm)
    );
    updateHistoryDisplay(filtered);
}

// Batch Download
async function addBatchToQueue() {
    const urls = elements.batchUrls.value
        .split('\n')
        .map(url => url.trim())
        .filter(url => isValidURL(url));
    
    if (urls.length === 0) {
        showToast('No valid URLs found', 'error');
        return;
    }
    
    elements.batchDownloadBtn.disabled = true;
    elements.batchDownloadBtn.textContent = 'Adding to queue...';
    
    let successCount = 0;
    for (const url of urls) {
        try {
            const response = await fetch(`${API_URL}/download`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    url,
                    quality: elements.qualitySelect.value,
                    format: elements.formatSelect.value,
                    user_id: getUserId()
                })
            });
            
            if (response.ok) {
                successCount++;
                const result = await response.json();
                addToQueue(result);
            }
        } catch (error) {
            console.error(`Failed to add ${url}:`, error);
        }
    }
    
    showToast(`Added ${successCount} of ${urls.length} videos to queue`, 'success');
    elements.batchUrls.value = '';
    elements.batchDownloadBtn.disabled = false;
    elements.batchDownloadBtn.textContent = 'Add to Queue';
}

// Utility Functions
function isValidURL(string) {
    try {
        new URL(string);
        return true;
    } catch (_) {
        return false;
    }
}

function formatDuration(seconds) {
    if (!seconds) return 'Unknown';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
        return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

function getUserId() {
    let userId = localStorage.getItem('userId');
    if (!userId) {
        userId = 'user_' + Math.random().toString(36).substr(2, 9);
        localStorage.setItem('userId', userId);
    }
    return userId;
}

function updateQualityOptions(qualities) {
    elements.qualitySelect.innerHTML = qualities.map(q => 
        `<option value="${q}">${q}</option>`
    ).join('');
}

function handleAudioOnlyToggle() {
    if (elements.audioOnly.checked) {
        elements.formatSelect.disabled = true;
        elements.formatSelect.value = 'mp3';
    } else {
        elements.formatSelect.disabled = false;
        elements.formatSelect.value = 'mp4';
    }
}

function updateStats() {
    elements.queueCount.textContent = currentQueue.length;
    elements.downloadCount.textContent = downloadHistory.length;
}

// Toast Notifications
function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    
    elements.toastContainer.appendChild(toast);
    
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// Queue Polling
let pollingInterval = null;

function startQueuePolling() {
    if (pollingInterval) return;
    
    pollingInterval = setInterval(async () => {
        await loadQueue();
        
        // Stop polling if queue is empty
        if (currentQueue.filter(item => item.status === 'processing').length === 0) {
            stopQueuePolling();
        }
    }, 2000);
}

function stopQueuePolling() {
    if (pollingInterval) {
        clearInterval(pollingInterval);
        pollingInterval = null;
    }
}

// Re-download from history
window.redownload = function(url) {
    elements.urlInput.value = url;
    analyzeURL();
};

// Export for global access
window.cancelDownload = cancelDownload;