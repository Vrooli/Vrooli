// Text Tools Application JavaScript

const API_BASE_URL = `http://localhost:${window.TEXT_TOOLS_PORT || 14000}/api/v1/text`;

// State management
const state = {
    currentTool: 'diff',
    transformPipeline: [],
    pipelineSteps: []
};

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    initializeTabs();
    initializeTools();
    checkAPIStatus();
    initializeEventListeners();
});

// Tab navigation
function initializeTabs() {
    const tabs = document.querySelectorAll('.nav-tab');
    const panels = document.querySelectorAll('.tool-panel');
    
    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tool = tab.dataset.tool;
            
            // Update tabs
            tabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            
            // Update panels
            panels.forEach(p => p.classList.remove('active'));
            document.getElementById(`${tool}-panel`).classList.add('active');
            
            state.currentTool = tool;
        });
    });
}

// Initialize tool-specific features
function initializeTools() {
    initializeDiffTool();
    initializeSearchTool();
    initializeTransformTool();
    initializeExtractTool();
    initializeAnalyzeTool();
    initializePipelineTool();
}

// Check API status
async function checkAPIStatus() {
    const indicator = document.getElementById('status-indicator');
    const dot = indicator.querySelector('.status-dot');
    const text = indicator.querySelector('.status-text');
    
    try {
        const response = await fetch(`${API_BASE_URL.replace('/api/v1/text', '/health')}`);
        if (response.ok) {
            dot.classList.add('connected');
            text.textContent = 'Connected';
        } else {
            text.textContent = 'API Error';
        }
    } catch (error) {
        text.textContent = 'Disconnected';
    }
}

// Initialize event listeners
function initializeEventListeners() {
    // API Documentation modal
    document.getElementById('api-docs-btn').addEventListener('click', () => {
        document.getElementById('api-docs-modal').classList.add('show');
    });
    
    document.querySelector('.modal-close').addEventListener('click', () => {
        document.getElementById('api-docs-modal').classList.remove('show');
    });
    
    // Close modal on outside click
    document.getElementById('api-docs-modal').addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            e.target.classList.remove('show');
        }
    });
}

// Diff Tool
function initializeDiffTool() {
    const compareBtn = document.getElementById('diff-compare-btn');
    
    compareBtn.addEventListener('click', async () => {
        const text1 = document.getElementById('diff-text1').value;
        const text2 = document.getElementById('diff-text2').value;
        const type = document.getElementById('diff-type').value;
        const ignoreWhitespace = document.getElementById('diff-ignore-whitespace').checked;
        const ignoreCase = document.getElementById('diff-ignore-case').checked;
        
        if (!text1 || !text2) {
            showNotification('Please enter both texts to compare', 'warning');
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE_URL}/diff`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text1,
                    text2,
                    options: {
                        type,
                        ignore_whitespace: ignoreWhitespace,
                        ignore_case: ignoreCase
                    }
                })
            });
            
            const data = await response.json();
            displayDiffResults(data);
        } catch (error) {
            showNotification('Failed to compare texts', 'error');
            console.error(error);
        }
    });
}

function displayDiffResults(data) {
    const resultsContainer = document.getElementById('diff-results');
    const similaritySpan = document.getElementById('diff-similarity');
    
    similaritySpan.textContent = `Similarity: ${(data.similarity_score * 100).toFixed(1)}%`;
    
    let html = '<div class="diff-results">';
    html += `<div class="diff-summary">${data.summary}</div>`;
    
    if (data.changes && data.changes.length > 0) {
        html += '<div class="diff-changes">';
        data.changes.forEach(change => {
            const typeClass = change.type === 'add' ? 'add' : change.type === 'remove' ? 'remove' : 'modify';
            html += `
                <div class="diff-change ${typeClass}">
                    <span class="change-type">${change.type.toUpperCase()}</span>
                    <span class="change-line">Line ${change.line_start}</span>
                    <span class="change-content">${escapeHtml(change.content)}</span>
                </div>
            `;
        });
        html += '</div>';
    }
    
    html += '</div>';
    resultsContainer.innerHTML = html;
}

// Search Tool
function initializeSearchTool() {
    const searchBtn = document.getElementById('search-btn');
    
    searchBtn.addEventListener('click', async () => {
        const pattern = document.getElementById('search-pattern').value;
        const text = document.getElementById('search-text').value;
        const regex = document.getElementById('search-regex').checked;
        const caseSensitive = document.getElementById('search-case-sensitive').checked;
        const wholeWord = document.getElementById('search-whole-word').checked;
        const fuzzy = document.getElementById('search-fuzzy').checked;
        const semantic = document.getElementById('search-semantic').checked;
        
        if (!pattern || !text) {
            showNotification('Please enter search pattern and text', 'warning');
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE_URL}/search`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text,
                    pattern,
                    options: {
                        regex,
                        case_sensitive: caseSensitive,
                        whole_word: wholeWord,
                        fuzzy,
                        semantic
                    }
                })
            });
            
            const data = await response.json();
            displaySearchResults(data);
        } catch (error) {
            showNotification('Search failed', 'error');
            console.error(error);
        }
    });
}

