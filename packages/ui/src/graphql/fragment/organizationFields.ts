import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationTagFields on Tag {
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
            ...organizationTagFields
        }
        stars
        isStarred
        role
    }
`