/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL query operation: customers
// ====================================================

export interface customers_customers_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesDeliveryUpdates: boolean;
}

export interface customers_customers_roles_role {
  __typename: "Role";
  title: string;
}

export interface customers_customers_roles {
  __typename: "CustomerRole";
  role: customers_customers_roles_role;
}

export interface customers_customers {
  __typename: "Customer";
  id: string;
  username: string;
  emails: customers_customers_emails[];
  status: AccountStatus;
  roles: customers_customers_roles[];
}

export interface customers {
  customers: customers_customers[];
}