function displaySearchResults(data) {
    const resultsContainer = document.getElementById('search-results');
    
    let html = `<div class="search-summary">Found ${data.total_matches} matches</div>`;
    
    if (data.matches && data.matches.length > 0) {
        html += '<div class="search-matches">';
        data.matches.forEach(match => {
            html += `
                <div class="search-match">
                    <span class="match-location">Line ${match.line}, Column ${match.column}</span>
                    <div class="match-context">${highlightMatch(match.context, match.column - 1, match.length)}</div>
                </div>
            `;
        });
        html += '</div>';
    }
    
    resultsContainer.innerHTML = html;
}

// Transform Tool
function initializeTransformTool() {
    const transformBtns = document.querySelectorAll('.transform-btn');
    const inputTextarea = document.getElementById('transform-input');
    const outputTextarea = document.getElementById('transform-output');
    const copyBtn = document.getElementById('transform-copy-btn');
    const applyPipelineBtn = document.getElementById('transform-apply-pipeline');
    
    // Update character count
    inputTextarea.addEventListener('input', () => {
        document.getElementById('transform-input-count').textContent = `${inputTextarea.value.length} characters`;
    });
    
    // Transform buttons
    transformBtns.forEach(btn => {
        btn.addEventListener('click', async () => {
            const transform = btn.dataset.transform;
            const text = inputTextarea.value;
            
            if (!text) {
                showNotification('Please enter text to transform', 'warning');
                return;
            }
            
            // Add to pipeline
            state.transformPipeline.push(transform);
            updatePipelineDisplay();
            
            // Apply transformation
            const result = await applyTransformation(text, transform);
            outputTextarea.value = result;
        });
    });
    
    // Copy button
    copyBtn.addEventListener('click', () => {
        outputTextarea.select();
        document.execCommand('copy');
        showNotification('Copied to clipboard', 'success');
    });
    
    // Apply pipeline
    applyPipelineBtn.addEventListener('click', async () => {
        const text = inputTextarea.value;
        if (!text || state.transformPipeline.length === 0) {
            showNotification('Please enter text and add transformations', 'warning');
            return;
        }
        
        let result = text;
        for (const transform of state.transformPipeline) {
            result = await applyTransformation(result, transform);
        }
        outputTextarea.value = result;
    });
}

async function applyTransformation(text, transformType) {
    const transformationMap = {
        'upper': { type: 'case', parameters: { type: 'upper' } },
        'lower': { type: 'case', parameters: { type: 'lower' } },
        'title': { type: 'case', parameters: { type: 'title' } },
        'camel': { type: 'case', parameters: { type: 'camel' } },
        'snake': { type: 'case', parameters: { type: 'snake' } },
        'kebab': { type: 'case', parameters: { type: 'kebab' } },
        'base64-encode': { type: 'encode', parameters: { type: 'base64' } },
        'base64-decode': { type: 'decode', parameters: { type: 'base64' } },
        'url-encode': { type: 'encode', parameters: { type: 'url' } },
        'url-decode': { type: 'decode', parameters: { type: 'url' } },
        'html-escape': { type: 'encode', parameters: { type: 'html' } },
        'sanitize': { type: 'sanitize', parameters: {} }
    };
    
    const transformation = transformationMap[transformType];
    
    try {
        const response = await fetch(`${API_BASE_URL}/transform`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                text,
                transformations: [transformation]
            })
        });
        
        const data = await response.json();
        return data.result;
    } catch (error) {
        console.error('Transform error:', error);
        return text;
    }
}

