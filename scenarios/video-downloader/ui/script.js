import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

(function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window.parent === window || window.__videoDownloaderBridgeInitialized) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[VideoDownloader] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'video-downloader' });
    window.__videoDownloaderBridgeInitialized = true;
})();

// Video Downloader - Client-side JavaScript

const API_URL = window.location.hostname === 'localhost' 
    ? 'http://localhost:8850/api' 
    : '/api';

// State management
let currentQueue = [];
let downloadHistory = [];
let currentVideoInfo = null;

// DOM Elements - Enhanced for Media Intelligence Platform
const elements = {
    // URL Input and Analysis
    urlInput: document.getElementById('urlInput'),
    analyzeBtn: document.getElementById('analyzeBtn'),
    downloadBtn: document.getElementById('downloadBtn'),
    downloadBtnText: document.getElementById('downloadBtnText'),
    downloadEstimate: document.getElementById('downloadEstimate'),
    estimateTime: document.getElementById('estimateTime'),
    urlPreview: document.getElementById('urlPreview'),
    previewThumbnail: document.getElementById('previewThumbnail'),
    previewTitle: document.getElementById('previewTitle'),
    previewDescription: document.getElementById('previewDescription'),
    previewDuration: document.getElementById('previewDuration'),
    previewPlatform: document.getElementById('previewPlatform'),
    
    // Enhanced Format Options
    qualitySelect: document.getElementById('qualitySelect'),
    formatSelect: document.getElementById('formatSelect'),
    audioOnly: document.getElementById('audioOnly'),
    audioOptions: document.getElementById('audioOptions'),
    audioFormatSelect: document.getElementById('audioFormatSelect'),
    audioQualitySelect: document.getElementById('audioQualitySelect'),
    
    // Transcript Options
    generateTranscript: document.getElementById('generateTranscript'),
    transcriptSettings: document.getElementById('transcriptSettings'),
    whisperModelSelect: document.getElementById('whisperModelSelect'),
    targetLanguageSelect: document.getElementById('targetLanguageSelect'),
    
    // Tab Content Areas
    queueList: document.getElementById('queueList'),
    historyList: document.getElementById('historyList'),
    transcriptsList: document.getElementById('transcriptsList'),
    searchInput: document.getElementById('searchInput'),
    transcriptSearchInput: document.getElementById('transcriptSearchInput'),
    languageFilter: document.getElementById('languageFilter'),
    
    // Batch Operations
    batchUrls: document.getElementById('batchUrls'),
    batchDownloadBtn: document.getElementById('batchDownloadBtn'),
    clearBatchBtn: document.getElementById('clearBatchBtn'),
    processQueueBtn: document.getElementById('processQueueBtn'),
    
    // Stats and UI
    queueCount: document.getElementById('queueCount'),
    downloadCount: document.getElementById('downloadCount'),
    toastContainer: document.getElementById('toastContainer'),
    
    // Transcript Modal Elements
    transcriptModal: document.getElementById('transcriptModal'),
    transcriptTitle: document.getElementById('transcriptTitle'),
    transcriptLanguage: document.getElementById('transcriptLanguage'),
    transcriptConfidence: document.getElementById('transcriptConfidence'),
    transcriptDuration: document.getElementById('transcriptDuration'),
    transcriptText: document.getElementById('transcriptText'),
    searchInTranscriptBtn: document.getElementById('searchInTranscriptBtn'),
    exportTranscriptBtn: document.getElementById('exportTranscriptBtn'),
    closeTranscriptBtn: document.getElementById('closeTranscriptBtn'),
    transcriptSearchBar: document.getElementById('transcriptSearchBar'),
    transcriptSearchQuery: document.getElementById('transcriptSearchQuery'),
    searchResultsCount: document.getElementById('searchResultsCount'),
    prevSearchResult: document.getElementById('prevSearchResult'),
    nextSearchResult: document.getElementById('nextSearchResult'),
    transcriptExportOptions: document.getElementById('transcriptExportOptions'),
    includeTimestamps: document.getElementById('includeTimestamps'),
    includeConfidence: document.getElementById('includeConfidence'),
    downloadExportBtn: document.getElementById('downloadExportBtn')
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

// Enhanced Event Listeners for Media Intelligence Platform
function initializeEventListeners() {
    // Core URL and Download functionality
    elements.analyzeBtn.addEventListener('click', analyzeURL);
    elements.downloadBtn.addEventListener('click', startDownload);
    elements.batchDownloadBtn.addEventListener('click', addBatchToQueue);
    elements.clearBatchBtn.addEventListener('click', () => {
        elements.batchUrls.value = '';
        showToast('Batch URLs cleared', 'success');
    });
    elements.processQueueBtn.addEventListener('click', processQueue);
    
    // Search and Filter functionality
    elements.searchInput.addEventListener('input', filterHistory);
    elements.transcriptSearchInput.addEventListener('input', filterTranscripts);
    elements.languageFilter.addEventListener('change', filterTranscripts);
    
    // Enhanced Audio and Format Options
    elements.audioOnly.addEventListener('change', handleAudioOnlyToggle);
    elements.audioFormatSelect.addEventListener('change', updateAudioQualityOptions);
    elements.generateTranscript.addEventListener('change', handleTranscriptToggle);
    elements.whisperModelSelect.addEventListener('change', updateDownloadEstimate);
    
    // Transcript Modal functionality
    elements.searchInTranscriptBtn.addEventListener('click', toggleTranscriptSearch);
    elements.exportTranscriptBtn.addEventListener('click', toggleExportOptions);
    elements.closeTranscriptBtn.addEventListener('click', closeTranscriptModal);
    elements.transcriptSearchQuery.addEventListener('input', searchWithinTranscript);
    elements.prevSearchResult.addEventListener('click', () => navigateSearchResults('prev'));
    elements.nextSearchResult.addEventListener('click', () => navigateSearchResults('next'));
    elements.downloadExportBtn.addEventListener('click', downloadTranscriptExport);
    
    // Export format selection
    document.querySelectorAll('.format-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            document.querySelectorAll('.format-btn').forEach(b => b.classList.remove('active'));
            e.target.classList.add('active');
        });
    });
    
    // Modal backdrop close
    elements.transcriptModal.querySelector('.modal-backdrop').addEventListener('click', closeTranscriptModal);
    
    // Keyboard shortcuts
    document.addEventListener('keydown', handleKeyboardShortcuts);
    
    // Enter key handlers
    elements.urlInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            analyzeURL();
        }
    });
    
    elements.transcriptSearchQuery.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            searchWithinTranscript();
        }
    });
    
    // Real-time download estimate updates
    [elements.qualitySelect, elements.formatSelect, elements.audioQualitySelect].forEach(select => {
        select.addEventListener('change', updateDownloadEstimate);
    });
    
    // Initialize enhanced tabs with transcript support
    initializeEnhancedTabs();
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

