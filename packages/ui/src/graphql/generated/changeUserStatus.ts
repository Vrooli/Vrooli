/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: changeUserStatus
// ====================================================

export interface changeUserStatus {
  changeUserStatus: boolean | null;
}

export interface changeUserStatusVariables {
  id: string;
  status: AccountStatus;
}
