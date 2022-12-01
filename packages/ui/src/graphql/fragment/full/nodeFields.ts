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
    fragment nodeRoutineVersionFields on Routine {
        id
        complexity
        created_at
        isAutomatable
        isInternal
        simplicity
        permissionsRoutine {
            canDelete
            canEdit
            canFork
            canStar
            canReport
            canRun
            canVote
        }
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
                    routineVersion {
                        ...nodeRoutineVersionFields
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