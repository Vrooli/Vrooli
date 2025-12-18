/**
 * Navigation type definitions.
 *
 * These types define the possible views and tabs in the application.
 * Used by components that need to understand the current navigation state
 * without depending on routing implementation details.
 */

/**
 * Possible top-level views in the application.
 * Maps to route paths in routes.tsx.
 */
export type AppView =
  | "dashboard"
  | "project-detail"
  | "project-workflow"
  | "settings"
  | "all-workflows"
  | "all-executions"
  | "record-mode";

/**
 * Dashboard tab options.
 * These map to URL query parameters: /?tab=<value>
 */
export type DashboardTab = "home" | "executions" | "exports" | "projects" | "schedules";

/**
 * Settings tab options.
 * These map to URL query parameters: /settings?tab=<value>
 */
export type SettingsTab =
  | "display"
  | "replay"
  | "branding"
  | "workflow"
  | "apikeys"
  | "data"
  | "sessions"
  | "subscription"
  | "schedules";
