// Analyze Module - Handles text analysis functionality
class AnalyzeModule {
    constructor(apiClient) {
        this.apiClient = apiClient;
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
        this.inputTextarea = document.getElementById('analyze-input');
        this.entitiesCheckbox = document.getElementById('analyze-entities');
        this.sentimentCheckbox = document.getElementById('analyze-sentiment');
        this.keywordsCheckbox = document.getElementById('analyze-keywords');
        this.languageCheckbox = document.getElementById('analyze-language');
        this.summaryCheckbox = document.getElementById('analyze-summary');
        this.summaryLengthInput = document.getElementById('summary-length');
        this.useAICheckbox = document.getElementById('analyze-use-ai');
        this.analyzeButton = document.getElementById('analyze-btn');
        this.progressIndicator = document.getElementById('analysis-progress');
        this.resultsSection = document.getElementById('analyze-results');
        this.fileInput = document.getElementById('analyze-file');

        // Result cards
        this.entitiesCard = document.getElementById('entities-card');
        this.sentimentCard = document.getElementById('sentiment-card');
        this.keywordsCard = document.getElementById('keywords-card');
        this.languageCard = document.getElementById('language-card');
        this.summaryCard = document.getElementById('summary-card');
        this.statisticsCard = document.getElementById('statistics-card');

        // Result content containers
        this.entitiesList = document.getElementById('entities-list');
        this.sentimentResult = document.getElementById('sentiment-result');
        this.keywordsCloud = document.getElementById('keywords-cloud');
        this.languageResult = document.getElementById('language-result');
        this.summaryResult = document.getElementById('summary-result');
        this.statisticsResult = document.getElementById('statistics-result');
    }

