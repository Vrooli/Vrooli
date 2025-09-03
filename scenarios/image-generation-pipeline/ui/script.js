/**
 * Enterprise Image Generation Pipeline
 * Creative Gallery-Style UI JavaScript
 */

class ImageGenerationApp {
    constructor() {
        this.apiUrl = window.location.protocol + '//' + window.location.hostname + ':24000';
        this.currentTab = 'generation-studio';
        this.generationInProgress = false;
        this.voiceFile = null;
        this.generatedImages = [];
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.setupTabNavigation();
        this.setupVoiceUpload();
        this.setupSliders();
        this.loadInitialData();
        this.updateStats();
    }

    setupEventListeners() {
        // Generation controls
        document.getElementById('process-voice')?.addEventListener('click', () => this.processVoice());
        document.getElementById('optimize-prompt')?.addEventListener('click', () => this.optimizePrompt());
        document.getElementById('generate-btn')?.addEventListener('click', () => this.generateImages());
        document.getElementById('batch-generate-btn')?.addEventListener('click', () => this.batchGenerate());
        
        // Campaign management
        document.getElementById('new-campaign-btn')?.addEventListener('click', () => this.createNewCampaign());
        
        // Brand management
        document.getElementById('add-brand-btn')?.addEventListener('click', () => this.addNewBrand());
        
        // Gallery search
        document.getElementById('gallery-search')?.addEventListener('input', (e) => this.searchGallery(e.target.value));
        document.getElementById('gallery-filter')?.addEventListener('change', (e) => this.filterGallery(e.target.value));
        
        // Prompt input auto-resize
        const promptInput = document.getElementById('prompt-input');
        if (promptInput) {
            promptInput.addEventListener('input', () => this.autoResizeTextarea(promptInput));
        }
    }

    setupTabNavigation() {
        const tabs = document.querySelectorAll('.nav-tab');
        const contents = document.querySelectorAll('.tab-content');
        
        tabs.forEach(tab => {
            tab.addEventListener('click', () => {
                const targetTab = tab.dataset.tab;
                
                // Update active tab
                tabs.forEach(t => t.classList.remove('active'));
                tab.classList.add('active');
                
                // Update active content
                contents.forEach(content => {
                    content.classList.remove('active');
                    if (content.id === targetTab) {
                        content.classList.add('active');
                    }
                });
                
                this.currentTab = targetTab;
                this.onTabChange(targetTab);
            });
        });
    }

