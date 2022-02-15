/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FeedbackInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: feedbackCreate
// ====================================================

export interface feedbackCreate_feedbackCreate {
  __typename: "Success";
  success: boolean | null;
}

export interface feedbackCreate {
  feedbackCreate: feedbackCreate_feedbackCreate;
}

export interface feedbackCreateVariables {
  input: FeedbackInput;
}
