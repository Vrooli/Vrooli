// Auto Steer Profile Management Module
import { logger } from '../utils/logger.js';

/**
 * Manages Auto Steer profiles including CRUD operations, UI rendering,
 * and profile builder interactions
 */
export class AutoSteerManager {
    constructor(apiBase, showToast) {
        this.apiBase = apiBase;
        this.showToast = showToast;
        this.profiles = [];
        this.templates = [];
        this.currentEditingProfile = null;
        this.conditionBuilderState = null;
    }

    /**
     * Initialize the Auto Steer manager and load initial data
     */
    async initialize() {
        try {
            await Promise.all([
                this.loadProfiles(),
                this.loadTemplates()
            ]);
            this.renderProfilesList();
            logger.debug('Auto Steer manager initialized');
        } catch (error) {
            logger.error('Failed to initialize Auto Steer manager:', error);
            this.showToast('Failed to load Auto Steer profiles', 'error');
        }
    }

    /**
     * Load all profiles from the API
     */
    async loadProfiles() {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/profiles`);
            if (!response.ok) {
                throw new Error(`Failed to load profiles: ${response.statusText}`);
            }
            this.profiles = await response.json();
            logger.debug(`Loaded ${this.profiles.length} profiles`);
            return this.profiles;
        } catch (error) {
            logger.error('Failed to load profiles:', error);
            throw error;
        }
    }

    /**
     * Load built-in templates from the API
     */
    async loadTemplates() {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/templates`);
            if (!response.ok) {
                throw new Error(`Failed to load templates: ${response.statusText}`);
            }
            this.templates = await response.json();
            logger.debug(`Loaded ${this.templates.length} templates`);
            return this.templates;
        } catch (error) {
            logger.error('Failed to load templates:', error);
            throw error;
        }
    }

    /**
     * Create a new profile
     */
    async createProfile(profileData) {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/profiles`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(profileData)
            });

            if (!response.ok) {
                const error = await response.text();
                throw new Error(error || response.statusText);
            }

            const newProfile = await response.json();
            this.profiles.push(newProfile);
            this.renderProfilesList();
            this.showToast('Profile created successfully', 'success');
            return newProfile;
        } catch (error) {
            logger.error('Failed to create profile:', error);
            this.showToast(`Failed to create profile: ${error.message}`, 'error');
            throw error;
        }
    }

    /**
     * Update an existing profile
     */
    async updateProfile(id, updates) {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/profiles/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(updates)
            });

            if (!response.ok) {
                const error = await response.text();
                throw new Error(error || response.statusText);
            }

            const updatedProfile = await response.json();
            const index = this.profiles.findIndex(p => p.id === id);
            if (index !== -1) {
                this.profiles[index] = updatedProfile;
            }
            this.renderProfilesList();
            this.showToast('Profile updated successfully', 'success');
            return updatedProfile;
        } catch (error) {
            logger.error('Failed to update profile:', error);
            this.showToast(`Failed to update profile: ${error.message}`, 'error');
            throw error;
        }
    }

    /**
     * Delete a profile
     */
    async deleteProfile(id) {
        try {
            const response = await fetch(`${this.apiBase}/auto-steer/profiles/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) {
                throw new Error(`Failed to delete profile: ${response.statusText}`);
            }

            this.profiles = this.profiles.filter(p => p.id !== id);
            this.renderProfilesList();
            this.showToast('Profile deleted successfully', 'success');
        } catch (error) {
            logger.error('Failed to delete profile:', error);
            this.showToast(`Failed to delete profile: ${error.message}`, 'error');
            throw error;
        }
    }

    /**
     * Duplicate a profile
     */
    async duplicateProfile(id) {
        const profile = this.profiles.find(p => p.id === id);
        if (!profile) {
            this.showToast('Profile not found', 'error');
            return;
        }

        const duplicate = {
            ...profile,
            id: undefined,
            name: `${profile.name} (Copy)`,
            created_at: undefined,
            updated_at: undefined
        };

        await this.createProfile(duplicate);
    }

    /**
     * Render the profiles list in the UI
     */
    renderProfilesList() {
        const container = document.getElementById('autosteer-profiles-list');
        if (!container) {
            logger.warn('Profiles list container not found');
            return;
        }

        if (this.profiles.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <i class="fas fa-route fa-3x"></i>
                    <p>No profiles yet</p>
                    <p class="text-muted">Create a profile or use a template to get started</p>
                </div>
            `;
            return;
        }

        container.innerHTML = this.profiles.map(profile => this.renderProfileCard(profile)).join('');
    }

    /**
     * Render a single profile card
     */
    renderProfileCard(profile) {
        const phasesCount = profile.phases?.length || 0;
        const tags = (profile.tags || []).map(tag =>
            `<span class="badge badge-secondary">${this.escapeHtml(tag)}</span>`
        ).join('');

        const phaseIcons = (profile.phases || []).slice(0, 5).map(phase => {
            const icon = this.getModeIcon(phase.mode);
            return `<span class="phase-icon" title="${this.escapeHtml(phase.mode)}">${icon}</span>`;
        }).join('');

        const morePhases = phasesCount > 5 ?
            `<span class="phase-more">+${phasesCount - 5}</span>` : '';

        return `
            <div class="profile-card" data-profile-id="${this.escapeHtml(profile.id)}">
                <div class="profile-card-header">
                    <h4 class="profile-card-title">${this.escapeHtml(profile.name)}</h4>
                    <div class="profile-card-tags">${tags}</div>
                </div>

                <p class="profile-card-description">${this.escapeHtml(profile.description || '')}</p>

                <div class="profile-card-phases">
                    <div class="profile-card-phases-label">
                        <strong>${phasesCount}</strong> phase${phasesCount !== 1 ? 's' : ''}
                    </div>
                    <div class="profile-card-phases-icons">
                        ${phaseIcons}${morePhases}
                    </div>
                </div>

                <div class="profile-card-actions">
                    <button
                        class="btn btn-sm btn-secondary"
                        onclick="ecosystemManager.autoSteerManager.openProfileEditor('${this.escapeHtml(profile.id)}')"
                        title="Edit profile">
                        <i class="fas fa-edit"></i> Edit
                    </button>
                    <button
                        class="btn btn-sm btn-secondary"
                        onclick="ecosystemManager.autoSteerManager.duplicateProfile('${this.escapeHtml(profile.id)}')"
                        title="Duplicate profile">
                        <i class="fas fa-copy"></i> Duplicate
                    </button>
                    <button
                        class="btn btn-sm btn-danger"
                        onclick="ecosystemManager.autoSteerManager.confirmDeleteProfile('${this.escapeHtml(profile.id)}')"
                        title="Delete profile">
                        <i class="fas fa-trash"></i> Delete
                    </button>
                </div>
            </div>
        `;
    }

    /**
     * Render the templates gallery
     */
    renderTemplatesGallery() {
        const container = document.getElementById('autosteer-templates-gallery');
        if (!container) return;

        if (this.templates.length === 0) {
            container.innerHTML = '<p class="text-muted">No templates available</p>';
            return;
        }

        container.innerHTML = this.templates.map(template => this.renderTemplateCard(template)).join('');
    }

    /**
     * Render a single template card
     */
    renderTemplateCard(template) {
        const tags = (template.tags || []).map(tag =>
            `<span class="badge badge-info">${this.escapeHtml(tag)}</span>`
        ).join('');

        const phasesCount = template.phases?.length || 0;

        return `
            <div class="template-card">
                <div class="template-card-header">
                    <h5>${this.escapeHtml(template.name)}</h5>
                    <div class="template-card-tags">${tags}</div>
                </div>
                <p class="template-card-description">${this.escapeHtml(template.description || '')}</p>
                <div class="template-card-meta">
                    <span><i class="fas fa-layer-group"></i> ${phasesCount} phases</span>
                </div>
                <button
                    class="btn btn-sm btn-primary"
                    onclick="ecosystemManager.autoSteerManager.useTemplate('${this.escapeHtml(template.id || template.name)}')"
                    title="Use this template">
                    <i class="fas fa-magic"></i> Use Template
                </button>
            </div>
        `;
    }

    /**
     * Use a template to create a new profile
     */
    useTemplate(templateIdentifier) {
        const template = this.templates.find(t =>
            t.id === templateIdentifier || t.name === templateIdentifier
        );

        if (!template) {
            this.showToast('Template not found', 'error');
            return;
        }

        // Clone the template and prepare for editing
        const newProfile = {
            name: `${template.name} (Custom)`,
            description: template.description,
            phases: JSON.parse(JSON.stringify(template.phases || [])),
            quality_gates: JSON.parse(JSON.stringify(template.quality_gates || [])),
            tags: [...(template.tags || [])]
        };

        this.openProfileEditor(null, newProfile);
    }

    /**
     * Open the profile editor dialog
     */
    openProfileEditor(profileId, prefillData = null) {
        let profile = prefillData;

        if (profileId && !prefillData) {
            profile = this.profiles.find(p => p.id === profileId);
            if (!profile) {
                this.showToast('Profile not found', 'error');
                return;
            }
        }

        this.currentEditingProfile = profile ? { ...profile } : {
            name: '',
            description: '',
            phases: [],
            quality_gates: [],
            tags: []
        };

        this.renderProfileEditor(!!profileId);
        this.showModal('autosteer-profile-editor-modal');
    }

    /**
     * Render the profile editor modal content
     */
    renderProfileEditor(isEdit) {
        const modal = document.getElementById('autosteer-profile-editor-modal');
        if (!modal) {
            logger.error('Profile editor modal not found');
            return;
        }

        const profile = this.currentEditingProfile;

        modal.querySelector('.modal-title-text').textContent = isEdit ? 'Edit Profile' : 'Create Profile';

        const editorContent = modal.querySelector('.profile-editor-content');
        editorContent.innerHTML = `
            <div class="form-group">
                <label for="profile-name">Profile Name *</label>
                <input
                    type="text"
                    id="profile-name"
                    class="form-control"
                    value="${this.escapeHtml(profile.name || '')}"
                    placeholder="e.g., Production Ready"
                    required>
            </div>

            <div class="form-group">
                <label for="profile-description">Description</label>
                <textarea
                    id="profile-description"
                    class="form-control"
                    rows="3"
                    placeholder="Describe the purpose and focus of this profile">${this.escapeHtml(profile.description || '')}</textarea>
            </div>

            <div class="form-group">
                <label for="profile-tags">Tags (comma-separated)</label>
                <input
                    type="text"
                    id="profile-tags"
                    class="form-control"
                    value="${(profile.tags || []).map(t => this.escapeHtml(t)).join(', ')}"
                    placeholder="e.g., balanced, production, rapid">
            </div>

            <hr>

            <div class="form-section">
                <div class="form-section-header">
                    <h4>Phases</h4>
                    <button
                        type="button"
                        class="btn btn-sm btn-primary"
                        onclick="ecosystemManager.autoSteerManager.addPhase()">
                        <i class="fas fa-plus"></i> Add Phase
                    </button>
                </div>

                <div id="phases-list" class="phases-list">
                    ${this.renderPhasesList(profile.phases || [])}
                </div>
            </div>

            <hr>

            <div class="form-section">
                <div class="form-section-header">
                    <h4>Quality Gates</h4>
                    <button
                        type="button"
                        class="btn btn-sm btn-primary"
                        onclick="ecosystemManager.autoSteerManager.addQualityGate()">
                        <i class="fas fa-plus"></i> Add Quality Gate
                    </button>
                </div>

                <div id="quality-gates-list" class="quality-gates-list">
                    ${this.renderQualityGatesList(profile.quality_gates || [])}
                </div>
            </div>
        `;
    }

    /**
     * Render the list of phases in the editor
     */
    renderPhasesList(phases) {
        if (phases.length === 0) {
            return '<p class="text-muted">No phases yet. Add a phase to get started.</p>';
        }

        return phases.map((phase, index) => this.renderPhaseEditor(phase, index)).join('');
    }

    /**
     * Render a single phase editor
     */
    renderPhaseEditor(phase, index) {
        const modes = ['progress', 'ux', 'refactor', 'test', 'explore', 'polish', 'integration', 'performance', 'security'];

        return `
            <div class="phase-editor" data-phase-index="${index}">
                <div class="phase-editor-header">
                    <span class="phase-editor-number">#${index + 1}</span>
                    <button
                        type="button"
                        class="btn btn-sm btn-icon btn-danger"
                        onclick="ecosystemManager.autoSteerManager.removePhase(${index})"
                        title="Remove phase">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label>Mode *</label>
                        <select class="form-control phase-mode" data-phase-index="${index}">
                            ${modes.map(mode => `
                                <option value="${mode}" ${phase.mode === mode ? 'selected' : ''}>
                                    ${this.formatModeName(mode)}
                                </option>
                            `).join('')}
                        </select>
                    </div>

                    <div class="form-group">
                        <label>Max Iterations *</label>
                        <input
                            type="number"
                            class="form-control phase-max-iterations"
                            data-phase-index="${index}"
                            value="${phase.max_iterations || 10}"
                            min="1"
                            max="100"
                            required>
                    </div>
                </div>

                <div class="form-group">
                    <label>Description (optional)</label>
                    <input
                        type="text"
                        class="form-control phase-description"
                        data-phase-index="${index}"
                        value="${this.escapeHtml(phase.description || '')}"
                        placeholder="Optional description for this phase">
                </div>

                <div class="form-group">
                    <label>Stop Conditions</label>
                    <button
                        type="button"
                        class="btn btn-sm btn-secondary mb-2"
                        onclick="ecosystemManager.autoSteerManager.openConditionBuilder(${index})">
                        <i class="fas fa-code-branch"></i> Configure Conditions
                    </button>
                    <div class="phase-conditions-preview" id="phase-${index}-conditions-preview">
                        ${this.renderConditionsPreview(phase.stop_conditions || [])}
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render the list of quality gates in the editor
     */
    renderQualityGatesList(gates) {
        if (gates.length === 0) {
            return '<p class="text-muted">No quality gates configured.</p>';
        }

        return gates.map((gate, index) => this.renderQualityGateEditor(gate, index)).join('');
    }

    /**
     * Render a single quality gate editor
     */
    renderQualityGateEditor(gate, index) {
        const actions = ['halt', 'skip_phase', 'warn'];

        return `
            <div class="quality-gate-editor" data-gate-index="${index}">
                <div class="quality-gate-header">
                    <span class="quality-gate-number">Gate #${index + 1}</span>
                    <button
                        type="button"
                        class="btn btn-sm btn-icon btn-danger"
                        onclick="ecosystemManager.autoSteerManager.removeQualityGate(${index})"
                        title="Remove quality gate">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>

                <div class="form-group">
                    <label>Name *</label>
                    <input
                        type="text"
                        class="form-control gate-name"
                        data-gate-index="${index}"
                        value="${this.escapeHtml(gate.name || '')}"
                        placeholder="e.g., build_health"
                        required>
                </div>

                <div class="form-group">
                    <label>Failure Action *</label>
                    <select class="form-control gate-action" data-gate-index="${index}">
                        ${actions.map(action => `
                            <option value="${action}" ${gate.failure_action === action ? 'selected' : ''}>
                                ${this.formatActionName(action)}
                            </option>
                        `).join('')}
                    </select>
                </div>

                <div class="form-group">
                    <label>Message</label>
                    <input
                        type="text"
                        class="form-control gate-message"
                        data-gate-index="${index}"
                        value="${this.escapeHtml(gate.message || '')}"
                        placeholder="Message to display when gate fails">
                </div>

                <div class="form-group">
                    <label>Condition</label>
                    <button
                        type="button"
                        class="btn btn-sm btn-secondary mb-2"
                        onclick="ecosystemManager.autoSteerManager.openQualityGateConditionBuilder(${index})">
                        <i class="fas fa-code-branch"></i> Configure Condition
                    </button>
                    <div class="gate-condition-preview" id="gate-${index}-condition-preview">
                        ${gate.condition ? this.renderConditionsPreview([gate.condition]) : '<p class="text-muted">No condition set</p>'}
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Render a preview of conditions
     */
    renderConditionsPreview(conditions) {
        if (!conditions || conditions.length === 0) {
            return '<p class="text-muted">No conditions set</p>';
        }

        return `<div class="conditions-preview">${conditions.map(c => this.renderConditionPreview(c)).join('')}</div>`;
    }

    /**
     * Render a single condition preview (recursive for compound conditions)
     */
    renderConditionPreview(condition, level = 0) {
        const indent = level * 20;

        if (condition.type === 'simple') {
            return `
                <div class="condition-preview-item" style="margin-left: ${indent}px;">
                    <code>${this.escapeHtml(condition.metric)} ${this.escapeHtml(condition.compare_operator)} ${condition.value}</code>
                </div>
            `;
        } else if (condition.type === 'compound') {
            const operator = condition.operator === 'AND' ? 'AND' : 'OR';
            const subConditions = (condition.conditions || [])
                .map(c => this.renderConditionPreview(c, level + 1))
                .join(`<div class="condition-operator" style="margin-left: ${indent + 10}px;"><strong>${operator}</strong></div>`);

            return `
                <div class="condition-group" style="margin-left: ${indent}px;">
                    <div class="condition-group-label">(</div>
                    ${subConditions}
                    <div class="condition-group-label">)</div>
                </div>
            `;
        }

        return '';
    }

    /**
     * Open the condition builder dialog
     */
    openConditionBuilder(phaseIndex) {
        logger.debug(`Opening condition builder for phase ${phaseIndex}`);

        const phase = this.currentEditingProfile.phases[phaseIndex];
        if (!phase) {
            this.showToast('Phase not found', 'error');
            return;
        }

        this.conditionBuilderState = {
            type: 'phase',
            index: phaseIndex,
            conditions: JSON.parse(JSON.stringify(phase.stop_conditions || []))
        };

        this.renderConditionBuilder();
        this.showModal('condition-builder-modal');
    }

    /**
     * Open the quality gate condition builder
     */
    openQualityGateConditionBuilder(gateIndex) {
        logger.debug(`Opening condition builder for quality gate ${gateIndex}`);

        const gate = this.currentEditingProfile.quality_gates[gateIndex];
        if (!gate) {
            this.showToast('Quality gate not found', 'error');
            return;
        }

        this.conditionBuilderState = {
            type: 'gate',
            index: gateIndex,
            conditions: gate.condition ? [JSON.parse(JSON.stringify(gate.condition))] : []
        };

        this.renderConditionBuilder();
        this.showModal('condition-builder-modal');
    }

    /**
     * Render the condition builder tree
     */
    renderConditionBuilder() {
        const container = document.getElementById('condition-builder-tree');
        if (!container) return;

        const { conditions } = this.conditionBuilderState;

        if (!conditions || conditions.length === 0) {
            container.innerHTML = `
                <div class="condition-builder-empty">
                    <i class="fas fa-code-branch fa-3x"></i>
                    <p>No conditions yet</p>
                    <p class="text-muted">Add a condition or group to get started</p>
                </div>
            `;
            return;
        }

        container.innerHTML = `
            <div class="condition-tree-root">
                ${conditions.map((condition, index) =>
                    this.renderConditionNode(condition, [index])
                ).join('')}
            </div>
        `;
    }

    /**
     * Render a single condition node (recursive for nested conditions)
     * @param {Object} condition - The condition to render
     * @param {Array} path - Path to this condition in the tree (array of indices)
     * @param {Number} depth - Nesting depth for styling
     */
    renderConditionNode(condition, path, depth = 0) {
        const pathStr = JSON.stringify(path);
        const indent = depth * 24;

        if (condition.type === 'simple') {
            return this.renderSimpleCondition(condition, path, indent);
        } else if (condition.type === 'compound') {
            return this.renderCompoundCondition(condition, path, indent, depth);
        }

        return '';
    }

    /**
     * Render a simple condition
     */
    renderSimpleCondition(condition, path, indent) {
        const pathStr = this.escapeHtml(JSON.stringify(path));
        const metrics = this.getAvailableMetrics();
        const operators = ['>', '<', '>=', '<=', '==', '!='];

        return `
            <div class="condition-node condition-simple" style="margin-left: ${indent}px;" data-path="${pathStr}">
                <div class="condition-controls">
                    <button
                        type="button"
                        class="btn-icon btn-danger btn-sm"
                        onclick="ecosystemManager.autoSteerManager?.removeConditionNode(${pathStr})"
                        title="Remove condition">
                        <i class="fas fa-times"></i>
                    </button>
                </div>

                <div class="condition-editor-simple">
                    <select
                        class="form-control form-control-sm condition-metric"
                        onchange="ecosystemManager.autoSteerManager?.updateConditionField(${pathStr}, 'metric', this.value)">
                        <option value="">Select metric...</option>
                        ${metrics.map(m => `
                            <option value="${m.value}" ${condition.metric === m.value ? 'selected' : ''}>
                                ${this.escapeHtml(m.label)}
                            </option>
                        `).join('')}
                    </select>

                    <select
                        class="form-control form-control-sm condition-operator"
                        onchange="ecosystemManager.autoSteerManager?.updateConditionField(${pathStr}, 'compare_operator', this.value)">
                        ${operators.map(op => `
                            <option value="${op}" ${condition.compare_operator === op ? 'selected' : ''}>
                                ${op}
                            </option>
                        `).join('')}
                    </select>

                    <input
                        type="number"
                        class="form-control form-control-sm condition-value"
                        value="${condition.value || 0}"
                        step="any"
                        onchange="ecosystemManager.autoSteerManager?.updateConditionField(${pathStr}, 'value', parseFloat(this.value))"
                        placeholder="Value">
                </div>
            </div>
        `;
    }

    /**
     * Render a compound condition (group)
     */
    renderCompoundCondition(condition, path, indent, depth) {
        const pathStr = this.escapeHtml(JSON.stringify(path));
        const subConditions = condition.conditions || [];

        return `
            <div class="condition-node condition-compound" style="margin-left: ${indent}px;" data-path="${pathStr}">
                <div class="condition-compound-header">
                    <div class="condition-compound-controls">
                        <div class="condition-operator-toggle">
                            <button
                                type="button"
                                class="btn btn-sm ${condition.operator === 'AND' ? 'btn-primary' : 'btn-secondary'}"
                                onclick="ecosystemManager.autoSteerManager?.updateConditionField(${pathStr}, 'operator', 'AND')">
                                AND
                            </button>
                            <button
                                type="button"
                                class="btn btn-sm ${condition.operator === 'OR' ? 'btn-primary' : 'btn-secondary'}"
                                onclick="ecosystemManager.autoSteerManager?.updateConditionField(${pathStr}, 'operator', 'OR')">
                                OR
                            </button>
                        </div>

                        <button
                            type="button"
                            class="btn-icon btn-danger btn-sm"
                            onclick="ecosystemManager.autoSteerManager?.removeConditionNode(${pathStr})"
                            title="Remove group">
                            <i class="fas fa-times"></i>
                        </button>
                    </div>
                </div>

                <div class="condition-compound-body">
                    ${subConditions.length === 0 ? `
                        <div class="condition-compound-empty">
                            <p class="text-muted">Empty group - add conditions below</p>
                        </div>
                    ` : `
                        ${subConditions.map((subCondition, index) =>
                            this.renderConditionNode(subCondition, [...path, index], depth + 1)
                        ).join('')}
                    `}

                    <div class="condition-compound-add" style="margin-left: ${(depth + 1) * 24}px;">
                        <button
                            type="button"
                            class="btn btn-sm btn-secondary"
                            onclick="ecosystemManager.autoSteerManager?.addConditionToPath(${pathStr}, 'simple')">
                            <i class="fas fa-plus"></i> Add Condition
                        </button>
                        <button
                            type="button"
                            class="btn btn-sm btn-secondary"
                            onclick="ecosystemManager.autoSteerManager?.addConditionToPath(${pathStr}, 'compound')">
                            <i class="fas fa-folder-plus"></i> Add Group
                        </button>
                    </div>
                </div>
            </div>
        `;
    }

    /**
     * Get available metrics for dropdown
     */
    getAvailableMetrics() {
        return [
            // Universal metrics
            { value: 'loops', label: 'Loops (iterations)' },
            { value: 'build_status', label: 'Build Status (0=fail, 1=pass)' },
            { value: 'operational_targets_total', label: 'Operational Targets - Total' },
            { value: 'operational_targets_passing', label: 'Operational Targets - Passing' },
            { value: 'operational_targets_percentage', label: 'Operational Targets - Percentage' },

            // UX metrics
            { value: 'accessibility_score', label: 'UX: Accessibility Score' },
            { value: 'ui_test_coverage', label: 'UX: UI Test Coverage' },
            { value: 'responsive_breakpoints', label: 'UX: Responsive Breakpoints' },
            { value: 'user_flows_implemented', label: 'UX: User Flows Implemented' },
            { value: 'loading_states_count', label: 'UX: Loading States Count' },
            { value: 'error_handling_coverage', label: 'UX: Error Handling Coverage' },

            // Refactor metrics
            { value: 'cyclomatic_complexity_avg', label: 'Refactor: Avg Cyclomatic Complexity' },
            { value: 'duplication_percentage', label: 'Refactor: Code Duplication %' },
            { value: 'standards_violations', label: 'Refactor: Standards Violations' },
            { value: 'tidiness_score', label: 'Refactor: Tidiness Score' },
            { value: 'tech_debt_items', label: 'Refactor: Tech Debt Items' },

            // Test metrics
            { value: 'unit_test_coverage', label: 'Test: Unit Test Coverage' },
            { value: 'integration_test_coverage', label: 'Test: Integration Test Coverage' },
            { value: 'ui_test_coverage', label: 'Test: UI Test Coverage' },
            { value: 'edge_cases_covered', label: 'Test: Edge Cases Covered' },
            { value: 'flaky_tests', label: 'Test: Flaky Tests' },
            { value: 'test_quality_score', label: 'Test: Quality Score' },

            // Performance metrics
            { value: 'bundle_size_kb', label: 'Performance: Bundle Size (KB)' },
            { value: 'initial_load_time_ms', label: 'Performance: Initial Load Time (ms)' },
            { value: 'lcp_ms', label: 'Performance: Largest Contentful Paint (ms)' },
            { value: 'fid_ms', label: 'Performance: First Input Delay (ms)' },
            { value: 'cls_score', label: 'Performance: Cumulative Layout Shift' },

            // Security metrics
            { value: 'vulnerability_count', label: 'Security: Vulnerability Count' },
            { value: 'input_validation_coverage', label: 'Security: Input Validation Coverage' },
            { value: 'auth_implementation_score', label: 'Security: Auth Implementation Score' },
            { value: 'security_scan_score', label: 'Security: Scan Score' }
        ];
    }

    /**
     * Add a condition to the builder at the root level
     */
    addConditionToBuilder(type) {
        if (!this.conditionBuilderState) return;

        const newCondition = type === 'simple' ? {
            type: 'simple',
            metric: 'loops',
            compare_operator: '>',
            value: 10
        } : {
            type: 'compound',
            operator: 'AND',
            conditions: []
        };

        if (!this.conditionBuilderState.conditions) {
            this.conditionBuilderState.conditions = [];
        }

        this.conditionBuilderState.conditions.push(newCondition);
        this.renderConditionBuilder();
    }

    /**
     * Add a condition to a specific path (for nested groups)
     */
    addConditionToPath(pathStr, type) {
        try {
            const path = JSON.parse(pathStr);
            const condition = this.getConditionAtPath(path);

            if (!condition || condition.type !== 'compound') return;

            const newCondition = type === 'simple' ? {
                type: 'simple',
                metric: 'loops',
                compare_operator: '>',
                value: 10
            } : {
                type: 'compound',
                operator: 'AND',
                conditions: []
            };

            if (!condition.conditions) {
                condition.conditions = [];
            }

            condition.conditions.push(newCondition);
            this.renderConditionBuilder();
        } catch (error) {
            logger.error('Failed to add condition to path:', error);
        }
    }

    /**
     * Remove a condition node
     */
    removeConditionNode(pathStr) {
        try {
            const path = JSON.parse(pathStr);

            if (path.length === 1) {
                // Root level - remove from conditions array
                this.conditionBuilderState.conditions.splice(path[0], 1);
            } else {
                // Nested - navigate to parent and remove
                const parentPath = path.slice(0, -1);
                const parent = this.getConditionAtPath(parentPath);

                if (parent && parent.conditions) {
                    const index = path[path.length - 1];
                    parent.conditions.splice(index, 1);
                }
            }

            this.renderConditionBuilder();
        } catch (error) {
            logger.error('Failed to remove condition:', error);
        }
    }

    /**
     * Update a field in a condition
     */
    updateConditionField(pathStr, field, value) {
        try {
            const path = JSON.parse(pathStr);
            const condition = this.getConditionAtPath(path);

            if (!condition) return;

            condition[field] = value;

            // Re-render to show updated state
            this.renderConditionBuilder();
        } catch (error) {
            logger.error('Failed to update condition field:', error);
        }
    }

    /**
     * Get condition at a specific path in the tree
     */
    getConditionAtPath(path) {
        let current = { conditions: this.conditionBuilderState.conditions };

        for (let i = 0; i < path.length; i++) {
            if (!current.conditions || !current.conditions[path[i]]) {
                return null;
            }
            current = current.conditions[path[i]];
        }

        return current;
    }

    /**
     * Save conditions from builder back to the profile
     */
    saveConditionsFromBuilder() {
        if (!this.conditionBuilderState) return;

        const { type, index, conditions } = this.conditionBuilderState;

        // Validate that we have at least one condition
        if (!conditions || conditions.length === 0) {
            if (!confirm('No conditions configured. The phase will only stop when max iterations is reached. Continue?')) {
                return;
            }
        }

        try {
            if (type === 'phase') {
                // Save to phase
                this.currentEditingProfile.phases[index].stop_conditions = conditions;

                // Update preview
                const preview = document.getElementById(`phase-${index}-conditions-preview`);
                if (preview) {
                    preview.innerHTML = this.renderConditionsPreview(conditions);
                }
            } else if (type === 'gate') {
                // Save to quality gate (single condition)
                this.currentEditingProfile.quality_gates[index].condition =
                    conditions.length > 0 ? conditions[0] : null;

                // Update preview
                const preview = document.getElementById(`gate-${index}-condition-preview`);
                if (preview) {
                    preview.innerHTML = conditions.length > 0 ?
                        this.renderConditionsPreview([conditions[0]]) :
                        '<p class="text-muted">No condition set</p>';
                }
            }

            this.closeConditionBuilder(true);
            this.showToast('Conditions updated successfully', 'success');
        } catch (error) {
            logger.error('Failed to save conditions:', error);
            this.showToast('Failed to save conditions: ' + error.message, 'error');
        }
    }

    /**
     * Close the condition builder
     */
    closeConditionBuilder(saved) {
        this.conditionBuilderState = null;
        this.closeModal('condition-builder-modal');
    }

    /**
     * Add a new phase to the current editing profile
     */
    addPhase() {
        this.currentEditingProfile.phases.push({
            id: `phase-${Date.now()}`,
            mode: 'progress',
            max_iterations: 10,
            stop_conditions: [],
            description: ''
        });

        const phasesList = document.getElementById('phases-list');
        if (phasesList) {
            phasesList.innerHTML = this.renderPhasesList(this.currentEditingProfile.phases);
        }
    }

    /**
     * Remove a phase from the current editing profile
     */
    removePhase(index) {
        this.currentEditingProfile.phases.splice(index, 1);

        const phasesList = document.getElementById('phases-list');
        if (phasesList) {
            phasesList.innerHTML = this.renderPhasesList(this.currentEditingProfile.phases);
        }
    }

    /**
     * Add a new quality gate to the current editing profile
     */
    addQualityGate() {
        this.currentEditingProfile.quality_gates.push({
            name: '',
            failure_action: 'halt',
            message: '',
            condition: null
        });

        const gatesList = document.getElementById('quality-gates-list');
        if (gatesList) {
            gatesList.innerHTML = this.renderQualityGatesList(this.currentEditingProfile.quality_gates);
        }
    }

    /**
     * Remove a quality gate from the current editing profile
     */
    removeQualityGate(index) {
        this.currentEditingProfile.quality_gates.splice(index, 1);

        const gatesList = document.getElementById('quality-gates-list');
        if (gatesList) {
            gatesList.innerHTML = this.renderQualityGatesList(this.currentEditingProfile.quality_gates);
        }
    }

    /**
     * Save the current editing profile
     */
    async saveProfile() {
        try {
            // Collect data from form
            const name = document.getElementById('profile-name')?.value?.trim();
            const description = document.getElementById('profile-description')?.value?.trim();
            const tagsInput = document.getElementById('profile-tags')?.value?.trim();
            const tags = tagsInput ? tagsInput.split(',').map(t => t.trim()).filter(t => t) : [];

            if (!name) {
                this.showToast('Profile name is required', 'error');
                return;
            }

            // Collect phases data from form
            const phases = this.collectPhasesData();

            if (phases.length === 0) {
                this.showToast('At least one phase is required', 'error');
                return;
            }

            // Collect quality gates data
            const qualityGates = this.collectQualityGatesData();

            const profileData = {
                name,
                description,
                tags,
                phases,
                quality_gates: qualityGates
            };

            if (this.currentEditingProfile.id) {
                await this.updateProfile(this.currentEditingProfile.id, profileData);
            } else {
                await this.createProfile(profileData);
            }

            this.closeModal('autosteer-profile-editor-modal');
        } catch (error) {
            logger.error('Failed to save profile:', error);
            // Error already shown in createProfile/updateProfile
        }
    }

    /**
     * Collect phases data from the form
     */
    collectPhasesData() {
        const phases = [];
        const phaseEditors = document.querySelectorAll('.phase-editor');

        phaseEditors.forEach((editor, index) => {
            const mode = editor.querySelector('.phase-mode')?.value;
            const maxIterations = parseInt(editor.querySelector('.phase-max-iterations')?.value) || 10;
            const description = editor.querySelector('.phase-description')?.value?.trim();
            const stopConditions = this.currentEditingProfile.phases[index]?.stop_conditions || [];

            if (mode) {
                phases.push({
                    id: this.currentEditingProfile.phases[index]?.id || `phase-${Date.now()}-${index}`,
                    mode,
                    max_iterations: maxIterations,
                    stop_conditions: stopConditions,
                    description: description || undefined
                });
            }
        });

        return phases;
    }

    /**
     * Collect quality gates data from the form
     */
    collectQualityGatesData() {
        const gates = [];
        const gateEditors = document.querySelectorAll('.quality-gate-editor');

        gateEditors.forEach((editor, index) => {
            const name = editor.querySelector('.gate-name')?.value?.trim();
            const action = editor.querySelector('.gate-action')?.value;
            const message = editor.querySelector('.gate-message')?.value?.trim();
            const condition = this.currentEditingProfile.quality_gates[index]?.condition;

            if (name && action) {
                gates.push({
                    name,
                    failure_action: action,
                    message: message || '',
                    condition: condition || { type: 'simple', metric: 'build_status', compare_operator: '==', value: 1 }
                });
            }
        });

        return gates;
    }

    /**
     * Confirm profile deletion with user
     */
    confirmDeleteProfile(id) {
        const profile = this.profiles.find(p => p.id === id);
        if (!profile) return;

        if (confirm(`Are you sure you want to delete the profile "${profile.name}"? This action cannot be undone.`)) {
            this.deleteProfile(id);
        }
    }

    /**
     * Get icon for a mode
     */
    getModeIcon(mode) {
        const icons = {
            progress: '<i class="fas fa-tasks"></i>',
            ux: '<i class="fas fa-palette"></i>',
            refactor: '<i class="fas fa-code"></i>',
            test: '<i class="fas fa-vial"></i>',
            explore: '<i class="fas fa-flask"></i>',
            polish: '<i class="fas fa-gem"></i>',
            integration: '<i class="fas fa-plug"></i>',
            performance: '<i class="fas fa-tachometer-alt"></i>',
            security: '<i class="fas fa-shield-alt"></i>'
        };
        return icons[mode] || '<i class="fas fa-circle"></i>';
    }

    /**
     * Format mode name for display
     */
    formatModeName(mode) {
        const names = {
            progress: 'Progress',
            ux: 'UX',
            refactor: 'Refactor',
            test: 'Test',
            explore: 'Explore',
            polish: 'Polish',
            integration: 'Integration',
            performance: 'Performance',
            security: 'Security'
        };
        return names[mode] || mode;
    }

    /**
     * Format action name for display
     */
    formatActionName(action) {
        const names = {
            halt: 'Halt Execution',
            skip_phase: 'Skip Phase',
            warn: 'Warn Only'
        };
        return names[action] || action;
    }

    /**
     * Show a modal
     */
    showModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.style.display = 'flex';
        }
    }

    /**
     * Close a modal
     */
    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.style.display = 'none';
        }
    }

    /**
     * Escape HTML to prevent XSS
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
