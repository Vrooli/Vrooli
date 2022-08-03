import { gql } from 'graphql-tag';

export const organizationFields = gql`
    fragment organizationResourceListFields on ResourceList {
        id
        created_at
        index
        usedFor
        translations {
            id
            language
            description
            title
        }
        resources {
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
    }
    fragment organizationTagFields on Tag {
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
        handle
        isOpenToNewMembers
        isStarred
        role
        stars
        reportsCount
        resourceLists {
            ...organizationResourceListFields
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