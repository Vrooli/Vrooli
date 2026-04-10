export interface Resource {
  name: string;
  status: string;
  category: string;
  installed: string;
  last_updated: string;
}

export interface OnboardingProgress {
  id: number;
  user_id: string;
  current_step: number;
  completed_steps: number[];
  config_data: Record<string, unknown>;
  updated_at: string;
}

export interface ResourceHealthStatus {
  name: string;
  status: string;
  category: string;
  available: boolean;
  last_checked: string;
}

export interface ResourceHealthResponse {
  resources: ResourceHealthStatus[];
  total: number;
  healthy_count: number;
  checked_at: string;
}

export interface GlossaryEntry {
  term: string;
  description: string;
  category: string;
}

export interface GlossaryResponse {
  entries: GlossaryEntry[];
  count: number;
  query?: string;
}

export interface SetupOrderEntry {
  name: string;
  category: string;
  order: number;
  dependencies: string[];
}

export interface SetupOrderResponse {
  setup_order: SetupOrderEntry[];
  total: number;
}

export interface ValidationResult {
  valid: boolean;
  errors?: string[];
  warnings?: string[];
}

/** Labels for each wizard step, in order. */
export const STEP_LABELS = ["Welcome", "Select Resources", "Review", "Complete"] as const;

/** Total number of wizard steps. */
export const TOTAL_STEPS = STEP_LABELS.length;
