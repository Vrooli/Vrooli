import { gql } from 'graphql-tag';
import { organizationFields } from 'graphql/fragment';

export const organizationsQuery = gql`
    ${organizationFields}
    query organizations($input: OrganizationSearchInput!) {
        organizations(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...organizationFields
                }
            }
        }
    }
`