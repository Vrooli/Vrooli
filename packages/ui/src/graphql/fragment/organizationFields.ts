import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationResourceFields on Resource {
        id
        created_at
        index
        link
        updated_at
        usedFor
        translations {
            id
            language
            description
            title
        }
    }
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
        resources {
            ...organizationResourceFields
        }
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