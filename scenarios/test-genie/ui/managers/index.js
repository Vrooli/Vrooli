/**
 * Managers Module Index
 * Central export point for all UI management modules
 */

// NavigationManager - Page navigation and routing
export { NavigationManager, navigationManager } from './NavigationManager.js';

// DialogManager - Dialog and overlay management
export { DialogManager, dialogManager } from './DialogManager.js';

// SelectionManager - Multi-select checkbox management
export { SelectionManager, selectionManager } from './SelectionManager.js';

// NotificationManager - Toast notification system
export { NotificationManager, notificationManager } from './NotificationManager.js';

// Export default object with all manager singletons
export default {
    navigationManager,
    dialogManager,
    selectionManager,
    notificationManager
};
