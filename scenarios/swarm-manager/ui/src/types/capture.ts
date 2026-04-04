/**
 * Capture domain types.
 */

import type { BacklogKind } from "./backlog";

/**
 * Lifecycle status of a capture.
 */
export type CaptureStatus = "classifying" | "classified" | "failed";


/**
 * A raw, unclassified thought from the user.
 */
export interface Capture {
  id: string;
  text: string;
  attachments: string[];
  created: string;
  status: CaptureStatus;
  classification: CaptureClassification | null;
  note?: string;
}

/**
 * Classification result — contains 1-N suggested backlog items from a single capture.
 */
export interface CaptureClassification {
  items: CaptureClassificationItem[];
  classifiedAt: string;
}

/**
 * One suggested backlog item extracted from a capture.
 */
export interface CaptureClassificationItem {
  kind: BacklogKind;
  title: string;
  description: string;
  priority: number;
  tags: string[];
  confidence: number;
}
