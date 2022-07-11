import { gql } from 'graphql-tag';

export const tagFields = gql`
    fragment tagFields on Tag {
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