/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: profile
// ====================================================

export interface profile_profile_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface profile_profile_wallets {
  __typename: "Wallet";
  publicAddress: string;
  verified: boolean;
}

export interface profile_profile_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profile_profile_hiddenTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profile_profile {
  __typename: "Profile";
  id: string;
  username: string | null;
  bio: string | null;
  emails: profile_profile_emails[];
  wallets: profile_profile_wallets[];
  theme: string;
  starredTags: profile_profile_starredTags[] | null;
  hiddenTags: profile_profile_hiddenTags[] | null;
}

export interface profile {
  profile: profile_profile;
}
