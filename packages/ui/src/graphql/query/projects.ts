import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectsQuery = gql`
    ${projectFields}
    query projects($input: ProjectSearchInput!) {
        projects(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...projectFields
                }
            }
        }
    }
`