// Enhanced Start Download with Media Intelligence Platform support
async function startDownload() {
    if (!currentVideoInfo) {
        showToast('Please analyze a URL first', 'error');
        return;
    }
    
    // Build enhanced download request
    const downloadData = {
        url: elements.urlInput.value.trim(),
        quality: elements.qualitySelect.value,
        format: elements.formatSelect.value,
        user_id: getUserId()
    };
    
    // Enhanced audio options
    if (elements.audioOnly.checked) {
        downloadData.audio_only = true;
        downloadData.audio_format = elements.audioFormatSelect.value;
        downloadData.audio_quality = elements.audioQualitySelect.value;
    } else if (elements.audioFormatSelect.value) {
        // Extract audio even for video downloads if format specified
        downloadData.audio_format = elements.audioFormatSelect.value;
        downloadData.audio_quality = elements.audioQualitySelect.value;
    }
    
    // Transcript generation options
    if (elements.generateTranscript.checked) {
        downloadData.generate_transcript = true;
        downloadData.whisper_model = elements.whisperModelSelect.value;
        if (elements.targetLanguageSelect.value) {
            downloadData.target_language = elements.targetLanguageSelect.value;
        }
    }
    
    elements.downloadBtn.disabled = true;
    elements.downloadBtnText.textContent = 'Starting...';
    
    try {
        const response = await fetch(`${API_URL}/download`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(downloadData)
        });
        
        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.message || 'Failed to start download');
        }
        
        const result = await response.json();
        addToQueue(result);
        
        // Enhanced success message
        let message = 'Download added to queue';
        if (downloadData.generate_transcript) {
            message += ' with transcript generation';
        }
        if (downloadData.audio_format && !downloadData.audio_only) {
            message += ` and ${downloadData.audio_format.toUpperCase()} extraction`;
        }
        
        showToast(message, 'success');
        
        // Clear input and reset form
        elements.urlInput.value = '';
        elements.urlPreview.classList.add('hidden');
        currentVideoInfo = null;
        resetFormOptions();
        
    } catch (error) {
        console.error('Download error:', error);
        showToast(error.message || 'Failed to start download', 'error');
    } finally {
        elements.downloadBtn.disabled = false;
        elements.downloadBtnText.textContent = 'Download';
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

// =============================================================================
// ENHANCED MEDIA INTELLIGENCE PLATFORM FUNCTIONS
// =============================================================================

// Enhanced Audio and Format Handling
function handleAudioOnlyToggle() {
    const audioOnly = elements.audioOnly.checked;
    elements.audioOptions.classList.toggle('hidden', !audioOnly);
    updateDownloadEstimate();
}

function handleTranscriptToggle() {
    const generateTranscript = elements.generateTranscript.checked;
    elements.transcriptSettings.classList.toggle('hidden', !generateTranscript);
    updateDownloadEstimate();
}

function updateAudioQualityOptions() {
    const format = elements.audioFormatSelect.value;
    const qualitySelect = elements.audioQualitySelect;
    
    // Disable quality selection for lossless formats
    if (format === 'flac') {
        qualitySelect.disabled = true;
        qualitySelect.innerHTML = '<option value="lossless">Lossless</option>';
    } else {
        qualitySelect.disabled = false;
        qualitySelect.innerHTML = `
            <option value="320k">320 kbps (High)</option>
            <option value="192k" selected>192 kbps (Standard)</option>
            <option value="128k">128 kbps (Low)</option>
            <option value="96k">96 kbps (Very Low)</option>
        `;
    }
    updateDownloadEstimate();
}

function updateDownloadEstimate() {
    let estimateMinutes = 3; // Base download time
    
    // Add time for high quality video
    const quality = elements.qualitySelect.value;
    if (quality === 'best' || quality === '2160p') estimateMinutes += 5;
    else if (quality === '1080p') estimateMinutes += 2;
    
    // Add time for transcript generation
    if (elements.generateTranscript.checked) {
        const model = elements.whisperModelSelect.value;
        const modelTimeMap = { tiny: 2, base: 5, small: 8, medium: 12, large: 20 };
        estimateMinutes += modelTimeMap[model] || 5;
    }
    
    // Add time for audio extraction
    if (!elements.audioOnly.checked && elements.audioFormatSelect.value) {
        estimateMinutes += 1;
    }
    
    elements.estimateTime.textContent = `~${estimateMinutes} min`;
    elements.downloadEstimate.classList.remove('hidden');
}

function resetFormOptions() {
    elements.audioOnly.checked = false;
    elements.generateTranscript.checked = false;
    elements.audioOptions.classList.add('hidden');
    elements.transcriptSettings.classList.add('hidden');
    elements.downloadEstimate.classList.add('hidden');
}

// Enhanced Tab System with Transcript Support
function initializeEnhancedTabs() {
    const tabs = document.querySelectorAll('.tab');
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const targetTab = tab.getAttribute('data-tab');
            if (targetTab === 'transcripts') {
                loadTranscripts();
            }
        });
    });
}

