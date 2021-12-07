/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: userContactFields
// ====================================================

export interface userContactFields_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesDeliveryUpdates: boolean;
}

export interface userContactFields {
  __typename: "User";
  id: string;
  username: string;
  emails: userContactFields_emails[];
}
