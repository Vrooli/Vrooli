/**
 * NotificationManager - Toast Notification System
 * Handles displaying temporary toast notifications for success, error, and info messages
 */

import { eventBus, EVENT_TYPES } from '../core/EventBus.js';

/**
 * NotificationManager class - Manages toast notifications
 */
export class NotificationManager {
    constructor(eventBusInstance = eventBus) {
        this.eventBus = eventBusInstance;

        // Active notifications tracking
        this.activeNotifications = [];
        this.maxNotifications = 5;

        // Default durations (in milliseconds)
        this.durations = {
            success: 5000,
            error: 7000,
            info: 5000,
            warning: 6000
        };

        // Debug mode
        this.debug = false;

        this.initialize();
    }

    /**
     * Initialize notification manager
     */
    initialize() {
        this.setupEventListeners();

        if (this.debug) {
            console.log('[NotificationManager] Notification system initialized');
        }
    }

    /**
     * Setup event listeners
     * @private
     */
    setupEventListeners() {
        // Listen for notification requests
        this.eventBus.on(EVENT_TYPES.NOTIFICATION_SHOW, (event) => {
            const { message, type, duration } = event.data;
            this.show(message, type, duration);
        });

        this.eventBus.on(EVENT_TYPES.NOTIFICATION_SUCCESS, (event) => {
            this.showSuccess(event.data.message);
        });

        this.eventBus.on(EVENT_TYPES.NOTIFICATION_ERROR, (event) => {
            this.showError(event.data.message);
        });

        this.eventBus.on(EVENT_TYPES.NOTIFICATION_INFO, (event) => {
            this.showInfo(event.data.message);
        });
    }

    /**
     * Show a notification
     * @param {string} message - Notification message
     * @param {string} type - Notification type (success, error, info, warning)
     * @param {number} duration - Duration in milliseconds (optional)
     */
    show(message, type = 'info', duration = null) {
        if (!message) {
            console.warn('[NotificationManager] Cannot show notification without message');
            return;
        }

        if (this.debug) {
            console.log(`[NotificationManager] Showing ${type} notification:`, message);
        }

        // Remove oldest notification if at max
        if (this.activeNotifications.length >= this.maxNotifications) {
            const oldest = this.activeNotifications[0];
            this.remove(oldest);
        }

        // Create notification element
        const notification = this.createNotificationElement(message, type);

        // Add to DOM
        document.body.appendChild(notification);

        // Track active notification
        this.activeNotifications.push(notification);

        // Position notifications
        this.updateNotificationPositions();

        // Animate in
        setTimeout(() => {
            notification.style.transform = 'translateX(0)';
            notification.style.opacity = '1';
        }, 100);

        // Auto-remove after duration
        const displayDuration = duration || this.durations[type] || this.durations.info;
        setTimeout(() => {
            this.remove(notification);
        }, displayDuration);

        return notification;
    }

    /**
     * Show success notification
     * @param {string} message
     */
    showSuccess(message) {
        return this.show(message, 'success');
    }

    /**
     * Show error notification
     * @param {string} message
     */
    showError(message) {
        return this.show(message, 'error');
    }

    /**
     * Show info notification
     * @param {string} message
     */
    showInfo(message) {
        return this.show(message, 'info');
    }

    /**
     * Show warning notification
     * @param {string} message
     */
    showWarning(message) {
        return this.show(message, 'warning');
    }

    /**
     * Create notification DOM element
     * @private
     */
    createNotificationElement(message, type) {
        const notification = document.createElement('div');
        notification.className = 'toast-notification';
        notification.dataset.notificationType = type;

        // Get color based on type
        const colors = {
            success: 'var(--accent-success)',
            error: 'var(--accent-error)',
            info: 'var(--accent-secondary)',
            warning: 'var(--accent-warning, #ff9800)'
        };

        const backgroundColor = colors[type] || colors.info;

        notification.style.cssText = `
            position: fixed;
            right: 20px;
            padding: 1rem 1.5rem;
            background: ${backgroundColor};
            color: var(--bg-primary);
            border-radius: var(--panel-border-radius, 8px);
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
            z-index: 10000;
            font-weight: 500;
            max-width: 400px;
            transform: translateX(120%);
            opacity: 0;
            transition: transform 0.3s ease, opacity 0.3s ease, top 0.3s ease;
            cursor: pointer;
            user-select: none;
        `;

        notification.textContent = message;

        // Allow dismissing by clicking
        notification.addEventListener('click', () => {
            this.remove(notification);
        });

        return notification;
    }

    /**
     * Remove a notification
     * @param {HTMLElement} notification
     */
    remove(notification) {
        if (!notification || !document.body.contains(notification)) {
            return;
        }

        if (this.debug) {
            console.log('[NotificationManager] Removing notification');
        }

        // Animate out
        notification.style.transform = 'translateX(120%)';
        notification.style.opacity = '0';

        // Remove from tracking
        const index = this.activeNotifications.indexOf(notification);
        if (index > -1) {
            this.activeNotifications.splice(index, 1);
        }

        // Update positions of remaining notifications
        this.updateNotificationPositions();

        // Remove from DOM after animation
        setTimeout(() => {
            if (document.body.contains(notification)) {
                document.body.removeChild(notification);
            }
        }, 300);
    }

    /**
     * Update notification positions to stack them
     * @private
     */
    updateNotificationPositions() {
        const spacing = 10; // Gap between notifications
        let topPosition = 20; // Initial top position

        this.activeNotifications.forEach((notification, index) => {
            notification.style.top = `${topPosition}px`;
            topPosition += notification.offsetHeight + spacing;
        });
    }

    /**
     * Clear all notifications
     */
    clearAll() {
        if (this.debug) {
            console.log('[NotificationManager] Clearing all notifications');
        }

        [...this.activeNotifications].forEach(notification => {
            this.remove(notification);
        });
    }

    /**
     * Set custom duration for notification type
     * @param {string} type - Notification type
     * @param {number} duration - Duration in milliseconds
     */
    setDuration(type, duration) {
        if (typeof duration !== 'number' || duration <= 0) {
            console.warn(`[NotificationManager] Invalid duration: ${duration}`);
            return;
        }

        this.durations[type] = duration;

        if (this.debug) {
            console.log(`[NotificationManager] Set ${type} duration to ${duration}ms`);
        }
    }

    /**
     * Set max number of concurrent notifications
     * @param {number} max
     */
    setMaxNotifications(max) {
        if (typeof max !== 'number' || max <= 0) {
            console.warn(`[NotificationManager] Invalid max notifications: ${max}`);
            return;
        }

        this.maxNotifications = max;

        if (this.debug) {
            console.log(`[NotificationManager] Set max notifications to ${max}`);
        }
    }

    /**
     * Set debug mode
     * @param {boolean} enabled
     */
    setDebug(enabled) {
        this.debug = enabled;
        console.log(`[NotificationManager] Debug mode ${enabled ? 'enabled' : 'disabled'}`);
    }

    /**
     * Get debug information
     * @returns {Object}
     */
    getDebugInfo() {
        return {
            activeCount: this.activeNotifications.length,
            maxNotifications: this.maxNotifications,
            durations: { ...this.durations }
        };
    }
}

// Export singleton instance
export const notificationManager = new NotificationManager();

// Export default for convenience
export default notificationManager;
