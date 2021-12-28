/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentUpdate
// ====================================================

export interface commentUpdate_commentUpdate {
  __typename: "Comment";
  id: string;
  text: string | null;
  created_at: any;
  updated_at: any;
  userId: string | null;
  organizationId: string | null;
  projectId: string | null;
  resourceId: string | null;
  routineId: string | null;
  standardId: string | null;
}

export interface commentUpdate {
  commentUpdate: commentUpdate_commentUpdate;
}

export interface commentUpdateVariables {
  input: CommentInput;
}
