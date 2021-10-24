/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { CustomerInput, AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: updateCustomer
// ====================================================

export interface updateCustomer_updateCustomer_roles_role {
  __typename: "Role";
  title: string;
  description: string | null;
}

export interface updateCustomer_updateCustomer_roles {
  __typename: "CustomerRole";
  role: updateCustomer_updateCustomer_roles_role;
}

export interface updateCustomer_updateCustomer {
  __typename: "Customer";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: updateCustomer_updateCustomer_roles[];
}

export interface updateCustomer {
  updateCustomer: updateCustomer_updateCustomer;
}

export interface updateCustomerVariables {
  input: CustomerInput;
  currentPassword: string;
  newPassword?: string | null;
}
