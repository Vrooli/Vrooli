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

export interface CustomerInput {
  id?: string | null;
  username?: string | null;
  pronouns?: string | null;
  emails?: EmailInput[] | null;
  theme?: string | null;
  status?: AccountStatus | null;
}

export interface EmailInput {
  id?: string | null;
  emailAddress: string;
  receivesDeliveryUpdates?: boolean | null;
  customerId?: string | null;
}

export interface ImageUpdate {
  hash: string;
  alt?: string | null;
  description?: string | null;
}

//==============================================================
// END Enums and Input Objects
//==============================================================
