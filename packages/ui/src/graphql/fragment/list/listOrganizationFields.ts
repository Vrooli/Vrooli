import { gql } from 'graphql-tag';

export const listOrganizationFields = gql`
    fragment listOrganizationTagFields on Tag {
        id
        created_at
        isStarred
        stars
        tag
        translations {
            id
            language
            description
        }
    }
    fragment listOrganizationFields on Organization {
        id
        handle
        stars
        isStarred
        role
        tags {
            ...listOrganizationTagFields
        }
        translations { 
            id
            language
            name
            bio
        }
    }
`