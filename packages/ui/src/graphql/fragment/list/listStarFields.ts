import { gql } from 'graphql-tag';

export const listStarFields = gql`
    fragment listStarTagFields on Tag {
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
    fragment listStarOrganizationFields on Organization {
        id
        handle
        stars
        isStarred
        role
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
        role
        score
        stars
        isUpvoted
        isStarred
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
        isInternal
        isComplete
        isStarred
        isUpvoted
        role
        score
        simplicity
        stars
        tags {
            ...listStarTagFields
        }
        translations {
            id
            language
            description
            title
        }
        version
    }
    fragment listStarStandardFields on Standard {
        id
        score
        stars
        isUpvoted
        isStarred
        name
        role
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
        }
    }
`