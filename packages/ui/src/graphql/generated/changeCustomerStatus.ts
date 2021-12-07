/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { AccountStatus } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: changeCustomerStatus
// ====================================================

export interface changeCustomerStatus {
  changeCustomerStatus: boolean | null;
}

export interface changeCustomerStatusVariables {
  id: string;
  status: AccountStatus;
}
