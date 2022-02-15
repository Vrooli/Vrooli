/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { StarInput } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: star
// ====================================================

export interface star_star {
  __typename: "Success";
  success: boolean | null;
}

export interface star {
  star: star_star;
}

export interface starVariables {
  input: StarInput;
}
