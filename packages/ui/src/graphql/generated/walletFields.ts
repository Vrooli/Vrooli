/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: walletFields
// ====================================================

export interface walletFields_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface walletFields {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: walletFields_handles[];
  verified: boolean;
}
