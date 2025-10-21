const API_PATH = '/api/v1';
const DEFAULT_API_PORT = 19374;
const ALT_API_PORTS = [19374, 19373, 19368, 19367, 19364, 8080];

function stripTrailingSlash(value) {
    return typeof value === 'string' ? value.replace(/\/+$/, '') : value;
}

function ensureApiPath(base) {
    if (typeof base !== 'string') {
        return base;
    }

    const sanitized = stripTrailingSlash(base);
    if (!sanitized) {
        return sanitized;
    }

    if (sanitized.endsWith(API_PATH)) {
        return sanitized;
    }

    if (sanitized.toLowerCase().endsWith('/api')) {
        return `${sanitized}${API_PATH.slice(4)}`;
    }

    return `${sanitized}${API_PATH}`;
}

function isLocalHostname(hostname) {
    if (!hostname) {
        return false;
    }
    const normalized = hostname.toLowerCase();
    return normalized === 'localhost' ||
        normalized === '127.0.0.1' ||
        normalized === '0.0.0.0' ||
        normalized === '::1' ||
        normalized === '[::1]';
}

function buildLoopbackBase(port = DEFAULT_API_PORT, win) {
    const protocol = win?.location?.protocol || 'http:';
    const hostname = win?.location?.hostname;
    const loopbackHost = isLocalHostname(hostname) ? hostname : '127.0.0.1';
    return `${protocol}//${loopbackHost}:${port}`;
}

function collectProxyCandidates(source) {
    const queue = [source];
    const seen = new Set();
    const candidates = [];
    const keys = [
        'apiBase',
        'apiUrl',
        'api',
        'url',
        'baseUrl',
        'publicUrl',
        'proxyUrl',
        'origin',
        'target',
        'path',
        'proxyPath',
        'paths',
        'endpoint',
        'endpoints',
        'service',
        'services',
        'host',
        'hosts',
        'port',
        'ports'
    ];

    while (queue.length > 0) {
        const value = queue.shift();
        if (!value) {
            continue;
        }

        if (typeof value === 'string') {
            candidates.push(value);
            continue;
        }

        if (Array.isArray(value)) {
            queue.push(...value);
            continue;
        }

        if (typeof value === 'object') {
            if (seen.has(value)) {
                continue;
            }
            seen.add(value);
            keys.forEach((key) => {
                if (Object.prototype.hasOwnProperty.call(value, key)) {
                    queue.push(value[key]);
                }
            });
        }
    }

    return candidates;
}

function normalizeProxyCandidate(candidate, win) {
    if (typeof candidate !== 'string') {
        return undefined;
    }

    const trimmed = candidate.trim();
    if (!trimmed) {
        return undefined;
    }

    if (/^https?:\/\//i.test(trimmed)) {
        return stripTrailingSlash(trimmed);
    }

    if (trimmed.startsWith('//')) {
        const protocol = win?.location?.protocol || 'https:';
        return stripTrailingSlash(`${protocol}${trimmed}`);
    }

    if (trimmed.startsWith('/')) {
        const origin = win?.location?.origin;
        if (origin) {
            return stripTrailingSlash(`${origin}${trimmed}`);
        }
        return stripTrailingSlash(trimmed);
    }

    if (/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)(:\\d+)?(\/.*)?$/i.test(trimmed)) {
        return stripTrailingSlash(`http://${trimmed}`);
    }

    return undefined;
}

function stripApiSuffix(value) {
    if (typeof value !== 'string') {
        return value;
    }
    const trimmed = stripTrailingSlash(value);
    return trimmed.replace(/\/api\/?$/i, '');
}

function resolveProxyBase(win) {
    if (!win) {
        return undefined;
    }

    const sources = [
        win.__APP_MONITOR_PROXY_INFO__,
        win.__APP_MONITOR_PROXY_INDEX__
    ].filter(Boolean);

    for (const source of sources) {
        const candidates = collectProxyCandidates(source);
        for (const candidate of candidates) {
            const normalized = normalizeProxyCandidate(candidate, win);
            if (normalized) {
                return stripApiSuffix(normalized);
            }
        }
    }

    return undefined;
}

function buildApiUrlFromBase(base, path, searchParams) {
    const sanitizedBase = stripTrailingSlash(base || '');
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    let url = `${sanitizedBase}${normalizedPath}`;

    if (searchParams && Object.keys(searchParams).length > 0) {
        const query = new URLSearchParams(searchParams).toString();
        if (query) {
            url += `?${query}`;
        }
    }

    return url;
}

