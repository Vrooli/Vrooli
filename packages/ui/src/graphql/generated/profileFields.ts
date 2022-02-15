/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: profileFields
// ====================================================

export interface profileFields_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
}

export interface profileFields_wallets {
  __typename: "Wallet";
  publicAddress: string;
  verified: boolean;
}

export interface profileFields_starredTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profileFields_hiddenTags {
  __typename: "Tag";
  id: string;
  tag: string;
  description: string | null;
  created_at: any;
  stars: number;
  isStarred: boolean;
}

export interface profileFields {
  __typename: "Profile";
  id: string;
  username: string | null;
  bio: string | null;
  emails: profileFields_emails[];
  wallets: profileFields_wallets[];
  theme: string;
  starredTags: profileFields_starredTags[] | null;
  hiddenTags: profileFields_hiddenTags[] | null;
}
