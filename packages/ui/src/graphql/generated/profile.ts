/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: profile
// ====================================================

export interface profile_profile_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profile_profile_pushDevices {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface profile_profile_wallets_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface profile_profile_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: profile_profile_wallets_handles[];
  verified: boolean;
}

export interface profile_profile_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profile_profile_schedules {
  __typename: "UserSchedule";
  id: string;
}

export interface profile_profile {
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
  emails: profile_profile_emails[] | null;
  pushDevices: profile_profile_pushDevices[] | null;
  wallets: profile_profile_wallets[] | null;
  theme: string | null;
  translations: profile_profile_translations[];
  schedules: profile_profile_schedules[] | null;
}

export interface profile {
  profile: profile_profile;
}
