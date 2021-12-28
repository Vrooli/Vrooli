/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { VoteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: commentVote
// ====================================================

export interface commentVote_commentVote {
  __typename: "Success";
  success: boolean | null;
}

export interface commentVote {
  commentVote: commentVote_commentVote;
}

export interface commentVoteVariables {
  input: VoteInput;
}
