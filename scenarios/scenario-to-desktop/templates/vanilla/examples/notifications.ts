/**
 * System Notifications Example for Electron Desktop Apps
 *
 * This file demonstrates how to show native OS notifications from your desktop app.
 *
 * USAGE:
 * 1. Copy the preload code to your preload.ts file
 * 2. Copy the main process code to your main.ts file
 * 3. Use the React hooks and components as templates
 */

// ============================================================================
// STEP 1: Add to preload.ts
// ============================================================================

import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // Show basic notification
  showNotification: (title: string, body: string) =>
    ipcRenderer.invoke('show-notification', title, body),

  // Show notification with actions
  showNotificationWithActions: (options: {
    title: string;
    body: string;
    actions?: { type: string; text: string }[];
  }) => ipcRenderer.invoke('show-notification-with-actions', options),

  // Listen for notification clicks
  onNotificationClick: (callback: (notificationId: string) => void) => {
    ipcRenderer.on('notification-click', (_event, id) => callback(id));
  },

  // Listen for notification action clicks
  onNotificationAction: (callback: (action: { id: string; actionType: string }) => void) => {
    ipcRenderer.on('notification-action', (_event, data) => callback(data));
  },

  // Remove notification listeners
  removeNotificationListeners: () => {
    ipcRenderer.removeAllListeners('notification-click');
    ipcRenderer.removeAllListeners('notification-action');
  }
});

// ============================================================================
// STEP 2: Add to main.ts
// ============================================================================

import { app, Notification, ipcMain } from 'electron';
import path from 'path';

// Store active notifications
const activeNotifications = new Map<string, Notification>();

// Basic notification
ipcMain.handle('show-notification', async (event, title, body) => {
  if (!Notification.isSupported()) {
    return { success: false, error: 'Notifications not supported' };
  }

  const notification = new Notification({
    title,
    body,
    icon: path.join(__dirname, '../assets/icon.png'), // Your app icon
    silent: false
  });

  notification.show();

  return { success: true };
});

// Notification with actions (Windows/macOS)
ipcMain.handle('show-notification-with-actions', async (event, options) => {
  if (!Notification.isSupported()) {
    return { success: false, error: 'Notifications not supported' };
  }

  const notificationId = Date.now().toString();

  const notification = new Notification({
    title: options.title,
    body: options.body,
    icon: path.join(__dirname, '../assets/icon.png'),
    actions: options.actions || [],
    silent: false
  });

  // Handle notification click
  notification.on('click', () => {
    event.sender.send('notification-click', notificationId);
  });

  // Handle action button clicks
  notification.on('action', (event, index) => {
    const action = options.actions?.[index];
    if (action) {
      event.sender.send('notification-action', {
        id: notificationId,
        actionType: action.type
      });
    }
  });

  // Handle notification close
  notification.on('close', () => {
    activeNotifications.delete(notificationId);
  });

  activeNotifications.set(notificationId, notification);
  notification.show();

  return { success: true, id: notificationId };
});

// ============================================================================
// STEP 3: TypeScript Declarations
// ============================================================================

declare global {
  interface Window {
    electronAPI: {
      showNotification: (title: string, body: string) => Promise<{
        success: boolean;
        error?: string;
      }>;
      showNotificationWithActions: (options: {
        title: string;
        body: string;
        actions?: { type: string; text: string }[];
      }) => Promise<{
        success: boolean;
        id?: string;
        error?: string;
      }>;
      onNotificationClick: (callback: (notificationId: string) => void) => void;
      onNotificationAction: (callback: (action: { id: string; actionType: string }) => void) => void;
      removeNotificationListeners: () => void;
    };
  }
}

// ============================================================================
// STEP 4: React Hooks
// ============================================================================

import { useEffect, useCallback } from 'react';

// Hook for basic notifications
export function useNotifications() {
  const showNotification = useCallback(async (title: string, body: string) => {
    if (!window.electronAPI) {
      // Fallback to browser notifications
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(title, { body });
      } else if ('Notification' in window && Notification.permission !== 'denied') {
        const permission = await Notification.requestPermission();
        if (permission === 'granted') {
          new Notification(title, { body });
        }
      }
      return;
    }

    // Desktop - use native notification
    const result = await window.electronAPI.showNotification(title, body);
    if (!result.success) {
      console.error('Failed to show notification:', result.error);
    }
  }, []);

  return { showNotification };
}

// Hook for notifications with actions
export function useNotificationsWithActions() {
  useEffect(() => {
    if (!window.electronAPI) return;

    const handleClick = (id: string) => {
      console.log('Notification clicked:', id);
      // Handle notification click
    };

    const handleAction = (action: { id: string; actionType: string }) => {
      console.log('Notification action:', action);
      // Handle specific action
    };

    window.electronAPI.onNotificationClick(handleClick);
    window.electronAPI.onNotificationAction(handleAction);

    return () => {
      window.electronAPI?.removeNotificationListeners();
    };
  }, []);

  const showNotificationWithActions = useCallback(async (
    title: string,
    body: string,
    actions: { type: string; text: string }[]
  ) => {
    if (!window.electronAPI) {
      // Web fallback - basic notification only (actions not supported)
      if ('Notification' in window) {
        new Notification(title, { body });
      }
      return;
    }

    const result = await window.electronAPI.showNotificationWithActions({
      title,
      body,
      actions
    });

    if (!result.success) {
      console.error('Failed to show notification:', result.error);
    }

    return result.id;
  }, []);

  return { showNotificationWithActions };
}

