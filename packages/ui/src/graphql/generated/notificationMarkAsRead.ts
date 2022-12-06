/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { FindByIdInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: notificationMarkAsRead
// ====================================================

export interface notificationMarkAsRead_notificationMarkAsRead {
  __typename: "Success";
  success: boolean;
}

export interface notificationMarkAsRead {
  notificationMarkAsRead: notificationMarkAsRead_notificationMarkAsRead;
}

export interface notificationMarkAsReadVariables {
  input: FindByIdInput;
}
