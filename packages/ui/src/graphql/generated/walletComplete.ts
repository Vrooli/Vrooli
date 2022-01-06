/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { WalletCompleteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletComplete
// ====================================================

export interface walletComplete_walletComplete {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: string[];
}

export interface walletComplete {
  walletComplete: walletComplete_walletComplete;
}

export interface walletCompleteVariables {
  input: WalletCompleteInput;
}
