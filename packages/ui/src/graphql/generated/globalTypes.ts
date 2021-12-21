/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

//==============================================================
// START Enums and Input Objects
//==============================================================

export enum AccountStatus {
  DELETED = "DELETED",
  HARD_LOCKED = "HARD_LOCKED",
  SOFT_LOCKED = "SOFT_LOCKED",
  UNLOCKED = "UNLOCKED",
}

export enum ProjectSortBy {
  AlphabeticalAsc = "AlphabeticalAsc",
  AlphabeticalDesc = "AlphabeticalDesc",
  CommentsAsc = "CommentsAsc",
  CommentsDesc = "CommentsDesc",
  DateCreatedAsc = "DateCreatedAsc",
  DateCreatedDesc = "DateCreatedDesc",
  DateUpdatedAsc = "DateUpdatedAsc",
  DateUpdatedDesc = "DateUpdatedDesc",
  ForksAsc = "ForksAsc",
  ForksDesc = "ForksDesc",
  StarsAsc = "StarsAsc",
  StarsDesc = "StarsDesc",
  VotesAsc = "VotesAsc",
  VotesDesc = "VotesDesc",
}

export enum ResourceFor {
  ORGANIZATION = "ORGANIZATION",
  PROJECT = "PROJECT",
  ROUTINE_CONTEXTUAL = "ROUTINE_CONTEXTUAL",
  ROUTINE_DONATION = "ROUTINE_DONATION",
  ROUTINE_EXTERNAL = "ROUTINE_EXTERNAL",
  USER = "USER",
}

export interface DeleteUserInput {
  id: string;
  password: string;
}

export interface EmailInput {
  id?: string | null;
  emailAddress: string;
  receivesAccountUpdates?: boolean | null;
  receivesBusinessUpdates?: boolean | null;
  userId?: string | null;
}

export interface EmailLogInInput {
  email?: string | null;
  password?: string | null;
  verificationCode?: string | null;
}

export interface EmailRequestPasswordChangeInput {
  email: string;
}

export interface EmailResetPasswordInput {
  id: string;
  code: string;
  newPassword: string;
}

export interface EmailSignUpInput {
  username: string;
  pronouns?: string | null;
  email: string;
  theme: string;
  marketingEmails: boolean;
  password: string;
}

export interface OrganizationInput {
  id?: string | null;
  name: string;
  description?: string | null;
  resources?: ResourceInput[] | null;
}

export interface ProjectInput {
  id?: string | null;
  name: string;
  description?: string | null;
  organizations?: OrganizationInput[] | null;
  users?: UserInput[] | null;
  resources?: ResourceInput[] | null;
}

export interface ProjectsQueryInput {
  userId?: number | null;
  ids?: string[] | null;
  sortBy?: ProjectSortBy | null;
  searchString?: string | null;
  first?: number | null;
  skip?: number | null;
}

export interface ResourceInput {
  id?: string | null;
  name: string;
  description?: string | null;
  link: string;
  displayUrl?: string | null;
  createdFor: ResourceFor;
  forId: string;
}

export interface UpdateUserInput {
  data: UserInput;
  currentPassword: string;
  newPassword?: string | null;
}

export interface UserInput {
  id?: string | null;
  username?: string | null;
  pronouns?: string | null;
  emails?: EmailInput[] | null;
  theme?: string | null;
  status?: AccountStatus | null;
}

export interface WalletCompleteInput {
  publicAddress: string;
  signedMessage: string;
}

export interface WalletInitInput {
  publicAddress: string;
  nonceDescription?: string | null;
}

//==============================================================
// END Enums and Input Objects
//==============================================================
