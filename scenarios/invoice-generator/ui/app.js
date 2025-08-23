// ACCOUNTING WIZARD '95 - System Core
const API_PORT = window.SERVICE_PORT || 8100;
const API_BASE = `http://localhost:${API_PORT}`;

// Invoice statistics tracker
let invoiceStats = {
    count: 0,
    totalRevenue: 0,
    totalTax: 0,
    invoices: []
};

// Initialize on load
document.addEventListener('DOMContentLoaded', function() {
    initializeInvoice();
    setupEventListeners();
    updatePreview();
    startClock();
    loadStats();
    playStartupSound();
});

// Retro clock display
function startClock() {
    const updateClock = () => {
        const now = new Date();
        const timeStr = now.toLocaleTimeString('en-US', { hour12: false });
        const dateStr = now.toLocaleDateString('en-US', { 
            year: 'numeric', 
            month: '2-digit', 
            day: '2-digit' 
        });
        const clockEl = document.getElementById('clock');
        if (clockEl) {
            clockEl.textContent = `[${dateStr} ${timeStr}]`;
        }
    };
    updateClock();
    setInterval(updateClock, 1000);
}

// Load saved statistics
function loadStats() {
    const saved = localStorage.getItem('invoiceStats');
    if (saved) {
        invoiceStats = JSON.parse(saved);
        updateStatsDisplay();
    }
}

// Update statistics display
function updateStatsDisplay() {
    document.getElementById('total-invoices').textContent = invoiceStats.count;
    document.getElementById('total-revenue').textContent = `$${invoiceStats.totalRevenue.toFixed(2)}`;
    const avgInvoice = invoiceStats.count > 0 ? invoiceStats.totalRevenue / invoiceStats.count : 0;
    document.getElementById('avg-invoice').textContent = `$${avgInvoice.toFixed(2)}`;
    document.getElementById('tax-collected').textContent = `$${invoiceStats.totalTax.toFixed(2)}`;
}

// Play retro startup sound (simulated)
function playStartupSound() {
    // Create a simple beep sound using Web Audio API
    try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();
        
        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);
        
        oscillator.frequency.value = 800;
        oscillator.type = 'square';
        gainNode.gain.value = 0.1;
        
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.1);
    } catch (e) {
        // Silently fail if audio doesn't work
    }
}

// Initialize invoice with defaults
function initializeInvoice() {
    const today = new Date();
    const dueDate = new Date(today);
    dueDate.setDate(dueDate.getDate() + 30);
    
    document.getElementById('issue_date').value = today.toISOString().split('T')[0];
    document.getElementById('due_date').value = dueDate.toISOString().split('T')[0];
    document.getElementById('invoice_number').value = `INV-${Date.now().toString().slice(-6)}`;
}

// Setup event listeners
function setupEventListeners() {
    // Update preview on any input change
    document.querySelectorAll('input, textarea, select').forEach(element => {
        element.addEventListener('input', updatePreview);
        element.addEventListener('change', updatePreview);
    });
    
    // Template selection
    document.querySelectorAll('.template-option').forEach(option => {
        option.addEventListener('click', function() {
            document.querySelectorAll('.template-option').forEach(o => o.classList.remove('selected'));
            this.classList.add('selected');
            updatePreview();
        });
    });
}

// Add new line item
function addItem() {
    const container = document.getElementById('items-container');
    const itemRow = document.createElement('div');
    itemRow.className = 'item-row';
    itemRow.innerHTML = `
        <input type="text" placeholder="Description" class="item-description">
        <input type="number" placeholder="Qty" class="item-quantity" value="1" min="1">
        <input type="number" placeholder="Price" class="item-price" value="0" min="0" step="0.01">
        <input type="number" placeholder="Tax %" class="item-tax" value="8" min="0" max="100">
        <button class="btn btn-danger" onclick="removeItem(this)">🗑️</button>
    `;
    container.appendChild(itemRow);
    
    // Add event listeners to new inputs
    itemRow.querySelectorAll('input').forEach(input => {
        input.addEventListener('input', updatePreview);
    });
    
    updatePreview();
}

// Remove line item
function removeItem(button) {
    const row = button.parentElement;
    row.style.animation = 'slideOut 0.3s ease';
    setTimeout(() => {
        row.remove();
        updatePreview();
    }, 300);
}

