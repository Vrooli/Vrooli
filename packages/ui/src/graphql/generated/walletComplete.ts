/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { WalletCompleteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletComplete
// ====================================================

export interface walletComplete_walletComplete_roles {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface walletComplete_walletComplete {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: walletComplete_walletComplete_roles[];
}

export interface walletComplete {
  walletComplete: walletComplete_walletComplete;
}

export interface walletCompleteVariables {
  input: WalletCompleteInput;
}