// ============================================================================
// STEP 5: Example React Components
// ============================================================================

import React from 'react';

// Example 1: Task completion notification
export function TaskCompleteNotifier({ taskName }: { taskName: string }) {
  const { showNotification } = useNotifications();

  const notifyComplete = async () => {
    await showNotification(
      'Task Complete',
      `${taskName} has finished successfully!`
    );
  };

  return (
    <button onClick={notifyComplete}>
      Mark Task Complete
    </button>
  );
}

// Example 2: Download complete notification with action
export function DownloadCompleteNotifier({ filename }: { filename: string }) {
  const { showNotificationWithActions } = useNotificationsWithActions();

  const notifyDownloadComplete = async () => {
    await showNotificationWithActions(
      'Download Complete',
      `${filename} has been downloaded successfully`,
      [
        { type: 'open', text: 'Open' },
        { type: 'show-in-folder', text: 'Show in Folder' }
      ]
    );
  };

  return (
    <button onClick={notifyDownloadComplete}>
      Download File
    </button>
  );
}

// Example 3: Progress notification
export function ProgressNotifier() {
  const { showNotification } = useNotifications();
  const [progress, setProgress] = React.useState(0);

  React.useEffect(() => {
    if (progress === 100) {
      showNotification(
        'Export Complete',
        'Your data has been exported successfully!'
      );
    }
  }, [progress, showNotification]);

  return (
    <div>
      <div>Progress: {progress}%</div>
      <button onClick={() => setProgress(100)}>
        Complete Export
      </button>
    </div>
  );
}

// Example 4: Error notification
export function ErrorNotifier({ error }: { error: Error | null }) {
  const { showNotification } = useNotifications();

  React.useEffect(() => {
    if (error) {
      showNotification(
        'Error Occurred',
        error.message || 'An unexpected error occurred'
      );
    }
  }, [error, showNotification]);

  return null;
}

// Example 5: Time-based reminder
export function ReminderNotifier() {
  const { showNotification } = useNotifications();

  const setReminder = (minutes: number, message: string) => {
    setTimeout(() => {
      showNotification('Reminder', message);
    }, minutes * 60 * 1000);
  };

  return (
    <button onClick={() => setReminder(5, 'Time to take a break!')}>
      Set 5-Minute Reminder
    </button>
  );
}

// ============================================================================
// STEP 6: Advanced - Notification Manager
// ============================================================================

class NotificationManager {
  private static instance: NotificationManager;
  private queue: Array<{ title: string; body: string; priority: number }> = [];
  private isShowing = false;

  static getInstance() {
    if (!NotificationManager.instance) {
      NotificationManager.instance = new NotificationManager();
    }
    return NotificationManager.instance;
  }

  async notify(title: string, body: string, priority = 0) {
    this.queue.push({ title, body, priority });
    this.queue.sort((a, b) => b.priority - a.priority);

    if (!this.isShowing) {
      await this.processQueue();
    }
  }

  private async processQueue() {
    while (this.queue.length > 0) {
      const notification = this.queue.shift();
      if (!notification) break;

      this.isShowing = true;

      if (window.electronAPI) {
        await window.electronAPI.showNotification(notification.title, notification.body);
      } else if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(notification.title, { body: notification.body });
      }

      // Wait a bit between notifications to avoid spam
      await new Promise(resolve => setTimeout(resolve, 1000));
    }

    this.isShowing = false;
  }
}

// Usage:
export function useNotificationManager() {
  return NotificationManager.getInstance();
}

// ============================================================================
// USAGE TIPS
// ============================================================================

/**
 * 1. Permission Handling:
 *    - Desktop apps have notification permission by default
 *    - Web apps need to request permission
 *    - Always provide graceful fallbacks
 *
 * 2. Notification Frequency:
 *    - Don't spam users with notifications
 *    - Group similar notifications
 *    - Respect user's notification settings
 *
 * 3. Actions (Windows/macOS):
 *    - Limit to 2-3 actions per notification
 *    - Use clear, action-oriented text
 *    - Handle action clicks appropriately
 *
 * 4. Icons and Images:
 *    - Use your app icon for brand consistency
 *    - Keep images small and optimized
 *    - Provide fallbacks if images fail to load
 *
 * 5. Cross-Platform Considerations:
 *    - macOS: Notifications go to Notification Center
 *    - Windows: Notifications go to Action Center
 *    - Linux: Depends on desktop environment
 *    - Web: Browser notifications (different API)
 *
 * 6. Best Practices:
 *    - Keep titles short and descriptive
 *    - Body text should be concise
 *    - Use notifications for important events only
 *    - Allow users to disable notifications in settings
 */
