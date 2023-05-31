import gql from "graphql-tag";
import { Label_list } from "../fragments/Label_list";
import { Organization_list } from "../fragments/Organization_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Project_list } from "../fragments/Project_list";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const projectOrOrganizationFindMany = gql`${Label_list}
${Organization_list}
${Organization_nav}
${Project_list}
${Tag_list}
${User_nav}

query projectOrOrganizations($input: ProjectOrOrganizationSearchInput!) {
  projectOrOrganizations(input: $input) {
    edges {
        cursor
        node {
            ... on Project {
                ...Project_list
            }
            ... on Organization {
                ...Organization_list
            }
        }
    }
    pageInfo {
        hasNextPage
        endCursorProject
        endCursorOrganization
    }
  }
}`;

