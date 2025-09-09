class ImageToolsApp {
    constructor() {
        this.apiUrl = window.location.protocol + '//' + window.location.hostname + ':' + (window.API_PORT || '8080');
        this.currentFile = null;
        this.currentOperation = 'compress';
        this.processedImageUrl = null;
        
        this.initializeElements();
        this.attachEventListeners();
        this.updateControlsVisibility();
    }
    
    initializeElements() {
        // Drop zone
        this.dropZone = document.getElementById('drop-zone');
        this.fileInput = document.getElementById('file-input');
        
        // Preview elements
        this.previewContainer = document.getElementById('preview-container');
        this.originalImage = document.getElementById('original-image');
        this.processedImage = document.getElementById('processed-image');
        this.originalSize = document.getElementById('original-size');
        this.processedSize = document.getElementById('processed-size');
        
        // Stats
        this.compressionRatio = document.getElementById('compression-ratio');
        this.savingsPercent = document.getElementById('savings-percent');
        this.dimensions = document.getElementById('dimensions');
        
        // Controls
        this.operationButtons = document.querySelectorAll('.toggle-switch');
        this.processBtn = document.getElementById('process-btn');
        this.qualityDial = document.getElementById('quality-dial');
        this.widthInput = document.getElementById('width-input');
        this.heightInput = document.getElementById('height-input');
        this.maintainAspect = document.getElementById('maintain-aspect');
        
        // Control groups
        this.compressControls = document.getElementById('compress-controls');
        this.resizeControls = document.getElementById('resize-controls');
        this.convertControls = document.getElementById('convert-controls');
        this.metadataControls = document.getElementById('metadata-controls');
        
        // Metadata viewer
        this.metadataViewer = document.getElementById('metadata-viewer');
        this.metadataContent = document.getElementById('metadata-content');
        
        // Status lights
        this.statusLights = document.querySelectorAll('.status-light');
    }
    
    attachEventListeners() {
        // File input
        this.dropZone.addEventListener('click', () => this.fileInput.click());
        this.fileInput.addEventListener('change', (e) => this.handleFileSelect(e.target.files[0]));
        
        // Drag and drop
        this.dropZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            this.dropZone.classList.add('dragover');
        });
        
        this.dropZone.addEventListener('dragleave', () => {
            this.dropZone.classList.remove('dragover');
        });
        
        this.dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            this.dropZone.classList.remove('dragover');
            const file = e.dataTransfer.files[0];
            if (file && file.type.startsWith('image/')) {
                this.handleFileSelect(file);
            }
        });
        
        // Operation toggles
        this.operationButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                this.operationButtons.forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                this.currentOperation = btn.dataset.operation;
                this.updateControlsVisibility();
            });
        });
        
        // Quality dial
        this.qualityDial.addEventListener('input', (e) => {
            const display = e.target.nextElementSibling;
            display.textContent = e.target.value + '%';
        });
        
        // Format buttons
        document.querySelectorAll('.format-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.format-btn').forEach(b => b.classList.remove('selected'));
                btn.classList.add('selected');
            });
        });
        
        // Process button
        this.processBtn.addEventListener('click', () => this.processImage());
        
        // Metadata buttons
        document.querySelector('.strip-btn')?.addEventListener('click', () => this.stripMetadata());
        document.querySelector('.view-btn')?.addEventListener('click', () => this.viewMetadata());
        
        // Split handle dragging
        const splitHandle = document.getElementById('split-handle');
        if (splitHandle) {
            let isDragging = false;
            
            splitHandle.addEventListener('mousedown', () => {
                isDragging = true;
                document.body.style.cursor = 'col-resize';
            });
            
            document.addEventListener('mousemove', (e) => {
                if (!isDragging) return;
                
                const container = document.querySelector('.preview-split');
                const rect = container.getBoundingClientRect();
                const x = e.clientX - rect.left;
                const percentage = (x / rect.width) * 100;
                
                if (percentage > 10 && percentage < 90) {
                    splitHandle.style.left = percentage + '%';
                    document.querySelector('.preview-side.original').style.flex = `0 0 ${percentage}%`;
                    document.querySelector('.preview-side.processed').style.flex = `0 0 ${100 - percentage}%`;
                }
            });
            
            document.addEventListener('mouseup', () => {
                isDragging = false;
                document.body.style.cursor = 'default';
            });
        }
    }
    
    updateControlsVisibility() {
        const controls = [
            this.compressControls,
            this.resizeControls,
            this.convertControls,
            this.metadataControls
        ];
        
        controls.forEach(control => control?.classList.add('hidden'));
        
        switch(this.currentOperation) {
            case 'compress':
                this.compressControls?.classList.remove('hidden');
                break;
            case 'resize':
                this.resizeControls?.classList.remove('hidden');
                break;
            case 'convert':
                this.convertControls?.classList.remove('hidden');
                break;
            case 'metadata':
                this.metadataControls?.classList.remove('hidden');
                break;
        }
    }
    
    handleFileSelect(file) {
        if (!file || !file.type.startsWith('image/')) {
            this.showError('Please select a valid image file');
            return;
        }
        
        this.currentFile = file;
        
        // Show original image
        const reader = new FileReader();
        reader.onload = (e) => {
            this.originalImage.src = e.target.result;
            this.dropZone.classList.add('hidden');
            this.previewContainer.classList.remove('hidden');
            this.originalSize.textContent = this.formatFileSize(file.size);
            
            // Get image dimensions
            const img = new Image();
            img.onload = () => {
                this.dimensions.textContent = `${img.width} × ${img.height}`;
                this.widthInput.placeholder = img.width;
                this.heightInput.placeholder = img.height;
            };
            img.src = e.target.result;
        };
        reader.readAsDataURL(file);
        
        // Reset processed image
        this.processedImage.src = '';
        this.processedSize.textContent = '--';
        this.compressionRatio.textContent = '--';
        this.savingsPercent.textContent = '--';
    }
    
    async processImage() {
        if (!this.currentFile) {
            this.showError('Please select an image first');
            return;
        }
        
        this.showProcessing(true);
        
        try {
            let result;
            
            switch(this.currentOperation) {
                case 'compress':
                    result = await this.compressImage();
                    break;
                case 'resize':
                    result = await this.resizeImage();
                    break;
                case 'convert':
                    result = await this.convertImage();
                    break;
                case 'metadata':
                    result = await this.stripMetadata();
                    break;
            }
            
            if (result) {
                this.displayResult(result);
            }
        } catch (error) {
            this.showError(error.message);
        } finally {
            this.showProcessing(false);
        }
    }
    
    async compressImage() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('quality', this.qualityDial.value);
        
        const response = await fetch(`${this.apiUrl}/api/v1/image/compress`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Compression failed');
        }
        
        return await response.json();
    }
    
    async resizeImage() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('width', this.widthInput.value || this.widthInput.placeholder);
        formData.append('height', this.heightInput.value || this.heightInput.placeholder);
        formData.append('maintain_aspect', this.maintainAspect.checked);
        
        const response = await fetch(`${this.apiUrl}/api/v1/image/resize`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Resize failed');
        }
        
        return await response.json();
    }
    
    async convertImage() {
        const selectedFormat = document.querySelector('.format-btn.selected');
        if (!selectedFormat) {
            throw new Error('Please select a target format');
        }
        
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('target_format', selectedFormat.dataset.format);
        
        const response = await fetch(`${this.apiUrl}/api/v1/image/convert`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Conversion failed');
        }
        
        return await response.json();
    }
    
    async stripMetadata() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        
        const response = await fetch(`${this.apiUrl}/api/v1/image/metadata?action=strip`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Metadata stripping failed');
        }
        
        return await response.json();
    }
    
    async viewMetadata() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        
        const response = await fetch(`${this.apiUrl}/api/v1/image/metadata?action=read`, {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Metadata reading failed');
        }
        
        const metadata = await response.json();
        this.displayMetadata(metadata);
    }
    
    displayResult(result) {
        // Update processed image
        if (result.url) {
            // Convert file:// URLs to blob URLs for display
            if (result.url.startsWith('file://')) {
                // For demo purposes, we'll use a placeholder
                this.processedImage.src = this.originalImage.src;
            } else {
                this.processedImage.src = result.url;
            }
        }
        
        // Update stats
        if (result.compressed_size || result.size) {
            const size = result.compressed_size || result.size;
            this.processedSize.textContent = this.formatFileSize(size);
        }
        
        if (result.savings_percent) {
            this.savingsPercent.textContent = result.savings_percent.toFixed(1) + '%';
            this.compressionRatio.textContent = (100 - result.savings_percent).toFixed(1) + '%';
        }
        
        if (result.dimensions) {
            this.dimensions.textContent = `${result.dimensions.width} × ${result.dimensions.height}`;
        }
    }
    
    displayMetadata(metadata) {
        this.metadataViewer.classList.remove('hidden');
        this.metadataContent.innerHTML = '';
        
        const displayData = {
            'Format': metadata.format,
            'Dimensions': `${metadata.width} × ${metadata.height}`,
            'Size': this.formatFileSize(metadata.size_bytes),
            'Color Space': metadata.color_space
        };
        
        if (metadata.metadata) {
            Object.entries(metadata.metadata).forEach(([key, value]) => {
                if (value) {
                    displayData[key.charAt(0).toUpperCase() + key.slice(1)] = value;
                }
            });
        }
        
        Object.entries(displayData).forEach(([key, value]) => {
            const item = document.createElement('div');
            item.className = 'metadata-item';
            item.innerHTML = `
                <span class="metadata-key">${key}:</span>
                <span class="metadata-value">${value}</span>
            `;
            this.metadataContent.appendChild(item);
        });
    }
    
    formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
    }
    
    showProcessing(show) {
        const developingIndicator = this.processBtn.querySelector('.developing-indicator');
        const btnText = this.processBtn.querySelector('.btn-text');
        
        if (show) {
            developingIndicator?.classList.remove('hidden');
            btnText.textContent = 'DEVELOPING';
            this.processBtn.disabled = true;
            
            // Update status lights
            document.querySelector('.status-light.ready')?.classList.add('hidden');
            document.querySelector('.status-light.processing')?.classList.remove('hidden');
        } else {
            developingIndicator?.classList.add('hidden');
            btnText.textContent = 'DEVELOP';
            this.processBtn.disabled = false;
            
            // Update status lights
            document.querySelector('.status-light.processing')?.classList.add('hidden');
            document.querySelector('.status-light.ready')?.classList.remove('hidden');
        }
    }
    
    showError(message) {
        // Show error status light
        document.querySelector('.status-light.error')?.classList.remove('hidden');
        setTimeout(() => {
            document.querySelector('.status-light.error')?.classList.add('hidden');
        }, 3000);
        
        console.error(message);
        alert(message);
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new ImageToolsApp();
});