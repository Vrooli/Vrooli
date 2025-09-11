// Pipeline Module - Handles text processing pipelines
class PipelineModule {
    constructor(apiClient) {
        this.apiClient = apiClient;
        this.steps = [];
        this.nextStepId = 1;
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
        this.inputTextarea = document.getElementById('pipeline-input');
        this.outputTextarea = document.getElementById('pipeline-output');
        this.addStepButton = document.getElementById('pipeline-add-step');
        this.clearButton = document.getElementById('pipeline-clear');
        this.saveTemplateButton = document.getElementById('pipeline-save-template');
        this.loadTemplateButton = document.getElementById('pipeline-load-template');
        this.executeButton = document.getElementById('pipeline-execute-btn');
        this.progressIndicator = document.getElementById('pipeline-progress');
        this.stepsContainer = document.getElementById('steps-container');
        this.stepCountElement = document.querySelector('.step-count');
        this.resultsTimeline = document.getElementById('results-timeline');
        this.fileInput = document.getElementById('pipeline-file');
        this.stepTemplate = document.getElementById('pipeline-step-template');
    }

    attachEventListeners() {
        if (this.addStepButton) {
            this.addStepButton.addEventListener('click', () => this.addStep());
        }

        if (this.clearButton) {
            this.clearButton.addEventListener('click', () => this.clearPipeline());
        }

        if (this.saveTemplateButton) {
            this.saveTemplateButton.addEventListener('click', () => this.saveTemplate());
        }

        if (this.loadTemplateButton) {
            this.loadTemplateButton.addEventListener('click', () => this.loadTemplate());
        }

        if (this.executeButton) {
            this.executeButton.addEventListener('click', () => this.executePipeline());
        }

        if (this.fileInput) {
            this.fileInput.addEventListener('change', (e) => this.handleFileUpload(e));
        }

        // Keyboard shortcuts
        if (this.inputTextarea) {
            this.inputTextarea.addEventListener('keydown', (e) => {
                if (e.ctrlKey && e.shiftKey && e.key === 'Enter') {
                    this.executePipeline();
                }
            });
        }
    }

    addStep() {
        if (!this.stepTemplate || !this.stepsContainer) return;

        // Hide no-steps message if present
        const noStepsMessage = this.stepsContainer.querySelector('.no-steps-message');
        if (noStepsMessage) {
            noStepsMessage.style.display = 'none';
        }

        // Clone template
        const stepElement = this.stepTemplate.content.cloneNode(true);
        const stepContainer = stepElement.querySelector('.pipeline-step');
        
        const stepId = this.nextStepId++;
        stepContainer.dataset.stepId = stepId;

        // Set step number
        const stepNumber = stepContainer.querySelector('.step-number');
        if (stepNumber) {
            stepNumber.textContent = this.steps.length + 1;
        }

        // Add event listeners for this step
        this.attachStepEventListeners(stepContainer, stepId);

        // Add to DOM
        this.stepsContainer.appendChild(stepElement);

        // Add to steps array
        this.steps.push({
            id: stepId,
            name: `Step ${this.steps.length + 1}`,
            operation: '',
            parameters: {}
        });

        this.updateUI();
    }

    attachStepEventListeners(stepContainer, stepId) {
        // Operation selection
        const operationSelect = stepContainer.querySelector('.step-operation');
        if (operationSelect) {
            operationSelect.addEventListener('change', (e) => {
                this.updateStepOperation(stepId, e.target.value);
            });
        }

        // Remove step button
        const removeButton = stepContainer.querySelector('.step-remove');
        if (removeButton) {
            removeButton.addEventListener('click', () => this.removeStep(stepId));
        }
    }

    updateStepOperation(stepId, operation) {
        const step = this.steps.find(s => s.id === stepId);
        if (!step) return;

        step.operation = operation;
        
        // Update step configuration UI
        const stepContainer = document.querySelector(`[data-step-id="${stepId}"]`);
        if (stepContainer) {
            this.updateStepConfig(stepContainer, operation);
        }

        this.updateUI();
    }

