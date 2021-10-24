/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL mutation operation: addImages
// ====================================================

export interface addImages_addImages {
  __typename: "AddImageResponse";
  success: boolean;
  src: string | null;
  hash: string | null;
}

export interface addImages {
  addImages: addImages_addImages[];
}

export interface addImagesVariables {
  files: any[];
  alts?: (string | null)[] | null;
  labels?: string[] | null;
}
