/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { VoteRemoveInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: voteRemove
// ====================================================

export interface voteRemove_voteRemove {
  __typename: "Success";
  success: boolean | null;
}

export interface voteRemove {
  voteRemove: voteRemove_voteRemove;
}

export interface voteRemoveVariables {
  input: VoteRemoveInput;
}
