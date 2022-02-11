import { gql } from 'graphql-tag';

export const nodeFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment routineFields on Routine {
        id
        version
        title
        description
        created_at
        isAutomatable
        role
        tags {
            ...tagFields
        }
    }
    fragment nodeFields on Node {
        id
        created_at
        description
        role
        next
        previous
        title
        type
        updated_at
        data {
            ... on NodeCombine {
                id
                from
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
                        ...routineFields
                    }
                }
            }
        }
    }
`