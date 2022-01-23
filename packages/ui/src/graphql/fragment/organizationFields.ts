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
        bio
        created_at
        tags {
            ...tagFields
        }
        stars
        isStarred
    }
`