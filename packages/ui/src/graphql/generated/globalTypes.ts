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

export interface CompleteValidateWalletInput {
  publicAddress: string;
  signedMessage: string;
}

export interface DeleteUserInput {
  id: string;
  password?: string | null;
}

export interface EmailInput {
  id?: string | null;
  emailAddress: string;
  receivesAccountUpdates?: boolean | null;
  receivesBusinessUpdates?: boolean | null;
  userId?: string | null;
}

export interface InitValidateWalletInput {
  publicAddress: string;
  nonceDescription?: string | null;
}

export interface LogInInput {
  email?: string | null;
  password?: string | null;
  verificationCode?: string | null;
}

export interface RequestPasswordChangeInput {
  email: string;
}

export interface ResetPasswordInput {
  id: string;
  code: string;
  newPassword: string;
}

export interface SignUpInput {
  username: string;
  pronouns?: string | null;
  email: string;
  theme: string;
  marketingEmails: boolean;
  password: string;
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

//==============================================================
// END Enums and Input Objects
//==============================================================
