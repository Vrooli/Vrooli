import { gql } from 'graphql-tag';

export const deepRoutineFields = gql`
    fragment deepRoutineTagFields on Tag {
        id
        description
        tag
    }
    fragment deepRoutineInputFields on InputItem {
        id
        standard {
            id
            default
            description
            isFile
            name
            schema
            tags {
                ...deepRoutineTagFields
            }
        }
    }
    fragment deepRoutineOutputFields on OutputItem {
        id
        standard {
            id
            default
            description
            isFile
            name
            schema
            tags {
                ...deepRoutineTagFields
            }
        }
    }
    fragment deepRoutineNodeFields on Node {
        id
        columnIndex,
        created_at
        description
        rowIndex,
        title
        type
        updated_at
        data {
            ... on NodeEnd {
                id
                wasSuccessful
            }
            ... on NodeLoop {
                id
            }
            ... on NodeRoutineList {
                id
                isOptional
                isOrdered
                routines {
                    id
                    title
                    description
                    isOptional
                    routine {
                        id
                        isInternal
                        role
                        title
                    }
                }
            }
        }
    }
    fragment deepRoutineNodeLinkFields on NodeLink {
        id
        fromId
        toId
        conditions {
            id
            description
            title
            when {
                id
                condition
            }
        }
    }
    fragment deepRoutineResourceFields on Resource {
        id
        created_at
        description
        link
        title
        updated_at
    }
    fragment deepRoutineFields on Routine {
        id
        completedAt
        created_at
        description
        inputs {
            ...deepRoutineInputFields
        }
        instructions
        isAutomatable
        isComplete
        isInternal
        isStarred
        isUpvoted
        nodeLinks {
            ...deepRoutineNodeLinkFields
        }
        nodes {
            ...deepRoutineNodeFields
        }
        outputs {
            ...deepRoutineOutputFields
        }
        owner {
            ... on Organization {
                id
                name
            }
            ... on User {
                id
                username
            }
        }
        parent {
            id
            title
        }
        resources {
            ...deepRoutineResourceFields
        }
        score
        stars
        role
        tags {
            ...deepRoutineTagFields
        }
        title
        updated_at
        version
    }
`