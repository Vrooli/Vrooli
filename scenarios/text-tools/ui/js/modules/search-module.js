// Search Module - Handles text search functionality
class SearchModule {
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
        this.patternInput = document.getElementById('search-pattern');
        this.textInput = document.getElementById('search-text');
        this.regexCheckbox = document.getElementById('search-regex');
        this.caseSensitiveCheckbox = document.getElementById('search-case-sensitive');
        this.wholeWordCheckbox = document.getElementById('search-whole-word');
        this.fuzzyCheckbox = document.getElementById('search-fuzzy');
        this.semanticCheckbox = document.getElementById('search-semantic');
        this.searchButton = document.getElementById('search-btn');
        this.resultsContainer = document.getElementById('search-results');
    }

    attachEventListeners() {
        if (this.searchButton) {
            this.searchButton.addEventListener('click', () => this.performSearch());
        }

        // Add keyboard shortcuts
        if (this.patternInput) {
            this.patternInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    this.performSearch();
                }
            });
        }

        if (this.textInput) {
            this.textInput.addEventListener('keydown', (e) => {
                if (e.ctrlKey && e.key === 'Enter') {
                    this.performSearch();
                }
            });
        }
    }

    async performSearch() {
        const pattern = this.patternInput?.value;
        const text = this.textInput?.value;

        if (!pattern) {
            this.showError('Please enter a search pattern');
            return;
        }

        if (!text) {
            this.showError('Please enter text to search in');
            return;
        }

        const options = {
            regex: this.regexCheckbox?.checked || false,
            case_sensitive: this.caseSensitiveCheckbox?.checked || false,
            whole_word: this.wholeWordCheckbox?.checked || false,
            fuzzy: this.fuzzyCheckbox?.checked || false,
            semantic: this.semanticCheckbox?.checked || false
        };

        this.showLoading();

        try {
            const result = await this.apiClient.searchText(text, pattern, options);
            this.displayResults(result);
        } catch (error) {
            console.error('Search error:', error);
            this.showError(error.message || 'Search failed');
        }
    }

    displayResults(result) {
        if (!this.resultsContainer) return;

        const { matches, total_matches } = result;

        if (total_matches === 0) {
            this.resultsContainer.innerHTML = `
                <div class="no-results">
                    <h4>No matches found</h4>
                    <p>No results for your search pattern.</p>
                </div>
            `;
            return;
        }

        let html = `
            <div class="results-header">
                <h4>${total_matches} match${total_matches !== 1 ? 'es' : ''} found</h4>
            </div>
            <div class="matches-list">
        `;

        matches.forEach((match, index) => {
            const { line, column, length, context, score } = match;
            
            // Highlight the match in context
            const beforeMatch = context.substring(0, column - 1);
            const matchText = context.substring(column - 1, column - 1 + length);
            const afterMatch = context.substring(column - 1 + length);

            html += `
                <div class="match-item" data-match="${index}">
                    <div class="match-info">
                        <span class="match-location">Line ${line}, Column ${column}</span>
                        ${score ? `<span class="match-score">Score: ${(score * 100).toFixed(1)}%</span>` : ''}
                    </div>
                    <div class="match-context">
                        <code>
                            ${this.escapeHtml(beforeMatch)}<mark class="match-highlight">${this.escapeHtml(matchText)}</mark>${this.escapeHtml(afterMatch)}
                        </code>
                    </div>
                </div>
            `;
        });

        html += '</div>';
        this.resultsContainer.innerHTML = html;
    }

    showLoading() {
        if (!this.resultsContainer) return;
        
        this.resultsContainer.innerHTML = `
            <div class="loading-state">
                <div class="loading-spinner"></div>
                <p>Searching...</p>
            </div>
        `;
    }

    showError(message) {
        if (!this.resultsContainer) return;
        
        this.resultsContainer.innerHTML = `
            <div class="error-state">
                <h4>Search Error</h4>
                <p>${this.escapeHtml(message)}</p>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Public methods for external access
    clearSearch() {
        if (this.patternInput) this.patternInput.value = '';
        if (this.textInput) this.textInput.value = '';
        if (this.resultsContainer) this.resultsContainer.innerHTML = '';
        
        // Reset checkboxes
        [this.regexCheckbox, this.caseSensitiveCheckbox, this.wholeWordCheckbox, 
         this.fuzzyCheckbox, this.semanticCheckbox].forEach(checkbox => {
            if (checkbox) checkbox.checked = false;
        });
    }

    setSearchPattern(pattern) {
        if (this.patternInput) {
            this.patternInput.value = pattern;
        }
    }

    setSearchText(text) {
        if (this.textInput) {
            this.textInput.value = text;
        }
    }
}

// Export for use in other modules
window.SearchModule = SearchModule;