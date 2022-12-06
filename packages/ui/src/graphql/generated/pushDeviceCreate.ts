/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { PushDeviceCreateInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: pushDeviceCreate
// ====================================================

export interface pushDeviceCreate_pushDeviceCreate {
  __typename: "PushDevice";
  id: string;
  expires: any | null;
  name: string | null;
}

export interface pushDeviceCreate {
  pushDeviceCreate: pushDeviceCreate_pushDeviceCreate;
}

export interface pushDeviceCreateVariables {
  input: PushDeviceCreateInput;
}
