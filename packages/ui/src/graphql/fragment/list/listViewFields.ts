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
        isStarred
        role
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
        role
        score
        stars
        isUpvoted
        isStarred
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
        isInternal
        isComplete
        isStarred
        isUpvoted
        role
        score
        simplicity
        stars
        tags {
            ...listViewTagFields
        }
        translations {
            id
            language
            description
            title
        }
        version
    }
    fragment listViewStandardFields on Standard {
        id
        score
        stars
        isUpvoted
        isStarred
        name
        role
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
        lastViewed
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