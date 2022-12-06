/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { PushDeviceUpdateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: pushDeviceUpdate
// ====================================================

export interface pushDeviceUpdate_pushDeviceUpdate {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface pushDeviceUpdate {
  pushDeviceUpdate: pushDeviceUpdate_pushDeviceUpdate;
}

export interface pushDeviceUpdateVariables {
  input: PushDeviceUpdateInput;
}
