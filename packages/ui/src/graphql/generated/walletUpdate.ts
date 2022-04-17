/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { WalletUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: walletUpdate
// ====================================================

export interface walletUpdate_walletUpdate_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface walletUpdate_walletUpdate {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: walletUpdate_walletUpdate_handles[];
  verified: boolean;
}

export interface walletUpdate {
  walletUpdate: walletUpdate_walletUpdate;
}

export interface walletUpdateVariables {
  input: WalletUpdateInput;
}
