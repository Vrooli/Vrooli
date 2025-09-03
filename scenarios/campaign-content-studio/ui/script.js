// Campaign Content Studio JavaScript
class CampaignStudio {
    constructor() {
        this.campaigns = [];
        this.currentCampaign = null;
        this.documents = [];
        this.generatedContent = [];
        this.templates = [];
        this.currentDate = new Date();
        
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadInitialData();
        this.renderCalendar();
        this.loadTemplates();
        this.updateStats();
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const view = item.dataset.view;
                if (view) {
                    this.switchView(view);
                }
            });
        });

        // Content type buttons
        document.querySelectorAll('.content-type-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.content-type-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
            });
        });

        // Template categories
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('template-category')) {
                document.querySelectorAll('.template-category').forEach(c => c.classList.remove('active'));
                e.target.classList.add('active');
                this.filterTemplates(e.target.dataset.category);
            }
        });

        // File upload
        const fileInput = document.getElementById('fileInput');
        if (fileInput) {
            fileInput.addEventListener('change', this.handleFileUpload.bind(this));
        }

        // Search functionality
        const searchInput = document.querySelector('.search-bar input');
        if (searchInput) {
            searchInput.addEventListener('input', this.handleSearch.bind(this));
        }

        // Document search
        const docSearchInput = document.getElementById('documentSearch');
        if (docSearchInput) {
            docSearchInput.addEventListener('input', this.searchDocuments.bind(this));
        }
    }

    switchView(viewName) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-view="${viewName}"]`)?.classList.add('active');

        // Update views
        document.querySelectorAll('.view').forEach(view => {
            view.classList.remove('active');
        });
        document.querySelector(`.${viewName}-view`)?.classList.add('active');

        // Load view-specific data
        switch(viewName) {
            case 'calendar':
                this.renderCalendar();
                break;
            case 'documents':
                this.loadDocuments();
                break;
            case 'analytics':
                this.loadAnalytics();
                break;
        }
    }

    loadInitialData() {
        // Load sample campaigns
        this.campaigns = [
            {
                id: 1,
                name: 'Summer Product Launch',
                description: 'Marketing campaign for new product line',
                created: new Date('2024-10-01'),
                status: 'active'
            },
            {
                id: 2,
                name: 'Holiday Season 2024',
                description: 'Holiday marketing and promotions',
                created: new Date('2024-10-15'),
                status: 'active'
            }
        ];

        // Load sample documents
        this.documents = [
            {
                id: 1,
                campaignId: 1,
                name: 'Brand Guidelines.pdf',
                type: 'pdf',
                uploadDate: new Date('2024-10-01'),
                size: '2.5 MB'
            },
            {
                id: 2,
                campaignId: 1,
                name: 'Product Specs.docx',
                type: 'docx',
                uploadDate: new Date('2024-10-02'),
                size: '1.2 MB'
            }
        ];

        // Sample generated content
        this.generatedContent = [
            {
                id: 1,
                campaignId: 1,
                type: 'blog',
                title: 'Introducing Our Revolutionary New Product',
                created: new Date('2024-10-01'),
                status: 'published'
            }
        ];

        this.renderCampaigns();
        this.renderRecentActivity();
    }

    renderCampaigns() {
        const campaignList = document.getElementById('campaignList');
        if (!campaignList) return;

        campaignList.innerHTML = '';

        this.campaigns.forEach(campaign => {
            const campaignElement = document.createElement('div');
            campaignElement.className = 'campaign-item';
            campaignElement.innerHTML = `
                <div class="campaign-info">
                    <h5>${campaign.name}</h5>
                    <p>${campaign.description}</p>
                </div>
                <div class="campaign-status"></div>
            `;
            campaignElement.addEventListener('click', () => this.selectCampaign(campaign));
            campaignList.appendChild(campaignElement);
        });

        // Update campaign selects
        this.updateCampaignSelects();
    }

    updateCampaignSelects() {
        const selects = ['campaignSelect', 'documentCampaignFilter'];
        selects.forEach(selectId => {
            const select = document.getElementById(selectId);
            if (!select) return;

            // Clear existing options (except first one)
            while (select.children.length > 1) {
                select.removeChild(select.lastChild);
            }

            // Add campaign options
            this.campaigns.forEach(campaign => {
                const option = document.createElement('option');
                option.value = campaign.id;
                option.textContent = campaign.name;
                select.appendChild(option);
            });
        });
    }

    selectCampaign(campaign) {
        // Update UI to show selected campaign
        document.querySelectorAll('.campaign-item').forEach(item => {
            item.classList.remove('active');
        });
        event.target.closest('.campaign-item').classList.add('active');

        this.currentCampaign = campaign;
        this.loadCampaignDocuments(campaign.id);
    }

    loadCampaignDocuments(campaignId) {
        const campaignDocs = this.documents.filter(doc => doc.campaignId === campaignId);
        this.renderContextDocuments(campaignDocs);
    }

    renderContextDocuments(documents) {
        const contextDocuments = document.getElementById('contextDocuments');
        if (!contextDocuments) return;

        contextDocuments.innerHTML = '';

        documents.forEach(doc => {
            const docElement = document.createElement('div');
            docElement.className = 'context-document';
            docElement.innerHTML = `
                <h6>${doc.name}</h6>
                <p>Relevant content from this document will appear here when selected.</p>
            `;
            contextDocuments.appendChild(docElement);
        });
    }

    renderRecentActivity() {
        const activityList = document.getElementById('recentActivity');
        if (!activityList) return;

        const activities = [
            { icon: 'fas fa-file-alt', title: 'Blog post generated', desc: 'Summer Product Launch campaign', time: '2 hours ago' },
            { icon: 'fas fa-upload', title: 'Document uploaded', desc: 'Brand Guidelines.pdf', time: '4 hours ago' },
            { icon: 'fas fa-bullhorn', title: 'Campaign created', desc: 'Holiday Season 2024', time: '1 day ago' },
            { icon: 'fas fa-image', title: 'Images generated', desc: '3 social media graphics', time: '2 days ago' }
        ];

        activityList.innerHTML = '';

        activities.forEach(activity => {
            const activityElement = document.createElement('div');
            activityElement.className = 'activity-item';
            activityElement.innerHTML = `
                <div class="activity-icon">
                    <i class="${activity.icon}"></i>
                </div>
                <div class="activity-content">
                    <h5>${activity.title}</h5>
                    <p>${activity.desc}</p>
                </div>
                <div class="activity-time">${activity.time}</div>
            `;
            activityList.appendChild(activityElement);
        });
    }

    updateStats() {
        const stats = {
            activeCampaigns: this.campaigns.filter(c => c.status === 'active').length,
            contentGenerated: this.generatedContent.length,
            documentsUploaded: this.documents.length,
            timeSaved: Math.floor(this.generatedContent.length * 2.5) // Estimated hours saved
        };

        // Update stat displays
        Object.keys(stats).forEach(key => {
            const element = document.getElementById(key);
            if (element) {
                element.textContent = key === 'timeSaved' ? `${stats[key]}h` : stats[key];
            }
        });
    }

    renderCalendar() {
        const calendarGrid = document.getElementById('calendarGrid');
        const currentMonth = document.getElementById('currentMonth');
        
        if (!calendarGrid || !currentMonth) return;

        // Update month display
        const monthNames = ['January', 'February', 'March', 'April', 'May', 'June',
                           'July', 'August', 'September', 'October', 'November', 'December'];
        currentMonth.textContent = `${monthNames[this.currentDate.getMonth()]} ${this.currentDate.getFullYear()}`;

        // Clear previous calendar
        calendarGrid.innerHTML = '';

        // Add day headers
        const dayHeaders = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
        dayHeaders.forEach(day => {
            const headerElement = document.createElement('div');
            headerElement.className = 'calendar-day-header';
            headerElement.textContent = day;
            headerElement.style.cssText = 'background: #f5f5f5; font-weight: 600; padding: 1rem; text-align: center;';
            calendarGrid.appendChild(headerElement);
        });

        // Get first day of month and number of days
        const firstDay = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1);
        const lastDay = new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0);
        const startDate = new Date(firstDay);
        startDate.setDate(startDate.getDate() - firstDay.getDay());

        // Generate calendar days
        for (let i = 0; i < 42; i++) {
            const cellDate = new Date(startDate);
            cellDate.setDate(startDate.getDate() + i);

            const dayElement = document.createElement('div');
            dayElement.className = 'calendar-day';
            dayElement.textContent = cellDate.getDate();

            if (cellDate.getMonth() !== this.currentDate.getMonth()) {
                dayElement.style.opacity = '0.3';
            }

            // Add content indicators for dates with scheduled content
            if (Math.random() < 0.3) { // Simulate content on some days
                dayElement.classList.add('has-content');
            }

            calendarGrid.appendChild(dayElement);
        }
    }

    previousMonth() {
        this.currentDate.setMonth(this.currentDate.getMonth() - 1);
        this.renderCalendar();
    }

    nextMonth() {
        this.currentDate.setMonth(this.currentDate.getMonth() + 1);
        this.renderCalendar();
    }

    loadDocuments() {
        const documentsGrid = document.getElementById('documentsGrid');
        if (!documentsGrid) return;

        documentsGrid.innerHTML = '';

        this.documents.forEach(doc => {
            const docElement = document.createElement('div');
            docElement.className = 'document-card';
            
            const iconClass = this.getDocumentIcon(doc.type);
            const campaign = this.campaigns.find(c => c.id === doc.campaignId);
            
            docElement.innerHTML = `
                <div class="document-header">
                    <div class="document-icon">
                        <i class="${iconClass}"></i>
                    </div>
                    <div class="document-info">
                        <h5>${doc.name}</h5>
                        <p>${doc.size} • ${campaign?.name || 'No Campaign'}</p>
                    </div>
                </div>
                <div class="document-meta">
                    <small>Uploaded ${this.formatDate(doc.uploadDate)}</small>
                </div>
            `;
            
            documentsGrid.appendChild(docElement);
        });
    }

    getDocumentIcon(type) {
        const icons = {
            pdf: 'fas fa-file-pdf',
            docx: 'fas fa-file-word',
            txt: 'fas fa-file-alt',
            xlsx: 'fas fa-file-excel',
            default: 'fas fa-file'
        };
        return icons[type] || icons.default;
    }

    formatDate(date) {
        return date.toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric'
        });
    }

    loadTemplates() {
        this.templates = [
            {
                id: 1,
                name: 'Product Launch Blog Post',
                description: 'Comprehensive template for announcing new products',
                category: 'blog',
                tags: ['product', 'launch', 'announcement']
            },
            {
                id: 2,
                name: 'Social Media Campaign',
                description: 'Multi-platform social media content strategy',
                category: 'social',
                tags: ['social', 'engagement', 'viral']
            },
            {
                id: 3,
                name: 'Email Newsletter',
                description: 'Weekly newsletter template with engaging content',
                category: 'email',
                tags: ['newsletter', 'engagement', 'retention']
            },
            {
                id: 4,
                name: 'Facebook Ad Campaign',
                description: 'High-converting Facebook advertisement template',
                category: 'ads',
                tags: ['facebook', 'conversion', 'roi']
            }
        ];
    }

    filterTemplates(category) {
        const templateGrid = document.getElementById('templateGrid');
        if (!templateGrid) return;

        templateGrid.innerHTML = '';

        const filteredTemplates = category === 'all' 
            ? this.templates 
            : this.templates.filter(t => t.category === category);

        filteredTemplates.forEach(template => {
            const templateElement = document.createElement('div');
            templateElement.className = 'template-card';
            templateElement.innerHTML = `
                <h4>${template.name}</h4>
                <p>${template.description}</p>
                <div class="template-tags">
                    ${template.tags.map(tag => `<span class="template-tag">${tag}</span>`).join('')}
                </div>
            `;
            templateElement.addEventListener('click', () => this.selectTemplate(template));
            templateGrid.appendChild(templateElement);
        });
    }

    selectTemplate(template) {
        // Close template modal and populate generator with template
        this.closeTemplateModal();
        
        // Switch to generator view
        this.switchView('generator');
        
        // Set content type based on template category
        const contentTypeBtn = document.querySelector(`[data-type="${template.category}"]`);
        if (contentTypeBtn) {
            document.querySelectorAll('.content-type-btn').forEach(b => b.classList.remove('active'));
            contentTypeBtn.classList.add('active');
        }

        // Populate prompt with template-based suggestion
        const promptTextarea = document.getElementById('contentPrompt');
        if (promptTextarea) {
            promptTextarea.value = `Using the "${template.name}" template, create content that focuses on ${template.tags.join(', ')}.`;
        }
    }

    async generateContent() {
        const campaignSelect = document.getElementById('campaignSelect');
        const contentPrompt = document.getElementById('contentPrompt');
        const includeImages = document.getElementById('includeImages');

        if (!campaignSelect.value) {
            alert('Please select a campaign first');
            return;
        }

        if (!contentPrompt.value.trim()) {
            alert('Please enter a content prompt');
            return;
        }

        // Show loading overlay
        this.showLoading('Generating content with AI...');

        try {
            // Simulate API call to backend
            await this.delay(3000);

            // Get selected content type
            const selectedType = document.querySelector('.content-type-btn.active')?.dataset.type || 'blog';

            // Generate mock content based on type
            const generatedContent = this.generateMockContent(selectedType, contentPrompt.value);

            // Show generated content
            this.displayGeneratedContent(generatedContent);

            // Add to generated content history
            this.generatedContent.push({
                id: Date.now(),
                campaignId: parseInt(campaignSelect.value),
                type: selectedType,
                title: generatedContent.title,
                content: generatedContent.content,
                created: new Date(),
                status: 'draft'
            });

            this.updateStats();

        } catch (error) {
            console.error('Content generation failed:', error);
            alert('Content generation failed. Please try again.');
        } finally {
            this.hideLoading();
        }
    }

    generateMockContent(type, prompt) {
        const contentTemplates = {
            blog: {
                title: 'Revolutionizing Your Marketing Strategy',
                content: `# Revolutionizing Your Marketing Strategy

