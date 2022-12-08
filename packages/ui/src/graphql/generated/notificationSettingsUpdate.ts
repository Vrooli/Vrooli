/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { NotificationSettingsUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: notificationSettingsUpdate
// ====================================================

export interface notificationSettingsUpdate_notificationSettingsUpdate_categories {
  __typename: "NotificationSettingsCategory";
  category: string;
  enabled: boolean;
  dailyLimit: number | null;
  toEmails: boolean | null;
  toSms: boolean | null;
  toPush: boolean | null;
}

export interface notificationSettingsUpdate_notificationSettingsUpdate {
  __typename: "NotificationSettings";
  categories: notificationSettingsUpdate_notificationSettingsUpdate_categories[] | null;
  dailyLimit: number | null;
  enabled: boolean;
  includedEmails: string[] | null;
  includedPush: string[] | null;
  includedSms: string[] | null;
  toEmails: boolean | null;
  toPush: boolean | null;
  toSms: boolean | null;
}

export interface notificationSettingsUpdate {
  notificationSettingsUpdate: notificationSettingsUpdate_notificationSettingsUpdate;
}

export interface notificationSettingsUpdateVariables {
  input: NotificationSettingsUpdateInput;
}
