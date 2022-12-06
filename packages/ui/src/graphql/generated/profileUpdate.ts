/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileUpdate
// ====================================================

export interface profileUpdate_profileUpdate_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profileUpdate_profileUpdate_pushDevices {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface profileUpdate_profileUpdate_wallets_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface profileUpdate_profileUpdate_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: profileUpdate_profileUpdate_wallets_handles[];
  verified: boolean;
}

export interface profileUpdate_profileUpdate_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileUpdate_profileUpdate_schedules {
  __typename: "UserSchedule";
  id: string;
}

export interface profileUpdate_profileUpdate {
  __typename: "User";
  id: string;
  handle: string | null;
  isPrivate: boolean;
  isPrivateApis: boolean;
  isPrivateApisCreated: boolean;
  isPrivateMemberships: boolean;
  isPrivateOrganizationsCreated: boolean;
  isPrivateProjects: boolean;
  isPrivateProjectsCreated: boolean;
  isPrivatePullRequests: boolean;
  isPrivateQuestionsAnswered: boolean;
  isPrivateQuestionsAsked: boolean;
  isPrivateQuizzesCreated: boolean;
  isPrivateRoles: boolean;
  isPrivateRoutines: boolean;
  isPrivateRoutinesCreated: boolean;
  isPrivateSmartContracts: boolean;
  isPrivateStandards: boolean;
  isPrivateStandardsCreated: boolean;
  isPrivateStars: boolean;
  isPrivateVotes: boolean;
  name: string;
  emails: profileUpdate_profileUpdate_emails[] | null;
  pushDevices: profileUpdate_profileUpdate_pushDevices[] | null;
  wallets: profileUpdate_profileUpdate_wallets[] | null;
  theme: string | null;
  translations: profileUpdate_profileUpdate_translations[];
  schedules: profileUpdate_profileUpdate_schedules[] | null;
}

export interface profileUpdate {
  profileUpdate: profileUpdate_profileUpdate;
}

export interface profileUpdateVariables {
  input: ProfileUpdateInput;
}
