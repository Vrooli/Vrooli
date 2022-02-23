import { gql } from 'graphql-tag';

export const nodeFields = gql`
    fragment nodeTagFields on Tag {
        id
        description
        tag
    }
    fragment nodeRoutineFields on Routine {
        id
        version
        title
        description
        created_at
        isAutomatable
        role
        tags {
            ...nodeTagFields
        }
    }
    fragment nodeFields on Node {
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
                        ...nodeRoutineFields
                    }
                }
            }
        }
    }
`