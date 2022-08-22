import { gql } from 'graphql-tag';
import { listProjectFields, listOrganizationFields } from 'graphql/fragment';

export const projectOrOrganizationsQuery = gql`
    ${listProjectFields}
    ${listOrganizationFields}
    query projectOrOrganizations($input: ProjectOrOrganizationSearchInput!) {
        projectOrOrganizations(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ... on Project {
                        ...listProjectFields
                    }
                    ... on Organization {
                        ...listOrganizationFields
                    }
                }
            }
        }
    }
`