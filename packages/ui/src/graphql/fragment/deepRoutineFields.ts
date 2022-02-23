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
        created_at
        description
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
        created_at
        instructions
        isAutomatable
        title
        description
        updated_at
        version
        stars
        score
        isUpvoted
        role
        isStarred
        inputs {
            ...deepRoutineInputFields
        }
        nodes {
            ...deepRoutineNodeFields
        }
        nodeLinks {
            ...deepRoutineNodeLinkFields
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
        outputs {
            ...deepRoutineOutputFields
        }
        parent {
            id
            title
        }
        contextualResources {
            ...deepRoutineResourceFields
        }
        externalResources {
            ...deepRoutineResourceFields
        }
        tags {
            ...deepRoutineTagFields
        }
    }
`