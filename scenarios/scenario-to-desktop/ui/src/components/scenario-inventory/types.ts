/**
 * Shared types for scenario inventory components
 *
 * Note: Core domain types (DesktopBuildArtifact, etc.) are now defined in
 * the domain layer (../../domain/types.ts) and re-exported here for
 * backward compatibility.
 */

// Re-export domain types for backward compatibility
export type { DesktopBuildArtifact } from "../../domain/types";

export interface DesktopConnectionConfig {
  proxy_url?: string;
  server_url?: string;
  api_url?: string;
  app_display_name?: string;
  app_description?: string;
  icon?: string;
  deployment_mode?: string;
  auto_manage_vrooli?: boolean;
  vrooli_binary_path?: string;
  server_type?: string;
  bundle_manifest_path?: string;
  updated_at?: string;
}

export interface ScenarioDesktopStatus {
  name: string;
  display_name?: string;
  service_display_name?: string;
  service_description?: string;
  service_icon_path?: string;
  has_desktop: boolean;
  desktop_path?: string;
  version?: string;
  platforms?: string[];
  built?: boolean;
  dist_path?: string;
  last_modified?: string;
  package_size?: number;
  connection_config?: DesktopConnectionConfig;
  build_artifacts?: DesktopBuildArtifact[];
  artifacts_source?: string;
  artifacts_path?: string;
  artifacts_expected_path?: string;
  record_id?: string;
  record_output_path?: string;
  record_location_mode?: string;
  record_updated_at?: string;
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
