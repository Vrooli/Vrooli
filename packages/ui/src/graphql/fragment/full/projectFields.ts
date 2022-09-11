import { gql } from 'graphql-tag';

export const projectFields = gql`
    fragment projectResourceListFields on ResourceList {
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
    fragment projectTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment projectFields on Project {
        id
        completedAt
        created_at
        handle
        isComplete
        isPrivate
        isStarred
        isUpvoted
        score
        stars
        permissionsProject {
            canComment
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        resourceLists {
            ...projectResourceListFields
        }
        tags {
            ...projectTagFields
        }
        translations {
            id
            language
            description
            name
        }
        owner {
            ... on Organization {
                id
                handle
                translations {
                    id
                    language
                    name
                }
                permissionsOrganization {
                    canAddMembers
                    canDelete
                    canEdit
                    canStar
                    canReport
                    isMember
                }
            }
            ... on User {
                id
                name
                handle
            }
        }
    }
`