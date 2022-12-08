/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: notificationSettingsFields
// ====================================================

export interface notificationSettingsFields_categories {
  __typename: "NotificationSettingsCategory";
  category: string;
  enabled: boolean;
  dailyLimit: number | null;
  toEmails: boolean | null;
  toSms: boolean | null;
  toPush: boolean | null;
}

export interface notificationSettingsFields {
  __typename: "NotificationSettings";
  categories: notificationSettingsFields_categories[] | null;
  dailyLimit: number | null;
  enabled: boolean;
  includedEmails: string[] | null;
  includedPush: string[] | null;
  includedSms: string[] | null;
  toEmails: boolean | null;
  toPush: boolean | null;
  toSms: boolean | null;
}
