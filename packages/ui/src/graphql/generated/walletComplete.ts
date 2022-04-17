/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { WalletCompleteInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletComplete
// ====================================================

export interface walletComplete_walletComplete_session {
  __typename: "Session";
  id: string | null;
  theme: string;
  roles: string[];
  languages: string[] | null;
}

export interface walletComplete_walletComplete_wallet_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface walletComplete_walletComplete_wallet {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: walletComplete_walletComplete_wallet_handles[];
  verified: boolean;
}

export interface walletComplete_walletComplete {
  __typename: "WalletComplete";
  firstLogIn: boolean;
  session: walletComplete_walletComplete_session | null;
  wallet: walletComplete_walletComplete_wallet | null;
}

export interface walletComplete {
  walletComplete: walletComplete_walletComplete;
}

export interface walletCompleteVariables {
  input: WalletCompleteInput;
}
