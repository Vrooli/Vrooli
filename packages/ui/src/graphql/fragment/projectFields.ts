import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment projectFields on Project {
        id
        name
        description
        created_at
        tags {
            ...tagFields
        }
    }
`