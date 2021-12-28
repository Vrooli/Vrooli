/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CommentInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentAdd
// ====================================================

export interface commentAdd_commentAdd {
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

export interface commentAdd {
  commentAdd: commentAdd_commentAdd;
}

export interface commentAddVariables {
  input: CommentInput;
}
