// Data Backup Manager - Frontend Application
const API_BASE = `http://localhost:${window.API_PORT || 20010}/api/v1`;

class BackupManager {
    constructor() {
        this.currentTab = 'dashboard';
        this.backups = [];
        this.schedules = [];
        this.init();
    }

    async init() {
        this.setupTabNavigation();
        this.loadDashboard();
        this.startPolling();
    }

    setupTabNavigation() {
        document.querySelectorAll('.tab-button').forEach(button => {
            button.addEventListener('click', () => {
                const tabId = button.dataset.tab;
                this.switchTab(tabId);
            });
        });
    }

    switchTab(tabId) {
        // Update button states
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[data-tab="${tabId}"]`).classList.add('active');

        // Update content visibility
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(tabId).classList.add('active');

        this.currentTab = tabId;

        // Load tab-specific data
        switch(tabId) {
            case 'dashboard':
                this.loadDashboard();
                break;
            case 'backups':
                this.loadBackups();
                break;
            case 'schedules':
                this.loadSchedules();
                break;
            case 'compliance':
                this.loadCompliance();
                break;
            case 'settings':
                this.loadSettings();
                break;
        }
    }

    async loadDashboard() {
        try {
            // Load system status
            const statusResponse = await fetch(`${API_BASE}/backup/status`);
            const status = await statusResponse.json();
            
            this.updateSystemStatus(status);
            this.updateMetrics(status.metrics);
            this.updateActivityList(status.recent_activity);

        } catch (error) {
            console.error('Failed to load dashboard:', error);
            this.showError('Failed to load dashboard data');
        }
    }

    updateSystemStatus(status) {
        const statusEl = document.getElementById('system-status');
        const statusDot = statusEl.querySelector('.status-dot');
        const statusText = statusEl.querySelector('.status-text');
        
        if (status.healthy) {
            statusEl.classList.add('online');
            statusEl.classList.remove('error');
            statusText.textContent = 'System Online';
        } else {
            statusEl.classList.add('error');
            statusEl.classList.remove('online');
            statusText.textContent = 'System Error';
        }

        // Update last backup time
        if (status.last_backup) {
            document.getElementById('last-backup').textContent = 
                `Last backup: ${this.formatTimeAgo(status.last_backup)}`;
        }
    }

    updateMetrics(metrics) {
        // Success Rate
        const successRate = metrics?.success_rate || 0;
        document.getElementById('success-rate').textContent = Math.round(successRate);
        
        const successTrend = document.getElementById('success-trend');
        if (successRate >= 95) {
            successTrend.textContent = '↑ Excellent';
            successTrend.style.color = 'var(--success-color)';
        } else if (successRate >= 80) {
            successTrend.textContent = '→ Good';
            successTrend.style.color = 'var(--warning-color)';
        } else {
            successTrend.textContent = '↓ Needs attention';
            successTrend.style.color = 'var(--danger-color)';
        }

        // Storage Usage
        const storageUsed = metrics?.storage_used_gb || 0;
        const storageTotal = metrics?.storage_total_gb || 100;
        const storagePercent = (storageUsed / storageTotal) * 100;
        
        document.getElementById('storage-used').textContent = storageUsed.toFixed(1);
        document.getElementById('storage-progress').style.width = `${storagePercent}%`;

        // Compliance Score
        const complianceScore = metrics?.compliance_score || 0;
        document.getElementById('compliance-score').textContent = Math.round(complianceScore);
        
        const complianceStatus = document.getElementById('compliance-status');
        if (complianceScore >= 90) {
            complianceStatus.textContent = 'Compliant';
            complianceStatus.className = 'compliance-indicator good';
        } else if (complianceScore >= 70) {
            complianceStatus.textContent = 'Partially Compliant';
            complianceStatus.className = 'compliance-indicator warning';
        } else {
            complianceStatus.textContent = 'Non-Compliant';
            complianceStatus.className = 'compliance-indicator critical';
        }

        // Protected Items
        const protectedCount = metrics?.protected_items || 0;
        document.getElementById('protected-count').textContent = protectedCount;
        
        // Item breakdown
        const breakdown = document.getElementById('item-breakdown');
        if (metrics?.item_breakdown) {
            breakdown.innerHTML = Object.entries(metrics.item_breakdown)
                .map(([type, count]) => `<div>${type}: ${count}</div>`)
                .join('');
        }
    }

    updateActivityList(activities) {
        const listEl = document.getElementById('activity-list');
        
        if (!activities || activities.length === 0) {
            listEl.innerHTML = '<div class="activity-placeholder">No recent activity</div>';
            return;
        }

        listEl.innerHTML = activities.slice(0, 10).map(activity => {
            const iconClass = this.getActivityIconClass(activity.type);
            const icon = this.getActivityIcon(activity.type);
            
            return `
                <div class="activity-item">
                    <div class="activity-icon ${iconClass}">${icon}</div>
                    <div class="activity-details">
                        <div class="activity-title">${activity.title}</div>
                        <div class="activity-time">${this.formatTimeAgo(activity.timestamp)}</div>
                    </div>
                </div>
            `;
        }).join('');
    }

    getActivityIconClass(type) {
        switch(type) {
            case 'backup_success':
            case 'restore_success':
                return 'success';
            case 'backup_failed':
            case 'restore_failed':
                return 'error';
            default:
                return 'warning';
        }
    }

    getActivityIcon(type) {
        switch(type) {
            case 'backup_success':
                return '✓';
            case 'backup_failed':
                return '✗';
            case 'restore_success':
                return '♻';
            case 'restore_failed':
                return '⚠';
            case 'schedule_created':
                return '📅';
            case 'compliance_check':
                return '📋';
            default:
                return '•';
        }
    }

    async loadBackups() {
        try {
            const response = await fetch(`${API_BASE}/backup/list`);
            this.backups = await response.json();
            this.renderBackupsList();
        } catch (error) {
            console.error('Failed to load backups:', error);
            this.showError('Failed to load backups');
        }
    }

    renderBackupsList() {
        const listEl = document.getElementById('backups-list');
        
        if (this.backups.length === 0) {
            listEl.innerHTML = '<div class="backup-placeholder">No backups available</div>';
            return;
        }

        listEl.innerHTML = this.backups.map(backup => `
            <div class="backup-item">
                <div class="backup-name">${backup.name}</div>
                <div class="backup-size">${this.formatFileSize(backup.size)}</div>
                <div class="backup-date">${this.formatDate(backup.created_at)}</div>
                <div class="backup-status ${backup.verified ? 'verified' : 'pending'}">
                    ${backup.verified ? 'Verified' : 'Pending'}
                </div>
                <div class="backup-actions">
                    <button class="backup-action" onclick="backup.verifyBackup('${backup.id}')">
                        Verify
                    </button>
                    <button class="backup-action" onclick="backup.downloadBackup('${backup.id}')">
                        Download
                    </button>
                    <button class="backup-action" onclick="backup.restoreBackup('${backup.id}')">
                        Restore
                    </button>
                </div>
            </div>
        `).join('');
    }

    async loadSchedules() {
        try {
            const response = await fetch(`${API_BASE}/schedules`);
            this.schedules = await response.json();
            this.renderSchedulesList();
        } catch (error) {
            console.error('Failed to load schedules:', error);
            this.showError('Failed to load schedules');
        }
    }

    renderSchedulesList() {
        const listEl = document.getElementById('schedules-list');
        
        if (this.schedules.length === 0) {
            listEl.innerHTML = '<div class="schedule-placeholder">No schedules configured</div>';
            return;
        }

        listEl.innerHTML = this.schedules.map(schedule => `
            <div class="schedule-item">
                <div class="schedule-name">${schedule.name}</div>
                <div class="schedule-frequency">${this.formatCron(schedule.cron)}</div>
                <div class="schedule-next-run">${this.formatDate(schedule.next_run)}</div>
                <div class="schedule-toggle">
                    <input type="checkbox" id="schedule-${schedule.id}" 
                           ${schedule.enabled ? 'checked' : ''}
                           onchange="backup.toggleSchedule('${schedule.id}', this.checked)">
                    <label for="schedule-${schedule.id}"></label>
                </div>
                <div class="backup-actions">
                    <button class="backup-action" onclick="backup.editSchedule('${schedule.id}')">
                        Edit
                    </button>
                    <button class="backup-action" onclick="backup.deleteSchedule('${schedule.id}')">
                        Delete
                    </button>
                </div>
            </div>
        `).join('');
    }

    async loadCompliance() {
        try {
            const response = await fetch(`${API_BASE}/compliance/report`);
            const compliance = await response.json();
            this.renderComplianceReport(compliance);
        } catch (error) {
            console.error('Failed to load compliance:', error);
            this.showError('Failed to load compliance report');
        }
    }

    renderComplianceReport(compliance) {
        // Update compliance details
        const detailsEl = document.getElementById('compliance-details');
        detailsEl.innerHTML = `
            <div class="compliance-stat">
                <span class="compliance-stat-label">Total Resources</span>
                <span class="compliance-stat-value">${compliance.total_resources || 0}</span>
            </div>
            <div class="compliance-stat">
                <span class="compliance-stat-label">Compliant</span>
                <span class="compliance-stat-value">${compliance.compliant || 0}</span>
            </div>
            <div class="compliance-stat">
                <span class="compliance-stat-label">Non-Compliant</span>
                <span class="compliance-stat-value">${compliance.non_compliant || 0}</span>
            </div>
            <div class="compliance-stat">
                <span class="compliance-stat-label">Last Scan</span>
                <span class="compliance-stat-value">${this.formatTimeAgo(compliance.last_scan)}</span>
            </div>
        `;

        // Update issues list
        const issuesEl = document.getElementById('compliance-issues');
        if (compliance.issues && compliance.issues.length > 0) {
            issuesEl.innerHTML = compliance.issues.map(issue => `
                <div class="issue-item">
                    <div class="issue-severity ${issue.severity}">
                        ${issue.severity[0].toUpperCase()}
                    </div>
                    <div class="issue-details">
                        <div class="issue-title">${issue.title}</div>
                        <div class="issue-path">${issue.path}</div>
                    </div>
                    <button class="issue-action" onclick="backup.fixComplianceIssue('${issue.id}')">
                        Fix
                    </button>
                </div>
            `).join('');
        } else {
            issuesEl.innerHTML = '<div class="issues-placeholder">No compliance issues found</div>';
        }
    }

    async loadSettings() {
        try {
            const response = await fetch(`${API_BASE}/settings`);
            const settings = await response.json();
            this.applySettings(settings);
        } catch (error) {
            console.error('Failed to load settings:', error);
        }
    }

    applySettings(settings) {
        document.getElementById('backup-path').value = settings.backup_path || '/data/backups';
        document.getElementById('retention-days').value = settings.retention_days || 30;
        document.getElementById('compression-level').value = settings.compression_level || 'medium';
        document.getElementById('enable-encryption').checked = settings.enable_encryption !== false;
        document.getElementById('enable-agent').checked = settings.enable_agent !== false;
    }

    async createBackup() {
        try {
            const response = await fetch(`${API_BASE}/backup/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: 'full',
                    targets: ['postgres', 'files', 'scenarios', 'resources']
                })
            });

            if (response.ok) {
                this.showSuccess('Backup initiated successfully');
                this.loadDashboard();
            } else {
                throw new Error('Backup failed');
            }
        } catch (error) {
            console.error('Failed to create backup:', error);
            this.showError('Failed to create backup');
        }
    }

    async verifyBackups() {
        try {
            const response = await fetch(`${API_BASE}/backup/verify-all`, {
                method: 'POST'
            });

            if (response.ok) {
                this.showSuccess('Verification started');
                this.loadBackups();
            } else {
                throw new Error('Verification failed');
            }
        } catch (error) {
            console.error('Failed to verify backups:', error);
            this.showError('Failed to verify backups');
        }
    }

    async runComplianceCheck() {
        try {
            const response = await fetch(`${API_BASE}/compliance/scan`, {
                method: 'POST'
            });

            if (response.ok) {
                this.showSuccess('Compliance check started');
                this.loadCompliance();
            } else {
                throw new Error('Compliance check failed');
            }
        } catch (error) {
            console.error('Failed to run compliance check:', error);
            this.showError('Failed to run compliance check');
        }
    }

    showRestoreModal() {
        const modal = document.getElementById('restore-modal');
        modal.classList.add('active');
        this.loadRestoreOptions();
    }

    async loadRestoreOptions() {
        const selectEl = document.getElementById('restore-backup-select');
        selectEl.innerHTML = '<option>Loading backups...</option>';
        
        try {
            const response = await fetch(`${API_BASE}/backup/list?verified=true`);
            const backups = await response.json();
            
            selectEl.innerHTML = '<option>Select backup to restore...</option>' +
                backups.map(backup => 
                    `<option value="${backup.id}">${backup.name} - ${this.formatDate(backup.created_at)}</option>`
                ).join('');
        } catch (error) {
            console.error('Failed to load restore options:', error);
        }
    }

    async confirmRestore() {
        const backupId = document.getElementById('restore-backup-select').value;
        
        if (!backupId || backupId === 'Select backup to restore...') {
            this.showError('Please select a backup to restore');
            return;
        }

        if (!confirm('Are you sure? This will overwrite existing data.')) {
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/restore/create`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ backup_id: backupId })
            });

            if (response.ok) {
                this.showSuccess('Restore initiated successfully');
                this.closeModal('restore-modal');
                this.loadDashboard();
            } else {
                throw new Error('Restore failed');
            }
        } catch (error) {
            console.error('Failed to restore:', error);
            this.showError('Failed to restore backup');
        }
    }

    showScheduleModal() {
        const modal = document.getElementById('schedule-modal');
        modal.classList.add('active');
    }

    async createSchedule() {
        const name = document.getElementById('schedule-name').value;
        const type = document.getElementById('schedule-type').value;
        const cron = document.getElementById('schedule-cron').value;
        const targets = Array.from(document.querySelectorAll('input[name="target"]:checked'))
            .map(cb => cb.value);

        if (!name || !cron || targets.length === 0) {
            this.showError('Please fill all required fields');
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/schedules`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, type, cron, targets })
            });

            if (response.ok) {
                this.showSuccess('Schedule created successfully');
                this.closeModal('schedule-modal');
                this.loadSchedules();
            } else {
                throw new Error('Failed to create schedule');
            }
        } catch (error) {
            console.error('Failed to create schedule:', error);
            this.showError('Failed to create schedule');
        }
    }

    async saveSettings() {
        const settings = {
            backup_path: document.getElementById('backup-path').value,
            retention_days: parseInt(document.getElementById('retention-days').value),
            compression_level: document.getElementById('compression-level').value,
            enable_encryption: document.getElementById('enable-encryption').checked,
            enable_agent: document.getElementById('enable-agent').checked
        };

        try {
            const response = await fetch(`${API_BASE}/settings`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(settings)
            });

            if (response.ok) {
                this.showSuccess('Settings saved successfully');
            } else {
                throw new Error('Failed to save settings');
            }
        } catch (error) {
            console.error('Failed to save settings:', error);
            this.showError('Failed to save settings');
        }
    }

    closeModal(modalId) {
        document.getElementById(modalId).classList.remove('active');
    }

    startPolling() {
        // Poll for updates every 30 seconds
        setInterval(() => {
            if (this.currentTab === 'dashboard') {
                this.loadDashboard();
            }
        }, 30000);
    }

    // Utility functions
    formatTimeAgo(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const seconds = Math.floor((now - date) / 1000);
        
        if (seconds < 60) return 'Just now';
        if (seconds < 3600) return `${Math.floor(seconds / 60)} minutes ago`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)} hours ago`;
        return `${Math.floor(seconds / 86400)} days ago`;
    }

    formatDate(timestamp) {
        return new Date(timestamp).toLocaleString();
    }

    formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
        if (bytes < 1073741824) return (bytes / 1048576).toFixed(1) + ' MB';
        return (bytes / 1073741824).toFixed(2) + ' GB';
    }

    formatCron(cron) {
        // Simple cron description
        if (cron === '0 2 * * *') return 'Daily at 2:00 AM';
        if (cron === '0 */6 * * *') return 'Every 6 hours';
        if (cron === '0 0 * * 0') return 'Weekly on Sunday';
        if (cron === '0 0 1 * *') return 'Monthly on 1st';
        return cron;
    }

    showSuccess(message) {
        console.log('Success:', message);
        // TODO: Implement toast notifications
    }

    showError(message) {
        console.error('Error:', message);
        // TODO: Implement toast notifications
    }
}

// Initialize the application
const backup = new BackupManager();