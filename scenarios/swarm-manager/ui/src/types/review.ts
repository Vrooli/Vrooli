/**
 * Review domain types.
 */

export type ReviewStatus = "approved" | "flagged" | "unreviewed";

export interface ReviewAction {
  review_status: ReviewStatus;
  review_comment?: string;
}

export interface ReviewUpdate {
  id: string;
  type: "target" | "requirement";
  module_id?: string;
  review_status: ReviewStatus;
  review_comment?: string;
}