    updateStepConfig(stepContainer, operation) {
        const configContainer = stepContainer.querySelector('.step-config');
        if (!configContainer) return;

        // Clear existing config
        configContainer.innerHTML = '';
        
        if (!operation) {
            configContainer.style.display = 'none';
            return;
        }

        configContainer.style.display = 'block';

        // Generate configuration UI based on operation type
        switch (operation) {
            case 'transform':
                configContainer.innerHTML = `
                    <div class="config-group">
                        <label>Transformation:</label>
                        <select class="transform-type">
                            <option value="">Select transformation</option>
                            <option value="upper">Uppercase</option>
                            <option value="lower">Lowercase</option>
                            <option value="title">Title Case</option>
                            <option value="sanitize">Sanitize HTML</option>
                        </select>
                    </div>
                `;
                break;

            case 'search':
                configContainer.innerHTML = `
                    <div class="config-group">
                        <label>Search Pattern:</label>
                        <input type="text" class="search-pattern" placeholder="Enter pattern to search for">
                    </div>
                    <div class="config-group">
                        <label>Replace With:</label>
                        <input type="text" class="replace-text" placeholder="Enter replacement text">
                    </div>
                    <div class="config-group">
                        <label>
                            <input type="checkbox" class="regex-enabled">
                            Use Regex
                        </label>
                    </div>
                `;
                break;

            case 'extract':
                configContainer.innerHTML = `
                    <div class="config-group">
                        <label>Extract Type:</label>
                        <select class="extract-type">
                            <option value="entities">Named Entities</option>
                            <option value="urls">URLs</option>
                            <option value="emails">Email Addresses</option>
                            <option value="numbers">Numbers</option>
                        </select>
                    </div>
                `;
                break;

            case 'analyze':
                configContainer.innerHTML = `
                    <div class="config-group">
                        <label>Analysis Type:</label>
                        <select class="analyze-type">
                            <option value="summary">Generate Summary</option>
                            <option value="sentiment">Sentiment Analysis</option>
                            <option value="keywords">Extract Keywords</option>
                            <option value="language">Detect Language</option>
                        </select>
                    </div>
                    <div class="config-group summary-options" style="display: none;">
                        <label>Summary Length (words):</label>
                        <input type="number" class="summary-length" value="50" min="10" max="500">
                    </div>
                `;

                // Show/hide summary options
                const analyzeSelect = configContainer.querySelector('.analyze-type');
                const summaryOptions = configContainer.querySelector('.summary-options');
                if (analyzeSelect && summaryOptions) {
                    analyzeSelect.addEventListener('change', (e) => {
                        summaryOptions.style.display = e.target.value === 'summary' ? 'block' : 'none';
                    });
                }
                break;
        }
    }

    removeStep(stepId) {
        // Remove from steps array
        this.steps = this.steps.filter(s => s.id !== stepId);

        // Remove from DOM
        const stepContainer = document.querySelector(`[data-step-id="${stepId}"]`);
        if (stepContainer) {
            stepContainer.remove();
        }

        // Update step numbers
        this.updateStepNumbers();
        this.updateUI();
    }

    updateStepNumbers() {
        const stepContainers = document.querySelectorAll('.pipeline-step');
        stepContainers.forEach((container, index) => {
            const stepNumber = container.querySelector('.step-number');
            if (stepNumber) {
                stepNumber.textContent = index + 1;
            }
        });
    }

    clearPipeline() {
        this.steps = [];
        this.nextStepId = 1;
        
        if (this.stepsContainer) {
            this.stepsContainer.innerHTML = `
                <div class="no-steps-message">
                    <p>No processing steps configured.</p>
                    <p>Click "Add Step" to create your text processing pipeline.</p>
                </div>
            `;
        }

        if (this.inputTextarea) this.inputTextarea.value = '';
        if (this.outputTextarea) this.outputTextarea.value = '';
        if (this.resultsTimeline) this.resultsTimeline.innerHTML = '';

        this.updateUI();
    }

