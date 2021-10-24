/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL fragment: imageFields
// ====================================================

export interface imageFields_files {
  __typename: "ImageFile";
  src: string;
  width: number;
  height: number;
}

export interface imageFields {
  __typename: "Image";
  hash: string;
  alt: string | null;
  description: string | null;
  files: imageFields_files[] | null;
}
