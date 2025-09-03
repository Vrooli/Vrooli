// Brand Manager Application Logic
class BrandManager {
    constructor() {
        this.apiBaseUrl = '/api';
        this.brands = [];
        this.currentBrand = null;
        this.isGenerating = false;
        
        this.init();
    }

    async init() {
        console.log('🎨 Initializing Brand Manager...');
        
        // Initialize event listeners
        this.initEventListeners();
        
        // Check API connection
        await this.checkAPIConnection();
        
        // Load initial data
        await this.loadBrands();
        
        console.log('✅ Brand Manager initialized');
    }

    initEventListeners() {
        // Navigation buttons
        document.getElementById('newBrandBtn')?.addEventListener('click', () => this.openBrandGenerator());
        document.getElementById('generateBrandBtn')?.addEventListener('click', () => this.openBrandGenerator());
        document.getElementById('createFirstBrandBtn')?.addEventListener('click', () => this.openBrandGenerator());
        
        // Modal controls
        document.getElementById('closeGeneratorModal')?.addEventListener('click', () => this.closeBrandGenerator());
        document.getElementById('closeBrandDetailModal')?.addEventListener('click', () => this.closeBrandDetail());
        
        // Form navigation
        document.getElementById('step1Next')?.addEventListener('click', () => this.nextStep());
        document.getElementById('step2Previous')?.addEventListener('click', () => this.previousStep());
        
        // Brand generator form
        document.getElementById('brandGeneratorForm')?.addEventListener('submit', (e) => this.handleBrandGeneration(e));
        
        // Template selection
        document.querySelectorAll('.template-option').forEach(option => {
            option.addEventListener('click', () => this.selectTemplate(option));
        });
        
        // Tab navigation
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', () => this.switchTab(btn.dataset.tab));
        });
        
        // Modal overlay click to close
        document.querySelectorAll('.modal-overlay').forEach(overlay => {
            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) {
                    this.closeAllModals();
                }
            });
        });
        
        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.closeAllModals();
            }
            if (e.ctrlKey && e.key === 'n') {
                e.preventDefault();
                this.openBrandGenerator();
            }
        });
        
        // Form validation
        document.getElementById('brandName')?.addEventListener('input', () => this.validateForm());
        document.getElementById('industry')?.addEventListener('change', () => this.validateForm());
    }

    async checkAPIConnection() {
        const statusIndicator = document.getElementById('apiStatus');
        const statusDot = statusIndicator?.querySelector('.status-dot');
        const statusText = statusIndicator?.querySelector('.status-text');
        
        try {
            const response = await fetch('/health');
            const data = await response.json();
            
            if (response.ok) {
                statusDot?.classList.add('connected');
                statusDot?.classList.remove('error');
                statusText.textContent = 'Connected';
                console.log('✅ API connection successful');
            } else {
                throw new Error('API health check failed');
            }
        } catch (error) {
            console.error('❌ API connection failed:', error);
            statusDot?.classList.add('error');
            statusDot?.classList.remove('connected');
            statusText.textContent = 'Disconnected';
            
            this.showNotification('API connection failed. Some features may not work.', 'error');
        }
    }

    async loadBrands() {
        try {
            console.log('📦 Loading brands...');
            const response = await fetch(`${this.apiBaseUrl}/brands`);
            
            if (response.ok) {
                this.brands = await response.json();
                this.renderBrands();
                this.updateStats();
                console.log(`✅ Loaded ${this.brands.length} brands`);
            } else {
                throw new Error(`Failed to load brands: ${response.statusText}`);
            }
        } catch (error) {
            console.error('❌ Failed to load brands:', error);
            this.showEmptyState();
            this.showNotification('Failed to load brands. Please check your connection.', 'error');
        }
    }

    renderBrands() {
        const brandGrid = document.getElementById('brandGrid');
        const emptyState = document.getElementById('emptyState');
        
        if (!this.brands || this.brands.length === 0) {
            this.showEmptyState();
            return;
        }
        
        // Hide empty state and show grid
        emptyState.style.display = 'none';
        brandGrid.style.display = 'grid';
        
        // Clear existing skeleton cards
        brandGrid.innerHTML = '';
        
        // Render brand cards
        this.brands.forEach(brand => {
            const brandCard = this.createBrandCard(brand);
            brandGrid.appendChild(brandCard);
        });
    }

    createBrandCard(brand) {
        const card = document.createElement('div');
        card.className = 'brand-card';
        card.addEventListener('click', () => this.openBrandDetail(brand));
        
        // Extract colors for display
        const colors = brand.brand_colors || {};
        const colorSwatches = Object.values(colors).slice(0, 4).map(color => 
            `<div class="color-swatch" style="background-color: ${color}"></div>`
        ).join('');
        
        card.innerHTML = `
            <div class="brand-preview">
                ${brand.logo_url ? `<img src="${brand.logo_url}" alt="${brand.name} logo" class="brand-logo">` : 
                  `<div style="color: white; font-size: 2rem; font-weight: bold;">${brand.name.charAt(0)}</div>`}
            </div>
            <div class="brand-info">
                <h3 class="brand-name">${brand.name}</h3>
                <p class="brand-details">${brand.description || brand.slogan || 'No description available'}</p>
                <div class="brand-colors">
                    ${colorSwatches}
                </div>
                <div class="brand-actions">
                    <button class="brand-action-btn" onclick="event.stopPropagation(); window.brandManager.editBrand('${brand.id}')">
                        <i class="fas fa-edit"></i> Edit
                    </button>
                    <button class="brand-action-btn" onclick="event.stopPropagation(); window.brandManager.integrateBrand('${brand.id}')">
                        <i class="fas fa-plug"></i> Integrate
                    </button>
                </div>
            </div>
        `;
        
        return card;
    }

    showEmptyState() {
        const brandGrid = document.getElementById('brandGrid');
        const emptyState = document.getElementById('emptyState');
        
        brandGrid.style.display = 'none';
        emptyState.style.display = 'block';
    }

    updateStats() {
        // Update brand count
        const totalBrandsEl = document.getElementById('totalBrands');
        if (totalBrandsEl) {
            totalBrandsEl.textContent = this.brands.length.toString();
        }
        
        // Update activity (simplified for now)
        this.updateRecentActivity();
    }

    updateRecentActivity() {
        const activityList = document.getElementById('recentActivity');
        if (!activityList) return;
        
        activityList.innerHTML = '';
        
        // Show recent brands (last 3)
        const recentBrands = this.brands.slice(-3).reverse();
        
        if (recentBrands.length === 0) {
            activityList.innerHTML = `
                <div class="activity-item">
                    <div class="activity-icon">
                        <i class="fas fa-info-circle"></i>
                    </div>
                    <div class="activity-content">
                        <div class="activity-title">No recent activity</div>
                    </div>
                </div>
            `;
            return;
        }
        
        recentBrands.forEach(brand => {
            const activityItem = document.createElement('div');
            activityItem.className = 'activity-item';
            activityItem.innerHTML = `
                <div class="activity-icon">
                    <i class="fas fa-palette"></i>
                </div>
                <div class="activity-content">
                    <div class="activity-title">Created "${brand.name}"</div>
                    <div class="activity-time">${this.formatTimeAgo(brand.created_at)}</div>
                </div>
            `;
            activityList.appendChild(activityItem);
        });
    }

    formatTimeAgo(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffMs = now - date;
        const diffMins = Math.floor(diffMs / (1000 * 60));
        const diffHours = Math.floor(diffMins / 60);
        const diffDays = Math.floor(diffHours / 24);
        
        if (diffMins < 1) return 'Just now';
        if (diffMins < 60) return `${diffMins}m ago`;
        if (diffHours < 24) return `${diffHours}h ago`;
        if (diffDays < 7) return `${diffDays}d ago`;
        return date.toLocaleDateString();
    }

    // Modal Management
    openBrandGenerator() {
        document.getElementById('brandGeneratorModal').style.display = 'flex';
        this.resetGeneratorForm();
        this.showStep(1);
    }

    closeBrandGenerator() {
        document.getElementById('brandGeneratorModal').style.display = 'none';
        this.resetGeneratorForm();
    }

    openBrandDetail(brand) {
        this.currentBrand = brand;
        document.getElementById('brandDetailModal').style.display = 'flex';
        document.getElementById('brandDetailTitle').innerHTML = `
            <i class="fas fa-eye"></i>
            ${brand.name}
        `;
        this.populateBrandDetail(brand);
        this.switchTab('assets');
    }

    closeBrandDetail() {
        document.getElementById('brandDetailModal').style.display = 'none';
        this.currentBrand = null;
    }

    closeAllModals() {
        document.querySelectorAll('.modal-overlay').forEach(modal => {
            modal.style.display = 'none';
        });
        this.currentBrand = null;
    }

    populateBrandDetail(brand) {
        // Populate assets panel
        const assetsPanel = document.getElementById('brandAssets');
        assetsPanel.innerHTML = `
            <div class="asset-item">
                <h4>Logo</h4>
                ${brand.logo_url ? `<img src="${brand.logo_url}" alt="Logo" style="max-width: 200px;">` : '<p>No logo available</p>'}
            </div>
            <div class="asset-item">
                <h4>Favicon</h4>
                ${brand.favicon_url ? `<img src="${brand.favicon_url}" alt="Favicon" style="max-width: 50px;">` : '<p>No favicon available</p>'}
            </div>
        `;
        
        // Populate colors panel
        const colorsPanel = document.getElementById('brandColors');
        if (brand.brand_colors && Object.keys(brand.brand_colors).length > 0) {
            const colorHTML = Object.entries(brand.brand_colors).map(([name, color]) => `
                <div class="color-item">
                    <div class="color-preview" style="background-color: ${color}"></div>
                    <div class="color-info">
                        <div class="color-name">${name}</div>
                        <div class="color-value">${color}</div>
                    </div>
                </div>
            `).join('');
            colorsPanel.innerHTML = `<div class="color-palette-grid">${colorHTML}</div>`;
        } else {
            colorsPanel.innerHTML = '<p>No color palette available</p>';
        }
        
        // Populate copy panel
        const copyPanel = document.getElementById('brandCopyContent');
        copyPanel.innerHTML = `
            <div class="copy-section">
                <h4>Slogan</h4>
                <p>${brand.slogan || 'No slogan available'}</p>
            </div>
            <div class="copy-section">
                <h4>Description</h4>
                <p>${brand.description || 'No description available'}</p>
            </div>
            <div class="copy-section">
                <h4>Ad Copy</h4>
                <p>${brand.ad_copy || 'No ad copy available'}</p>
            </div>
        `;
    }

    // Form Management
    resetGeneratorForm() {
        document.getElementById('brandGeneratorForm').reset();
        document.querySelectorAll('.template-option').forEach(option => {
            option.classList.remove('selected');
        });
        document.querySelector('.template-option[data-template="modern-tech"]')?.classList.add('selected');
        document.getElementById('template').value = 'modern-tech';
        this.hideGenerationProgress();
    }

    validateForm() {
        const brandName = document.getElementById('brandName').value.trim();
        const industry = document.getElementById('industry').value;
        const nextBtn = document.getElementById('step1Next');
        
        if (brandName && industry) {
            nextBtn.disabled = false;
        } else {
            nextBtn.disabled = true;
        }
    }

    showStep(step) {
        document.querySelectorAll('.form-step').forEach((stepEl, index) => {
            stepEl.classList.toggle('active', index + 1 === step);
        });
    }

    nextStep() {
        this.showStep(2);
    }

    previousStep() {
        this.showStep(1);
    }

    selectTemplate(option) {
        document.querySelectorAll('.template-option').forEach(opt => {
            opt.classList.remove('selected');
        });
        option.classList.add('selected');
        document.getElementById('template').value = option.dataset.template;
    }

    switchTab(tabName) {
        // Update tab buttons
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        
        // Update tab panels
        document.querySelectorAll('.tab-panel').forEach(panel => {
            panel.classList.toggle('active', panel.id === `${tabName}Panel`);
        });
    }

    // Brand Generation
    async handleBrandGeneration(event) {
        event.preventDefault();
        
        if (this.isGenerating) return;
        
        const formData = new FormData(event.target);
        const brandData = {
            brand_name: formData.get('brandName'),
            short_name: formData.get('shortName'),
            industry: formData.get('industry'),
            template: formData.get('template'),
            logo_style: formData.get('logoStyle'),
            color_scheme: formData.get('colorScheme')
        };
        
        console.log('🎨 Starting brand generation:', brandData);
        
        try {
            this.isGenerating = true;
            this.showGenerationProgress();
            
            // Start brand generation
            const response = await fetch(`${this.apiBaseUrl}/brands`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(brandData)
            });
            
            if (!response.ok) {
                throw new Error(`Brand generation failed: ${response.statusText}`);
            }
            
            const result = await response.json();
            console.log('📡 Brand generation started:', result);
            
            // Start polling for completion
            this.pollBrandGeneration(brandData.brand_name);
            
        } catch (error) {
            console.error('❌ Brand generation failed:', error);
            this.hideGenerationProgress();
            this.showNotification('Brand generation failed. Please try again.', 'error');
            this.isGenerating = false;
        }
    }

    showGenerationProgress() {
        document.getElementById('brandGeneratorForm').style.display = 'none';
        document.getElementById('generationProgress').style.display = 'block';
        
        // Animate progress steps
        this.animateProgressSteps();
    }

    hideGenerationProgress() {
        document.getElementById('generationProgress').style.display = 'none';
        document.getElementById('brandGeneratorForm').style.display = 'block';
    }

    animateProgressSteps() {
        const steps = document.querySelectorAll('.progress-step');
        const progressFill = document.getElementById('progressFill');
        const progressStatus = document.getElementById('progressStatus');
        
        const stepMessages = [
            'Analyzing your requirements...',
            'Generating color palette...',
            'Creating logo design...',
            'Writing brand copy...'
        ];
        
        let currentStep = 0;
        
        const nextStep = () => {
            if (currentStep < steps.length) {
                // Mark current step as active
                steps[currentStep].classList.add('active');
                
                // Update progress bar
                const progress = ((currentStep + 1) / steps.length) * 100;
                progressFill.style.width = `${progress}%`;
                
                // Update status message
                progressStatus.textContent = stepMessages[currentStep];
                
                // Mark previous steps as completed
                for (let i = 0; i < currentStep; i++) {
                    steps[i].classList.add('completed');
                    steps[i].classList.remove('active');
                }
                
                currentStep++;
                
                // Continue to next step after delay
                if (currentStep < steps.length) {
                    setTimeout(nextStep, 2000 + Math.random() * 1000); // 2-3 seconds
                }
            }
        };
        
        // Start animation
        setTimeout(nextStep, 500);
    }

    async pollBrandGeneration(brandName) {
        const maxAttempts = 30; // 30 attempts = ~90 seconds
        let attempts = 0;
        
        const poll = async () => {
            attempts++;
            
            try {
                const response = await fetch(`${this.apiBaseUrl}/brands/status/${encodeURIComponent(brandName)}`);
                const result = await response.json();
                
                if (result.status === 'completed' && result.brand) {
                    console.log('✅ Brand generation completed!', result.brand);
                    this.isGenerating = false;
                    this.hideGenerationProgress();
                    this.closeBrandGenerator();
                    this.showNotification(`Brand "${result.brand.name}" created successfully!`, 'success');
                    
                    // Refresh brands list
                    await this.loadBrands();
                    
                } else if (result.status === 'in_progress') {
                    // Continue polling
                    if (attempts < maxAttempts) {
                        setTimeout(poll, 3000); // Poll every 3 seconds
                    } else {
                        throw new Error('Brand generation timeout');
                    }
                } else {
                    throw new Error('Brand generation failed');
                }
                
            } catch (error) {
                console.error('❌ Polling error:', error);
                this.isGenerating = false;
                this.hideGenerationProgress();
                this.showNotification('Brand generation may have failed. Please check your brands list.', 'warning');
            }
        };
        
        // Start polling after a short delay
        setTimeout(poll, 3000);
    }

    // Utility Methods
    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.innerHTML = `
            <div class="notification-content">
                <i class="fas fa-${type === 'success' ? 'check-circle' : type === 'error' ? 'exclamation-circle' : type === 'warning' ? 'exclamation-triangle' : 'info-circle'}"></i>
                <span>${message}</span>
            </div>
            <button class="notification-close">
                <i class="fas fa-times"></i>
            </button>
        `;
        
        // Add styles if not already present
        if (!document.getElementById('notification-styles')) {
            const styles = document.createElement('style');
            styles.id = 'notification-styles';
            styles.textContent = `
                .notification {
                    position: fixed;
                    top: 100px;
                    right: 20px;
                    background: var(--glass-bg);
                    backdrop-filter: var(--glass-backdrop);
                    border: 1px solid var(--glass-border);
                    border-radius: var(--radius-lg);
                    padding: var(--spacing-md);
                    box-shadow: var(--shadow-lg);
                    z-index: 3000;
                    display: flex;
                    align-items: center;
                    gap: var(--spacing-sm);
                    max-width: 400px;
                    animation: slideInRight 0.3s ease;
                    color: white;
                }
                
                .notification.success { border-left: 4px solid var(--success); }
                .notification.error { border-left: 4px solid var(--error); }
                .notification.warning { border-left: 4px solid var(--warning); }
                .notification.info { border-left: 4px solid var(--accent); }
                
                .notification-content {
                    display: flex;
                    align-items: center;
                    gap: var(--spacing-sm);
                    flex: 1;
                }
                
                .notification-close {
                    background: none;
                    border: none;
                    color: rgba(255, 255, 255, 0.6);
                    cursor: pointer;
                    padding: var(--spacing-xs);
                    border-radius: var(--radius-sm);
                }
                
                .notification-close:hover {
                    background: rgba(255, 255, 255, 0.1);
                    color: white;
                }
                
                @keyframes slideInRight {
                    from { transform: translateX(100%); opacity: 0; }
                    to { transform: translateX(0); opacity: 1; }
                }
                
                @keyframes slideOutRight {
                    from { transform: translateX(0); opacity: 1; }
                    to { transform: translateX(100%); opacity: 0; }
                }
            `;
            document.head.appendChild(styles);
        }
        
        // Add close functionality
        notification.querySelector('.notification-close').addEventListener('click', () => {
            notification.style.animation = 'slideOutRight 0.3s ease';
            setTimeout(() => notification.remove(), 300);
        });
        
        // Add to DOM
        document.body.appendChild(notification);
        
        // Auto remove after 5 seconds
        setTimeout(() => {
            if (notification.parentNode) {
                notification.style.animation = 'slideOutRight 0.3s ease';
                setTimeout(() => notification.remove(), 300);
            }
        }, 5000);
    }

    // Placeholder methods for future functionality
    editBrand(brandId) {
        console.log('Edit brand:', brandId);
        this.showNotification('Brand editing functionality coming soon!', 'info');
    }

    integrateBrand(brandId) {
        const brand = this.brands.find(b => b.id === brandId);
        if (brand) {
            this.openBrandDetail(brand);
            this.switchTab('integrate');
        }
    }
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    window.brandManager = new BrandManager();
});

