// Diff Module
class DiffModule {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.initializeElements();
        this.attachEventListeners();
    }

    initializeElements() {
        this.text1Input = document.getElementById('diff-text1');
        this.text2Input = document.getElementById('diff-text2');
        this.diffTypeSelect = document.getElementById('diff-type');
        this.ignoreWhitespaceCheckbox = document.getElementById('diff-ignore-whitespace');
        this.ignoreCaseCheckbox = document.getElementById('diff-ignore-case');
        this.compareButton = document.getElementById('diff-compare-btn');
        this.similaritySpan = document.getElementById('diff-similarity');
        this.resultsContainer = document.getElementById('diff-results');
    }

    attachEventListeners() {
        if (this.compareButton) {
            this.compareButton.addEventListener('click', () => this.performDiff());
        }

        // Add keyboard shortcut (Ctrl+Enter to compare)
        if (this.text1Input && this.text2Input) {
            [this.text1Input, this.text2Input].forEach(input => {
                input.addEventListener('keydown', (e) => {
                    if (e.ctrlKey && e.key === 'Enter') {
                        this.performDiff();
                    }
                });
            });
        }
    }

    async performDiff() {
        const text1 = this.text1Input?.value;
        const text2 = this.text2Input?.value;

        if (!text1 || !text2) {
            this.showError('Please enter text in both fields');
            return;
        }

        const options = {
            type: this.diffTypeSelect?.value || 'line',
            ignore_whitespace: this.ignoreWhitespaceCheckbox?.checked || false,
            ignore_case: this.ignoreCaseCheckbox?.checked || false,
        };

        try {
            this.setLoading(true);
            const result = await this.apiClient.diff(text1, text2, options);
            this.displayResults(result);
        } catch (error) {
            this.showError(`Diff failed: ${error.message}`);
        } finally {
            this.setLoading(false);
        }
    }

    displayResults(result) {
        if (!this.resultsContainer) return;

        // Display similarity score
        if (this.similaritySpan) {
            const percentage = (result.similarity_score * 100).toFixed(1);
            this.similaritySpan.textContent = `${percentage}% Similar`;
            this.similaritySpan.className = `similarity-score ${
                result.similarity_score > 0.8 ? 'high' : 
                result.similarity_score > 0.5 ? 'medium' : 'low'
            }`;
        }

        // Display changes
        let html = '<div class="diff-results">';
        html += `<div class="diff-summary">${result.summary}</div>`;
        
        if (result.changes && result.changes.length > 0) {
            html += '<div class="diff-changes">';
            result.changes.forEach(change => {
                const typeClass = `change-${change.type}`;
                const icon = change.type === 'add' ? '+' : 
                            change.type === 'remove' ? '-' : '~';
                
                html += `
                    <div class="change-item ${typeClass}">
                        <span class="change-icon">${icon}</span>
                        <span class="change-line">Line ${change.line_start}${
                            change.line_end !== change.line_start ? `-${change.line_end}` : ''
                        }</span>
                        <pre class="change-content">${this.escapeHtml(change.content)}</pre>
                    </div>
                `;
            });
            html += '</div>';
        } else {
            html += '<div class="no-changes">No differences found</div>';
        }
        
        html += '</div>';
        this.resultsContainer.innerHTML = html;
    }

    showError(message) {
        if (this.resultsContainer) {
            this.resultsContainer.innerHTML = `
                <div class="error-message">
                    <span class="error-icon">⚠️</span>
                    <span>${message}</span>
                </div>
            `;
        }
    }

    setLoading(loading) {
        if (this.compareButton) {
            this.compareButton.disabled = loading;
            this.compareButton.textContent = loading ? 'Comparing...' : 'Compare';
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // File upload support
    async loadFile(inputElement, targetTextarea) {
        const file = inputElement.files[0];
        if (!file) return;

        try {
            const text = await file.text();
            targetTextarea.value = text;
        } catch (error) {
            this.showError(`Failed to load file: ${error.message}`);
        }
    }
}

// Export for use
window.DiffModule = DiffModule;