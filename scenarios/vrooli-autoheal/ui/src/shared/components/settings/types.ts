import type { CheckInfo } from "../../../lib/api";

export type SettingsTab = "general" | "checks" | "monitoring" | "import-export";

export interface CheckWithConfig extends CheckInfo {
  config: {
    enabled: boolean;
    autoHeal: boolean;
  };
}

export type CategoryIcon = React.ComponentType<{ size?: string | number; className?: string }>;
