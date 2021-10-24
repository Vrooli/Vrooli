/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: customerContactFields
// ====================================================

export interface customerContactFields_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesDeliveryUpdates: boolean;
}

export interface customerContactFields {
  __typename: "Customer";
  id: string;
  username: string;
  emails: customerContactFields_emails[];
}