In today's rapidly evolving digital landscape, businesses must adapt their marketing strategies to stay competitive. Based on your campaign context and the prompt "${prompt}", here's a comprehensive approach to enhancing your marketing efforts.

## Key Strategies

1. **Data-Driven Decision Making**: Leverage analytics to understand your audience better
2. **Personalized Content**: Create tailored experiences for different customer segments
3. **Multi-Channel Integration**: Ensure consistent messaging across all platforms

## Implementation Steps

- Audit your current marketing efforts
- Identify key performance indicators
- Develop targeted content calendars
- Monitor and optimize continuously

This approach will help you achieve measurable results and drive business growth.`
            },
            social: {
                title: 'Social Media Content Pack',
                content: `🚀 Exciting news from our campaign!

📱 **Instagram Post**: "Ready to transform your business? Our latest innovation is here to help you reach new heights! #Innovation #Business #Growth"

🐦 **Twitter Post**: "Game-changing announcement coming your way! Stay tuned for something that will revolutionize how you think about [your industry]. #ComingSoon #Innovation"

💼 **LinkedIn Post**: "We're thrilled to share insights from our latest campaign. The results speak for themselves - increased engagement, better ROI, and happier customers. What's your biggest marketing challenge? Let's discuss in the comments."

📘 **Facebook Post**: "Behind the scenes of our amazing team working on something special. Can you guess what it is? Drop your thoughts below! 👇"

