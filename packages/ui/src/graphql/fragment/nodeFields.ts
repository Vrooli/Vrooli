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
        tags {
            ...tagFields
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
                toId
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
            ... on NodeRedirect {
                id
            }
            ... on NodeStart {
                id
            }
        }
    }
`