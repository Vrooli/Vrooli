/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FeedbackInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: feedbackAdd
// ====================================================

export interface feedbackAdd_feedbackAdd {
  __typename: "Success";
  success: boolean | null;
}

export interface feedbackAdd {
  feedbackAdd: feedbackAdd_feedbackAdd;
}

export interface feedbackAddVariables {
  input: FeedbackInput;
}
