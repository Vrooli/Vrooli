/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { VoteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: voteAdd
// ====================================================

export interface voteAdd_voteAdd {
  __typename: "Success";
  success: boolean | null;
}

export interface voteAdd {
  voteAdd: voteAdd_voteAdd;
}

export interface voteAddVariables {
  input: VoteInput;
}