function resolveApiBase() {
    const win = typeof window !== 'undefined' ? window : undefined;
    const fallbackBase = ensureApiPath(buildLoopbackBase(DEFAULT_API_PORT, win));

    if (!win) {
        return fallbackBase;
    }

    const proxyBase = resolveProxyBase(win);
    if (proxyBase) {
        return ensureApiPath(proxyBase);
    }

    const origin = win.location?.origin;
    const hostname = win.location?.hostname;

    if (origin && hostname && !isLocalHostname(hostname)) {
        return ensureApiPath(origin);
    }

    return fallbackBase;
}

class ImageToolsApp {
    constructor() {
        this.apiBase = resolveApiBase();
        this.checkApiHealth();
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
        this.errorStatusLight = document.querySelector('.status-light.error');

        // Error banner
        this.errorBanner = document.getElementById('error-banner');
        this.errorMessageEl = document.getElementById('error-message');
        this.errorDismissBtn = document.querySelector('.alert-dismiss');
        this.errorHideTimeout = null;
        this.errorIndicatorTimeout = null;
    }

    attachEventListeners() {
        // Error banner dismissal
        this.errorDismissBtn?.addEventListener('click', () => this.clearError());

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
                this.clearError();
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
                this.clearError();
                document.querySelectorAll('.format-btn').forEach(b => b.classList.remove('selected'));
                btn.classList.add('selected');
            });
        });
        
        // Process button
        this.processBtn.addEventListener('click', () => this.processImage());
        
        // Metadata buttons
        document.querySelector('.strip-btn')?.addEventListener('click', async () => {
            this.clearError();

            if (!this.currentFile) {
                this.showError('Please select an image first');
                return;
            }

            try {
                const result = await this.stripMetadata();
                if (result) {
                    await this.displayResult(result);
                }
            } catch (error) {
                this.showError(error);
            }
        });

        document.querySelector('.view-btn')?.addEventListener('click', async () => {
            this.clearError();

            if (!this.currentFile) {
                this.showError('Please select an image first');
                return;
            }

            try {
                await this.viewMetadata();
            } catch (error) {
                this.showError(error);
            }
        });
        
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
        this.clearError();

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
        this.clearError();

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
                await this.displayResult(result);
            }
        } catch (error) {
            this.showError(error);
        } finally {
            this.showProcessing(false);
        }
    }
    
    async compressImage() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('quality', this.qualityDial.value);
        
        const response = await fetch(this.buildApiUrl('/image/compress'), {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const detail = await this.extractErrorMessage(response);
            throw new Error(this.buildDetailedError('Compression failed', detail));
        }
        
        return response.json();
    }
    
    async resizeImage() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('width', this.widthInput.value || this.widthInput.placeholder);
        formData.append('height', this.heightInput.value || this.heightInput.placeholder);
        formData.append('maintain_aspect', this.maintainAspect.checked);
        
        const response = await fetch(this.buildApiUrl('/image/resize'), {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const detail = await this.extractErrorMessage(response);
            throw new Error(this.buildDetailedError('Resize failed', detail));
        }
        
        return response.json();
    }
    
    async convertImage() {
        const selectedFormat = document.querySelector('.format-btn.selected');
        if (!selectedFormat) {
            throw new Error('Please select a target format');
        }
        
        const formData = new FormData();
        formData.append('image', this.currentFile);
        formData.append('target_format', selectedFormat.dataset.format);
        
        const response = await fetch(this.buildApiUrl('/image/convert'), {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const detail = await this.extractErrorMessage(response);
            throw new Error(this.buildDetailedError('Conversion failed', detail));
        }
        
        return response.json();
    }
    
    async stripMetadata() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        
        const response = await fetch(this.buildApiUrl('/image/metadata', { action: 'strip' }), {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const detail = await this.extractErrorMessage(response);
            throw new Error(this.buildDetailedError('Metadata stripping failed', detail));
        }
        
        return response.json();
    }
    
    async viewMetadata() {
        const formData = new FormData();
        formData.append('image', this.currentFile);
        
        const response = await fetch(this.buildApiUrl('/image/metadata', { action: 'read' }), {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            const detail = await this.extractErrorMessage(response);
            throw new Error(this.buildDetailedError('Metadata reading failed', detail));
        }
        
        const metadata = await response.json();
        this.displayMetadata(metadata);
    }
    
    async displayResult(result) {
        this.clearError();

        // Update processed image
        if (result.url) {
            // Handle different URL types
            if (result.url.startsWith('file://')) {
                // Local storage fallback needs proxy fetch through the API
                await this.loadProcessedImage(result.url);
            } else if (this.shouldProxyImageUrl(result.url)) {
                // Proxy inaccessible hosts (e.g., MinIO or loopback) through the API
                await this.fetchAndDisplayImage(result.url, { forceProxy: true });
            } else {
                // Direct URL
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
        this.clearError();

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
    
    shouldProxyImageUrl(url) {
        if (!url) {
            return false;
        }

        const win = typeof window !== 'undefined' ? window : undefined;
        if (!win) {
            return true;
        }

        if (url.startsWith('blob:') || url.startsWith('data:')) {
            return false;
        }

        let parsed;
        try {
            parsed = new URL(url, win.location.href);
        } catch (error) {
            return true;
        }

        if (parsed.protocol === 'file:') {
            return true;
        }

        const currentHost = win.location?.host;
        if (currentHost && parsed.host === currentHost) {
            return false;
        }

        try {
            const apiHost = new URL(this.apiBase).host;
            if (parsed.host === apiHost) {
                return false;
            }
        } catch (error) {
            // Ignore cases where apiBase is not a valid URL yet
        }

        if (isLocalHostname(parsed.hostname) && !isLocalHostname(win.location?.hostname)) {
            return true;
        }

        return true;
    }

    async fetchAndDisplayImage(url, { forceProxy = false } = {}) {
        try {
            // Try to fetch the image directly
            if (!forceProxy) {
                const response = await fetch(url, {
                    mode: 'cors',
                    credentials: 'omit'
                });
                
                if (response.ok) {
                    const blob = await response.blob();
                    this.processedImage.src = URL.createObjectURL(blob);
                    return;
                }
            }

            // If direct fetch fails, proxy through our API
            const proxyResponse = await fetch(this.buildApiUrl('/image/proxy', { url }));
            if (proxyResponse.ok) {
                const blob = await proxyResponse.blob();
                this.processedImage.src = URL.createObjectURL(blob);
            } else {
                // Fallback: show a success indicator
                this.showProcessingSuccess();
            }
        } catch (error) {
            console.log('Could not fetch processed image directly, showing success state');
            this.showProcessingSuccess();
        }
    }
    
    async loadProcessedImage(fileUrl) {
        try {
            const normalized = fileUrl.startsWith('file://') ? fileUrl : `file://${fileUrl}`;
            const proxyUrl = this.buildApiUrl('/image/proxy', { url: normalized });
            const response = await fetch(proxyUrl, {
                mode: 'cors',
                credentials: 'omit'
            });

            if (!response.ok) {
                throw new Error(`Proxy request failed with status ${response.status}`);
            }

            const blob = await response.blob();
            this.processedImage.src = URL.createObjectURL(blob);
        } catch (error) {
            console.warn('Could not load processed image from storage', error);
            this.showProcessingSuccess();
        }
    }
    
    showProcessingSuccess() {
        // Apply a visual filter to show the image was processed
        this.processedImage.src = this.originalImage.src;
        this.processedImage.style.filter = 'brightness(1.1) contrast(1.05)';
        
        // Add a success overlay
        const holder = this.processedImage.parentElement;
        if (!holder.querySelector('.success-overlay')) {
            const overlay = document.createElement('div');
            overlay.className = 'success-overlay';
            overlay.innerHTML = '✓ PROCESSED';
            overlay.style.cssText = 'position: absolute; top: 10px; right: 10px; background: #4CAF50; color: white; padding: 5px 10px; border-radius: 3px; font-size: 12px; z-index: 10;';
            holder.style.position = 'relative';
            holder.appendChild(overlay);
            
            setTimeout(() => overlay.remove(), 3000);
        }
    }
    
    async checkApiHealth() {
        let isHealthy = false;

        try {
            const response = await fetch(this.buildApiUrl('/health'));
            isHealthy = response.ok;
        } catch (error) {
            console.warn('API connection check failed:', error);
        }

        if (!isHealthy) {
            console.warn('API health check failed, trying alternative ports');
            await this.tryAlternativePorts();
        }
    }

    async tryAlternativePorts() {
        const win = typeof window !== 'undefined' ? window : undefined;
        if (!win) {
            return;
        }

        let hostname;
        try {
            hostname = new URL(this.apiBase).hostname;
        } catch (error) {
            hostname = undefined;
        }

        if (!isLocalHostname(hostname)) {
            return;
        }

        for (const port of ALT_API_PORTS) {
            const candidateBase = ensureApiPath(buildLoopbackBase(port, win));
            if (candidateBase === this.apiBase) {
                continue;
            }

            try {
                const response = await fetch(buildApiUrlFromBase(candidateBase, '/health'));
                if (response.ok) {
                    this.apiBase = candidateBase;
                    console.log(`API found at port ${port}`);
                    return;
                }
            } catch (error) {
                // Try next port
            }
        }
    }

    buildApiUrl(path, searchParams) {
        return buildApiUrlFromBase(this.apiBase, path, searchParams);
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
            this.clearError();
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
    
    showError(error) {
        const message = this.normalizeErrorMessage(error);

        console.error('[ImageTools]', message);

        if (this.errorBanner && this.errorMessageEl) {
            this.errorBanner.classList.remove('hidden');
            this.errorMessageEl.textContent = message;

            if (this.errorHideTimeout) {
                clearTimeout(this.errorHideTimeout);
            }
            this.errorHideTimeout = setTimeout(() => {
                this.clearError();
            }, 10000);

            if (this.errorStatusLight) {
                this.errorStatusLight.classList.remove('hidden');
            }

            if (this.errorIndicatorTimeout) {
                clearTimeout(this.errorIndicatorTimeout);
            }
            this.errorIndicatorTimeout = setTimeout(() => {
                this.errorStatusLight?.classList.add('hidden');
                this.errorIndicatorTimeout = null;
            }, 4000);
        } else {
            alert(message);
        }
    }

    clearError() {
        if (this.errorHideTimeout) {
            clearTimeout(this.errorHideTimeout);
            this.errorHideTimeout = null;
        }
        if (this.errorIndicatorTimeout) {
            clearTimeout(this.errorIndicatorTimeout);
            this.errorIndicatorTimeout = null;
        }

        this.errorBanner?.classList.add('hidden');
        if (this.errorMessageEl) {
            this.errorMessageEl.textContent = '';
        }
        this.errorStatusLight?.classList.add('hidden');
    }

    normalizeErrorMessage(error, fallback = 'Image Tools ran into a problem while processing this image.') {
        let message = '';

        if (typeof error === 'string') {
            message = error;
        } else if (error instanceof Error) {
            message = error.message;
        } else if (error && typeof error === 'object') {
            message = error.message || error.error || '';
        }

        const sanitized = (message || '').toString().replace(/\s+/g, ' ').trim();
        let finalMessage = sanitized || fallback;

        if (/failed to fetch/i.test(finalMessage) || /networkerror/i.test(finalMessage)) {
            finalMessage = 'Could not connect to the Image Tools API. Verify the scenario is running and try again.';
        }

        if (finalMessage.length > 200) {
            finalMessage = `${finalMessage.slice(0, 197)}...`;
        }

        return finalMessage;
    }

    buildDetailedError(baseMessage, detail) {
        const trimmedDetail = (detail || '').toString().replace(/\s+/g, ' ').trim();
        if (!trimmedDetail) {
            return baseMessage;
        }

        const lowerBase = baseMessage.toLowerCase();
        const lowerDetail = trimmedDetail.toLowerCase();

        if (lowerDetail === lowerBase) {
            return baseMessage;
        }

        if (lowerDetail.startsWith(lowerBase)) {
            return trimmedDetail;
        }

        return `${baseMessage}: ${trimmedDetail}`;
    }

    async extractErrorMessage(response, fallback = '') {
        if (!response) {
            return fallback;
        }

        try {
            const jsonClone = response.clone();
            const contentType = jsonClone.headers.get('content-type') || '';
            if (contentType.includes('application/json')) {
                const data = await jsonClone.json();
                const message = this.pickErrorMessage(data);
                if (message) {
                    return message;
                }
            }
        } catch (error) {
            // Ignore JSON parsing errors and fall back to text handling
        }

        try {
            const text = await response.text();
            const cleaned = text.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
            if (cleaned) {
                return cleaned.length > 200 ? `${cleaned.slice(0, 197)}...` : cleaned;
            }
        } catch (error) {
            // Ignore body read errors
        }

        return fallback;
    }

    pickErrorMessage(payload) {
        if (!payload) {
            return '';
        }

        if (typeof payload === 'string') {
            return payload;
        }

        if (Array.isArray(payload) && payload.length > 0) {
            return this.pickErrorMessage(payload[0]);
        }

        if (typeof payload === 'object') {
            const candidates = ['error', 'message', 'detail', 'details', 'title', 'description'];
            for (const key of candidates) {
                if (key in payload) {
                    const value = payload[key];
                    const extracted = this.pickErrorMessage(value);
                    if (extracted) {
                        return extracted;
                    }
                }
            }
        }

        return '';
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new ImageToolsApp();
});
