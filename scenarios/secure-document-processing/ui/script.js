import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

// Initialize the iframe bridge when embedded inside the orchestrator
const bootstrapIframeBridge = () => {
    if (window.__secureDocumentBridgeInitialized) {
        return;
    }

    initIframeBridgeChild({ appId: 'secure-document-processing-ui' });
    window.__secureDocumentBridgeInitialized = true;
};

if (typeof window !== 'undefined' && window.parent !== window) {
    bootstrapIframeBridge();
}

// Global state
let documentQueue = [];
let processingJobs = [];
let auditEntries = [];

// API Configuration
const API_BASE = 'http://localhost:8090/api';
// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    setupWebSocket();
    loadDocuments();
    updateStats();
});

// Initialize application
function initializeApp() {
    console.log('SecureVault Pro initialized');
    addAuditEntry('success', 'Application initialized successfully');
    
    // Set up periodic updates
    setInterval(updateStats, 5000);
    setInterval(checkJobStatus, 3000);
}

// File handling
function handleFileSelect(event) {
    const files = event.target.files;
    processFiles(files);
}

function handleDrop(event) {
    event.preventDefault();
    const uploadZone = document.getElementById('uploadZone');
    uploadZone.classList.remove('dragover');
    
    const files = event.dataTransfer.files;
    processFiles(files);
}

function handleDragOver(event) {
    event.preventDefault();
    const uploadZone = document.getElementById('uploadZone');
    uploadZone.classList.add('dragover');
}

function handleDragLeave(event) {
    event.preventDefault();
    const uploadZone = document.getElementById('uploadZone');
    uploadZone.classList.remove('dragover');
}

// Process files
function processFiles(files) {
    for (let file of files) {
        const fileObj = {
            id: generateId(),
            name: file.name,
            size: formatFileSize(file.size),
            type: file.type,
            status: 'pending',
            uploadedAt: new Date().toISOString(),
            file: file
        };
        
        documentQueue.push(fileObj);
        addDocumentToList(fileObj);
        addAuditEntry('success', `File "${file.name}" added to queue`);
    }
    
    updateStats();
}

// Process document queue
async function processDocuments() {
    const pendingDocs = documentQueue.filter(doc => doc.status === 'pending');
    
    if (pendingDocs.length === 0) {
        addAuditEntry('warning', 'No documents in queue to process');
        return;
    }
    
    addAuditEntry('success', `Processing ${pendingDocs.length} documents...`);
    
    for (let doc of pendingDocs) {
        doc.status = 'processing';
        updateDocumentStatus(doc.id, 'processing');
        
        try {
            const formData = new FormData();
            formData.append('file', doc.file);
            formData.append('encryption', document.getElementById('encryptionLevel').value);
            formData.append('compliance', document.getElementById('complianceFramework').value);
            formData.append('audit', document.getElementById('enableAudit').checked);
            formData.append('versioning', document.getElementById('enableVersioning').checked);
            formData.append('redaction', document.getElementById('enableRedaction').checked);
            
            const response = await fetch(`${API_BASE}/documents/upload`, {
                method: 'POST',
                body: formData
            });
            
            if (response.ok) {
                doc.status = 'encrypted';
                updateDocumentStatus(doc.id, 'encrypted');
                addAuditEntry('success', `Document "${doc.name}" encrypted successfully`);
            } else {
                throw new Error('Upload failed');
            }
        } catch (error) {
            doc.status = 'error';
            updateDocumentStatus(doc.id, 'error');
            addAuditEntry('danger', `Failed to process "${doc.name}": ${error.message}`);
        }
    }
    
    updateStats();
}

// Add document to UI list
function addDocumentToList(doc) {
    const documentList = document.getElementById('documentList');
    
    const documentItem = document.createElement('div');
    documentItem.className = 'document-item';
    documentItem.id = `doc-${doc.id}`;
    
    documentItem.innerHTML = `
        <div class="document-info">
            <div class="document-icon">
                <i class="fas fa-file-alt"></i>
            </div>
            <div class="document-details">
                <div class="document-name">${doc.name}</div>
                <div class="document-meta">${doc.size} • ${formatTime(doc.uploadedAt)}</div>
            </div>
        </div>
        <div class="document-status status-${doc.status}" id="status-${doc.id}">
            <i class="fas fa-${getStatusIcon(doc.status)}"></i>
            <span>${getStatusText(doc.status)}</span>
        </div>
    `;
    
    documentList.insertBefore(documentItem, documentList.firstChild);
    
    // Limit to 10 visible documents
    while (documentList.children.length > 10) {
        documentList.removeChild(documentList.lastChild);
    }
}

// Update document status
function updateDocumentStatus(docId, status) {
    const statusElement = document.getElementById(`status-${docId}`);
    if (statusElement) {
        statusElement.className = `document-status status-${status}`;
        statusElement.innerHTML = `
            <i class="fas fa-${getStatusIcon(status)}"></i>
            <span>${getStatusText(status)}</span>
        `;
    }
}

