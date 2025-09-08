// Vrooli Orchestrator Dashboard Application
class OrchestratorDashboard {
    constructor() {
        this.apiBase = '/api/v1';
        this.profiles = [];
        this.activeProfile = null;
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.loadData();
        
        // Refresh data every 30 seconds
        setInterval(() => this.loadData(), 30000);
    }

    setupEventListeners() {
        // Search functionality
        document.getElementById('searchBox').addEventListener('input', (e) => {
            this.filterProfiles(e.target.value);
        });

        // Control buttons
        document.getElementById('refreshBtn').addEventListener('click', () => this.loadData());
        document.getElementById('createBtn').addEventListener('click', () => this.showCreateModal());
        document.getElementById('deactivateBtn').addEventListener('click', () => this.deactivateProfile());

        // Modal functionality
        document.getElementById('cancelBtn').addEventListener('click', () => this.hideModal());
        document.getElementById('profileForm').addEventListener('submit', (e) => this.saveProfile(e));
        
        // Close modal when clicking outside
        document.getElementById('profileModal').addEventListener('click', (e) => {
            if (e.target.id === 'profileModal') {
                this.hideModal();
            }
        });
    }

    async loadData() {
        try {
            await Promise.all([
                this.loadStatus(),
                this.loadProfiles()
            ]);
        } catch (error) {
            console.error('Failed to load data:', error);
            this.showMessage('Failed to load dashboard data. Check if the orchestrator is running.', 'error');
        }
    }

    async loadStatus() {
        try {
            const response = await fetch(`${this.apiBase}/status`);
            const status = await response.json();
            
            this.updateStatusBar(status);
            this.activeProfile = status.active_profile;
            
        } catch (error) {
            document.getElementById('apiStatus').textContent = 'Error';
            document.getElementById('apiStatus').style.color = '#ff4444';
            throw error;
        }
    }

    async loadProfiles() {
        try {
            const response = await fetch(`${this.apiBase}/profiles`);
            const data = await response.json();
            
            this.profiles = data.profiles || [];
            this.renderProfiles();
            
        } catch (error) {
            document.getElementById('profilesContainer').innerHTML = 
                '<div class="error">Failed to load profiles. Make sure the orchestrator API is running.</div>';
            throw error;
        }
    }

    updateStatusBar(status) {
        // Update active profile
        const activeProfileEl = document.getElementById('activeProfile');
        if (status.active_profile) {
            activeProfileEl.textContent = status.active_profile.display_name || status.active_profile.name;
            activeProfileEl.className = 'status-value active-profile';
        } else {
            activeProfileEl.textContent = 'None';
            activeProfileEl.className = 'status-value';
        }

        // Update counters
        document.getElementById('resourceCount').textContent = status.resource_count || 0;
        document.getElementById('scenarioCount').textContent = status.scenario_count || 0;

        // Update API status
        const apiStatusEl = document.getElementById('apiStatus');
        if (status.status === 'healthy') {
            apiStatusEl.textContent = 'Healthy';
            apiStatusEl.style.color = '#00ff88';
        } else {
            apiStatusEl.textContent = 'Error';
            apiStatusEl.style.color = '#ff4444';
        }
    }

    renderProfiles() {
        const container = document.getElementById('profilesContainer');
        
        if (this.profiles.length === 0) {
            container.innerHTML = `
                <div class="loading">
                    No profiles found. Create your first profile to get started!
                </div>
            `;
            return;
        }

        const profilesHtml = this.profiles.map(profile => this.renderProfileCard(profile)).join('');
        
        container.innerHTML = `
            <div class="profiles-grid">
                ${profilesHtml}
            </div>
        `;

        // Add event listeners to action buttons
        this.attachProfileEventListeners();
    }

