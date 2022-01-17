import { gql } from 'graphql-tag';

export const tagFields = gql`
    fragment tagFields on Tag {
        id
        tag
        description
        created_at
        stars
        score
        isUpvoted
    }
`