✨ **Story Content**: "Swipe up to see the magic happening behind the scenes of our latest project!"`
            },
            email: {
                title: 'Engaging Email Campaign',
                content: `Subject: You won't believe what we've been working on...

Hi [First Name],

I hope this email finds you well! I'm excited to share something special that our team has been developing based on your interests and our campaign goals.

**What's New?**
We've been listening to your feedback and working tirelessly to create something that addresses your biggest challenges. Without giving too much away, let me tell you that this is going to change how you approach [relevant topic].

**Why This Matters to You:**
- Solve problems faster than ever before
- Save time on repetitive tasks
- Achieve better results with less effort

**What's Next?**
Stay tuned for our official announcement next week. You'll be among the first to know when we launch!

Best regards,
The Campaign Studio Team

P.S. Keep an eye on your inbox - we have some exclusive early access opportunities coming your way!

[Unsubscribe] | [Update Preferences]`
            },
            ad: {
                title: 'High-Converting Advertisement',
                content: `🎯 **Ad Headline**: "Ready to 10x Your Results?"

**Primary Text**: 
Tired of slow growth and missed opportunities? Our proven system has helped thousands of businesses achieve extraordinary results in record time.

✅ Increase efficiency by 300%
✅ Reduce costs by up to 50%
✅ See results in as little as 30 days

