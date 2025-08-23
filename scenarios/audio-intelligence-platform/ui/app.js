const API_BASE_URL = window.location.protocol + '//' + window.location.hostname + ':' + (window.API_PORT || '5680');
const N8N_BASE_URL = window.location.protocol + '//' + window.location.hostname + ':' + (window.N8N_PORT || '5678');

let currentTranscriptionId = null;
let audioPlayer = null;
let waveformData = [];

document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
});

function initializeApp() {
    setupDropZone();
    setupTabs();
    setupAnalysisType();
    setupButtons();
    loadTranscriptions();
    
    audioPlayer = document.getElementById('audio-player');
}

function setupDropZone() {
    const dropZone = document.getElementById('drop-zone');
    const fileInput = document.getElementById('file-input');
    
    dropZone.addEventListener('click', () => fileInput.click());
    
    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropZone.classList.add('dragover');
    });
    
    dropZone.addEventListener('dragleave', () => {
        dropZone.classList.remove('dragover');
    });
    
    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropZone.classList.remove('dragover');
        handleFiles(e.dataTransfer.files);
    });
    
    fileInput.addEventListener('change', (e) => {
        handleFiles(e.target.files);
    });
}

function setupTabs() {
    const tabButtons = document.querySelectorAll('.tab-btn');
    const tabContents = document.querySelectorAll('.tab-content');
    
    tabButtons.forEach(btn => {
        btn.addEventListener('click', () => {
            const targetTab = btn.dataset.tab;
            
            tabButtons.forEach(b => b.classList.remove('active'));
            tabContents.forEach(c => c.classList.remove('active'));
            
            btn.classList.add('active');
            document.getElementById(targetTab + '-tab').classList.add('active');
        });
    });
}

function setupAnalysisType() {
    const analysisType = document.getElementById('analysis-type');
    const customPrompt = document.getElementById('custom-prompt');
    
    analysisType.addEventListener('change', () => {
        if (analysisType.value === 'custom') {
            customPrompt.classList.remove('hidden');
        } else {
            customPrompt.classList.add('hidden');
        }
    });
}

function setupButtons() {
    document.getElementById('copy-transcript').addEventListener('click', copyTranscript);
    document.getElementById('analyze-btn').addEventListener('click', analyzeTranscription);
    document.getElementById('search-btn').addEventListener('click', searchTranscriptions);
    document.getElementById('play-btn').addEventListener('click', playAudio);
    document.getElementById('pause-btn').addEventListener('click', pauseAudio);
    
    document.getElementById('search-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') searchTranscriptions();
    });
}

async function handleFiles(files) {
    for (const file of files) {
        await uploadFile(file);
    }
}

async function uploadFile(file) {
    const progressSection = document.getElementById('upload-progress');
    const progressFill = document.getElementById('progress-fill');
    const progressText = document.getElementById('progress-text');
    
    progressSection.classList.remove('hidden');
    progressText.textContent = `Uploading ${file.name}...`;
    
    try {
        const reader = new FileReader();
        const fileData = await new Promise((resolve) => {
            reader.onload = (e) => resolve(e.target.result.split(',')[1]);
            reader.readAsDataURL(file);
        });
        
        progressFill.style.width = '30%';
        progressText.textContent = 'Transcribing audio...';
        
        const response = await fetch(`${N8N_BASE_URL}/webhook/transcription-upload`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                filename: file.name,
                fileData: fileData,
                contentType: file.type,
                fileSizeBytes: file.size
            })
        });
        
        progressFill.style.width = '100%';
        
        if (response.ok) {
            const result = await response.json();
            progressText.textContent = 'Transcription completed!';
            
            currentTranscriptionId = result.transcriptionId;
            displayTranscription(result);
            loadTranscriptions();
            updateStats();
            
            setTimeout(() => {
                progressSection.classList.add('hidden');
                progressFill.style.width = '0%';
            }, 2000);
        } else {
            throw new Error('Upload failed');
        }
    } catch (error) {
        console.error('Upload error:', error);
        progressText.textContent = 'Upload failed. Please try again.';
        progressFill.style.backgroundColor = '#ff4444';
        
        setTimeout(() => {
            progressSection.classList.add('hidden');
            progressFill.style.width = '0%';
            progressFill.style.backgroundColor = '';
        }, 3000);
    }
}

function displayTranscription(data) {
    const transcriptionText = document.getElementById('transcription-text');
    transcriptionText.innerHTML = `
        <div class="transcription-content">
            <p>${data.transcriptionText || 'No transcription available'}</p>
            <div class="transcription-metadata">
                <span>Language: ${data.language || 'Unknown'}</span>
                <span>Confidence: ${(data.confidence * 100).toFixed(1)}%</span>
                <span>Duration: ${formatDuration(data.duration)}</span>
            </div>
        </div>
    `;
    
    drawWaveform();
}

async function loadTranscriptions() {
    try {
        const response = await fetch(`${API_BASE_URL}/api/transcriptions`);
        if (response.ok) {
            const transcriptions = await response.json();
            displayTranscriptionsList(transcriptions);
        }
    } catch (error) {
        console.error('Failed to load transcriptions:', error);
    }
}

function displayTranscriptionsList(transcriptions) {
    const listContainer = document.getElementById('transcriptions-list');
    
    if (!transcriptions || transcriptions.length === 0) {
        listContainer.innerHTML = '<p class="placeholder-text">No transcriptions yet</p>';
        return;
    }
    
    listContainer.innerHTML = transcriptions.map(t => `
        <div class="transcription-item" data-id="${t.id}" onclick="loadTranscription('${t.id}')">
            <div class="transcription-info">
                <div class="transcription-filename">${t.filename}</div>
                <div class="transcription-meta">
                    ${new Date(t.created_at).toLocaleString()} • ${formatDuration(t.duration_seconds)}
                </div>
            </div>
            <span class="transcription-status status-${t.status}">${t.status}</span>
        </div>
    `).join('');
}

