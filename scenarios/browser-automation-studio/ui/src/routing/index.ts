/**
 * Routing module - handles navigation state and URL management
 *
 * This is the SINGLE SOURCE OF TRUTH for app navigation.
 * Components should use useAppNavigation() instead of managing their own
 * navigation state or directly manipulating window.history.
 */
export {
  useAppNavigation,
  transformWorkflow,
  type AppView,
  type DashboardTab,
  type DashboardNavigationOptions,
  type NormalizedWorkflow,
  type NavigationState,
  type NavigationActions,
} from "./useAppNavigation";