    setupVoiceUpload() {
        const uploadArea = document.getElementById('voice-upload');
        const fileInput = document.getElementById('voice-file');
        const processBtn = document.getElementById('process-voice');
        
        if (!uploadArea || !fileInput || !processBtn) return;
        
        uploadArea.addEventListener('click', () => fileInput.click());
        
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('dragover');
        });
        
        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('dragover');
        });
        
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('dragover');
            
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleVoiceFile(files[0]);
            }
        });
        
        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                this.handleVoiceFile(e.target.files[0]);
            }
        });
    }

    setupSliders() {
        const variationsSlider = document.getElementById('variations-slider');
        const variationsCount = document.getElementById('variations-count');
        
        if (variationsSlider && variationsCount) {
            variationsSlider.addEventListener('input', (e) => {
                variationsCount.textContent = e.target.value;
            });
        }
    }

    handleVoiceFile(file) {
        if (!file.type.startsWith('audio/')) {
            this.showNotification('Please select a valid audio file', 'error');
            return;
        }
        
        if (file.size > 50 * 1024 * 1024) { // 50MB limit
            this.showNotification('File size must be less than 50MB', 'error');
            return;
        }
        
        this.voiceFile = file;
        
        // Update UI
        const uploadContent = document.querySelector('.upload-content');
        if (uploadContent) {
            uploadContent.innerHTML = `
                <i class="fas fa-file-audio upload-icon" style="color: var(--primary-orange);"></i>
                <p class="upload-text" style="color: var(--primary-orange);">Audio file ready: ${file.name}</p>
                <p class="upload-hint">Click "Process Voice Brief" to analyze</p>
            `;
        }
        
        document.getElementById('process-voice').disabled = false;
    }

    async processVoice() {
        if (!this.voiceFile) return;
        
        const processBtn = document.getElementById('process-voice');
        const processingDiv = document.getElementById('voice-processing');
        
        try {
            processBtn.disabled = true;
            processBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Processing...';
            processingDiv.style.display = 'flex';
            
            // Simulate voice processing (replace with actual API call)
            await this.simulateVoiceProcessing();
            
            // Update UI with results
            this.displayVoiceResults({
                transcription: "I need professional marketing images for our Q4 product launch. The style should be modern and tech-focused with blue and white colors. We need hero images for the website and social media assets for LinkedIn and Instagram campaigns.",
                structured: {
                    objective: "Q4 Product Launch Marketing Campaign",
                    targetAudience: "Tech professionals and enterprise clients",
                    keyMessages: ["Innovation", "Reliability", "Professional Excellence"],
                    visualStyle: "Modern, tech-focused, blue and white color scheme"
                }
            });
            
        } catch (error) {
            console.error('Voice processing error:', error);
            this.showNotification('Voice processing failed', 'error');
        } finally {
            processBtn.disabled = false;
            processBtn.innerHTML = '<i class="fas fa-brain"></i> Process Voice Brief';
            processingDiv.style.display = 'none';
        }
    }

    async simulateVoiceProcessing() {
        return new Promise(resolve => setTimeout(resolve, 3000));
    }

    displayVoiceResults(results) {
        // Auto-fill prompt based on voice analysis
        const promptInput = document.getElementById('prompt-input');
        if (promptInput) {
            promptInput.value = `Professional ${results.structured.visualStyle.toLowerCase()} marketing image for ${results.structured.objective}. Style: ${results.structured.visualStyle}. Target audience: ${results.structured.targetAudience}.`;
            this.autoResizeTextarea(promptInput);
        }
        
        this.showNotification('Voice brief processed successfully! Prompt updated.', 'success');
    }

    async optimizePrompt() {
        const promptInput = document.getElementById('prompt-input');
        const optimizeBtn = document.getElementById('optimize-prompt');
        
        if (!promptInput || !promptInput.value.trim()) {
            this.showNotification('Please enter a prompt first', 'error');
            return;
        }
        
        try {
            optimizeBtn.disabled = true;
            optimizeBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Optimizing...';
            
            // Simulate AI optimization (replace with actual API call)
            await this.simulateOptimization();
            
            const originalPrompt = promptInput.value;
            const optimizedPrompt = this.generateOptimizedPrompt(originalPrompt);
            
            promptInput.value = optimizedPrompt;
            this.autoResizeTextarea(promptInput);
            
            this.showNotification('Prompt optimized successfully!', 'success');
            
        } catch (error) {
            console.error('Prompt optimization error:', error);
            this.showNotification('Prompt optimization failed', 'error');
        } finally {
            optimizeBtn.disabled = false;
            optimizeBtn.innerHTML = '<i class="fas fa-lightbulb"></i> AI Optimize Prompt';
        }
    }

    generateOptimizedPrompt(original) {
        // Simple prompt enhancement (replace with actual AI optimization)
        const enhancements = [
            "high quality, professional photography",
            "studio lighting, sharp focus",
            "8K resolution, detailed",
            "commercial grade, marketing ready"
        ];
        
        return `${original}, ${enhancements.join(', ')}, trending on artstation, award winning composition`;
    }

    async simulateOptimization() {
        return new Promise(resolve => setTimeout(resolve, 2000));
    }

    async generateImages() {
        const promptInput = document.getElementById('prompt-input');
        const generateBtn = document.getElementById('generate-btn');
        const progressDiv = document.getElementById('generation-progress');
        
        if (!promptInput || !promptInput.value.trim()) {
            this.showNotification('Please enter a prompt', 'error');
            return;
        }
        
        if (this.generationInProgress) return;
        
        try {
            this.generationInProgress = true;
            generateBtn.disabled = true;
            generateBtn.innerHTML = '<i class="fas fa-spinner fa-spin"></i> Generating...';
            progressDiv.style.display = 'block';
            
            // Clear previous results
            this.clearGallery();
            
            // Simulate generation steps
            await this.simulateGenerationSteps();
            
            // Generate mock images
            await this.generateMockImages();
            
        } catch (error) {
            console.error('Image generation error:', error);
            this.showNotification('Image generation failed', 'error');
        } finally {
            this.generationInProgress = false;
            generateBtn.disabled = false;
            generateBtn.innerHTML = '<i class="fas fa-sparkles"></i> Generate Images';
            progressDiv.style.display = 'none';
        }
    }

    async simulateGenerationSteps() {
        const steps = document.querySelectorAll('.step');
        
        for (let i = 0; i < steps.length; i++) {
            const step = steps[i];
            step.classList.add('active');
            
            await new Promise(resolve => setTimeout(resolve, 1500));
            
            step.classList.remove('active');
            step.classList.add('completed');
        }
    }

    async generateMockImages() {
        const variations = parseInt(document.getElementById('variations-slider').value);
        const size = document.getElementById('image-size').value;
        const quality = document.getElementById('quality-level').value;
        
        const images = [];
        
        for (let i = 0; i < variations; i++) {
            const mockImage = {
                id: Date.now() + i,
                url: `https://picsum.photos/600/600?random=${Date.now() + i}`,
                prompt: document.getElementById('prompt-input').value,
                size: size,
                quality: quality,
                qualityScore: Math.floor(Math.random() * 20) + 80, // 80-100
                complianceScore: Math.floor(Math.random() * 15) + 85, // 85-100
                timestamp: new Date()
            };
            
            images.push(mockImage);
            
            // Add image to gallery with animation delay
            setTimeout(() => {
                this.addImageToGallery(mockImage);
            }, i * 500);
        }
        
        this.generatedImages.push(...images);
        this.updateStats();
    }

    addImageToGallery(image) {
        const gallery = document.getElementById('image-gallery');
        const emptyGallery = gallery.querySelector('.empty-gallery');
        
        if (emptyGallery) {
            emptyGallery.remove();
        }
        
        // Create or find gallery grid
        let galleryGrid = gallery.querySelector('.gallery-grid');
        if (!galleryGrid) {
            galleryGrid = document.createElement('div');
            galleryGrid.className = 'gallery-grid';
            gallery.appendChild(galleryGrid);
        }
        
        const imageCard = document.createElement('div');
        imageCard.className = 'image-card';
        imageCard.style.opacity = '0';
        imageCard.style.transform = 'translateY(20px)';
        
        imageCard.innerHTML = `
            <div class="image-container">
                <img src="${image.url}" alt="Generated image" loading="lazy">
                <div class="image-overlay">
                    <div class="image-actions">
                        <button class="btn btn-primary" onclick="app.downloadImage('${image.id}')" title="Download">
                            <i class="fas fa-download"></i>
                        </button>
                        <button class="btn btn-secondary" onclick="app.approveImage('${image.id}')" title="Approve">
                            <i class="fas fa-check"></i>
                        </button>
                        <button class="btn btn-outline" onclick="app.generateVariations('${image.id}')" title="Variations">
                            <i class="fas fa-sync"></i>
                        </button>
                        <button class="btn btn-outline" onclick="app.viewImage('${image.id}')" title="View Full">
                            <i class="fas fa-expand"></i>
                        </button>
                    </div>
                </div>
            </div>
            <div class="image-info">
                <div class="image-metrics">
                    <span>Quality: ${image.qualityScore}/100</span>
                    <span>Brand: ${image.complianceScore}/100</span>
                </div>
                <div class="image-prompt">${image.prompt}</div>
            </div>
        `;
        
        galleryGrid.appendChild(imageCard);
        
        // Animate in
        setTimeout(() => {
            imageCard.style.transition = 'all 0.5s ease';
            imageCard.style.opacity = '1';
            imageCard.style.transform = 'translateY(0)';
        }, 100);
    }

    clearGallery() {
        const gallery = document.getElementById('image-gallery');
        const galleryGrid = gallery.querySelector('.gallery-grid');
        if (galleryGrid) {
            galleryGrid.remove();
        }
        
        // Show empty state
        if (!gallery.querySelector('.empty-gallery')) {
            gallery.innerHTML = `
                <div class="empty-gallery">
                    <i class="fas fa-images empty-icon"></i>
                    <h3>Ready to Generate</h3>
                    <p>Enter a prompt and click generate to see your AI-created images appear here</p>
                </div>
            `;
        }
    }

    async batchGenerate() {
        this.showNotification('Batch generation coming soon!', 'info');
    }

    createNewCampaign() {
        this.showNotification('Campaign creation coming soon!', 'info');
    }

    addNewBrand() {
        this.showNotification('Brand management coming soon!', 'info');
    }

    downloadImage(imageId) {
        const image = this.generatedImages.find(img => img.id == imageId);
        if (image) {
            // Create download link
            const link = document.createElement('a');
            link.href = image.url;
            link.download = `generated-image-${imageId}.jpg`;
            link.click();
            
            this.showNotification('Image download started', 'success');
        }
    }

    approveImage(imageId) {
        this.showNotification(`Image ${imageId} approved!`, 'success');
    }

    generateVariations(imageId) {
        this.showNotification('Generating variations...', 'info');
        // Simulate variation generation
        setTimeout(() => {
            this.showNotification('Variations generated!', 'success');
        }, 2000);
    }

    viewImage(imageId) {
        const image = this.generatedImages.find(img => img.id == imageId);
        if (image) {
            // Create modal to view image
            this.showImageModal(image);
        }
    }

    showImageModal(image) {
        const overlay = document.getElementById('modal-overlay');
        overlay.innerHTML = `
            <div class="image-modal" style="max-width: 90vw; max-height: 90vh; background: white; border-radius: 1rem; overflow: hidden; position: relative;">
                <button onclick="app.closeModal()" style="position: absolute; top: 1rem; right: 1rem; z-index: 10; background: rgba(0,0,0,0.5); color: white; border: none; border-radius: 50%; width: 40px; height: 40px; cursor: pointer;">
                    <i class="fas fa-times"></i>
                </button>
                <img src="${image.url}" alt="Generated image" style="width: 100%; height: auto; display: block;">
                <div style="padding: 1rem;">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 1rem;">
                        <span>Quality: ${image.qualityScore}/100</span>
                        <span>Brand Compliance: ${image.complianceScore}/100</span>
                    </div>
                    <p style="font-size: 0.875rem; color: var(--gray-600);">${image.prompt}</p>
                </div>
            </div>
        `;
        overlay.classList.add('active');
    }

    closeModal() {
        const overlay = document.getElementById('modal-overlay');
        overlay.classList.remove('active');
        setTimeout(() => overlay.innerHTML = '', 300);
    }

    searchGallery(query) {
        // Gallery search functionality
        console.log('Searching gallery:', query);
    }

    filterGallery(filter) {
        // Gallery filter functionality
        console.log('Filtering gallery:', filter);
    }

    onTabChange(tab) {
        switch(tab) {
            case 'gallery':
                this.loadGalleryImages();
                break;
            case 'campaign-manager':
                this.loadCampaigns();
                break;
            case 'brand-center':
                this.loadBrands();
                break;
            case 'analytics':
                this.updateAnalytics();
                break;
        }
    }

    loadGalleryImages() {
        // Load gallery images (placeholder)
        const gallery = document.getElementById('masonry-gallery');
        if (gallery && this.generatedImages.length > 0) {
            gallery.innerHTML = this.generatedImages.map(image => `
                <div class="masonry-item">
                    <img src="${image.url}" alt="Generated image" style="width: 100%; display: block;">
                    <div style="padding: 1rem;">
                        <div style="display: flex; justify-content: space-between; font-size: 0.75rem; color: var(--gray-500); margin-bottom: 0.5rem;">
                            <span>Q: ${image.qualityScore}</span>
                            <span>B: ${image.complianceScore}</span>
                        </div>
                        <p style="font-size: 0.75rem; color: var(--gray-600);">${image.prompt.substring(0, 80)}...</p>
                    </div>
                </div>
            `).join('');
        }
    }

    loadCampaigns() {
        // Load campaigns (placeholder)
        console.log('Loading campaigns...');
    }

    loadBrands() {
        // Load brands (placeholder)
        console.log('Loading brands...');
    }

    updateAnalytics() {
        // Update analytics (placeholder)
        console.log('Updating analytics...');
    }

    loadInitialData() {
        // Load any initial data needed
        console.log('Loading initial data...');
    }

    updateStats() {
        // Update hero stats
        document.getElementById('total-images').textContent = this.generatedImages.length;
        document.getElementById('analytics-images').textContent = this.generatedImages.length;
        
        if (this.generatedImages.length > 0) {
            const avgQuality = this.generatedImages.reduce((sum, img) => sum + img.qualityScore, 0) / this.generatedImages.length;
            document.getElementById('quality-score').textContent = avgQuality.toFixed(1);
            document.getElementById('analytics-quality').textContent = avgQuality.toFixed(1);
        }
    }

    autoResizeTextarea(textarea) {
        textarea.style.height = 'auto';
        textarea.style.height = (textarea.scrollHeight) + 'px';
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            z-index: 1001;
            padding: 1rem 1.5rem;
            background: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : '#3b82f6'};
            color: white;
            border-radius: 0.5rem;
            box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
            animation: slideIn 0.3s ease;
            max-width: 350px;
            font-weight: 500;
        `;
        
        notification.textContent = message;
        document.body.appendChild(notification);
        
        // Remove after 5 seconds
        setTimeout(() => {
            notification.style.animation = 'slideOut 0.3s ease forwards';
            setTimeout(() => notification.remove(), 300);
        }, 5000);
        
        // Add CSS animations if not already present
        if (!document.getElementById('notification-styles')) {
            const styles = document.createElement('style');
            styles.id = 'notification-styles';
            styles.textContent = `
                @keyframes slideIn {
                    from { transform: translateX(100%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
                @keyframes slideOut {
                    from { transform: translateX(0); opacity: 1; }
                    to { transform: translateX(100%); opacity: 0; }
                }
            `;
            document.head.appendChild(styles);
        }
    }
}

// Initialize the application
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new ImageGenerationApp();
});

// Handle clicks outside modal to close
document.addEventListener('click', (e) => {
    if (e.target.id === 'modal-overlay') {
        app?.closeModal();
    }
});

// Handle keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && document.getElementById('modal-overlay').classList.contains('active')) {
        app?.closeModal();
    }
});