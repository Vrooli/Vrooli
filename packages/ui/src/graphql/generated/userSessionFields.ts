/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL fragment: userSessionFields
// ====================================================

export interface userSessionFields_roles_role {
  __typename: "Role";
  title: string;
  description?: string | null;
}

export interface userSessionFields_roles {
  __typename: "UserRole";
  role: userSessionFields_roles_role;
}

export interface userSessionFields {
  __typename: "User";
  id: string;
  status: AccountStatus;
  theme: string;
  roles: userSessionFields_roles[];
}