**Call-to-Action**: "Get Started Free Today"

**Ad Copy Variations:**

**Variation A (Problem-Focused):**
"Struggling with [common problem]? You're not alone. 87% of businesses face this exact challenge. Here's how we solved it..."

**Variation B (Benefit-Focused):**
"Imagine achieving your biggest goals while working half as hard. It's not a dream - it's what our clients experience every day."

**Variation C (Social Proof):**
"Join 10,000+ successful businesses who've already transformed their operations with our solution."

**Target Audience**: Business owners, marketers, entrepreneurs aged 25-55
**Budget Recommendation**: $50-100/day
**Placement**: Facebook & Instagram feeds, stories`
            }
        };

        return contentTemplates[type] || contentTemplates.blog;
    }

    displayGeneratedContent(content) {
        const generatedContentCard = document.getElementById('generatedContentCard');
        const generatedContent = document.getElementById('generatedContent');

        if (!generatedContentCard || !generatedContent) return;

        generatedContent.innerHTML = `
            <h3>${content.title}</h3>
            <div style="white-space: pre-wrap; margin-top: 1rem;">${content.content}</div>
        `;

        generatedContentCard.style.display = 'block';
        generatedContentCard.scrollIntoView({ behavior: 'smooth' });
    }

    async regenerateContent() {
        await this.generateContent();
    }

    saveContent() {
        alert('Content saved successfully!');
        // In a real app, this would save to the backend
    }

    handleFileUpload(event) {
        const files = Array.from(event.target.files);
        
        files.forEach(file => {
            const newDoc = {
                id: Date.now() + Math.random(),
                campaignId: this.currentCampaign?.id || 1,
                name: file.name,
                type: file.name.split('.').pop().toLowerCase(),
                uploadDate: new Date(),
                size: this.formatFileSize(file.size)
            };

            this.documents.push(newDoc);
        });

        this.updateStats();
        this.loadDocuments();
        
        // Show success message
        alert(`${files.length} file(s) uploaded successfully!`);
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    searchDocuments() {
        const query = document.getElementById('documentSearch').value.toLowerCase();
        const filteredDocs = this.documents.filter(doc => 
            doc.name.toLowerCase().includes(query)
        );
        this.renderContextDocuments(filteredDocs);
    }

    handleSearch(event) {
        const query = event.target.value.toLowerCase();
        // Implement global search functionality
        console.log('Searching for:', query);
    }

    showLoading(message = 'Loading...') {
        const overlay = document.getElementById('loadingOverlay');
        const text = overlay.querySelector('p');
        if (text) text.textContent = message;
        overlay.classList.add('show');
    }

    hideLoading() {
        document.getElementById('loadingOverlay').classList.remove('show');
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    loadAnalytics() {
        // Simulate loading analytics data
        console.log('Loading analytics...');
    }
}

// Modal functions (called from HTML)
function createCampaign() {
    document.getElementById('campaignModal').classList.add('show');
}

function closeCampaignModal() {
    document.getElementById('campaignModal').classList.remove('show');
    // Clear form
    document.getElementById('campaignName').value = '';
    document.getElementById('campaignDescription').value = '';
    document.getElementById('campaignGuidelines').value = '';
}

function saveCampaign() {
    const name = document.getElementById('campaignName').value;
    const description = document.getElementById('campaignDescription').value;
    const guidelines = document.getElementById('campaignGuidelines').value;

    if (!name.trim()) {
        alert('Please enter a campaign name');
        return;
    }

    // Add new campaign
    const newCampaign = {
        id: Date.now(),
        name: name.trim(),
        description: description.trim(),
        guidelines: guidelines.trim(),
        created: new Date(),
        status: 'active'
    };

    window.campaignStudio.campaigns.push(newCampaign);
    window.campaignStudio.renderCampaigns();
    window.campaignStudio.updateStats();

    closeCampaignModal();
    alert('Campaign created successfully!');
}

function showTemplateGallery() {
    const modal = document.getElementById('templateModal');
    modal.classList.add('show');
    window.campaignStudio.filterTemplates('all');
}

function closeTemplateModal() {
    document.getElementById('templateModal').classList.remove('show');
}

function generateContent() {
    window.campaignStudio.generateContent();
}

function regenerateContent() {
    window.campaignStudio.regenerateContent();
}

function saveContent() {
    window.campaignStudio.saveContent();
}

function searchDocuments() {
    window.campaignStudio.searchDocuments();
}

function handleFileUpload(event) {
    window.campaignStudio.handleFileUpload(event);
}

function previousMonth() {
    window.campaignStudio.previousMonth();
}

function nextMonth() {
    window.campaignStudio.nextMonth();
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    window.campaignStudio = new CampaignStudio();
});

// Close modals when clicking outside
document.addEventListener('click', (e) => {
    if (e.target.classList.contains('modal')) {
        e.target.classList.remove('show');
    }
});