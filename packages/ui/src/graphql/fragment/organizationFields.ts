import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment organizationFields on Organization {
        id
        name
        description
        created_at
        tags {
            ...tagFields
        }
    }
`