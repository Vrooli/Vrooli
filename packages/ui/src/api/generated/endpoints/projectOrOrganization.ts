import gql from 'graphql-tag';
import { Organization_list } from '../fragments/Organization_list';
import { Project_list } from '../fragments/Project_list';

export const projectOrOrganizationFindMany = gql`${Organization_list}
${Project_list}

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
        endCursor
        hasNextPage
    }
  }
}`;