function updatePipelineDisplay() {
    const pipelineContainer = document.getElementById('transform-pipeline');
    
    pipelineContainer.innerHTML = state.transformPipeline.map((step, index) => `
        <div class="pipeline-step">
            ${step}
            <button onclick="removePipelineStep(${index})">×</button>
        </div>
    `).join('');
}

function removePipelineStep(index) {
    state.transformPipeline.splice(index, 1);
    updatePipelineDisplay();
}

// Extract Tool
function initializeExtractTool() {
    const uploadArea = document.getElementById('extract-upload');
    const fileInput = document.getElementById('extract-file-input');
    const urlBtn = document.getElementById('extract-url-btn');
    
    // File upload
    uploadArea.addEventListener('click', () => fileInput.click());
    
    uploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadArea.classList.add('dragover');
    });
    
    uploadArea.addEventListener('dragleave', () => {
        uploadArea.classList.remove('dragover');
    });
    
    uploadArea.addEventListener('drop', async (e) => {
        e.preventDefault();
        uploadArea.classList.remove('dragover');
        
        const file = e.dataTransfer.files[0];
        if (file) {
            await extractFromFile(file);
        }
    });
    
    fileInput.addEventListener('change', async (e) => {
        const file = e.target.files[0];
        if (file) {
            await extractFromFile(file);
        }
    });
    
    // URL extraction
    urlBtn.addEventListener('click', async () => {
        const url = document.getElementById('extract-url').value;
        if (!url) {
            showNotification('Please enter a URL', 'warning');
            return;
        }
        
        await extractFromURL(url);
    });
}

async function extractFromFile(file) {
    const format = document.getElementById('extract-format').value;
    const ocr = document.getElementById('extract-ocr').checked;
    const metadata = document.getElementById('extract-metadata').checked;
    
    // Convert file to base64
    const reader = new FileReader();
    reader.onload = async (e) => {
        const base64 = e.target.result.split(',')[1];
        
        try {
            const response = await fetch(`${API_BASE_URL}/extract`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    source: { file: base64 },
                    format,
                    options: {
                        ocr,
                        extract_metadata: metadata
                    }
                })
            });
            
            const data = await response.json();
            displayExtractResults(data);
        } catch (error) {
            showNotification('Extraction failed', 'error');
            console.error(error);
        }
    };
    reader.readAsDataURL(file);
}

async function extractFromURL(url) {
    const format = document.getElementById('extract-format').value;
    const ocr = document.getElementById('extract-ocr').checked;
    const metadata = document.getElementById('extract-metadata').checked;
    
    try {
        const response = await fetch(`${API_BASE_URL}/extract`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                source: { url },
                format,
                options: {
                    ocr,
                    extract_metadata: metadata
                }
            })
        });
        
        const data = await response.json();
        displayExtractResults(data);
    } catch (error) {
        showNotification('Extraction failed', 'error');
        console.error(error);
    }
}

function displayExtractResults(data) {
    const resultsContainer = document.getElementById('extract-results');
    
    let html = '<div class="extract-results">';
    
    if (data.metadata && Object.keys(data.metadata).length > 0) {
        html += '<div class="extract-metadata">';
        html += '<h4>Metadata</h4>';
        for (const [key, value] of Object.entries(data.metadata)) {
            html += `<div class="metadata-item"><strong>${key}:</strong> ${value}</div>`;
        }
        html += '</div>';
    }
    
    html += '<div class="extract-text">';
    html += '<h4>Extracted Text</h4>';
    html += `<pre>${escapeHtml(data.text)}</pre>`;
    html += '</div>';
    
    html += '</div>';
    resultsContainer.innerHTML = html;
}