// Add some additional CSS for notification and other dynamic styles
const additionalStyles = `
    /* Color palette grid */
    .color-palette-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: var(--spacing-md);
    }
    
    .color-item {
        text-align: center;
    }
    
    .color-preview {
        width: 100%;
        height: 60px;
        border-radius: var(--radius-md);
        margin-bottom: var(--spacing-sm);
        border: 2px solid rgba(255, 255, 255, 0.2);
    }
    
    .color-name {
        font-weight: 500;
        color: white;
        font-size: 0.875rem;
        margin-bottom: var(--spacing-xs);
        text-transform: capitalize;
    }
    
    .color-value {
        font-family: var(--font-mono);
        font-size: 0.75rem;
        color: rgba(255, 255, 255, 0.6);
    }
    
    /* Copy sections */
    .copy-section {
        margin-bottom: var(--spacing-xl);
        padding: var(--spacing-lg);
        background: rgba(255, 255, 255, 0.05);
        border-radius: var(--radius-lg);
        border: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .copy-section h4 {
        color: var(--accent);
        margin-bottom: var(--spacing-sm);
        font-size: 1rem;
        font-weight: 600;
    }
    
    .copy-section p {
        color: rgba(255, 255, 255, 0.8);
        line-height: 1.6;
    }
    
    /* Asset items */
    .asset-item {
        margin-bottom: var(--spacing-xl);
        text-align: center;
    }
    
    .asset-item h4 {
        color: white;
        margin-bottom: var(--spacing-md);
        font-size: 1.1rem;
    }
    
    .asset-item img {
        border-radius: var(--radius-md);
        box-shadow: var(--shadow-md);
        background: white;
        padding: var(--spacing-sm);
    }
    
    /* Integration center */
    .integration-center {
        text-align: center;
        padding: var(--spacing-xl);
    }
    
    .integration-center h3 {
        color: white;
        margin-bottom: var(--spacing-sm);
    }
    
    .integration-center p {
        color: rgba(255, 255, 255, 0.7);
        margin-bottom: var(--spacing-xl);
    }
`;

// Add the additional styles to the page
const styleSheet = document.createElement('style');
styleSheet.textContent = additionalStyles;
document.head.appendChild(styleSheet);