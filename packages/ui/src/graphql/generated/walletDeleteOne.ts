/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletDeleteOne
// ====================================================

export interface walletDeleteOne_walletDeleteOne {
  __typename: "Success";
  success: boolean | null;
}

export interface walletDeleteOne {
  walletDeleteOne: walletDeleteOne_walletDeleteOne;
}

export interface walletDeleteOneVariables {
  input: DeleteOneInput;
}
