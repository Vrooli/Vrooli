/* tslint:disable */
/* eslint-disable */
// @generated
// This file was automatically generated and should not be edited.

import { ProjectsQueryInput } from "./globalTypes";

// ====================================================
// GraphQL query operation: projects
// ====================================================

export interface projects_projects {
  __typename: "Project";
  id: string;
  name: string;
  description: string | null;
  created_at: any;
}

export interface projects {
  projects: projects_projects[];
}

export interface projectsVariables {
  input: ProjectsQueryInput;
}
