import { gql } from 'graphql-tag';
import { commentFields } from 'graphql/fragment';

export const commentsQuery = gql`
    ${commentFields}
    query comments($input: CommentSearchInput!) {
        comments(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...commentFields
                }
            }
        }
    }
`