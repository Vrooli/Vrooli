// UI Components Module
export class UIComponents {
    static createTaskCard(task, runningProcesses = {}) {
        const isRunning = !!runningProcesses[task.id];
        const targetMarkup = this.renderTaskTarget(task);
        const normalizedNotes = typeof task.notes === 'string' ? task.notes : '';
        
        const card = document.createElement('div');
        const statusClass = this.getStatusClass(task.status);
        const autoRequeueDisabled = task.processor_auto_requeue === false;
        const isArchived = task.status === 'archived';
        const cardClasses = [
            'task-card',
            task.priority,
            isRunning ? 'task-executing' : '',
            task.type || 'resource',
            task.operation || 'generator',
            statusClass,
            autoRequeueDisabled ? 'auto-requeue-disabled' : '',
            isArchived ? 'task-archived' : ''
        ].filter(Boolean);
        card.className = cardClasses.join(' ');
        card.id = `task-${task.id}`;
        card.draggable = true;
        card.dataset.notesText = normalizedNotes.toLowerCase();

        const completionCount = Number.isInteger(task.completion_count) ? task.completion_count : 0;
        const completionLabel = completionCount === 1 ? 'completed run' : 'completed runs';
        const lastCompletedDate = task.last_completed_at ? new Date(task.last_completed_at) : null;
        const hasValidLastCompleted = lastCompletedDate && !Number.isNaN(lastCompletedDate.getTime());
        const lastCompletedRelative = hasValidLastCompleted ? this.formatRelativeTime(task.last_completed_at) : '';
        const lastCompletedTooltip = hasValidLastCompleted ? lastCompletedDate.toLocaleString() : null;

        // Determine icon and label based on type and operation
        const typeInfo = this.getTaskTypeInfo(task);

        card.innerHTML = `
            ${autoRequeueDisabled ? `
                <div class="task-warning-banner" title="This task will not be picked up automatically by the queue processor.">
                    <i class="fas fa-exclamation-triangle"></i>
                    Auto Requeue Disabled
                </div>
            ` : ''}
            <div class="task-header">
                <span class="task-type ${task.type}">
                    <i class="${typeInfo.icon}"></i> ${typeInfo.label}
                </span>
                <div class="task-header-metrics">
                    <span class="task-completions" title="${completionCount} ${completionLabel}">
                        <i class="fas fa-rotate-right"></i> ${completionCount}
                    </span>
                    <span class="priority-chip priority-${task.priority}" title="Priority: ${task.priority}">
                        ${task.priority}
                    </span>
                </div>
            </div>
            <h3 class="task-title">${this.escapeHtml(task.title)}</h3>
            ${targetMarkup}
            ${task.category ? `<span class="task-category category-${task.category}">${task.category}</span>` : ''}
            <div class="task-meta-row">
                <span class="task-meta-label">Last run:</span>
                <span class="task-meta-value" ${lastCompletedTooltip ? `title="${this.escapeHtml(lastCompletedTooltip)}"` : ''}>
                    ${lastCompletedRelative || '—'}
                </span>
            </div>
            ${this.renderStreakInfo(task)}
            ${isRunning ? `
                <div class="task-execution-indicator">
                    <i class="fas fa-brain fa-spin"></i>
                    <div class="execution-details">
                        <span>Executing with Claude...</span>
                        ${runningProcesses[task.id] && runningProcesses[task.id].duration ? `<div class="duration-info">Running: ${this.formatDuration(runningProcesses[task.id].duration)}</div>` : ''}
                        ${runningProcesses[task.id] && runningProcesses[task.id].agent_id ? `<div class="agent-info">Agent: ${this.escapeHtml(runningProcesses[task.id].agent_id)}</div>` : ''}
                    </div>
                    <button class="btn-stop-execution" onclick="event.stopPropagation(); ecosystemManager.stopTaskExecution('${task.id}')" title="Stop execution">
                        <i class="fas fa-stop"></i>
                    </button>
                </div>
            ` : ''}
            ${isArchived ? `
                <button class="btn-icon task-restore-btn" data-task-id="${task.id}" data-task-status="${task.status}" title="Restore Task to Pending">
                    <i class="fas fa-box-open"></i>
                </button>
            ` : `
                <button class="btn-icon task-archive-btn" data-task-id="${task.id}" data-task-status="${task.status}" title="Archive Task">
                    <i class="fas fa-box-archive"></i>
                </button>
            `}
            <button class="btn-icon task-delete-btn" data-task-id="${task.id}" data-task-status="${task.status}" title="Delete Task">
                <i class="fas fa-trash"></i>
            </button>
        `;
        
        return card;
    }