// Update live preview
function updatePreview() {
    // Update header info
    document.getElementById('preview-company').textContent = 
        document.getElementById('company_name').value || 'Your Company';
    document.getElementById('preview-client').textContent = 
        document.getElementById('client_name').value || 'Client Name';
    document.getElementById('preview-invoice-number').textContent = 
        `Invoice #${document.getElementById('invoice_number').value || 'INV-001'}`;
    
    const dueDate = document.getElementById('due_date').value;
    const issueDate = document.getElementById('issue_date').value;
    if (dueDate && issueDate) {
        const days = Math.round((new Date(dueDate) - new Date(issueDate)) / (1000 * 60 * 60 * 24));
        document.getElementById('preview-dates').textContent = `Due in ${days} days`;
    }
    
    // Calculate totals
    let subtotal = 0;
    let totalTax = 0;
    const itemsHtml = [];
    
    document.querySelectorAll('.item-row').forEach(row => {
        const description = row.querySelector('.item-description').value;
        const quantity = parseFloat(row.querySelector('.item-quantity').value) || 0;
        const price = parseFloat(row.querySelector('.item-price').value) || 0;
        const taxRate = parseFloat(row.querySelector('.item-tax').value) || 0;
        
        if (description && quantity && price) {
            const lineTotal = quantity * price;
            const taxAmount = lineTotal * (taxRate / 100);
            subtotal += lineTotal;
            totalTax += taxAmount;
            
            itemsHtml.push(`
                <div style="display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px solid #e5e7eb;">
                    <span style="flex: 2;">${description}</span>
                    <span style="flex: 1; text-align: center;">${quantity}</span>
                    <span style="flex: 1; text-align: right;">$${price.toFixed(2)}</span>
                    <span style="flex: 1; text-align: right;">$${lineTotal.toFixed(2)}</span>
                </div>
            `);
        }
    });
    
    document.getElementById('preview-items').innerHTML = itemsHtml.join('');
    
    const currency = document.getElementById('currency').value;
    const currencySymbol = getCurrencySymbol(currency);
    
    document.getElementById('preview-subtotal').textContent = `${currencySymbol}${subtotal.toFixed(2)}`;
    document.getElementById('preview-tax').textContent = `${currencySymbol}${totalTax.toFixed(2)}`;
    document.getElementById('preview-total').textContent = `${currencySymbol}${(subtotal + totalTax).toFixed(2)}`;
}

// Get currency symbol
function getCurrencySymbol(currency) {
    const symbols = {
        'USD': '$',
        'EUR': '€',
        'GBP': '£',
        'CAD': 'C$'
    };
    return symbols[currency] || '$';
}

// Gather invoice data
function gatherInvoiceData() {
    const items = [];
    document.querySelectorAll('.item-row').forEach(row => {
        const description = row.querySelector('.item-description').value;
        const quantity = parseFloat(row.querySelector('.item-quantity').value) || 0;
        const unit_price = parseFloat(row.querySelector('.item-price').value) || 0;
        const tax_rate = (parseFloat(row.querySelector('.item-tax').value) || 0) / 100;
        
        if (description && quantity && unit_price) {
            items.push({ description, quantity, unit_price, tax_rate });
        }
    });
    
    return {
        company_name: document.getElementById('company_name').value,
        company_address: document.getElementById('company_address').value,
        company_email: document.getElementById('company_email').value,
        company_phone: document.getElementById('company_phone').value,
        client_name: document.getElementById('client_name').value,
        client_address: document.getElementById('client_address').value,
        client_email: document.getElementById('client_email').value,
        invoice_number: document.getElementById('invoice_number').value,
        issue_date: document.getElementById('issue_date').value,
        due_date: document.getElementById('due_date').value,
        currency: document.getElementById('currency').value,
        payment_terms: document.getElementById('payment_terms').value,
        notes: document.getElementById('notes').value,
        bank_details: document.getElementById('bank_details').value,
        items: items
    };
}

// Show receipt printer animation
function showReceiptPrinter(invoiceData) {
    const printer = document.getElementById('receipt-printer');
    const receiptContent = document.getElementById('receipt-content');
    
    // Calculate totals
    let subtotal = 0;
    let totalTax = 0;
    invoiceData.items.forEach(item => {
        const lineTotal = item.quantity * item.unit_price;
        const taxAmount = lineTotal * item.tax_rate;
        subtotal += lineTotal;
        totalTax += taxAmount;
    });
    const total = subtotal + totalTax;
    
    // Format receipt content
    receiptContent.innerHTML = `
        INV# ${invoiceData.invoice_number}<br>
        ${new Date().toLocaleDateString()}<br>
        ----------------<br>
        TOTAL: $${total.toFixed(2)}<br>
    `;
    
    printer.classList.add('active');
    
    // Play printing sound
    playPrintSound();
    
    // Hide after animation
    setTimeout(() => {
        printer.classList.remove('active');
    }, 3000);
}

