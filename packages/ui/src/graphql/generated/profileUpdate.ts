/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileUpdate
// ====================================================

export interface profileUpdate_profileUpdate_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface profileUpdate_profileUpdate_wallets {
  __typename: "Wallet";
  publicAddress: string;
  verified: boolean;
}

export interface profileUpdate_profileUpdate_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profileUpdate_profileUpdate_hiddenTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profileUpdate_profileUpdate {
  __typename: "Profile";
  id: string;
  username: string | null;
  bio: string | null;
  emails: profileUpdate_profileUpdate_emails[];
  wallets: profileUpdate_profileUpdate_wallets[];
  theme: string;
  starredTags: profileUpdate_profileUpdate_starredTags[] | null;
  hiddenTags: profileUpdate_profileUpdate_hiddenTags[] | null;
}

export interface profileUpdate {
  profileUpdate: profileUpdate_profileUpdate;
}

export interface profileUpdateVariables {
  input: ProfileUpdateInput;
}
