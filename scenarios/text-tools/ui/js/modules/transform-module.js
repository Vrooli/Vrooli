// Transform Module - Handles text transformation functionality
class TransformModule {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.transformations = [];
        this.initialize();
    }

    initialize() {
        // Wait for component to be loaded
        document.addEventListener('componentsLoaded', () => {
            this.initializeElements();
            this.attachEventListeners();
        });
    }

    initializeElements() {
        this.inputTextarea = document.getElementById('transform-input');
        this.outputTextarea = document.getElementById('transform-output');
        this.transformButton = document.getElementById('transform-btn');
        this.resultsContainer = document.getElementById('transform-results');
        this.transformationChain = document.getElementById('transformation-chain');
        
        // Transform option checkboxes
        this.upperCheckbox = document.getElementById('transform-upper');
        this.lowerCheckbox = document.getElementById('transform-lower');
        this.titleCheckbox = document.getElementById('transform-title');
        this.sanitizeCheckbox = document.getElementById('transform-sanitize');
        this.base64Checkbox = document.getElementById('transform-base64');
        this.fileInput = document.getElementById('transform-file');
    }

    attachEventListeners() {
        if (this.transformButton) {
            this.transformButton.addEventListener('click', () => this.performTransformation());
        }

        // File upload
        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => this.handleFileUpload(e));
        }

        // Transform checkboxes - update chain when changed
        [this.upperCheckbox, this.lowerCheckbox, this.titleCheckbox, 
         this.sanitizeCheckbox, this.base64Checkbox].forEach(checkbox => {
            if (checkbox) {
                checkbox.addEventListener('change', () => this.updateTransformationChain());
            }
        });

        // Keyboard shortcuts
        if (this.inputTextarea) {
            this.inputTextarea.addEventListener('keydown', (e) => {
                if (e.ctrlKey && e.key === 'Enter') {
                    this.performTransformation();
                }
            });
        }
    }

    updateTransformationChain() {
        this.transformations = [];

        if (this.upperCheckbox?.checked) {
            this.transformations.push({ type: 'upper', parameters: {} });
        }
        if (this.lowerCheckbox?.checked) {
            this.transformations.push({ type: 'lower', parameters: {} });
        }
        if (this.titleCheckbox?.checked) {
            this.transformations.push({ type: 'title', parameters: {} });
        }
        if (this.sanitizeCheckbox?.checked) {
            this.transformations.push({ type: 'sanitize', parameters: {} });
        }
        if (this.base64Checkbox?.checked) {
            this.transformations.push({ type: 'base64', parameters: {} });
        }

        this.displayTransformationChain();
    }

    displayTransformationChain() {
        if (!this.transformationChain) return;

        if (this.transformations.length === 0) {
            this.transformationChain.innerHTML = '<span class="chain-empty">No transformations selected</span>';
            return;
        }

        const chainHtml = this.transformations.map((transform, index) => {
            const displayName = this.getTransformDisplayName(transform.type);
            return `<span class="chain-step" data-index="${index}">${displayName}</span>`;
        }).join('<span class="chain-arrow">→</span>');

        this.transformationChain.innerHTML = chainHtml;
    }

    getTransformDisplayName(type) {
        const names = {
            'upper': 'UPPERCASE',
            'lower': 'lowercase',
            'title': 'Title Case',
            'sanitize': 'Sanitize HTML',
            'base64': 'Base64 Encode'
        };
        return names[type] || type;
    }

    async handleFileUpload(event) {
        const file = event.target.files[0];
        if (!file) return;

        try {
            const text = await this.readFile(file);
            if (this.inputTextarea) {
                this.inputTextarea.value = text;
            }
        } catch (error) {
            console.error('File upload error:', error);
            this.showError('Failed to read file: ' + error.message);
        }
    }

    readFile(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = (e) => resolve(e.target.result);
            reader.onerror = (e) => reject(new Error('File read failed'));
            reader.readAsText(file);
        });
    }

    async performTransformation() {
        const text = this.inputTextarea?.value;

        if (!text) {
            this.showError('Please enter text to transform');
            return;
        }

        this.updateTransformationChain();

        if (this.transformations.length === 0) {
            this.showError('Please select at least one transformation');
            return;
        }

        this.showLoading();

        try {
            const result = await this.apiClient.transformText(text, this.transformations);
            this.displayResults(result);
        } catch (error) {
            console.error('Transform error:', error);
            this.showError(error.message || 'Transformation failed');
        }
    }

    displayResults(result) {
        const { result: transformedText, transformations_applied, warnings } = result;

        // Update output textarea
        if (this.outputTextarea) {
            this.outputTextarea.value = transformedText;
        }

        // Show results info
        if (this.resultsContainer) {
            let html = `
                <div class="results-summary">
                    <h4>Transformation Complete</h4>
                    <div class="transformation-stats">
                        <span class="stat">
                            <strong>Applied:</strong> ${transformations_applied.join(', ')}
                        </span>
                        <span class="stat">
                            <strong>Original length:</strong> ${this.inputTextarea?.value.length || 0} characters
                        </span>
                        <span class="stat">
                            <strong>Result length:</strong> ${transformedText.length} characters
                        </span>
                    </div>
                </div>
            `;

            if (warnings && warnings.length > 0) {
                html += `
                    <div class="warnings">
                        <h5>Warnings:</h5>
                        <ul>
                            ${warnings.map(warning => `<li>${this.escapeHtml(warning)}</li>`).join('')}
                        </ul>
                    </div>
                `;
            }

            this.resultsContainer.innerHTML = html;
        }
    }

    showLoading() {
        if (this.outputTextarea) {
            this.outputTextarea.value = '';
        }
        
        if (this.resultsContainer) {
            this.resultsContainer.innerHTML = `
                <div class="loading-state">
                    <div class="loading-spinner"></div>
                    <p>Transforming text...</p>
                </div>
            `;
        }
    }

    showError(message) {
        if (this.resultsContainer) {
            this.resultsContainer.innerHTML = `
                <div class="error-state">
                    <h4>Transformation Error</h4>
                    <p>${this.escapeHtml(message)}</p>
                </div>
            `;
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Public methods for external access
    clearTransformation() {
        if (this.inputTextarea) this.inputTextarea.value = '';
        if (this.outputTextarea) this.outputTextarea.value = '';
        if (this.resultsContainer) this.resultsContainer.innerHTML = '';
        
        // Reset checkboxes
        [this.upperCheckbox, this.lowerCheckbox, this.titleCheckbox, 
         this.sanitizeCheckbox, this.base64Checkbox].forEach(checkbox => {
            if (checkbox) checkbox.checked = false;
        });
        
        this.updateTransformationChain();
    }

    setInputText(text) {
        if (this.inputTextarea) {
            this.inputTextarea.value = text;
        }
    }

    getOutputText() {
        return this.outputTextarea?.value || '';
    }
}

// Global utility functions for component
window.copyToClipboard = function(elementId) {
    const element = document.getElementById(elementId);
    if (element) {
        element.select();
        document.execCommand('copy');
        
        // Show feedback
        const button = event.target;
        const originalText = button.textContent;
        button.textContent = 'Copied!';
        setTimeout(() => {
            button.textContent = originalText;
        }, 2000);
    }
};

window.downloadText = function(elementId, filename) {
    const element = document.getElementById(elementId);
    if (element && element.value) {
        const blob = new Blob([element.value], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }
};

// Export for use in other modules
window.TransformModule = TransformModule;