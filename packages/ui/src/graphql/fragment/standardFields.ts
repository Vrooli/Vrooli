import { gql } from 'graphql-tag';

export const standardFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment standardFields on Standard {
        id
        name
        description
        type
        schema
        default
        isFile
        created_at
        tags {
            ...tagFields
        }
    }
`