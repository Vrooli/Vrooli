/**
 * Shared types for scenario inventory components
 */

export interface DesktopConnectionConfig {
  proxy_url?: string;
  server_url?: string;
  api_url?: string;
  deployment_mode?: string;
  auto_manage_vrooli?: boolean;
  vrooli_binary_path?: string;
  server_type?: string;
  bundle_manifest_path?: string;
  updated_at?: string;
}

export interface DesktopBuildArtifact {
  platform?: string;
  file_name: string;
  size_bytes?: number;
  modified_at?: string;
  absolute_path?: string;
  relative_path?: string;
}

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
  connection_config?: DesktopConnectionConfig;
  build_artifacts?: DesktopBuildArtifact[];
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
