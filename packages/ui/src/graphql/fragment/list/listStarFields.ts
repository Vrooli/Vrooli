import { gql } from 'graphql-tag';
import { organizationName, projectName, routineName, standardName, tagTranslation, userName } from 'graphql/partial';

export const listStarFields = gql`
    fragment listStarTagFields on Tag {
        id
        created_at
        isStarred
        stars
        tag
        translations ${tagTranslation}
    }
    fragment listStarCommentFields on Comment {
        id
        created_at
        updated_at
        score
        isUpvoted
        isStarred
        commentedOn {
            ... on Project ${projectName}
            ... on Routine ${routineName}
            ... on Standard ${standardName}
        }
        creator {
            ... on Organization ${organizationName}
            ... on User ${userName}
        }
        permissionsComment {
            canDelete
            canEdit
            canStar
            canReply
            canReport
            canVote
        }
        translations {
            id
            language
            text
        }
    }
    fragment listStarOrganizationFields on Organization {
        id
        handle
        stars
        isPrivate
        isStarred
        permissionsOrganization {
            canAddMembers
            canDelete
            canEdit
            canStar
            canReport
            isMember
        }
        tags {
            ...listStarTagFields
        }
        translations { 
            id
            language
            name
            bio
        }
    }
    fragment listStarProjectFields on Project {
        id
        handle
        score
        stars
        isPrivate
        isUpvoted
        isStarred
        permissionsProject {
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        tags {
            ...listStarTagFields
        }
        translations {
            id
            language
            name
            description
        }
    }
    fragment listStarRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        isAutomatable
        isDeleted
        isInternal
        isPrivate
        isComplete
        isStarred
        isUpvoted
        score
        simplicity
        stars
        permissionsRoutine {
            canDelete
            canEdit
            canFork
            canStar
            canReport
            canRun
            canVote
        }
        tags {
            ...listStarTagFields
        }
        translations {
            id
            language
            description
            title
        }
    }
    fragment listStarStandardFields on Standard {
        id
        score
        stars
        isDeleted
        isPrivate
        isUpvoted
        isStarred
        name
        permissionsStandard {
            canDelete
            canEdit
            canStar
            canReport
            canVote
        }
        tags {
            ...listStarTagFields
        }
        translations {
            id
            language
            description
        }
    }
    fragment listStarUserFields on User {
        id
        handle
        name
        stars
        isStarred
    }
    fragment listStarFields on Star {
        id
        to {
            ... on Comment {
                ...listStarCommentFields
            }
            ... on Organization {
                ...listStarOrganizationFields
            }
            ... on Project {
                ...listStarProjectFields
            }
            ... on Routine {
                ...listStarRoutineFields
            }
            ... on Standard {
                ...listStarStandardFields
            }
            ... on User {
                ...listStarUserFields
            }
            ... on Tag {
                ...listStarTagFields
            }
        }
    }
`