    renderProfileCard(profile) {
        const isActive = this.activeProfile && this.activeProfile.name === profile.name;
        const statusClass = isActive ? 'active' : 'inactive';
        const statusText = isActive ? 'Active' : 'Inactive';

        return `
            <div class="profile-card ${isActive ? 'active' : ''}" data-profile="${profile.name}">
                <div class="profile-header">
                    <div>
                        <div class="profile-title">${profile.display_name || profile.name}</div>
                        <div class="profile-name">${profile.name}</div>
                    </div>
                    <div class="profile-status status-${statusClass}">${statusText}</div>
                </div>
                
                <div class="profile-description">
                    ${profile.description || 'No description provided'}
                </div>
                
                <div class="profile-details">
                    <div class="detail-item">
                        <div class="detail-label">Resources</div>
                        <div class="detail-value">${(profile.resources || []).length}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">Scenarios</div>
                        <div class="detail-value">${(profile.scenarios || []).length}</div>
                    </div>
                    <div class="detail-item">
                        <div class="detail-label">URLs</div>
                        <div class="detail-value">${(profile.auto_browser || []).length}</div>
                    </div>
                </div>
                
                <div class="profile-actions">
                    ${!isActive ? `
                        <button class="btn btn-success btn-small activate-btn" data-profile="${profile.name}">
                            ▶️ Activate
                        </button>
                    ` : `
                        <button class="btn btn-danger btn-small deactivate-btn" data-profile="${profile.name}">
                            ⏹️ Deactivate
                        </button>
                    `}
                    <button class="btn btn-secondary btn-small edit-btn" data-profile="${profile.name}">
                        ✏️ Edit
                    </button>
                    <button class="btn btn-danger btn-small delete-btn" data-profile="${profile.name}">
                        🗑️ Delete
                    </button>
                </div>
                
                <div class="profile-metadata" style="margin-top: 15px; font-size: 0.8em; color: #666;">
                    ${profile.metadata ? `
                        <div><strong>Audience:</strong> ${profile.metadata.target_audience || 'General'}</div>
                        <div><strong>Footprint:</strong> ${profile.metadata.resource_footprint || 'Medium'}</div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    attachProfileEventListeners() {
        // Activate buttons
        document.querySelectorAll('.activate-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const profileName = e.target.dataset.profile;
                this.activateProfile(profileName);
            });
        });

        // Deactivate buttons
        document.querySelectorAll('.deactivate-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                this.deactivateProfile();
            });
        });

        // Edit buttons
        document.querySelectorAll('.edit-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const profileName = e.target.dataset.profile;
                this.editProfile(profileName);
            });
        });

        // Delete buttons
        document.querySelectorAll('.delete-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const profileName = e.target.dataset.profile;
                this.deleteProfile(profileName);
            });
        });
    }

    filterProfiles(searchTerm) {
        const cards = document.querySelectorAll('.profile-card');
        const term = searchTerm.toLowerCase();

        cards.forEach(card => {
            const profileName = card.dataset.profile;
            const profile = this.profiles.find(p => p.name === profileName);
            
            if (!profile) return;

            const searchableText = [
                profile.name,
                profile.display_name,
                profile.description,
                ...(profile.resources || []),
                ...(profile.scenarios || []),
                profile.metadata?.target_audience,
                profile.metadata?.use_case
            ].join(' ').toLowerCase();

            if (searchableText.includes(term)) {
                card.style.display = 'block';
            } else {
                card.style.display = 'none';
            }
        });
    }

    async activateProfile(profileName) {
        if (!confirm(`Activate profile "${profileName}"? This will start associated resources and scenarios.`)) {
            return;
        }

        try {
            this.showMessage(`Activating profile "${profileName}"... This may take 30-60 seconds.`, 'success');
            
            const response = await fetch(`${this.apiBase}/profiles/${profileName}/activate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ force: false })
            });

            const result = await response.json();

            if (response.ok && result.success) {
                this.showMessage(`✅ Profile "${profileName}" activated successfully!`, 'success');
                this.loadData(); // Refresh status
            } else {
                throw new Error(result.error || 'Activation failed');
            }

        } catch (error) {
            console.error('Activation error:', error);
            this.showMessage(`❌ Failed to activate profile: ${error.message}`, 'error');
        }
    }

    async deactivateProfile() {
        if (!this.activeProfile) {
            this.showMessage('No active profile to deactivate', 'error');
            return;
        }

        if (!confirm('Deactivate the current profile? This will stop running resources and scenarios.')) {
            return;
        }

        try {
            this.showMessage('Deactivating current profile...', 'success');
            
            const response = await fetch(`${this.apiBase}/profiles/current/deactivate`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' }
            });

            const result = await response.json();

            if (response.ok) {
                this.showMessage('✅ Profile deactivated successfully!', 'success');
                this.loadData(); // Refresh status
            } else {
                throw new Error(result.error || 'Deactivation failed');
            }

        } catch (error) {
            console.error('Deactivation error:', error);
            this.showMessage(`❌ Failed to deactivate profile: ${error.message}`, 'error');
        }
    }

    showCreateModal() {
        document.getElementById('modalTitle').textContent = 'Create New Profile';
        document.getElementById('profileForm').reset();
        document.getElementById('profileForm').dataset.mode = 'create';
        document.getElementById('profileModal').style.display = 'block';
    }

    editProfile(profileName) {
        const profile = this.profiles.find(p => p.name === profileName);
        if (!profile) return;

        document.getElementById('modalTitle').textContent = 'Edit Profile';
        document.getElementById('profileName').value = profile.name;
        document.getElementById('profileDisplayName').value = profile.display_name || '';
        document.getElementById('profileDescription').value = profile.description || '';
        document.getElementById('profileResources').value = (profile.resources || []).join(',');
        document.getElementById('profileScenarios').value = (profile.scenarios || []).join(',');
        document.getElementById('profileUrls').value = (profile.auto_browser || []).join(',');

        // Disable name editing for existing profiles
        document.getElementById('profileName').disabled = true;
        
        document.getElementById('profileForm').dataset.mode = 'edit';
        document.getElementById('profileForm').dataset.originalName = profileName;
        document.getElementById('profileModal').style.display = 'block';
    }

    async deleteProfile(profileName) {
        if (!confirm(`Delete profile "${profileName}"? This action cannot be undone.`)) {
            return;
        }

        try {
            const response = await fetch(`${this.apiBase}/profiles/${profileName}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                this.showMessage(`✅ Profile "${profileName}" deleted successfully!`, 'success');
                this.loadProfiles(); // Refresh profiles
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Delete failed');
            }

        } catch (error) {
            console.error('Delete error:', error);
            this.showMessage(`❌ Failed to delete profile: ${error.message}`, 'error');
        }
    }

    hideModal() {
        document.getElementById('profileModal').style.display = 'none';
        document.getElementById('profileName').disabled = false; // Re-enable for next use
    }

    async saveProfile(e) {
        e.preventDefault();

        const form = e.target;
        const mode = form.dataset.mode;
        
        const profileData = {
            name: document.getElementById('profileName').value,
            display_name: document.getElementById('profileDisplayName').value,
            description: document.getElementById('profileDescription').value,
            resources: this.parseCommaSeparated(document.getElementById('profileResources').value),
            scenarios: this.parseCommaSeparated(document.getElementById('profileScenarios').value),
            auto_browser: this.parseCommaSeparated(document.getElementById('profileUrls').value),
            metadata: {
                target_audience: 'custom',
                resource_footprint: 'medium',
                use_case: 'custom_profile'
            }
        };

        try {
            let response;
            
            if (mode === 'create') {
                response = await fetch(`${this.apiBase}/profiles`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(profileData)
                });
            } else {
                const originalName = form.dataset.originalName;
                response = await fetch(`${this.apiBase}/profiles/${originalName}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(profileData)
                });
            }

            if (response.ok) {
                const action = mode === 'create' ? 'created' : 'updated';
                this.showMessage(`✅ Profile "${profileData.name}" ${action} successfully!`, 'success');
                this.hideModal();
                this.loadProfiles(); // Refresh profiles
            } else {
                const error = await response.json();
                throw new Error(error.error || `${mode} failed`);
            }

        } catch (error) {
            console.error('Save error:', error);
            this.showMessage(`❌ Failed to save profile: ${error.message}`, 'error');
        }
    }

    parseCommaSeparated(value) {
        return value ? value.split(',').map(s => s.trim()).filter(s => s) : [];
    }

    showMessage(message, type = 'success') {
        const container = document.getElementById('messages');
        const messageEl = document.createElement('div');
        messageEl.className = type;
        messageEl.textContent = message;
        
        container.innerHTML = ''; // Clear previous messages
        container.appendChild(messageEl);

        // Auto-hide success messages after 5 seconds
        if (type === 'success') {
            setTimeout(() => {
                if (messageEl.parentNode) {
                    messageEl.remove();
                }
            }, 5000);
        }
    }
}

// Initialize dashboard when page loads
document.addEventListener('DOMContentLoaded', () => {
    new OrchestratorDashboard();
});