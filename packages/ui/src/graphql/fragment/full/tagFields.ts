import { gql } from 'graphql-tag';

export const tagFields = gql`
    fragment tagFields on Tag {
        id
        tag
        created_at
        stars
        isStarred
        translations {
            id
            language
            description
        }
    }
`