    attachEventListeners() {
        if (this.analyzeButton) {
            this.analyzeButton.addEventListener('click', () => this.performAnalysis());
        }

        // File upload
        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => this.handleFileUpload(e));
        }

        // Keyboard shortcuts
        if (this.inputTextarea) {
            this.inputTextarea.addEventListener('keydown', (e) => {
                if (e.ctrlKey && e.key === 'Enter') {
                    this.performAnalysis();
                }
            });
        }

        // Toggle summary options when summary checkbox changes
        if (this.summaryCheckbox && this.summaryLengthInput) {
            this.summaryCheckbox.addEventListener('change', () => {
                this.summaryLengthInput.disabled = !this.summaryCheckbox.checked;
            });
        }

        // Update result cards visibility when checkboxes change
        [this.entitiesCheckbox, this.sentimentCheckbox, this.keywordsCheckbox, 
         this.languageCheckbox, this.summaryCheckbox].forEach(checkbox => {
            if (checkbox) {
                checkbox.addEventListener('change', () => this.updateResultCardsVisibility());
            }
        });
    }

    updateResultCardsVisibility() {
        const cardMap = {
            [this.entitiesCheckbox?.id]: this.entitiesCard,
            [this.sentimentCheckbox?.id]: this.sentimentCard,
            [this.keywordsCheckbox?.id]: this.keywordsCard,
            [this.languageCheckbox?.id]: this.languageCard,
            [this.summaryCheckbox?.id]: this.summaryCard
        };

        Object.entries(cardMap).forEach(([checkboxId, card]) => {
            const checkbox = document.getElementById(checkboxId);
            if (checkbox && card) {
                card.style.display = checkbox.checked ? 'block' : 'none';
            }
        });

        // Always show statistics card
        if (this.statisticsCard) {
            this.statisticsCard.style.display = 'block';
        }
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

    async performAnalysis() {
        const text = this.inputTextarea?.value;

        if (!text || text.trim().length === 0) {
            this.showError('Please enter text to analyze');
            return;
        }

        // Collect selected analyses
        const analyses = [];
        if (this.entitiesCheckbox?.checked) analyses.push('entities');
        if (this.sentimentCheckbox?.checked) analyses.push('sentiment');
        if (this.keywordsCheckbox?.checked) analyses.push('keywords');
        if (this.languageCheckbox?.checked) analyses.push('language');
        if (this.summaryCheckbox?.checked) analyses.push('summary');

        if (analyses.length === 0) {
            this.showError('Please select at least one analysis type');
            return;
        }

        const options = {
            summary_length: parseInt(this.summaryLengthInput?.value || 50),
            use_ai: this.useAICheckbox?.checked || false
        };

        this.showLoading();
        this.updateResultCardsVisibility();

        try {
            const result = await this.apiClient.analyzeText(text, analyses, options);
            this.displayResults(result, text);
        } catch (error) {
            console.error('Analysis error:', error);
            this.showError(error.message || 'Text analysis failed');
        }
    }

    displayResults(result, originalText) {
        const { entities, sentiment, keywords, language, summary } = result;

        // Display entities
        if (entities && this.entitiesList) {
            this.displayEntities(entities);
        }

        // Display sentiment
        if (sentiment && this.sentimentResult) {
            this.displaySentiment(sentiment);
        }

        // Display keywords
        if (keywords && this.keywordsCloud) {
            this.displayKeywords(keywords);
        }

        // Display language
        if (language && this.languageResult) {
            this.displayLanguage(language);
        }

        // Display summary
        if (summary && this.summaryResult) {
            this.displaySummary(summary);
        }

        // Display statistics (always shown)
        if (this.statisticsResult) {
            this.displayStatistics(originalText);
        }

        this.hideLoading();
    }

    displayEntities(entities) {
        if (entities.length === 0) {
            this.entitiesList.innerHTML = '<p class="no-data">No entities found</p>';
            return;
        }

        const entitiesByType = {};
        entities.forEach(entity => {
            if (!entitiesByType[entity.type]) {
                entitiesByType[entity.type] = [];
            }
            entitiesByType[entity.type].push(entity);
        });

        let html = '';
        Object.entries(entitiesByType).forEach(([type, typeEntities]) => {
            html += `
                <div class="entity-group">
                    <h5 class="entity-type">${type.toUpperCase()}</h5>
                    <div class="entity-tags">
            `;
            
            typeEntities.forEach(entity => {
                const confidenceClass = entity.confidence > 0.8 ? 'high' : 
                                       entity.confidence > 0.5 ? 'medium' : 'low';
                html += `
                    <span class="entity-tag confidence-${confidenceClass}" 
                          title="Confidence: ${(entity.confidence * 100).toFixed(1)}%">
                        ${this.escapeHtml(entity.value)}
                    </span>
                `;
            });
            
            html += '</div></div>';
        });

        this.entitiesList.innerHTML = html;
    }

    displaySentiment(sentiment) {
        const { score, label } = sentiment;
        const percentage = (score * 100).toFixed(1);
        
        let sentimentClass = 'neutral';
        if (score > 0.6) sentimentClass = 'positive';
        else if (score < 0.4) sentimentClass = 'negative';
        
        this.sentimentResult.innerHTML = `
            <div class="sentiment-display">
                <div class="sentiment-score ${sentimentClass}">
                    <span class="sentiment-label">${label.toUpperCase()}</span>
                    <span class="sentiment-percentage">${percentage}%</span>
                </div>
                <div class="sentiment-bar">
                    <div class="sentiment-fill ${sentimentClass}" style="width: ${percentage}%"></div>
                </div>
            </div>
        `;
    }

    displayKeywords(keywords) {
        if (keywords.length === 0) {
            this.keywordsCloud.innerHTML = '<p class="no-data">No keywords found</p>';
            return;
        }

        // Sort keywords by score
        const sortedKeywords = keywords.sort((a, b) => b.score - a.score);
        
        let html = '<div class="keyword-cloud">';
        sortedKeywords.forEach(keyword => {
            // Scale font size based on score
            const fontSize = Math.max(0.8, Math.min(2, keyword.score * 10));
            html += `
                <span class="keyword-tag" 
                      style="font-size: ${fontSize}em"
                      title="Score: ${(keyword.score * 100).toFixed(1)}%">
                    ${this.escapeHtml(keyword.word)}
                </span>
            `;
        });
        html += '</div>';

        this.keywordsCloud.innerHTML = html;
    }

    displayLanguage(language) {
        const { code, name, confidence } = language;
        const percentage = (confidence * 100).toFixed(1);
        
        this.languageResult.innerHTML = `
            <div class="language-detection">
                <div class="language-info">
                    <span class="language-name">${this.escapeHtml(name)}</span>
                    <span class="language-code">(${this.escapeHtml(code)})</span>
                </div>
                <div class="language-confidence">
                    <span class="confidence-label">Confidence:</span>
                    <span class="confidence-value">${percentage}%</span>
                </div>
            </div>
        `;
    }

    displaySummary(summary) {
        this.summaryResult.innerHTML = `
            <div class="summary-content">
                <p>${this.escapeHtml(summary)}</p>
            </div>
        `;
    }

    displayStatistics(text) {
        const stats = this.calculateTextStatistics(text);
        
        this.statisticsResult.innerHTML = `
            <div class="statistics-grid">
                <div class="stat-item">
                    <span class="stat-value">${stats.characters}</span>
                    <span class="stat-label">Characters</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.words}</span>
                    <span class="stat-label">Words</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.sentences}</span>
                    <span class="stat-label">Sentences</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.paragraphs}</span>
                    <span class="stat-label">Paragraphs</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.avgWordsPerSentence}</span>
                    <span class="stat-label">Avg Words/Sentence</span>
                </div>
                <div class="stat-item">
                    <span class="stat-value">${stats.readingTime}</span>
                    <span class="stat-label">Reading Time (min)</span>
                </div>
            </div>
        `;
    }

    calculateTextStatistics(text) {
        const characters = text.length;
        const words = text.split(/\s+/).filter(word => word.length > 0).length;
        const sentences = text.split(/[.!?]+/).filter(s => s.trim().length > 0).length;
        const paragraphs = text.split(/\n\s*\n/).filter(p => p.trim().length > 0).length;
        const avgWordsPerSentence = sentences > 0 ? Math.round(words / sentences) : 0;
        const readingTime = Math.ceil(words / 200); // Assuming 200 words per minute
        
        return { characters, words, sentences, paragraphs, avgWordsPerSentence, readingTime };
    }

    showLoading() {
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = `
                <div class="loading-state">
                    <div class="loading-spinner"></div>
                    <p>Analyzing text...</p>
                </div>
            `;
        }
        
        if (this.analyzeButton) {
            this.analyzeButton.disabled = true;
            this.analyzeButton.textContent = 'Analyzing...';
        }
    }

    hideLoading() {
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = '';
        }
        
        if (this.analyzeButton) {
            this.analyzeButton.disabled = false;
            this.analyzeButton.textContent = 'Analyze Text';
        }
    }

    showError(message) {
        this.hideLoading();
        
        if (this.resultsSection) {
            this.resultsSection.innerHTML = `
                <div class="error-state">
                    <h4>Analysis Error</h4>
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
    clearAnalysis() {
        if (this.inputTextarea) this.inputTextarea.value = '';
        if (this.progressIndicator) this.progressIndicator.innerHTML = '';
        
        // Clear result containers
        [this.entitiesList, this.sentimentResult, this.keywordsCloud,
         this.languageResult, this.summaryResult, this.statisticsResult].forEach(container => {
            if (container) container.innerHTML = '';
        });
        
        // Reset checkboxes to default
        if (this.entitiesCheckbox) this.entitiesCheckbox.checked = true;
        if (this.sentimentCheckbox) this.sentimentCheckbox.checked = true;
        if (this.keywordsCheckbox) this.keywordsCheckbox.checked = true;
        if (this.languageCheckbox) this.languageCheckbox.checked = true;
        if (this.summaryCheckbox) this.summaryCheckbox.checked = false;
        if (this.useAICheckbox) this.useAICheckbox.checked = false;
        
        this.updateResultCardsVisibility();
    }

    setAnalysisText(text) {
        if (this.inputTextarea) {
            this.inputTextarea.value = text;
        }
    }
}

// Export for use in other modules
window.AnalyzeModule = AnalyzeModule;