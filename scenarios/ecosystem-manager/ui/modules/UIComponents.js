// UI Components Module
export class UIComponents {
    static createTaskCard(task, runningProcesses = {}) {
        const escapedNotes = this.escapeHtml(task.notes || 'No notes available');
        const truncatedNotes = escapedNotes.length > 150 ? 
            escapedNotes.substring(0, 150) + '...' : 
            escapedNotes;
        
        const isRunning = !!runningProcesses[task.id];
        
        const card = document.createElement('div');
        card.className = `task-card ${task.priority} ${isRunning ? 'task-executing' : ''} ${task.type || 'resource'} ${task.operation || 'generator'}`;
        card.id = `task-${task.id}`;
        card.draggable = true;
        
        // Determine icon and label based on type and operation
        const typeInfo = this.getTaskTypeInfo(task);
        
        card.innerHTML = `
            <div class="task-header">
                <span class="task-type ${task.type}">
                    <i class="${typeInfo.icon}"></i> ${typeInfo.label}
                </span>
                <span class="priority-chip priority-${task.priority}">
                    ${task.priority}
                </span>
            </div>
            <h3 class="task-title">${this.escapeHtml(task.title)}</h3>
            ${task.current_phase ? `<div class="task-phase"><i class="fas ${this.getPhaseIcon(task.current_phase)}"></i> ${task.current_phase}</div>` : ''}
            ${task.category ? `<span class="task-category category-${task.category}">${task.category}</span>` : ''}
            ${isRunning ? `
                <div class="task-execution-indicator">
                    <i class="fas fa-brain fa-spin"></i>
                    <span>Executing with Claude...</span>
                </div>
            ` : ''}
            ${task.results && task.status === 'completed' ? `
                <div class="task-result-indicator success">
                    <i class="fas fa-check-circle"></i>
                    <span>Completed</span>
                </div>
            ` : ''}
            ${task.results && task.status === 'failed' ? `
                <div class="task-result-indicator error">
                    <i class="fas fa-times-circle"></i>
                    <span>${task.results.timeout_failure ? 'Timeout' : 'Failed'}</span>
                </div>
            ` : ''}
            <button class="btn-icon task-delete-btn" data-task-id="${task.id}" data-task-status="${task.status}" title="Delete Task">
                <i class="fas fa-trash"></i>
            </button>
        `;
        
        return card;
    }

    static getTaskTypeInfo(task) {
        if (task.type === 'resource') {
            return task.operation === 'improver' ? 
                { icon: 'fas fa-tools', label: 'Resource Improver' } :
                { icon: 'fas fa-cube', label: 'Resource Generator' };
        } else {
            return task.operation === 'improver' ? 
                { icon: 'fas fa-sync', label: 'Scenario Improver' } :
                { icon: 'fas fa-project-diagram', label: 'Scenario Generator' };
        }
    }

    static getPhaseIcon(phase) {
        switch (phase) {
            case 'pending':
                return 'fa-clock';
            case 'in-progress':
                return 'fa-spinner';
            case 'completed':
                return 'fa-check-circle';
            case 'failed':
                return 'fa-times-circle';
            default:
                return 'fa-tasks';
        }
    }

    static showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.innerHTML = `
            <i class="fas ${this.getToastIcon(type)}"></i>
            <span>${message}</span>
        `;
        
        document.body.appendChild(toast);
        
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);
        
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                document.body.removeChild(toast);
            }, 300);
        }, 3000);
    }

    static getToastIcon(type) {
        switch (type) {
            case 'success': return 'fa-check-circle';
            case 'error': return 'fa-exclamation-circle';
            case 'warning': return 'fa-exclamation-triangle';
            default: return 'fa-info-circle';
        }
    }

    static showLoading(show) {
        const loader = document.getElementById('loading-overlay');
        if (loader) {
            loader.style.display = show ? 'flex' : 'none';
        }
    }

    static showRateLimitNotification(retryAfter) {
        const existingNotification = document.querySelector('.rate-limit-notification');
        if (existingNotification) {
            existingNotification.remove();
        }
        
        const notification = document.createElement('div');
        notification.className = 'rate-limit-notification';
        notification.innerHTML = `
            <div class="rate-limit-content">
                <i class="fas fa-hourglass-half"></i>
                <div class="rate-limit-text">
                    <strong>Rate Limit Reached</strong>
                    <div id="rate-limit-countdown">Please wait ${retryAfter} seconds before trying again</div>
                </div>
                <button class="btn btn-sm" onclick="ecosystemManager.dismissRateLimitNotification(this)">
                    <i class="fas fa-times"></i>
                </button>
            </div>
        `;
        
        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.classList.add('show');
        }, 10);
        
        // Countdown
        let secondsLeft = retryAfter;
        const countdownInterval = setInterval(() => {
            secondsLeft--;
            const countdownElement = document.getElementById('rate-limit-countdown');
            if (countdownElement) {
                if (secondsLeft > 0) {
                    countdownElement.textContent = `Please wait ${secondsLeft} seconds before trying again`;
                } else {
                    countdownElement.textContent = 'You can try again now';
                    clearInterval(countdownInterval);
                    setTimeout(() => {
                        notification.classList.remove('show');
                        setTimeout(() => {
                            notification.remove();
                        }, 300);
                    }, 2000);
                }
            } else {
                clearInterval(countdownInterval);
            }
        }, 1000);
    }

    static dismissRateLimitNotification(button) {
        const notification = button.closest('.rate-limit-notification');
        if (notification) {
            notification.classList.remove('show');
            setTimeout(() => {
                notification.remove();
            }, 300);
        }
    }

    static escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}