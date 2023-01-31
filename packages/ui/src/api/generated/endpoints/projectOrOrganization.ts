import gql from 'graphql-tag';
import { Project_list } from '../fragments/Project_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { Organization_list } from '../fragments/Organization_list';

export const projectOrOrganizationFindMany = gql`...${Project_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${Organization_list}

query projectOrOrganizations($input: ProjectOrOrganizationSearchInput!) {
  projectOrOrganizations(input: $input) {
    edges {
        cursor
        node {
            ... on Project {
            }
            ... on Organization {
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

