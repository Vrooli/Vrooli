/**
 * Archive / Operational Targets domain types.
 */

import type { ReviewStatus } from "./review";

export interface ArchiveTarget {
  id: string;
  criticality: string;
  title: string;
  notes: string;
  status: string;
  linked_requirement_ids: string[];
  reviewed_at?: string;
  review_comment?: string;
  review_status?: ReviewStatus;
}

export interface ArchiveRequirement {
  id: string;
  title: string;
  description: string;
  status: string;
  category: string;
  prd_ref: string;
  reviewed_at?: string;
  review_comment?: string;
  review_status?: ReviewStatus;
}

export interface ArchiveRequirementGroup {
  id: string;
  name: string;
  requirements: ArchiveRequirement[];
  children: ArchiveRequirementGroup[];
}

export interface ArchiveRequirementRecord {
  id: string;
  title: string;
  description: string;
  status: string;
  category: string;
  prd_ref: string;
  criticality?: string;
  validation?: Array<{ type: string; phase: string; status: string; ref: string }>;
  dependencies?: string[];
  notes?: string;
}

export interface ModuleFormValues {
  id: string;
  title: string;
  description: string;
}

export interface ArchiveTargetFormValues {
  id: string;
  criticality: string;
  title: string;
  notes: string;
  status: string;
  linked_requirement_ids: string[];
}

export interface ArchiveTargetsResponse {
  targets: ArchiveTarget[];
  requirements: ArchiveRequirementGroup[];
  has_archive: boolean;
}
