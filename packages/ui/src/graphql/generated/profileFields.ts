/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: profileFields
// ====================================================

export interface profileFields_emails {
  __typename: "Email";
  id: string;
  emailAddress: string;
  receivesAccountUpdates: boolean;
  receivesBusinessUpdates: boolean;
  verified: boolean;
}

export interface profileFields_pushDevices {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface profileFields_wallets_handles {
  __typename: "Handle";
  id: string;
  handle: string;
}

export interface profileFields_wallets {
  __typename: "Wallet";
  id: string;
  name: string | null;
  publicAddress: string | null;
  stakingAddress: string;
  handles: profileFields_wallets_handles[];
  verified: boolean;
}

export interface profileFields_translations {
  __typename: "UserTranslation";
  id: string;
  language: string;
  bio: string | null;
}

export interface profileFields_schedules {
  __typename: "UserSchedule";
  id: string;
}

export interface profileFields {
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
  emails: profileFields_emails[] | null;
  pushDevices: profileFields_pushDevices[] | null;
  wallets: profileFields_wallets[] | null;
  theme: string | null;
  translations: profileFields_translations[];
  schedules: profileFields_schedules[] | null;
}
