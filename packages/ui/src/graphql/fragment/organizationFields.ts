import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationTagFields on Tag {
        id
        description
        tag
    }
    fragment organizationFields on Organization {
        id
        bio
        created_at
        isOpenToNewMembers
        isStarred
        name
        role
        stars
        tags {
            ...organizationTagFields
        }
    }
`