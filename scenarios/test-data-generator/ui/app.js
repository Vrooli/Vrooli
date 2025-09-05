class TestDataGenerator {
    constructor() {
        this.apiUrl = `http://localhost:${window.location.port === '3002' ? '3001' : '3001'}`;
        this.dataTypes = {};
        this.currentData = null;
        
        this.initializeElements();
        this.attachEventListeners();
        this.loadDataTypes();
    }

    initializeElements() {
        this.elements = {
            dataType: document.getElementById('dataType'),
            count: document.getElementById('count'),
            format: document.getElementById('format'),
            seed: document.getElementById('seed'),
            fieldsSection: document.getElementById('fieldsSection'),
            fieldsList: document.getElementById('fieldsList'),
            customSchemaSection: document.getElementById('customSchemaSection'),
            schemaBuilder: document.getElementById('schemaBuilder'),
            addField: document.getElementById('addField'),
            generateBtn: document.getElementById('generateBtn'),
            clearBtn: document.getElementById('clearBtn'),
            loading: document.getElementById('loading'),
            output: document.getElementById('output'),
            outputContent: document.getElementById('outputContent'),
            copyBtn: document.getElementById('copyBtn'),
            downloadBtn: document.getElementById('downloadBtn')
        };
    }

    attachEventListeners() {
        this.elements.dataType.addEventListener('change', () => this.handleDataTypeChange());
        this.elements.generateBtn.addEventListener('click', () => this.generateData());
        this.elements.clearBtn.addEventListener('click', () => this.clearOutput());
        this.elements.copyBtn.addEventListener('click', () => this.copyToClipboard());
        this.elements.downloadBtn.addEventListener('click', () => this.downloadData());
        this.elements.addField.addEventListener('click', () => this.addSchemaField());
        
        // Handle Enter key in form inputs
        document.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && e.target.tagName !== 'TEXTAREA') {
                this.generateData();
            }
        });
    }

    async loadDataTypes() {
        try {
            const response = await fetch(`${this.apiUrl}/api/types`);
            const data = await response.json();
            this.dataTypes = data.definitions || {};
            this.updateFieldsList();
        } catch (error) {
            console.error('Failed to load data types:', error);
            this.showError('Failed to connect to API. Please ensure the API server is running.');
        }
    }

    handleDataTypeChange() {
        const selectedType = this.elements.dataType.value;
        
        if (selectedType === 'custom') {
            this.elements.fieldsSection.style.display = 'none';
            this.elements.customSchemaSection.style.display = 'block';
        } else {
            this.elements.fieldsSection.style.display = 'block';
            this.elements.customSchemaSection.style.display = 'none';
            this.updateFieldsList();
        }
    }

    updateFieldsList() {
        const selectedType = this.elements.dataType.value;
        const typeData = this.dataTypes[selectedType];
        
        if (!typeData || !typeData.fields) {
            this.elements.fieldsList.innerHTML = '<p>No fields available for this data type.</p>';
            return;
        }

        this.elements.fieldsList.innerHTML = typeData.fields.map(field => `
            <div class="field-checkbox">
                <input type="checkbox" id="field-${field}" value="${field}" checked>
                <label for="field-${field}">${this.formatFieldName(field)}</label>
            </div>
        `).join('');
    }

    formatFieldName(field) {
        return field.charAt(0).toUpperCase() + field.slice(1).replace(/([A-Z])/g, ' $1');
    }

    addSchemaField() {
        const fieldDiv = document.createElement('div');
        fieldDiv.className = 'schema-field';
        fieldDiv.innerHTML = `
            <input type="text" placeholder="Field name" class="field-name">
            <select class="field-type">
                <option value="string">String</option>
                <option value="integer">Integer</option>
                <option value="decimal">Decimal</option>
                <option value="boolean">Boolean</option>
                <option value="email">Email</option>
                <option value="phone">Phone</option>
                <option value="date">Date</option>
                <option value="uuid">UUID</option>
            </select>
            <button type="button" class="remove-field">Remove</button>
        `;

        const removeBtn = fieldDiv.querySelector('.remove-field');
        removeBtn.addEventListener('click', () => {
            fieldDiv.remove();
        });

        this.elements.schemaBuilder.appendChild(fieldDiv);
    }

    getSelectedFields() {
        const checkboxes = this.elements.fieldsList.querySelectorAll('input[type="checkbox"]:checked');
        return Array.from(checkboxes).map(cb => cb.value);
    }

    getCustomSchema() {
        const schemaFields = this.elements.schemaBuilder.querySelectorAll('.schema-field');
        const schema = {};

        schemaFields.forEach(field => {
            const name = field.querySelector('.field-name').value.trim();
            const type = field.querySelector('.field-type').value;

            if (name) {
                schema[name] = type;
            }
        });

        return schema;
    }

    async generateData() {
        const dataType = this.elements.dataType.value;
        const count = parseInt(this.elements.count.value) || 10;
        const format = this.elements.format.value;
        const seed = this.elements.seed.value.trim();

        if (count < 1 || count > 10000) {
            this.showError('Count must be between 1 and 10,000');
            return;
        }

        this.showLoading(true);
        this.elements.generateBtn.disabled = true;

        try {
            let requestData = { count, format };
            
            if (seed) {
                requestData.seed = seed;
            }

            let endpoint = `/api/generate/${dataType}`;

            if (dataType === 'custom') {
                const schema = this.getCustomSchema();
                if (Object.keys(schema).length === 0) {
                    throw new Error('Please add at least one field to the custom schema');
                }
                requestData.schema = schema;
                endpoint = '/api/generate/custom';
            } else {
                const selectedFields = this.getSelectedFields();
                if (selectedFields.length > 0) {
                    requestData.fields = selectedFields;
                }
            }

            const response = await fetch(`${this.apiUrl}${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestData)
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || `HTTP ${response.status}: ${response.statusText}`);
            }

            const result = await response.json();
            this.displayResult(result);

        } catch (error) {
            console.error('Generation failed:', error);
            this.showError(error.message || 'Failed to generate data');
        } finally {
            this.showLoading(false);
            this.elements.generateBtn.disabled = false;
        }
    }

    displayResult(result) {
        this.currentData = result;
        
        let displayData;
        if (result.format === 'json') {
            displayData = JSON.stringify(result.data, null, 2);
        } else {
            displayData = result.data;
        }

        this.elements.outputContent.innerHTML = `
            <div class="success">
                ✅ Generated ${result.count} ${result.type} record(s) in ${result.format.toUpperCase()} format
                ${result.timestamp ? `<br>Generated at: ${new Date(result.timestamp).toLocaleString()}` : ''}
            </div>
            <pre>${this.escapeHtml(displayData)}</pre>
        `;

        this.elements.copyBtn.style.display = 'inline-block';
        this.elements.downloadBtn.style.display = 'inline-block';
    }

    showError(message) {
        this.elements.outputContent.innerHTML = `
            <div class="error">
                ❌ Error: ${this.escapeHtml(message)}
            </div>
            <div class="placeholder">
                <p>Please check your API server is running on port 3001</p>
                <p>Try: <code>cd api && npm install && npm start</code></p>
            </div>
        `;
        
        this.elements.copyBtn.style.display = 'none';
        this.elements.downloadBtn.style.display = 'none';
    }

    showLoading(show) {
        this.elements.loading.style.display = show ? 'flex' : 'none';
        this.elements.output.style.display = show ? 'none' : 'flex';
    }

    clearOutput() {
        this.currentData = null;
        this.elements.outputContent.innerHTML = '<p class="placeholder">Generated data will appear here...</p>';
        this.elements.copyBtn.style.display = 'none';
        this.elements.downloadBtn.style.display = 'none';
    }

    async copyToClipboard() {
        if (!this.currentData) return;

        try {
            let textToCopy;
            if (this.currentData.format === 'json') {
                textToCopy = JSON.stringify(this.currentData.data, null, 2);
            } else {
                textToCopy = this.currentData.data;
            }

            await navigator.clipboard.writeText(textToCopy);
            
            // Show feedback
            const originalText = this.elements.copyBtn.textContent;
            this.elements.copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                this.elements.copyBtn.textContent = originalText;
            }, 2000);

        } catch (error) {
            console.error('Failed to copy:', error);
            this.showError('Failed to copy to clipboard');
        }
    }

    downloadData() {
        if (!this.currentData) return;

        let filename, content, mimeType;

        switch (this.currentData.format) {
            case 'json':
                filename = `test-data-${this.currentData.type}-${Date.now()}.json`;
                content = JSON.stringify(this.currentData.data, null, 2);
                mimeType = 'application/json';
                break;
            case 'csv':
                filename = `test-data-${this.currentData.type}-${Date.now()}.csv`;
                content = this.convertToCSV(this.currentData.data);
                mimeType = 'text/csv';
                break;
            case 'xml':
                filename = `test-data-${this.currentData.type}-${Date.now()}.xml`;
                content = this.currentData.data;
                mimeType = 'application/xml';
                break;
            case 'sql':
                filename = `test-data-${this.currentData.type}-${Date.now()}.sql`;
                content = this.currentData.data;
                mimeType = 'application/sql';
                break;
            default:
                filename = `test-data-${this.currentData.type}-${Date.now()}.txt`;
                content = JSON.stringify(this.currentData.data, null, 2);
                mimeType = 'text/plain';
        }

        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }

    convertToCSV(data) {
        if (!data || data.length === 0) return '';

        const headers = Object.keys(data[0]);
        const csvRows = [headers.join(',')];

        data.forEach(row => {
            const values = headers.map(header => {
                const value = row[header];
                // Handle nested objects and arrays
                const stringValue = typeof value === 'object' ? JSON.stringify(value) : String(value);
                // Escape quotes and wrap in quotes if contains comma or quotes
                return stringValue.includes(',') || stringValue.includes('"') 
                    ? `"${stringValue.replace(/"/g, '""')}"` 
                    : stringValue;
            });
            csvRows.push(values.join(','));
        });

        return csvRows.join('\n');
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    new TestDataGenerator();
});

// Service worker registration for offline support (optional)
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js')
            .then(registration => console.log('SW registered'))
            .catch(registrationError => console.log('SW registration failed'));
    });
}