// Analyze Tool
function initializeAnalyzeTool() {
    const analyzeBtn = document.getElementById('analyze-btn');
    const summaryCheckbox = document.getElementById('analyze-summary');
    const summaryControls = document.getElementById('summary-controls');
    
    // Toggle summary controls
    summaryCheckbox.addEventListener('change', () => {
        summaryControls.style.display = summaryCheckbox.checked ? 'flex' : 'none';
    });
    
    analyzeBtn.addEventListener('click', async () => {
        const text = document.getElementById('analyze-text').value;
        if (!text) {
            showNotification('Please enter text to analyze', 'warning');
            return;
        }
        
        const analyses = [];
        if (document.getElementById('analyze-entities').checked) analyses.push('entities');
        if (document.getElementById('analyze-sentiment').checked) analyses.push('sentiment');
        if (document.getElementById('analyze-keywords').checked) analyses.push('keywords');
        if (document.getElementById('analyze-language').checked) analyses.push('language');
        if (document.getElementById('analyze-summary').checked) analyses.push('summary');
        
        const summaryLength = parseInt(document.getElementById('summary-length').value);
        
        try {
            const response = await fetch(`${API_BASE_URL}/analyze`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    text,
                    analyses,
                    options: {
                        summary_length: summaryLength
                    }
                })
            });
            
            const data = await response.json();
            displayAnalysisResults(data);
        } catch (error) {
            showNotification('Analysis failed', 'error');
            console.error(error);
        }
    });
}

function displayAnalysisResults(data) {
    const statsContainer = document.getElementById('text-stats');
    const sectionsContainer = document.getElementById('analysis-sections');
    
    // Display basic stats
    statsContainer.innerHTML = `
        <div class="stat-card">
            <div class="stat-label">Language</div>
            <div class="stat-value">${data.language ? data.language.code.toUpperCase() : 'N/A'}</div>
        </div>
        <div class="stat-card">
            <div class="stat-label">Sentiment</div>
            <div class="stat-value">${data.sentiment ? data.sentiment.label : 'N/A'}</div>
        </div>
    `;
    
    // Display detailed sections
    let html = '';
    
    if (data.entities && data.entities.length > 0) {
        html += '<div class="analysis-section">';
        html += '<h3>Named Entities</h3>';
        html += '<div class="entity-list">';
        data.entities.forEach(entity => {
            html += `
                <div class="entity-item">
                    <span class="entity-type">${entity.type}</span>
                    <span class="entity-value">${entity.value}</span>
                    <span class="entity-confidence">${(entity.confidence * 100).toFixed(0)}%</span>
                </div>
            `;
        });
        html += '</div></div>';
    }
    
    if (data.keywords && data.keywords.length > 0) {
        html += '<div class="analysis-section">';
        html += '<h3>Keywords</h3>';
        html += '<div class="keyword-list">';
        data.keywords.forEach(keyword => {
            html += `
                <div class="keyword-item">
                    <span class="keyword-word">${keyword.word}</span>
                    <span class="keyword-score">${(keyword.score * 100).toFixed(1)}%</span>
                </div>
            `;
        });
        html += '</div></div>';
    }
    
    if (data.summary) {
        html += '<div class="analysis-section">';
        html += '<h3>Summary</h3>';
        html += `<p class="summary-text">${data.summary}</p>`;
        html += '</div>';
    }
    
    sectionsContainer.innerHTML = html;
}

// Pipeline Tool
function initializePipelineTool() {
    const operationItems = document.querySelectorAll('.operation-item');
    const pipelineFlow = document.getElementById('pipeline-flow');
    const runBtn = document.getElementById('pipeline-run-btn');
    const saveBtn = document.getElementById('pipeline-save-btn');
    
    // Drag and drop for pipeline
    operationItems.forEach(item => {
        item.addEventListener('dragstart', (e) => {
            e.dataTransfer.setData('operation', item.dataset.op);
            item.classList.add('dragging');
        });
        
        item.addEventListener('dragend', () => {
            item.classList.remove('dragging');
        });
    });
    
    pipelineFlow.addEventListener('dragover', (e) => {
        e.preventDefault();
    });
    
    pipelineFlow.addEventListener('drop', (e) => {
        e.preventDefault();
        const operation = e.dataTransfer.getData('operation');
        addPipelineOperation(operation);
    });
    
    // Run pipeline
    runBtn.addEventListener('click', async () => {
        const input = document.getElementById('pipeline-input').value;
        if (!input || state.pipelineSteps.length === 0) {
            showNotification('Please enter input and add operations', 'warning');
            return;
        }
        
        const result = await runPipeline(input);
        document.getElementById('pipeline-output').textContent = result;
    });
    
    // Save pipeline
    saveBtn.addEventListener('click', () => {
        const pipelineName = prompt('Enter pipeline name:');
        if (pipelineName) {
            savePipeline(pipelineName);
        }
    });
}

