/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProfileEmailUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: profileEmailUpdate
// ====================================================

export interface profileEmailUpdate_profileEmailUpdate_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profileEmailUpdate_profileEmailUpdate_pushDevices {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_wallets_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface profileEmailUpdate_profileEmailUpdate_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: profileEmailUpdate_profileEmailUpdate_wallets_handles[];
  verified: boolean;
}

export interface profileEmailUpdate_profileEmailUpdate_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileEmailUpdate_profileEmailUpdate_schedules {
  __typename: "UserSchedule";
  id: string;
}

export interface profileEmailUpdate_profileEmailUpdate {
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
  emails: profileEmailUpdate_profileEmailUpdate_emails[] | null;
  pushDevices: profileEmailUpdate_profileEmailUpdate_pushDevices[] | null;
  wallets: profileEmailUpdate_profileEmailUpdate_wallets[] | null;
  theme: string | null;
  translations: profileEmailUpdate_profileEmailUpdate_translations[];
  schedules: profileEmailUpdate_profileEmailUpdate_schedules[] | null;
}

export interface profileEmailUpdate {
  profileEmailUpdate: profileEmailUpdate_profileEmailUpdate;
}

export interface profileEmailUpdateVariables {
  input: ProfileEmailUpdateInput;
}