// Play printing sound effect
function playPrintSound() {
    try {
        const audioContext = new (window.AudioContext || window.webkitAudioContext)();
        const oscillator = audioContext.createOscillator();
        const gainNode = audioContext.createGain();
        
        oscillator.connect(gainNode);
        gainNode.connect(audioContext.destination);
        
        // Create printer-like sound
        oscillator.type = 'sawtooth';
        oscillator.frequency.setValueAtTime(100, audioContext.currentTime);
        oscillator.frequency.exponentialRampToValueAtTime(50, audioContext.currentTime + 0.5);
        
        gainNode.gain.setValueAtTime(0.05, audioContext.currentTime);
        gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.5);
        
        oscillator.start();
        oscillator.stop(audioContext.currentTime + 0.5);
    } catch (e) {
        // Silently fail
    }
}

// Save invoice to database
async function saveInvoice() {
    const loading = document.getElementById('loading');
    const successMsg = document.getElementById('success-message');
    
    loading.classList.add('active');
    successMsg.classList.remove('active');
    
    try {
        const invoiceData = gatherInvoiceData();
        
        const response = await fetch(`${API_BASE}/api/invoice`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                action: 'create',
                invoice: invoiceData
            })
        });
        
        if (!response.ok) {
            throw new Error('Failed to save invoice');
        }
        
        const result = await response.json();
        
        // Update statistics
        let subtotal = 0;
        let totalTax = 0;
        invoiceData.items.forEach(item => {
            const lineTotal = item.quantity * item.unit_price;
            const taxAmount = lineTotal * item.tax_rate;
            subtotal += lineTotal;
            totalTax += taxAmount;
        });
        const total = subtotal + totalTax;
        
        invoiceStats.count++;
        invoiceStats.totalRevenue += total;
        invoiceStats.totalTax += totalTax;
        invoiceStats.invoices.push({
            id: result.id || Date.now(),
            number: invoiceData.invoice_number,
            total: total,
            date: new Date().toISOString()
        });
        
        // Save stats to localStorage
        localStorage.setItem('invoiceStats', JSON.stringify(invoiceStats));
        updateStatsDisplay();
        
        loading.classList.remove('active');
        successMsg.textContent = `[SUCCESS] Invoice ${result.invoice_number} saved! Database ID: ${result.id || 'PENDING'}`;
        successMsg.classList.add('active');
        
        // Show receipt printer animation
        showReceiptPrinter(invoiceData);
        
        setTimeout(() => {
            successMsg.classList.remove('active');
        }, 5000);
        
    } catch (error) {
        loading.classList.remove('active');
        alert('[ERROR] Failed to save invoice: ' + error.message);
    }
}

// Generate PDF
async function generatePDF() {
    const loading = document.getElementById('loading');
    const successMsg = document.getElementById('success-message');
    
    loading.classList.add('active');
    successMsg.classList.remove('active');
    
    try {
        const invoiceData = gatherInvoiceData();
        const selectedTemplate = document.querySelector('.template-option.selected').dataset.template;
        
        const response = await fetch(`${API_BASE}/api/invoice/pdf`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                invoice: invoiceData,
                format: 'pdf',
                template: selectedTemplate
            })
        });
        
        if (!response.ok) {
            throw new Error('Failed to generate PDF');
        }
        
        const result = await response.json();
        
        // Download the PDF
        if (result.pdf_base64) {
            const link = document.createElement('a');
            link.href = 'data:application/pdf;base64,' + result.pdf_base64;
            link.download = result.filename || `invoice_${invoiceData.invoice_number}.pdf`;
            link.click();
        }
        
        loading.classList.remove('active');
        successMsg.textContent = `📄 PDF generated and downloaded successfully!`;
        successMsg.classList.add('active');
        
        setTimeout(() => {
            successMsg.classList.remove('active');
        }, 5000);
        
    } catch (error) {
        loading.classList.remove('active');
        alert('Error generating PDF: ' + error.message);
    }
}

// Add some fun animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideOut {
        to {
            opacity: 0;
            transform: translateX(-100%);
        }
    }
    
    @keyframes pulse {
        0%, 100% { transform: scale(1); }
        50% { transform: scale(1.05); }
    }
    
    .btn:active {
        animation: pulse 0.3s ease;
    }
`;
document.head.appendChild(style);