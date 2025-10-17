/**
 * AI Research Assistant - Professional Dashboard Application
 * Enterprise-grade research intelligence platform
 */

class ResearchAssistantApp {
    constructor() {
        this.apiBaseUrl = '/api';
        this.currentTab = 'dashboard';
        this.currentSettingsPanel = 'general';
        this.isDarkMode = localStorage.getItem('darkMode') === 'true';
        
        this.init();
    }

    async init() {
        this.setupEventListeners();
        this.loadDashboardData();
        this.initializeTheme();
        
        // Initialize WebSocket connection for real-time updates
        this.initWebSocket();
        
        console.log('🚀 AI Research Assistant Dashboard initialized');
    }

    setupEventListeners() {
        // Navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const tab = item.getAttribute('data-tab');
                this.switchTab(tab);
            });
        });

        // Settings navigation
        document.querySelectorAll('.settings-nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                const setting = item.getAttribute('data-setting');
                this.switchSettingsPanel(setting);
            });
        });

        // Form submissions
        document.getElementById('newReportForm')?.addEventListener('submit', (e) => {
            e.preventDefault();
            this.createNewReport();
        });

        // Chat input
        document.getElementById('chatInput')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });

        // Search and filters
        document.getElementById('reportSearch')?.addEventListener('input', (e) => {
            this.filterReports(e.target.value);
        });

        document.getElementById('statusFilter')?.addEventListener('change', (e) => {
            this.filterReportsByStatus(e.target.value);
        });

        // Theme toggle
        document.getElementById('themeSelect')?.addEventListener('change', (e) => {
            this.toggleTheme(e.target.value === 'dark');
        });

        // Modal close on outside click
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.closeModal(e.target.id);
            }
        });
    }

    // Navigation
    switchTab(tab) {
        // Update navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-tab="${tab}"]`).classList.add('active');

        // Update content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(tab).classList.add('active');

        this.currentTab = tab;

        // Load tab-specific data
        this.loadTabData(tab);
    }

    switchSettingsPanel(panel) {
        // Update navigation
        document.querySelectorAll('.settings-nav-item').forEach(item => {
            item.classList.remove('active');
        });
        document.querySelector(`[data-setting="${panel}"]`).classList.add('active');

        // Update content
        document.querySelectorAll('.settings-panel').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${panel}-settings`).classList.add('active');

        this.currentSettingsPanel = panel;
    }

    // Data Loading
    async loadTabData(tab) {
        switch(tab) {
            case 'dashboard':
                await this.loadDashboardData();
                break;
            case 'reports':
                await this.loadReports();
                break;
            case 'chat':
                await this.loadChatHistory();
                break;
            case 'schedules':
                await this.loadSchedules();
                break;
        }
    }

    async loadDashboardData() {
        try {
            this.showLoading(true);
            
            // Load key metrics
            const [reports, schedules, chatSessions, avgConfidence] = await Promise.all([
                this.apiRequest('/reports/count'),
                this.apiRequest('/schedules/count'),
                this.apiRequest('/chat/sessions/count'),
                this.apiRequest('/reports/confidence-average')
            ]);

            // Update metrics
            this.updateElement('totalReports', reports.count || 0);
            this.updateElement('activeSchedules', schedules.count || 0);
            this.updateElement('chatSessions', chatSessions.count || 0);
            this.updateElement('avgConfidence', `${Math.round(avgConfidence.average || 0)}%`);

            // Load recent reports
            await this.loadRecentReports();

        } catch (error) {
            this.showToast('Error loading dashboard data', 'error');
            console.error('Dashboard loading error:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async loadRecentReports() {
        try {
            const reports = await this.apiRequest('/reports?limit=5&status=completed');
            const tbody = document.querySelector('#recentReportsTable tbody');
            
            if (!tbody) return;

            tbody.innerHTML = reports.map(report => `
                <tr>
                    <td class="font-semibold">${this.truncateText(report.title, 30)}</td>
                    <td><span class="status-badge ${report.depth}">${report.depth}</span></td>
                    <td>${report.word_count?.toLocaleString() || 'N/A'}</td>
                    <td>${Math.round(report.confidence_score * 100)}%</td>
                    <td>${this.formatDate(report.completed_at)}</td>
                    <td>
                        <div class="action-buttons">
                            <button class="icon-button" onclick="app.viewReport('${report.id}')" title="View">
                                <i class="fas fa-eye"></i>
                            </button>
                            <button class="icon-button" onclick="app.downloadReport('${report.id}')" title="Download">
                                <i class="fas fa-download"></i>
                            </button>
                        </div>
                    </td>
                </tr>
            `).join('');

        } catch (error) {
            console.error('Error loading recent reports:', error);
        }
    }

    async loadReports() {
        try {
            this.showLoading(true);
            const reports = await this.apiRequest('/reports');
            const tbody = document.querySelector('#reportsTable tbody');
            
            if (!tbody) return;

            tbody.innerHTML = reports.map(report => `
                <tr>
                    <td class="font-semibold">${this.truncateText(report.title, 40)}</td>
                    <td>${this.truncateText(report.topic, 30)}</td>
                    <td><span class="status-badge ${report.depth}">${report.depth}</span></td>
                    <td>${report.word_count?.toLocaleString() || 'N/A'}</td>
                    <td>${report.sources_count || 'N/A'}</td>
                    <td>${Math.round(report.confidence_score * 100)}%</td>
                    <td><span class="status-badge ${report.status}">${report.status}</span></td>
                    <td>${this.formatDate(report.completed_at || report.created_at)}</td>
                    <td>
                        <div class="action-buttons">
                            <button class="icon-button" onclick="app.viewReport('${report.id}')" title="View">
                                <i class="fas fa-eye"></i>
                            </button>
                            <button class="icon-button" onclick="app.downloadReport('${report.id}')" title="Download">
                                <i class="fas fa-download"></i>
                            </button>
                            <button class="icon-button" onclick="app.chatAboutReport('${report.id}')" title="Chat">
                                <i class="fas fa-comments"></i>
                            </button>
                        </div>
                    </td>
                </tr>
            `).join('');

        } catch (error) {
            this.showToast('Error loading reports', 'error');
            console.error('Reports loading error:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async loadSchedules() {
        try {
            this.showLoading(true);
            const schedules = await this.apiRequest('/schedules');
            const tbody = document.querySelector('#schedulesTable tbody');
            
            if (!tbody) return;

            tbody.innerHTML = schedules.map(schedule => `
                <tr>
                    <td class="font-semibold">${schedule.name}</td>
                    <td>${this.truncateText(schedule.topic_template, 40)}</td>
                    <td>${this.formatCron(schedule.cron_expression)}</td>
                    <td><span class="status-badge ${schedule.depth}">${schedule.depth}</span></td>
                    <td>${this.formatDate(schedule.next_run_at)}</td>
                    <td><span class="status-badge ${schedule.is_active ? 'completed' : 'pending'}">${schedule.is_active ? 'Active' : 'Inactive'}</span></td>
                    <td>
                        <div class="action-buttons">
                            <button class="icon-button" onclick="app.editSchedule('${schedule.id}')" title="Edit">
                                <i class="fas fa-edit"></i>
                            </button>
                            <button class="icon-button" onclick="app.runScheduleNow('${schedule.id}')" title="Run Now">
                                <i class="fas fa-play"></i>
                            </button>
                            <button class="icon-button" onclick="app.toggleSchedule('${schedule.id}')" title="Toggle">
                                <i class="fas fa-power-off"></i>
                            </button>
                        </div>
                    </td>
                </tr>
            `).join('');

        } catch (error) {
            this.showToast('Error loading schedules', 'error');
            console.error('Schedules loading error:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async loadChatHistory() {
        try {
            const conversations = await this.apiRequest('/chat/conversations');
            const messages = await this.apiRequest('/chat/messages?limit=50');
            
            this.renderChatMessages(messages);
        } catch (error) {
            console.error('Error loading chat history:', error);
        }
    }

    // Report Management
    async createNewReport() {
        try {
            const formData = {
                topic: document.getElementById('reportTopic').value,
                depth: document.getElementById('reportDepth').value,
                target_length: parseInt(document.getElementById('reportLength').value)
            };

            if (!formData.topic.trim()) {
                this.showToast('Please enter a research topic', 'warning');
                return;
            }

            this.showLoading(true);
            this.closeModal('newReportModal');

            const response = await this.apiRequest('/reports', 'POST', formData);
            
            this.showToast('Research report generation started!', 'success');
            
            // Reset form
            document.getElementById('newReportForm').reset();
            
            // Refresh dashboard if we're on it
            if (this.currentTab === 'dashboard') {
                setTimeout(() => this.loadDashboardData(), 1000);
            }

        } catch (error) {
            this.showToast('Failed to create report. Please try again.', 'error');
            console.error('Report creation error:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async viewReport(reportId) {
        try {
            this.showLoading(true);
            const report = await this.apiRequest(`/reports/${reportId}`);
            
            // Create a modal to display the report
            this.showReportModal(report);
            
        } catch (error) {
            this.showToast('Error loading report', 'error');
            console.error('Report viewing error:', error);
        } finally {
            this.showLoading(false);
        }
    }

    async downloadReport(reportId) {
        try {
            const response = await fetch(`${this.apiBaseUrl}/reports/${reportId}/download`);
            
            if (!response.ok) throw new Error('Download failed');
            
            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `research-report-${reportId}.pdf`;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);
            
            this.showToast('Report downloaded successfully', 'success');
            
        } catch (error) {
            this.showToast('Error downloading report', 'error');
            console.error('Download error:', error);
        }
    }

    // Chat Functionality
    async sendMessage() {
        const input = document.getElementById('chatInput');
        const message = input.value.trim();
        
        if (!message) return;

        try {
            // Add user message to UI immediately
            this.addChatMessage('user', message);
            input.value = '';

            // Send to API
            const response = await this.apiRequest('/chat/message', 'POST', {
                message: message,
                conversation_id: this.currentConversationId || null
            });

            // Add AI response
            this.addChatMessage('assistant', response.response);

            // Update conversation ID
            if (response.conversation_id) {
                this.currentConversationId = response.conversation_id;
            }

        } catch (error) {
            this.showToast('Failed to send message', 'error');
            console.error('Chat error:', error);
        }
    }

    async sendQuickPrompt(prompt) {
        const input = document.getElementById('chatInput');
        input.value = prompt;
        await this.sendMessage();
    }

    addChatMessage(role, content) {
        const messagesContainer = document.getElementById('chatMessages');
        if (!messagesContainer) return;

        const messageDiv = document.createElement('div');
        messageDiv.className = `chat-message ${role}`;
        
        const avatar = role === 'assistant' ? '<i class="fas fa-robot"></i>' : '<i class="fas fa-user"></i>';
        
        messageDiv.innerHTML = `
            <div class="message-avatar">
                ${avatar}
            </div>
            <div class="message-content">
                <p>${content}</p>
                <div class="message-time">${this.formatTime(new Date())}</div>
            </div>
        `;

        messagesContainer.appendChild(messageDiv);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    renderChatMessages(messages) {
        const container = document.getElementById('chatMessages');
        if (!container || !messages) return;

        container.innerHTML = messages.map(msg => `
            <div class="chat-message ${msg.role}">
                <div class="message-avatar">
                    ${msg.role === 'assistant' ? '<i class="fas fa-robot"></i>' : '<i class="fas fa-user"></i>'}
                </div>
                <div class="message-content">
                    <p>${msg.content}</p>
                    <div class="message-time">${this.formatTime(new Date(msg.created_at))}</div>
                </div>
            </div>
        `).join('');

        container.scrollTop = container.scrollHeight;
    }

    clearChat() {
        if (confirm('Are you sure you want to clear the chat history?')) {
            document.getElementById('chatMessages').innerHTML = `
                <div class="chat-message assistant">
                    <div class="message-avatar">
                        <i class="fas fa-robot"></i>
                    </div>
                    <div class="message-content">
                        <p>Hello! I'm your AI Research Assistant. I can help you with questions about your research reports, analyze data, or start new research projects. What would you like to know?</p>
                        <div class="message-time">Just now</div>
                    </div>
                </div>
            `;
            this.currentConversationId = null;
        }
    }

    // Schedule Management
    async runScheduleNow(scheduleId) {
        try {
            await this.apiRequest(`/schedules/${scheduleId}/run`, 'POST');
            this.showToast('Schedule triggered successfully', 'success');
            
            setTimeout(() => this.loadSchedules(), 1000);
        } catch (error) {
            this.showToast('Failed to run schedule', 'error');
            console.error('Schedule run error:', error);
        }
    }

    async toggleSchedule(scheduleId) {
        try {
            await this.apiRequest(`/schedules/${scheduleId}/toggle`, 'POST');
            this.showToast('Schedule status updated', 'success');
            
            setTimeout(() => this.loadSchedules(), 1000);
        } catch (error) {
            this.showToast('Failed to toggle schedule', 'error');
            console.error('Schedule toggle error:', error);
        }
    }

    // Filtering
    filterReports(searchTerm) {
        const rows = document.querySelectorAll('#reportsTable tbody tr');
        const term = searchTerm.toLowerCase();

        rows.forEach(row => {
            const title = row.cells[0].textContent.toLowerCase();
            const topic = row.cells[1].textContent.toLowerCase();
            
            if (title.includes(term) || topic.includes(term)) {
                row.style.display = '';
            } else {
                row.style.display = 'none';
            }
        });
    }

    filterReportsByStatus(status) {
        const rows = document.querySelectorAll('#reportsTable tbody tr');

        rows.forEach(row => {
            const statusCell = row.cells[6].querySelector('.status-badge');
            const reportStatus = statusCell.textContent.toLowerCase();
            
            if (status === 'all' || reportStatus === status) {
                row.style.display = '';
            } else {
                row.style.display = 'none';
            }
        });
    }

    // UI Helpers
    showModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.add('active');
        }
    }

    closeModal(modalId) {
        const modal = document.getElementById(modalId);
        if (modal) {
            modal.classList.remove('active');
        }
    }

    showLoading(show) {
        const loader = document.getElementById('loadingIndicator');
        if (loader) {
            if (show) {
                loader.classList.add('active');
            } else {
                loader.classList.remove('active');
            }
        }
    }

    showToast(message, type = 'success') {
        const container = document.getElementById('toastContainer');
        if (!container) return;

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        
        const icons = {
            success: 'fas fa-check-circle',
            warning: 'fas fa-exclamation-triangle',
            error: 'fas fa-times-circle'
        };

        toast.innerHTML = `
            <i class="toast-icon ${icons[type]}"></i>
            <div class="toast-content">
                <h4>${type.charAt(0).toUpperCase() + type.slice(1)}</h4>
                <p>${message}</p>
            </div>
        `;

        container.appendChild(toast);

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (toast.parentNode) {
                toast.parentNode.removeChild(toast);
            }
        }, 5000);
    }

    // Theme Management
    initializeTheme() {
        if (this.isDarkMode) {
            document.body.classList.add('dark-mode');
            const themeSelect = document.getElementById('themeSelect');
            if (themeSelect) themeSelect.value = 'dark';
        }
    }

    toggleTheme(isDark) {
        this.isDarkMode = isDark;
        localStorage.setItem('darkMode', isDark);
        
        if (isDark) {
            document.body.classList.add('dark-mode');
        } else {
            document.body.classList.remove('dark-mode');
        }
    }

    toggleDarkMode() {
        this.toggleTheme(!this.isDarkMode);
        const themeSelect = document.getElementById('themeSelect');
        if (themeSelect) {
            themeSelect.value = this.isDarkMode ? 'dark' : 'light';
        }
    }

    // API Helper
    async apiRequest(endpoint, method = 'GET', data = null) {
        try {
            const config = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                }
            };

            if (data) {
                config.body = JSON.stringify(data);
            }

            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, config);
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error(`API Error (${method} ${endpoint}):`, error);
            throw error;
        }
    }

    // Utility Functions
    updateElement(id, content) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = content;
        }
    }

    truncateText(text, maxLength) {
        if (!text) return 'N/A';
        return text.length > maxLength ? text.substr(0, maxLength) + '...' : text;
    }

    formatDate(dateString) {
        if (!dateString) return 'N/A';
        const date = new Date(dateString);
        return date.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'short',
            day: 'numeric'
        });
    }

    formatTime(date) {
        return date.toLocaleTimeString('en-US', {
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    formatCron(cronExpression) {
        const patterns = {
            '0 0 * * *': 'Daily',
            '0 0 * * 0': 'Weekly',
            '0 0 1 * *': 'Monthly',
            '0 */6 * * *': 'Every 6 hours',
            '0 */12 * * *': 'Every 12 hours'
        };
        return patterns[cronExpression] || cronExpression;
    }

    // WebSocket for Real-time Updates
    initWebSocket() {
        // This would typically connect to a WebSocket endpoint
        // For now, we'll use periodic polling
        setInterval(() => {
            if (this.currentTab === 'dashboard') {
                this.loadDashboardData();
            }
        }, 30000); // Refresh every 30 seconds
    }

    // Global methods for onclick handlers
    chatAboutReport(reportId) {
        this.switchTab('chat');
        // Could pre-populate chat with report context
    }

    editSchedule(scheduleId) {
        this.showToast('Schedule editing coming soon', 'warning');
    }

    exportChat() {
        this.showToast('Chat export coming soon', 'warning');
    }

    showNotifications() {
        this.showToast('Notifications panel coming soon', 'warning');
    }
}

// Global functions for HTML onclick handlers
function showNewReportModal() {
    app.showModal('newReportModal');
}

function closeModal(modalId) {
    app.closeModal(modalId);
}

function switchTab(tab) {
    app.switchTab(tab);
}

function sendMessage() {
    app.sendMessage();
}

function sendQuickPrompt(prompt) {
    app.sendQuickPrompt(prompt);
}

function clearChat() {
    app.clearChat();
}

function exportChat() {
    app.exportChat();
}

function toggleDarkMode() {
    app.toggleDarkMode();
}

function showNotifications() {
    app.showNotifications();
}

function showNewScheduleModal() {
    app.showToast('Schedule creation coming soon', 'warning');
}

// Initialize the application
const app = new ResearchAssistantApp();

// Export for global access
window.app = app;