// Load existing documents
async function loadDocuments() {
    try {
        const response = await fetch(`${API_BASE}/documents`);
        if (response.ok) {
            const documents = await response.json();
            documents.forEach(doc => {
                addDocumentToList(doc);
                documentQueue.push(doc);
            });
        }
    } catch (error) {
        console.error('Failed to load documents:', error);
    }
}

// Update statistics
async function updateStats() {
    const totalDocs = documentQueue.length;
    const encryptedDocs = documentQueue.filter(d => d.status === 'encrypted').length;
    const activeJobs = documentQueue.filter(d => d.status === 'processing').length;
    
    document.getElementById('totalDocs').textContent = totalDocs;
    document.getElementById('encryptedDocs').textContent = encryptedDocs;
    document.getElementById('activeJobs').textContent = activeJobs;
}

// Check job status
async function checkJobStatus() {
    try {
        const response = await fetch(`${API_BASE}/jobs/active`);
        if (response.ok) {
            const jobs = await response.json();
            processingJobs = jobs;
            
            // Update document statuses based on job status
            jobs.forEach(job => {
                const doc = documentQueue.find(d => d.id === job.documentId);
                if (doc && doc.status !== job.status) {
                    doc.status = job.status;
                    updateDocumentStatus(doc.id, job.status);
                }
            });
        }
    } catch (error) {
        console.error('Failed to check job status:', error);
    }
}

// Audit log management
function addAuditEntry(type, message) {
    const auditLog = document.getElementById('auditLog');
    
    const entry = document.createElement('div');
    entry.className = `audit-entry audit-${type}`;
    
    const time = new Date().toLocaleString('en-US', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
    
    entry.innerHTML = `
        <div class="audit-time">${time}</div>
        <div class="audit-message">${message}</div>
    `;
    
    auditLog.insertBefore(entry, auditLog.firstChild);
    
    // Keep only last 20 entries
    while (auditLog.children.length > 20) {
        auditLog.removeChild(auditLog.lastChild);
    }
    
    // Store in audit entries array
    auditEntries.push({ type, message, timestamp: new Date() });
}

// Quick actions
async function exportAuditLog() {
    const csvContent = auditEntries.map(entry => 
        `"${entry.timestamp.toISOString()}","${entry.type}","${entry.message}"`
    ).join('\n');
    
    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-log-${new Date().toISOString()}.csv`;
    a.click();
    
    addAuditEntry('success', 'Audit log exported successfully');
}

async function rotateKeys() {
    addAuditEntry('warning', 'Initiating key rotation...');
    
    try {
        const response = await fetch(`${API_BASE}/security/rotate-keys`, {
            method: 'POST'
        });
        
        if (response.ok) {
            addAuditEntry('success', 'Encryption keys rotated successfully');
        } else {
            throw new Error('Key rotation failed');
        }
    } catch (error) {
        addAuditEntry('danger', `Key rotation failed: ${error.message}`);
    }
}

async function purgeExpired() {
    if (!confirm('Are you sure you want to purge all expired documents? This action cannot be undone.')) {
        return;
    }
    
    addAuditEntry('warning', 'Purging expired documents...');
    
    try {
        const response = await fetch(`${API_BASE}/documents/purge-expired`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            const result = await response.json();
            addAuditEntry('success', `Purged ${result.count} expired documents`);
            loadDocuments();
        } else {
            throw new Error('Purge failed');
        }
    } catch (error) {
        addAuditEntry('danger', `Purge failed: ${error.message}`);
    }
}

// WebSocket for real-time updates
function setupWebSocket() {
    const ws = new WebSocket('ws://localhost:8090/ws');
    
    ws.onopen = () => {
        console.log('WebSocket connected');
        addAuditEntry('success', 'Real-time connection established');
    };
    
    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        
        if (data.type === 'document-update') {
            const doc = documentQueue.find(d => d.id === data.documentId);
            if (doc) {
                doc.status = data.status;
                updateDocumentStatus(doc.id, data.status);
            }
        } else if (data.type === 'audit-log') {
            addAuditEntry(data.level, data.message);
        }
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        addAuditEntry('danger', 'Real-time connection error');
    };
    
    ws.onclose = () => {
        console.log('WebSocket disconnected');
        addAuditEntry('warning', 'Real-time connection lost, attempting reconnect...');
        setTimeout(setupWebSocket, 5000);
    };
}

// Utility functions
function generateId() {
    return Math.random().toString(36).substr(2, 9);
}

function formatFileSize(bytes) {
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    if (bytes === 0) return '0 Bytes';
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);
    
    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)} min ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} hours ago`;
    return date.toLocaleDateString();
}

function getStatusIcon(status) {
    const icons = {
        pending: 'clock',
        processing: 'spinner fa-spin',
        encrypted: 'lock',
        error: 'exclamation-triangle'
    };
    return icons[status] || 'file';
}

function getStatusText(status) {
    const texts = {
        pending: 'Pending',
        processing: 'Processing',
        encrypted: 'Encrypted',
        error: 'Error'
    };
    return texts[status] || status;
}
