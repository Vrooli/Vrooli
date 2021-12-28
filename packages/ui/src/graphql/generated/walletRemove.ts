/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { DeleteOneInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletRemove
// ====================================================

export interface walletRemove_walletRemove {
  __typename: "Success";
  success: boolean | null;
}

export interface walletRemove {
  walletRemove: walletRemove_walletRemove;
}

export interface walletRemoveVariables {
  input: DeleteOneInput;
}
