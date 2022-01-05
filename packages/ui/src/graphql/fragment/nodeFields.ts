import { gql } from 'graphql-tag';
import { routineFields } from '.';

export const nodeFields = gql`
    ${routineFields}
    fragment nodeFields on Node {
        id
        created_at
        updated_at
        routineId
        title
        description
        type
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
                    title
                    description
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
                isOrdered
                isOptional
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
        previous
        next
    }
`