/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { TagVoteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: tagVote
// ====================================================

export interface tagVote_tagVote {
  __typename: "Success";
  success: boolean | null;
}

export interface tagVote {
  tagVote: tagVote_tagVote;
}

export interface tagVoteVariables {
  input: TagVoteInput;
}
