/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

// ====================================================
// GraphQL query operation: imagesByLabel
// ====================================================

export interface imagesByLabel_imagesByLabel_files {
  __typename: "ImageFile";
  src: string;
  width: number;
  height: number;
}

export interface imagesByLabel_imagesByLabel {
  __typename: "Image";
  hash: string;
  alt: string | null;
  description: string | null;
  files: imagesByLabel_imagesByLabel_files[] | null;
}

export interface imagesByLabel {
  imagesByLabel: imagesByLabel_imagesByLabel[];
}

export interface imagesByLabelVariables {
  label: string;
}
