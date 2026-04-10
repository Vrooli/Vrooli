/**
 * Prompt Center domain types.
 */

export interface PromptTrace {
  purpose: string;
  prompt: string;
  prompt_revision?: string;
  used_fallback: boolean;
  captured_at: string;
  experiment_id?: string;
  variant_id?: string;
}

export interface PromptCatalogEntry {
  id: string;
  title: string;
  group: "capture" | "backlog" | "execution" | "archive" | "support";
  usage_type: "direct_runtime" | "generated_runtime" | "support_reference";
  source_type: "skill" | "generated";
  trigger: string;
  backlog_kinds?: string[];
  modes?: string[];
  operations?: string[];
  skill_id?: string;
  builder?: string;
  purpose: string;
  output_paths?: string[];
  variable_keys?: string[];
  reference_skill_ids?: string[];
  experiment_id?: string;
}

export interface PromptSkillSummary {
  id: string;
  name: string;
  description: string;
  default_scope?: string;
  draft: boolean;
  updated_at?: string;
  created_at?: string;
  usage_type: "direct_runtime" | "support_reference";
  groups?: string[];
  trigger_count: number;
  impact_summary: string;
  current_content?: string;
  required_missing?: string[];
}

export interface PromptSkillVersion {
  version: number;
  content: string;
  name: string;
  updatedAt: string;
  createdBy?: string;
}


export interface PromptSkillVersions {
  skillId: string;
  current: number;
  versions: PromptSkillVersion[];
}
