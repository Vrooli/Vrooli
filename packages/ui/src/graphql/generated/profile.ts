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

export interface profile_profile {
  __typename: "Profile";
  id: string;
  username: string | null;
  theme: string;
  emails: profile_profile_emails[];
}

export interface profile {
  profile: profile_profile;
}
