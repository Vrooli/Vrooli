import { gql } from 'graphql-tag';

export const nodeFields = gql`
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
                from {
                    id
                }
                to {
                    id
                }
            }
            ... on NodeDecision {
                id
                decisions {
                    id
                    title
                    when {
                        id
                        condition
                    }
                }
            }
            ... on NodeEnd {
                id
            }
            ... on NodeLoop {
                id
            }
            ... on NodeRoutineList {
                id
            }
            ... on NodeRedirect {
                id
            }
            ... on NodeStart {
                id
            }
        }
        previous {
            id
        }
        next {
            id
        }
    }
`