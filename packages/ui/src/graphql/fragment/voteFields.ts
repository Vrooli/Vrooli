import { gql } from 'graphql-tag';

export const voteFields = gql`
    fragment voteFields on Vote {
        id
        isUpvote
    }
`