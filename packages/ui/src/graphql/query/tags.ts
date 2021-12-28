import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagsQuery = gql`
    ${tagFields}
    query tags($input: TagSearchInput!) {
        tags(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...tagFields
                }
            }
        }
    }
`