// Transcript Management Functions
async function loadTranscripts() {
    try {
        const response = await fetch(`${API_URL}/history?has_transcript=true`);
        if (!response.ok) throw new Error('Failed to load transcripts');
        
        const transcripts = await response.json();
        displayTranscripts(transcripts);
    } catch (error) {
        console.error('Error loading transcripts:', error);
        showToast('Failed to load transcripts', 'error');
    }
}

function displayTranscripts(transcripts) {
    if (transcripts.length === 0) {
        elements.transcriptsList.innerHTML = `
            <div class="empty-state">
                <svg width="60" height="60" viewBox="0 0 60 60" fill="none" opacity="0.3">
                    <path d="M10 10H50V50H10V10Z" stroke="currentColor" stroke-width="2"/>
                    <path d="M18 20H42M18 26H42M18 32H35M18 38H28" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
                </svg>
                <p>No transcripts available</p>
                <small>Transcripts will appear here when you download videos with transcript generation enabled</small>
            </div>
        `;
        return;
    }
    
    elements.transcriptsList.innerHTML = transcripts.map(item => `
        <div class="transcript-item" onclick="openTranscriptModal(${item.id})">
            <div class="transcript-header">
                <h4 class="transcript-title">${escapeHtml(item.title || 'Unknown Title')}</h4>
                <div class="transcript-meta">
                    ${item.has_transcript ? '<span class="status-badge success">Transcribed</span>' : '<span class="status-badge pending">Processing</span>'}
                    ${item.language ? `<span class="meta-badge">${item.language.toUpperCase()}</span>` : ''}
                </div>
            </div>
            <div class="transcript-info">
                <span class="duration">${formatDuration(item.duration)}</span>
                <span class="created">${formatDate(item.created_at)}</span>
                ${item.confidence_score ? `<span class="confidence">${Math.round(item.confidence_score * 100)}% confidence</span>` : ''}
            </div>
        </div>
    `).join('');
}