async function loadTranscription(transcriptionId) {
    try {
        const response = await fetch(`${API_BASE_URL}/api/transcriptions/${transcriptionId}`);
        if (response.ok) {
            const data = await response.json();
            currentTranscriptionId = transcriptionId;
            displayTranscription(data);
        }
    } catch (error) {
        console.error('Failed to load transcription:', error);
    }
}

function copyTranscript() {
    const transcriptionText = document.querySelector('.transcription-content p');
    if (transcriptionText) {
        navigator.clipboard.writeText(transcriptionText.textContent);
        
        const copyBtn = document.getElementById('copy-transcript');
        const originalText = copyBtn.textContent;
        copyBtn.textContent = '✓ Copied!';
        setTimeout(() => {
            copyBtn.textContent = originalText;
        }, 2000);
    }
}

async function analyzeTranscription() {
    if (!currentTranscriptionId) {
        alert('Please select a transcription first');
        return;
    }
    
    const analysisType = document.getElementById('analysis-type').value;
    const customPrompt = document.getElementById('custom-prompt').value;
    const resultDiv = document.getElementById('analysis-result');
    
    resultDiv.innerHTML = '<p class="placeholder-text">Analyzing...</p>';
    
    try {
        const body = {
            transcriptionId: currentTranscriptionId,
            analysisType: analysisType
        };
        
        if (analysisType === 'custom') {
            body.customPrompt = customPrompt;
        }
        
        const response = await fetch(`${N8N_BASE_URL}/webhook/ai-analysis`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(body)
        });
        
        if (response.ok) {
            const result = await response.json();
            resultDiv.innerHTML = `
                <div class="analysis-content">
                    <h4>${analysisType === 'custom' ? 'Custom Analysis' : analysisType.replace('_', ' ').toUpperCase()}</h4>
                    <p>${result.analysis || 'No analysis available'}</p>
                </div>
            `;
        } else {
            throw new Error('Analysis failed');
        }
    } catch (error) {
        console.error('Analysis error:', error);
        resultDiv.innerHTML = '<p class="error-text">Analysis failed. Please try again.</p>';
    }
}

async function searchTranscriptions() {
    const query = document.getElementById('search-input').value.trim();
    if (!query) return;
    
    const resultsDiv = document.getElementById('search-results');
    resultsDiv.innerHTML = '<p class="placeholder-text">Searching...</p>';
    
    try {
        const response = await fetch(`${N8N_BASE_URL}/webhook/semantic-search`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ query: query })
        });
        
        if (response.ok) {
            const results = await response.json();
            displaySearchResults(results);
        } else {
            throw new Error('Search failed');
        }
    } catch (error) {
        console.error('Search error:', error);
        resultsDiv.innerHTML = '<p class="error-text">Search failed. Please try again.</p>';
    }
}

function displaySearchResults(results) {
    const resultsDiv = document.getElementById('search-results');
    
    if (!results || results.length === 0) {
        resultsDiv.innerHTML = '<p class="placeholder-text">No results found</p>';
        return;
    }
    
    resultsDiv.innerHTML = results.map(r => `
        <div class="search-result" onclick="loadTranscription('${r.transcription_id}')">
            <div class="result-filename">${r.filename}</div>
            <div class="result-excerpt">${r.excerpt}</div>
            <div class="result-score">Relevance: ${(r.score * 100).toFixed(1)}%</div>
        </div>
    `).join('');
}

function drawWaveform() {
    const canvas = document.getElementById('waveform-canvas');
    const ctx = canvas.getContext('2d');
    
    canvas.width = canvas.offsetWidth;
    canvas.height = 150;
    
    ctx.fillStyle = '#252540';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    
    const bars = 100;
    const barWidth = canvas.width / bars;
    const barGap = 2;
    
    for (let i = 0; i < bars; i++) {
        const barHeight = Math.random() * 100 + 20;
        const x = i * barWidth;
        const y = (canvas.height - barHeight) / 2;
        
        const gradient = ctx.createLinearGradient(0, y, 0, y + barHeight);
        gradient.addColorStop(0, '#00ff88');
        gradient.addColorStop(1, '#4a9eff');
        
        ctx.fillStyle = gradient;
        ctx.fillRect(x + barGap / 2, y, barWidth - barGap, barHeight);
    }
}

function playAudio() {
    if (audioPlayer && audioPlayer.src) {
        audioPlayer.play();
        document.getElementById('play-btn').style.display = 'none';
        document.getElementById('pause-btn').style.display = 'inline-block';
    }
}

function pauseAudio() {
    if (audioPlayer) {
        audioPlayer.pause();
        document.getElementById('play-btn').style.display = 'inline-block';
        document.getElementById('pause-btn').style.display = 'none';
    }
}

async function updateStats() {
    try {
        const response = await fetch(`${API_BASE_URL}/api/stats`);
        if (response.ok) {
            const stats = await response.json();
            document.getElementById('total-transcriptions').textContent = stats.totalTranscriptions || 0;
            document.getElementById('total-hours').textContent = formatHours(stats.totalHours || 0);
        }
    } catch (error) {
        console.error('Failed to update stats:', error);
    }
}

function formatDuration(seconds) {
    if (!seconds) return '0:00';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function formatHours(hours) {
    if (hours < 1) return `${Math.floor(hours * 60)}m`;
    return `${hours.toFixed(1)}h`;
}