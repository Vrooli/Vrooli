/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ImageUpdate } from "./globalTypes";

// ====================================================
// GraphQL mutation operation: updateImages
// ====================================================

export interface updateImages {
  updateImages: boolean;
}

export interface updateImagesVariables {
  data: ImageUpdate[];
  deleting?: string[] | null;
  label?: string | null;
}