    updateUI() {
        // Update step count
        if (this.stepCountElement) {
            const count = this.steps.length;
            this.stepCountElement.textContent = count === 0 ? 'No steps added' : 
                                              count === 1 ? '1 step configured' : 
                                              `${count} steps configured`;
        }

        // Update execute button state
        if (this.executeButton) {
            const hasInput = this.inputTextarea?.value.trim().length > 0;
            const hasSteps = this.steps.length > 0 && this.steps.some(step => step.operation);
            this.executeButton.disabled = !hasInput || !hasSteps;
        }
    }

    async handleFileUpload(event) {
        const file = event.target.files[0];
        if (!file) return;

        try {
            const text = await this.readFile(file);
            if (this.inputTextarea) {
                this.inputTextarea.value = text;
                this.updateUI();
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

    async executePipeline() {
        const input = this.inputTextarea?.value;
        if (!input) {
            this.showError('Please enter input text');
            return;
        }

        if (this.steps.length === 0) {
            this.showError('Please add at least one processing step');
            return;
        }

        // Collect configured steps
        const pipelineSteps = this.collectPipelineSteps();
        if (pipelineSteps.length === 0) {
            this.showError('Please configure at least one processing step');
            return;
        }

        this.showLoading();

        try {
            const result = await this.apiClient.processPipeline(input, pipelineSteps);
            this.displayResults(result);
        } catch (error) {
            console.error('Pipeline execution error:', error);
            this.showError(error.message || 'Pipeline execution failed');
        }
    }

    collectPipelineSteps() {
        const pipelineSteps = [];

        this.steps.forEach(step => {
            if (!step.operation) return;

            const stepContainer = document.querySelector(`[data-step-id="${step.id}"]`);
            if (!stepContainer) return;

            const parameters = {};

            switch (step.operation) {
                case 'transform':
                    const transformType = stepContainer.querySelector('.transform-type')?.value;
                    if (transformType) {
                        parameters.type = transformType;
                    }
                    break;

                case 'search':
                    const searchPattern = stepContainer.querySelector('.search-pattern')?.value;
                    const replaceText = stepContainer.querySelector('.replace-text')?.value;
                    const useRegex = stepContainer.querySelector('.regex-enabled')?.checked;
                    
                    if (searchPattern) {
                        parameters.pattern = searchPattern;
                        parameters.replacement = replaceText || '';
                        parameters.regex = useRegex || false;
                    }
                    break;

                case 'extract':
                    const extractType = stepContainer.querySelector('.extract-type')?.value;
                    if (extractType) {
                        parameters.type = extractType;
                    }
                    break;

                case 'analyze':
                    const analyzeType = stepContainer.querySelector('.analyze-type')?.value;
                    if (analyzeType) {
                        parameters.type = analyzeType;
                        if (analyzeType === 'summary') {
                            const summaryLength = stepContainer.querySelector('.summary-length')?.value;
                            parameters.length = parseInt(summaryLength) || 50;
                        }
                    }
                    break;
            }

            if (Object.keys(parameters).length > 0) {
                pipelineSteps.push({
                    name: step.name,
                    operation: step.operation,
                    parameters
                });
            }
        });

        return pipelineSteps;
    }

    displayResults(result) {
        const { final_output, steps } = result;

        // Update output
        if (this.outputTextarea) {
            this.outputTextarea.value = final_output;
        }

        // Display step-by-step results
        if (this.resultsTimeline && steps) {
            this.displayResultsTimeline(steps);
        }

        this.hideLoading();
    }

    displayResultsTimeline(steps) {
        let html = '<div class="timeline-steps">';

        steps.forEach((step, index) => {
            const { step_name, output, metadata } = step;
            
            html += `
                <div class="timeline-step">
                    <div class="step-marker">${index + 1}</div>
                    <div class="step-content">
                        <h5 class="step-title">${this.escapeHtml(step_name)}</h5>
                        <div class="step-output">
                            <div class="output-preview">
                                ${this.escapeHtml(output.substring(0, 200))}${output.length > 200 ? '...' : ''}
                            </div>
                            ${metadata ? `<div class="step-metadata">${JSON.stringify(metadata)}</div>` : ''}
                        </div>
                    </div>
                </div>
            `;
        });

        html += '</div>';
        this.resultsTimeline.innerHTML = html;
    }

    showLoading() {
        if (this.outputTextarea) {
            this.outputTextarea.value = '';
        }
        
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = `
                <div class="loading-state">
                    <div class="loading-spinner"></div>
                    <p>Executing pipeline...</p>
                </div>
            `;
        }
        
        if (this.executeButton) {
            this.executeButton.disabled = true;
            this.executeButton.textContent = 'Executing...';
        }
    }

    hideLoading() {
        if (this.progressIndicator) {
            this.progressIndicator.innerHTML = '';
        }
        
        if (this.executeButton) {
            this.executeButton.disabled = false;
            this.executeButton.textContent = 'Execute Pipeline';
        }

        this.updateUI();
    }

    showError(message) {
        this.hideLoading();
        
        if (this.resultsTimeline) {
            this.resultsTimeline.innerHTML = `
                <div class="error-state">
                    <h4>Pipeline Error</h4>
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

    // Template management
    saveTemplate() {
        if (this.steps.length === 0) {
            alert('No steps to save');
            return;
        }

        const templateName = prompt('Enter template name:');
        if (!templateName) return;

        const template = {
            name: templateName,
            steps: this.collectPipelineSteps(),
            created: new Date().toISOString()
        };

        const templates = JSON.parse(localStorage.getItem('pipelineTemplates') || '[]');
        templates.push(template);
        localStorage.setItem('pipelineTemplates', JSON.stringify(templates));

        alert(`Template "${templateName}" saved successfully`);
    }

    loadTemplate() {
        const templates = JSON.parse(localStorage.getItem('pipelineTemplates') || '[]');
        if (templates.length === 0) {
            alert('No saved templates found');
            return;
        }

        const templateList = templates.map((t, index) => `${index}: ${t.name} (${new Date(t.created).toLocaleDateString()})`).join('\n');
        const selection = prompt(`Select template by number:\n${templateList}`);
        
        if (selection === null) return;
        
        const index = parseInt(selection);
        if (isNaN(index) || index < 0 || index >= templates.length) {
            alert('Invalid selection');
            return;
        }

        const template = templates[index];
        this.loadTemplateSteps(template.steps);
    }

    loadTemplateSteps(steps) {
        this.clearPipeline();
        
        steps.forEach(step => {
            this.addStep();
            const stepId = this.steps[this.steps.length - 1].id;
            this.updateStepOperation(stepId, step.operation);
            
            // Set parameters
            const stepContainer = document.querySelector(`[data-step-id="${stepId}"]`);
            if (stepContainer) {
                this.setStepParameters(stepContainer, step.operation, step.parameters);
            }
        });
    }

    setStepParameters(stepContainer, operation, parameters) {
        switch (operation) {
            case 'transform':
                const transformSelect = stepContainer.querySelector('.transform-type');
                if (transformSelect && parameters.type) {
                    transformSelect.value = parameters.type;
                }
                break;

            case 'search':
                const patternInput = stepContainer.querySelector('.search-pattern');
                const replaceInput = stepContainer.querySelector('.replace-text');
                const regexCheckbox = stepContainer.querySelector('.regex-enabled');
                
                if (patternInput && parameters.pattern) patternInput.value = parameters.pattern;
                if (replaceInput && parameters.replacement) replaceInput.value = parameters.replacement;
                if (regexCheckbox && parameters.regex) regexCheckbox.checked = parameters.regex;
                break;

            // Add other parameter setting cases as needed
        }
    }

    // Public methods for external access
    setInputText(text) {
        if (this.inputTextarea) {
            this.inputTextarea.value = text;
            this.updateUI();
        }
    }

    getOutputText() {
        return this.outputTextarea?.value || '';
    }
}

// Export for use in other modules
window.PipelineModule = PipelineModule;