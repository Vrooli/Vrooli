/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { VoteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: vote
// ====================================================

export interface vote_vote {
  __typename: "Success";
  success: boolean | null;
}

export interface vote {
  vote: vote_vote;
}

export interface voteVariables {
  input: VoteInput;
}
