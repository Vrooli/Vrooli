// QR Code Generator JavaScript
const API_URL = window.location.hostname === 'localhost' 
    ? `http://localhost:${process.env.API_PORT || 9100}`
    : '/api';

let currentQRData = null;
let batchItems = [];

// Initialize event listeners
document.addEventListener('DOMContentLoaded', () => {
    // Main generator
    document.getElementById('generate-btn').addEventListener('click', generateQRCode);
    document.getElementById('download-btn').addEventListener('click', downloadQR);
    document.getElementById('copy-btn').addEventListener('click', copyURL);
    
    // Color pickers
    document.getElementById('qr-color').addEventListener('change', updateColorDisplay);
    document.getElementById('qr-bg').addEventListener('change', updateBgDisplay);
    
    // Batch mode
    document.getElementById('add-batch-item').addEventListener('click', addBatchItem);
    document.getElementById('process-batch').addEventListener('click', processBatch);
    
    // Initial color display
    updateColorDisplay();
    updateBgDisplay();
});

function updateColorDisplay() {
    const color = document.getElementById('qr-color').value;
    document.getElementById('color-hex').textContent = color.toUpperCase();
}

function updateBgDisplay() {
    const bg = document.getElementById('qr-bg').value;
    document.getElementById('bg-hex').textContent = bg.toUpperCase();
}

async function generateQRCode() {
    const text = document.getElementById('qr-text').value.trim();
    
    if (!text) {
        showStatus('ERROR: No text provided', 'error');
        return;
    }
    
    const button = document.getElementById('generate-btn');
    const originalText = button.innerHTML;
    button.innerHTML = '<span class="button-text">⏳ PROCESSING...</span>';
    button.disabled = true;
    
    showStatus('GENERATING QR CODE...', 'processing');
    
    const params = {
        text: text,
        size: parseInt(document.getElementById('qr-size').value),
        color: document.getElementById('qr-color').value,
        background: document.getElementById('qr-bg').value,
        errorCorrection: document.getElementById('qr-correction').value,
        format: 'png'
    };
    
    try {
        const response = await fetch(`${API_URL}/generate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(params)
        });
        
        const data = await response.json();
        
        if (data.success) {
            displayQRCode(data);
            showStatus('QR CODE GENERATED SUCCESSFULLY', 'success');
        } else {
            throw new Error(data.message || 'Generation failed');
        }
    } catch (error) {
        console.error('Error:', error);
        showStatus('ERROR: ' + error.message, 'error');
    } finally {
        button.innerHTML = originalText;
        button.disabled = false;
    }
}

function displayQRCode(data) {
    currentQRData = data;
    const display = document.getElementById('qr-display');
    
    // Create QR image element
    const img = document.createElement('img');
    img.src = data.base64 ? `data:image/png;base64,${data.base64}` : data.download_url;
    img.alt = 'Generated QR Code';
    img.style.maxWidth = '100%';
    img.style.imageRendering = 'pixelated';
    
    display.innerHTML = '';
    display.appendChild(img);
    
    // Show action buttons
    document.getElementById('action-buttons').style.display = 'flex';
}

function downloadQR() {
    if (!currentQRData) return;
    
    const link = document.createElement('a');
    link.href = currentQRData.base64 
        ? `data:image/png;base64,${currentQRData.base64}`
        : currentQRData.download_url;
    link.download = `qr_code_${Date.now()}.png`;
    link.click();
    
    showStatus('DOWNLOAD INITIATED', 'success');
}

function copyURL() {
    if (!currentQRData) return;
    
    const text = document.getElementById('qr-text').value.trim();
    navigator.clipboard.writeText(text).then(() => {
        showStatus('TEXT COPIED TO CLIPBOARD', 'success');
    }).catch(err => {
        showStatus('COPY FAILED', 'error');
    });
}

// Batch processing
function addBatchItem() {
    const itemDiv = document.createElement('div');
    itemDiv.className = 'batch-item';
    itemDiv.innerHTML = `
        <input type="text" placeholder="Enter text/URL" class="batch-input">
        <input type="text" placeholder="Label (optional)" class="batch-label">
        <button onclick="removeBatchItem(this)">✖</button>
    `;
    
    document.getElementById('batch-items').appendChild(itemDiv);
    updateBatchCount();
}

function removeBatchItem(button) {
    button.parentElement.remove();
    updateBatchCount();
}

function updateBatchCount() {
    const items = document.querySelectorAll('.batch-item');
    const count = items.length;
    document.getElementById('batch-count').textContent = count;
    document.getElementById('process-batch').style.display = count > 0 ? 'block' : 'none';
}

async function processBatch() {
    const items = [];
    document.querySelectorAll('.batch-item').forEach((item, index) => {
        const text = item.querySelector('.batch-input').value.trim();
        const label = item.querySelector('.batch-label').value.trim() || `QR_${index + 1}`;
        
        if (text) {
            items.push({ text, label });
        }
    });
    
    if (items.length === 0) {
        showStatus('ERROR: No valid items in batch', 'error');
        return;
    }
    
    const button = document.getElementById('process-batch');
    const originalText = button.innerHTML;
    button.innerHTML = '⏳ PROCESSING BATCH...';
    button.disabled = true;
    
    showStatus(`PROCESSING ${items.length} ITEMS...`, 'processing');
    
    const batchRequest = {
        items: items,
        options: {
            size: parseInt(document.getElementById('qr-size').value),
            color: document.getElementById('qr-color').value,
            background: document.getElementById('qr-bg').value,
            errorCorrection: document.getElementById('qr-correction').value,
            format: 'png'
        }
    };
    
    try {
        const response = await fetch(`${API_URL}/batch`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(batchRequest)
        });
        
        const data = await response.json();
        
        if (data.success) {
            showStatus(`BATCH COMPLETE: ${data.successful}/${data.total} GENERATED`, 'success');
            // Could display results in a modal or download as zip
        } else {
            throw new Error(data.message || 'Batch processing failed');
        }
    } catch (error) {
        console.error('Error:', error);
        showStatus('ERROR: ' + error.message, 'error');
    } finally {
        button.innerHTML = originalText;
        button.disabled = false;
    }
}

function showStatus(message, type = 'normal') {
    const status = document.getElementById('status');
    status.textContent = message;
    status.className = 'status-text';
    
    if (type === 'processing') {
        status.classList.add('processing');
    }
    
    if (type === 'error') {
        status.style.color = '#ff0000';
    } else if (type === 'success') {
        status.style.color = '#00ff00';
    }
    
    // Reset color after 3 seconds
    setTimeout(() => {
        status.style.color = '';
        status.className = 'status-text';
    }, 3000);
}