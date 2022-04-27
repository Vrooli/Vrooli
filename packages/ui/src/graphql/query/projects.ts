import { gql } from 'graphql-tag';
import { listProjectFields } from 'graphql/fragment';

export const projectsQuery = gql`
    ${listProjectFields}
    query projects($input: ProjectSearchInput!) {
        projects(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listProjectFields
                }
            }
        }
    }
`