    static renderTaskTarget(task) {
        if (!task || task.operation !== 'improver') {
            return '';
        }

        const targets = [];
        if (Array.isArray(task.targets) && task.targets.length > 0) {
            targets.push(...task.targets.filter(Boolean));
        } else if (task.target) {
            targets.push(task.target);
        }

        if (targets.length === 0) {
            return '';
        }

        const label = targets.length > 1 ? 'Targets:' : 'Target:';
        const formattedTargets = targets.map(value => this.escapeHtml(value)).join(', ');

        return `
            <div class="task-meta-row">
                <span class="task-meta-label">${label}</span>
                <span class="task-meta-value">${formattedTargets}</span>
            </div>
        `;
    }

    static formatRelativeTime(timestamp) {
        if (!timestamp) return '';

        const parsed = new Date(timestamp);
        if (Number.isNaN(parsed.getTime())) {
            return '';
        }

        const diffMs = Date.now() - parsed.getTime();
        if (diffMs < 0) {
            return 'just now';
        }

        const seconds = Math.floor(diffMs / 1000);
        if (seconds < 60) {
            return 'just now';
        }

        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) {
            return `${minutes}m ago`;
        }

        const hours = Math.floor(minutes / 60);
        if (hours < 24) {
            return `${hours}h ago`;
        }

        const days = Math.floor(hours / 24);
        if (days < 7) {
            return `${days}d ago`;
        }

        const weeks = Math.floor(days / 7);
        if (weeks < 5) {
            return `${weeks}w ago`;
        }

        const months = Math.floor(days / 30);
        if (months < 12) {
            return `${months}mo ago`;
        }

        const years = Math.floor(days / 365);
        return `${years}y ago`;
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

    static getStatusClass(status) {
        if (!status) return '';
        switch (status) {
            case 'completed-finalized':
                return 'task-finalized';
            case 'failed-blocked':
                return 'task-blocked';
            default:
                return `status-${status.replace(/[^a-z0-9-]/gi, '').toLowerCase()}`;
        }
    }

    static renderStreakInfo(task) {
        const completionStreak = Number.isInteger(task.consecutive_completion_claims) ? task.consecutive_completion_claims : 0;
        const failureStreak = Number.isInteger(task.consecutive_failures) ? task.consecutive_failures : 0;

        if (completionStreak <= 0 && failureStreak <= 0) {
            return '';
        }

        const chips = [];
        if (completionStreak > 0) {
            chips.push(`<span class="streak-chip success"><i class="fas fa-check"></i> ${completionStreak} complete</span>`);
        }
        if (failureStreak > 0) {
            chips.push(`<span class="streak-chip danger"><i class="fas fa-times"></i> ${failureStreak} failure</span>`);
        }

        return `<div class="task-streaks">${chips.join('')}</div>`;
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

    static dismissRateLimitNotification(target) {
        let notification = null;

        if (target && typeof target.closest === 'function') {
            notification = target.closest('.rate-limit-notification');
        }

        if (!notification && target && target.classList && target.classList.contains('rate-limit-notification')) {
            notification = target;
        }

        if (!notification) {
            notification = document.querySelector('.rate-limit-notification');
        }

        if (notification) {
            notification.classList.remove('show');
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.remove();
                }
            }, 300);
        }
    }

    static formatDuration(durationStr) {
        if (!durationStr) return '';
        
        // If it's already formatted nicely (like "4m 10s"), return as is
        if (durationStr.includes(' ')) {
            return durationStr;
        }
        
        // Parse durations like "4m10s" or "1h30m5s" and format them nicely
        const duration = durationStr.toString();
        
        // Extract hours, minutes, seconds
        const hourMatch = duration.match(/(\d+)h/);
        const minuteMatch = duration.match(/(\d+)m/);
        const secondMatch = duration.match(/(\d+)s/);
        
        const hours = hourMatch ? parseInt(hourMatch[1]) : 0;
        const minutes = minuteMatch ? parseInt(minuteMatch[1]) : 0;
        const seconds = secondMatch ? parseInt(secondMatch[1]) : 0;
        
        // Format nicely
        const parts = [];
        if (hours > 0) parts.push(`${hours}h`);
        if (minutes > 0) parts.push(`${minutes}m`);
        if (seconds > 0) parts.push(`${seconds}s`);
        
        return parts.join(' ') || '0s';
    }

    static escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
