import { gql } from 'graphql-tag';

export const deepRoutineFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment inputFields on InputItem {
        id
        standard {
            id
            default
            description
            isFile
            name
            schema
            tags {
                ...tagFields
            }
        }
    }
    fragment outputFields on OutputItem {
        id
        standard {
            id
            default
            description
            isFile
            name
            schema
            tags {
                ...tagFields
            }
        }
    }
    fragment nodeFields on Node {
        id
        created_at
        description
        next
        previous
        title
        type
        updated_at
        data {
            ... on NodeCombine {
                id
                from
                to
            }
            ... on NodeDecision {
                id
                decisions {
                    id
                    description
                    title
                    toId
                    when {
                        id
                        condition
                    }
                }
            }
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
    fragment resourceFields on Resource {
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
        isStarred
        inputs {
            ...inputFields
        }
        nodes {
            ...nodeFields
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
            ...outputFields
        }
        parent {
            id
            title
        }
        contextualResources {
            ...resourceFields
        }
        externalResources {
            ...resourceFields
        }
        tags {
            ...tagFields
        }
    }
`