// Extract Module - Handles text extraction from various formats
class ExtractModule {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.currentSource = 'file';
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
        this.formatSelect = document.getElementById('extract-format');
        this.ocrCheckbox = document.getElementById('extract-ocr');
        this.preserveFormattingCheckbox = document.getElementById('extract-preserve-formatting');
        this.metadataCheckbox = document.getElementById('extract-metadata');
        this.fileInput = document.getElementById('extract-file-input');
        this.urlInput = document.getElementById('extract-url');
        this.extractButton = document.getElementById('extract-btn');
        this.progressIndicator = document.getElementById('extract-progress');
        this.outputTextarea = document.getElementById('extract-output');
        this.metadataSection = document.getElementById('extract-metadata-section');
        this.metadataContent = document.getElementById('extract-metadata-content');
        this.resultsContainer = document.getElementById('extract-results');
        
        // Source tabs
        this.sourceTabs = document.querySelectorAll('.source-tab');
        this.sourcePanels = document.querySelectorAll('.source-panel');
    }

    attachEventListeners() {
        if (this.extractButton) {
            this.extractButton.addEventListener('click', () => this.performExtraction());
        }

        // Source tabs
        this.sourceTabs.forEach(tab => {
            tab.addEventListener('click', () => this.switchSourceTab(tab.dataset.source));
        });

        // File upload
        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => this.handleFileSelection(e));
        }

        // URL input
        if (this.urlInput) {
            this.urlInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    this.performExtraction();
                }
            });
        }

        // Format selection change
        if (this.formatSelect) {
            this.formatSelect.addEventListener('change', () => this.updateFormatOptions());
        }

        // Drag and drop for file input
        if (this.fileInput && this.fileInput.parentElement) {
            const dropZone = this.fileInput.parentElement;
            
            ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
                dropZone.addEventListener(eventName, (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                });
            });

            ['dragenter', 'dragover'].forEach(eventName => {
                dropZone.addEventListener(eventName, () => {
                    dropZone.classList.add('drag-active');
                });
            });

            ['dragleave', 'drop'].forEach(eventName => {
                dropZone.addEventListener(eventName, () => {
                    dropZone.classList.remove('drag-active');
                });
            });

            dropZone.addEventListener('drop', (e) => {
                const files = e.dataTransfer.files;
                if (files.length > 0) {
                    this.fileInput.files = files;
                    this.handleFileSelection({ target: { files } });
                }
            });
        }
    }

    switchSourceTab(source) {
        this.currentSource = source;

        // Update tab appearance
        this.sourceTabs.forEach(tab => {
            tab.classList.toggle('active', tab.dataset.source === source);
        });

        // Update panel visibility
        this.sourcePanels.forEach(panel => {
            panel.classList.toggle('active', panel.dataset.source === source);
        });
    }

    handleFileSelection(event) {
        const file = event.target.files[0];
        if (!file) return;

        // Auto-detect format based on file extension
        const extension = file.name.split('.').pop().toLowerCase();
        const formatMap = {
            'pdf': 'pdf',
            'doc': 'docx',
            'docx': 'docx',
            'html': 'html',
            'htm': 'html',
            'png': 'image',
            'jpg': 'image',
            'jpeg': 'image',
            'gif': 'image'
        };

        if (this.formatSelect && formatMap[extension]) {
            this.formatSelect.value = formatMap[extension];
            this.updateFormatOptions();
        }
    }

    updateFormatOptions() {
        const format = this.formatSelect?.value;
        
        // Show/hide OCR option for images
        if (this.ocrCheckbox && this.ocrCheckbox.parentElement) {
            this.ocrCheckbox.parentElement.style.display = 
                (format === 'image' || format === 'pdf') ? 'block' : 'none';
        }
    }

    async performExtraction() {
        let source;
        
        if (this.currentSource === 'file') {
            const file = this.fileInput?.files[0];
            if (!file) {
                this.showError('Please select a file to extract from');
                return;
            }
            
            // Convert file to base64
            try {
                const base64 = await this.fileToBase64(file);
                source = { file: base64 };
            } catch (error) {
                this.showError('Failed to read file: ' + error.message);
                return;
            }
        } else {
            const url = this.urlInput?.value;
            if (!url) {
                this.showError('Please enter a URL to extract from');
                return;
            }
            source = { url };
        }

        const options = {
            format: this.formatSelect?.value || 'auto',
            ocr: this.ocrCheckbox?.checked || false,
            preserve_formatting: this.preserveFormattingCheckbox?.checked || false,
            extract_metadata: this.metadataCheckbox?.checked || false
        };

        this.showLoading();

        try {
            const result = await this.apiClient.extractText(source, options);
            this.displayResults(result);
        } catch (error) {
            console.error('Extract error:', error);
            this.showError(error.message || 'Text extraction failed');
        }
    }

    fileToBase64(file) {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onload = () => {
                const base64 = reader.result.split(',')[1];
                resolve(base64);
            };
            reader.onerror = () => reject(new Error('Failed to read file'));
            reader.readAsDataURL(file);
        });
    }

    displayResults(result) {
        const { text, metadata, warnings } = result;

        // Update output textarea
        if (this.outputTextarea) {
            this.outputTextarea.value = text;
        }

        // Show metadata if extracted
        if (metadata && Object.keys(metadata).length > 0 && this.metadataContent) {
            this.displayMetadata(metadata);
            if (this.metadataSection) {
                this.metadataSection.style.display = 'block';
            }
        } else if (this.metadataSection) {
            this.metadataSection.style.display = 'none';
        }

        // Show results summary
        if (this.resultsContainer) {
            let html = `
                <div class="results-summary">
                    <h4>Text Extraction Complete</h4>
                    <div class="extraction-stats">
                        <span class="stat">
                            <strong>Characters extracted:</strong> ${text.length}
                        </span>
                        <span class="stat">
                            <strong>Words extracted:</strong> ${text.split(/\s+/).filter(word => word.length > 0).length}
                        </span>
                        <span class="stat">
                            <strong>Lines extracted:</strong> ${text.split('\n').length}
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

        this.hideLoading();
    }

    displayMetadata(metadata) {
        if (!this.metadataContent) return;

        let html = '<div class="metadata-grid">';
        
        Object.entries(metadata).forEach(([key, value]) => {
            const displayKey = key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
            let displayValue = value;
            
            if (typeof value === 'object') {
                displayValue = JSON.stringify(value, null, 2);
            }
            
            html += `
                <div class="metadata-item">
                    <dt class="metadata-key">${this.escapeHtml(displayKey)}</dt>
                    <dd class="metadata-value">${this.escapeHtml(String(displayValue))}</dd>
                </div>
            `;
        });
        
        html += '</div>';
        this.metadataContent.innerHTML = html;
    }

    showLoading() {
        if (this.outputTextarea) {
            this.outputTextarea.value = '';
        }
        
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = `
                <div class="loading-state">
                    <div class="loading-spinner"></div>
                    <p>Extracting text...</p>
                </div>
            `;
        }
        
        if (this.extractButton) {
            this.extractButton.disabled = true;
            this.extractButton.textContent = 'Extracting...';
        }
    }

    hideLoading() {
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = '';
        }
        
        if (this.extractButton) {
            this.extractButton.disabled = false;
            this.extractButton.textContent = 'Extract Text';
        }
    }

    showError(message) {
        this.hideLoading();
        
        if (this.resultsContainer) {
            this.resultsContainer.innerHTML = `
                <div class="error-state">
                    <h4>Extraction Error</h4>
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
    clearExtraction() {
        if (this.outputTextarea) this.outputTextarea.value = '';
        if (this.urlInput) this.urlInput.value = '';
        if (this.fileInput) this.fileInput.value = '';
        if (this.resultsContainer) this.resultsContainer.innerHTML = '';
        if (this.metadataSection) this.metadataSection.style.display = 'none';
        
        // Reset checkboxes
        [this.ocrCheckbox, this.preserveFormattingCheckbox, this.metadataCheckbox].forEach(checkbox => {
            if (checkbox) checkbox.checked = false;
        });
        
        // Reset format select
        if (this.formatSelect) this.formatSelect.value = 'auto';
    }

    getExtractedText() {
        return this.outputTextarea?.value || '';
    }
}

// Export for use in other modules
window.ExtractModule = ExtractModule;