import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment organizationFields on Organization {
        id
        created_at
        isOpenToNewMembers
        isStarred
        role
        stars
        tags {
            ...organizationTagFields
        }
        translations {
            id
            language
            bio
            name
        }
    }
`