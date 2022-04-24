import { gql } from 'graphql-tag';
import { listOrganizationFields } from 'graphql/fragment';

export const organizationsQuery = gql`
    ${listOrganizationFields}
    query organizations($input: OrganizationSearchInput!) {
        organizations(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listOrganizationFields
                }
            }
        }
    }
`