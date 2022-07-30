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
        commentsCount
        handle
        stars
        isOpenToNewMembers
        isPrivate
        isStarred
        membersCount
        reportsCount
        permissionsOrganization {
            canAddMembers
            canDelete
            canEdit
            canStar
            canReport
            isMember
        }
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