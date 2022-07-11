import { gql } from 'graphql-tag';

export const nodeFields = gql`
    fragment nodeTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment nodeRoutineFields on Routine {
        id
        complexity
        version
        created_at
        isAutomatable
        isInternal
        role
        simplicity
        tags {
            ...nodeTagFields
        }
        translations {
            id
            language
            description
            instructions
            title
        }
    }
    fragment nodeFields on Node {
        id
        columnIndex
        created_at
        rowIndex
        type
        updated_at
        data {
            ... on NodeEnd {
                id
                wasSuccessful
            }
            ... on NodeRoutineList {
                id
                isOptional
                isOrdered
                routines {
                    id
                    index
                    isOptional
                    routine {
                        ...nodeRoutineFields
                    }
                    translations {
                        id
                        language
                        description
                        title
                    }
                }
            }
        }
        loop {
            id
            loops
            maxLoops
            operation
            whiles {
                id
                condition
                translations {
                    id
                    language
                    description
                    title
                }
            }
        }
        translations {
            id
            language
            description
            title
        }
    }
`