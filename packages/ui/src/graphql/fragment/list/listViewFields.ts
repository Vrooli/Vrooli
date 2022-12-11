import { gql } from 'graphql-tag';

export const listViewFields = gql`
    fragment listViewTagFields on Tag {
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
    fragment listViewOrganizationFields on Organization {
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
            ...listViewTagFields
        }
        translations { 
            id
            language
            name
            bio
        }
    }
    fragment listViewProjectFields on Project {
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
            ...listViewTagFields
        }
        translations {
            id
            language
            name
            description
        }
    }
    fragment listViewRoutineFields on Routine {
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
            ...listViewTagFields
        }
        translations {
            id
            language
            description
            title
        }
    }
    fragment listViewStandardFields on Standard {
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
            ...listViewTagFields
        }
        translations {
            id
            language
            description
            jsonVariable
        }
    }
    fragment listViewUserFields on User {
        id
        handle
        name
        stars
        isStarred
    }
    fragment listViewFields on View {
        id
        lastViewedAt
        title
        to {
            ... on Organization {
                ...listViewOrganizationFields
            }
            ... on Project {
                ...listViewProjectFields
            }
            ... on Routine {
                ...listViewRoutineFields
            }
            ... on Standard {
                ...listViewStandardFields
            }
            ... on User {
                ...listViewUserFields
            }
        }
    }
`