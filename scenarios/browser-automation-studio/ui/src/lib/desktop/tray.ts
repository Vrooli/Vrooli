/**
 * Desktop tray integration utilities for workflow scheduling.
 * Only active when running in Electron (window.desktop available).
 *
 * These utilities communicate with the Electron main process via IPC
 * to update the system tray with schedule information.
 */

import type { WorkflowSchedule } from '@stores/scheduleStore';
import { formatNextRun } from '@stores/scheduleStore';

interface TrayMenuItem {
  label: string;
  action: string;
  enabled?: boolean;
}

interface DesktopTrayAPI {
  updateTooltip: (tooltip: string) => Promise<{ success: boolean }>;
  setBadge: (count: number) => Promise<{ success: boolean }>;
  updateContextMenu: (items: TrayMenuItem[]) => Promise<{ success: boolean }>;
}

interface DesktopAPI {
  tray?: DesktopTrayAPI;
  notify?: (title: string, body: string, options?: NotificationOptions) => void;
}

interface NotificationOptions {
  silent?: boolean;
  urgency?: 'low' | 'normal' | 'critical';
}

declare global {
  interface Window {
    desktop?: DesktopAPI;
  }
}

/**
 * Check if running in Electron desktop environment.
 */
export function isDesktopEnvironment(): boolean {
  return typeof window !== 'undefined' && typeof window.desktop?.tray !== 'undefined';
}

/**
 * Check if desktop notifications are available.
 */
export function hasDesktopNotifications(): boolean {
  return typeof window !== 'undefined' && typeof window.desktop?.notify === 'function';
}

/**
 * Updates the system tray tooltip to show schedule information.
 * Called whenever schedules change.
 */
export async function updateTrayWithSchedules(schedules: WorkflowSchedule[]): Promise<void> {
  if (!isDesktopEnvironment()) return;

  const activeSchedules = schedules.filter(s => s.is_active);
  const nextRunTimes = activeSchedules
    .map(s => s.next_run_at)
    .filter((t): t is string => Boolean(t))
    .sort();
  const nextRun = nextRunTimes[0];

  // Update tooltip
  let tooltip = 'Vrooli Ascension';
  if (activeSchedules.length > 0) {
    tooltip += `\n${activeSchedules.length} active schedule${activeSchedules.length === 1 ? '' : 's'}`;
    if (nextRun) {
      tooltip += `\nNext run: ${formatNextRun(nextRun)}`;
    }
  }

  try {
    await window.desktop?.tray?.updateTooltip(tooltip);
  } catch (err) {
    console.warn('[Tray] Failed to update tooltip:', err);
  }

  // Update context menu with upcoming schedules
  const menuItems: TrayMenuItem[] = activeSchedules
    .filter(s => s.next_run_at)
    .sort((a, b) => {
      const aTime = new Date(a.next_run_at!).getTime();
      const bTime = new Date(b.next_run_at!).getTime();
      return aTime - bTime;
    })
    .slice(0, 3)
    .map(s => ({
      label: `${s.name} - ${formatNextRun(s.next_run_at)}`,
      action: `view-schedule:${s.id}`,
    }));

  if (menuItems.length > 0) {
    menuItems.push({
      label: 'View all schedules...',
      action: 'open-schedules-tab',
    });
  }

  try {
    await window.desktop?.tray?.updateContextMenu(menuItems);
  } catch (err) {
    console.warn('[Tray] Failed to update context menu:', err);
  }
}

/**
 * Sets the dock/taskbar badge for pending schedule count.
 */
export async function updateTrayBadge(pendingCount: number): Promise<void> {
  if (!isDesktopEnvironment()) return;

  try {
    await window.desktop?.tray?.setBadge(pendingCount);
  } catch (err) {
    console.warn('[Tray] Failed to update badge:', err);
  }
}

/**
 * Show a desktop notification for schedule events.
 */
export function showScheduleNotification(
  title: string,
  body: string,
  options?: NotificationOptions
): void {
  if (!hasDesktopNotifications()) {
    // Fallback to browser notifications if available
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification(title, { body });
    }
    return;
  }

  try {
    window.desktop?.notify?.(title, body, options);
  } catch (err) {
    console.warn('[Desktop] Failed to show notification:', err);
  }
}

/**
 * Request notification permission if not already granted.
 * Returns true if notifications are available.
 */
export async function requestNotificationPermission(): Promise<boolean> {
  // Desktop notifications don't need permission
  if (hasDesktopNotifications()) {
    return true;
  }

  // Browser notifications need permission
  if ('Notification' in window) {
    const permission = await Notification.requestPermission();
    return permission === 'granted';
  }

  return false;
}
