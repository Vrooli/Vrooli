import { gql } from 'graphql-tag';
import { threadFields } from 'graphql/fragment';

export const commentsQuery = gql`
    ${threadFields}
    query comments($input: CommentSearchInput!) {
        comments(input: $input) {
            endCursor
            totalThreads
            threads {
                ...threadFields
            }
        }
    }
`