function filterTranscripts() {
    const searchTerm = elements.transcriptSearchInput.value.toLowerCase();
    const languageFilter = elements.languageFilter.value;
    
    // This would filter the displayed transcripts
    // Implementation depends on the current transcript data structure
}

// Transcript Modal Functions
async function openTranscriptModal(downloadId) {
    elements.transcriptModal.classList.remove('hidden');
    
    try {
        const response = await fetch(`${API_URL}/transcript/${downloadId}?include_segments=true`);
        if (!response.ok) throw new Error('Failed to load transcript');
        
        const transcript = await response.json();
        displayTranscriptContent(transcript);
    } catch (error) {
        console.error('Error loading transcript:', error);
        showToast('Failed to load transcript', 'error');
        closeTranscriptModal();
    }
}

function displayTranscriptContent(transcript) {
    elements.transcriptTitle.textContent = transcript.title || 'Transcript';
    elements.transcriptLanguage.textContent = transcript.language?.toUpperCase() || '';
    elements.transcriptConfidence.textContent = transcript.confidence_score ? 
        `${Math.round(transcript.confidence_score * 100)}% confidence` : '';
    elements.transcriptDuration.textContent = formatDuration(transcript.duration);
    
    // Display transcript with segments
    let transcriptHTML = '';
    if (transcript.segments && transcript.segments.length > 0) {
        transcriptHTML = transcript.segments.map(segment => `
            <div class="transcript-segment" data-start="${segment.start_time}" data-end="${segment.end_time}">
                <span class="timestamp">${formatTime(segment.start_time)}</span>
                <span class="text">${escapeHtml(segment.text)}</span>
            </div>
        `).join('');
    } else {
        transcriptHTML = `<div class="transcript-full-text">${escapeHtml(transcript.full_text || 'No transcript available')}</div>`;
    }
    
    elements.transcriptText.innerHTML = transcriptHTML;
}

function closeTranscriptModal() {
    elements.transcriptModal.classList.add('hidden');
    elements.transcriptSearchBar.classList.add('hidden');
    elements.transcriptExportOptions.classList.add('hidden');
}

function toggleTranscriptSearch() {
    elements.transcriptSearchBar.classList.toggle('hidden');
    if (!elements.transcriptSearchBar.classList.contains('hidden')) {
        elements.transcriptSearchQuery.focus();
    }
}

function toggleExportOptions() {
    elements.transcriptExportOptions.classList.toggle('hidden');
}

function searchWithinTranscript() {
    const query = elements.transcriptSearchQuery.value.toLowerCase();
    const segments = elements.transcriptText.querySelectorAll('.transcript-segment');
    let matches = 0;
    
    segments.forEach(segment => {
        const text = segment.querySelector('.text').textContent.toLowerCase();
        if (text.includes(query)) {
            segment.classList.add('search-match');
            matches++;
        } else {
            segment.classList.remove('search-match');
        }
    });
    
    elements.searchResultsCount.textContent = `${matches} results`;
}

function navigateSearchResults(direction) {
    // Implementation for navigating through search results
    const matches = elements.transcriptText.querySelectorAll('.search-match');
    // Navigation logic would go here
}

async function downloadTranscriptExport() {
    const format = document.querySelector('.format-btn.active').getAttribute('data-format');
    const includeTimestamps = elements.includeTimestamps.checked;
    const includeConfidence = elements.includeConfidence.checked;
    
    // Implementation for downloading transcript in selected format
    showToast(`Downloading transcript as ${format.toUpperCase()}...`, 'info');
}

function handleKeyboardShortcuts(e) {
    // Escape key to close modals
    if (e.key === 'Escape') {
        closeTranscriptModal();
    }
}

// Utility Functions
function formatTime(seconds) {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function formatDate(dateString) {
    return new Date(dateString).toLocaleDateString();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Global Functions for HTML onclick handlers
window.openTranscriptModal = openTranscriptModal;
window.generateTranscriptForDownload = async function(downloadId) {
    try {
        const response = await fetch(`${API_URL}/transcript/${downloadId}/generate`, { method: 'POST' });
        if (response.ok) {
            showToast('Transcript generation started', 'success');
            setTimeout(() => loadTranscripts(), 1000);
        }
    } catch (error) {
        showToast('Failed to generate transcript', 'error');
    }
};

// Export for global access
window.cancelDownload = cancelDownload;