function addPipelineOperation(operation) {
    state.pipelineSteps.push(operation);
    updatePipelineFlow();
}

function updatePipelineFlow() {
    const flow = document.getElementById('pipeline-flow');
    
    let html = '<div class="pipeline-node start-node"><span>Input</span></div>';
    
    if (state.pipelineSteps.length > 0) {
        state.pipelineSteps.forEach((step, index) => {
            html += `
                <div class="pipeline-node operation-node">
                    <span>${step}</span>
                    <button class="remove-node" onclick="removePipelineOperation(${index})">×</button>
                </div>
            `;
        });
    } else {
        html += '<div class="pipeline-placeholder">Drag operations here to build your pipeline</div>';
    }
    
    html += '<div class="pipeline-node end-node"><span>Output</span></div>';
    
    flow.innerHTML = html;
}

function removePipelineOperation(index) {
    state.pipelineSteps.splice(index, 1);
    updatePipelineFlow();
}

async function runPipeline(input) {
    let result = input;
    
    for (const operation of state.pipelineSteps) {
        // Apply each operation
        // This would call the appropriate API endpoint based on the operation
        result = await applyPipelineOperation(result, operation);
    }
    
    return result;
}

async function applyPipelineOperation(text, operation) {
    // Implement pipeline operation logic
    // This would map operations to API calls
    return text; // Placeholder
}

function savePipeline(name) {
    const pipeline = {
        name,
        steps: state.pipelineSteps,
        created: new Date().toISOString()
    };
    
    // Save to localStorage or API
    localStorage.setItem(`pipeline_${name}`, JSON.stringify(pipeline));
    showNotification('Pipeline saved successfully', 'success');
}

// Utility functions
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function highlightMatch(text, start, length) {
    const before = text.substring(0, start);
    const match = text.substring(start, start + length);
    const after = text.substring(start + length);
    
    return `${escapeHtml(before)}<mark>${escapeHtml(match)}</mark>${escapeHtml(after)}`;
}

function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `notification notification-${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 1rem 1.5rem;
        background: ${type === 'error' ? '#ff4444' : type === 'warning' ? '#ffaa00' : type === 'success' ? '#00ff88' : '#00aaff'};
        color: ${type === 'error' || type === 'warning' ? 'white' : 'black'};
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
        z-index: 10000;
        animation: slideIn 0.3s ease;
    `;
    
    document.body.appendChild(notification);
    
    // Remove after 3 seconds
    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
    
    .diff-change {
        padding: 0.5rem;
        margin: 0.25rem 0;
        border-radius: 4px;
        font-size: 0.875rem;
    }
    
    .diff-change.add {
        background: rgba(0, 255, 136, 0.1);
        border-left: 3px solid #00ff88;
    }
    
    .diff-change.remove {
        background: rgba(255, 68, 68, 0.1);
        border-left: 3px solid #ff4444;
    }
    
    .diff-change.modify {
        background: rgba(255, 170, 0, 0.1);
        border-left: 3px solid #ffaa00;
    }
    
    .search-match {
        padding: 0.75rem;
        background: var(--bg-tertiary);
        border-radius: 4px;
        margin: 0.5rem 0;
    }
    
    .match-location {
        color: var(--accent-info);
        font-size: 0.75rem;
    }
    
    .match-context {
        margin-top: 0.25rem;
        font-family: monospace;
    }
    
    mark {
        background: var(--accent-warning);
        color: var(--bg-primary);
        padding: 0 2px;
    }
    
    .entity-item, .keyword-item {
        display: inline-flex;
        align-items: center;
        gap: 0.5rem;
        padding: 0.25rem 0.75rem;
        background: var(--bg-tertiary);
        border-radius: 20px;
        margin: 0.25rem;
    }
    
    .entity-type {
        font-size: 0.75rem;
        color: var(--accent-info);
        text-transform: uppercase;
    }
    
    .pipeline-node {
        position: relative;
    }
    
    .remove-node {
        position: absolute;
        top: -8px;
        right: -8px;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: var(--accent-danger);
        color: white;
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 12px;
    }
`;
document.head.appendChild(style);

// Set API status check interval
setInterval(checkAPIStatus, 30000);