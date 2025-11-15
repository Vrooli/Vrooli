/**
 * Shared types for scenario inventory components
 */

export interface ScenarioDesktopStatus {
  name: string;
  display_name?: string;
  has_desktop: boolean;
  desktop_path?: string;
  version?: string;
  platforms?: string[];
  built?: boolean;
  dist_path?: string;
  last_modified?: string;
  package_size?: number;
}

export interface ScenariosResponse {
  scenarios: ScenarioDesktopStatus[];
  stats: {
    total: number;
    with_desktop: number;
    built: number;
    web_only: number;
  };
}

export type FilterStatus = "all" | "desktop" | "web";
