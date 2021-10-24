/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CustomerInput, AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: addCustomer
// ====================================================

export interface addCustomer_addCustomer_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesDeliveryUpdates: boolean;
}

export interface addCustomer_addCustomer_roles_role {
  __typename: "Role";
  title: string;
}

export interface addCustomer_addCustomer_roles {
  __typename: "CustomerRole";
  role: addCustomer_addCustomer_roles_role;
}

export interface addCustomer_addCustomer {
  __typename: "Customer";
  id: string;
  username: string;
  emails: addCustomer_addCustomer_emails[];
  status: AccountStatus;
  roles: addCustomer_addCustomer_roles[];
}

export interface addCustomer {
  addCustomer: addCustomer_addCustomer;
}

export interface addCustomerVariables {
  input: CustomerInput;
}
