/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: customerSessionFields
// ====================================================

export interface customerSessionFields_roles_role {
  __typename: "Role";
  title: string;
  description?: string | null;
}

export interface customerSessionFields_roles {
  __typename: "CustomerRole";
  role: customerSessionFields_roles_role;
}

export interface customerSessionFields {
  __typename: "Customer";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: customerSessionFields_roles[];
}
