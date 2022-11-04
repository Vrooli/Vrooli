import { gql } from 'graphql-tag';

export const routineNodeFields = gql`
    fragment routineNodeTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment routineNodeRoutineFields on Routine {
        id
        complexity
        version
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
    fragment routineNodeFields on Node {
        id
        columnIndex
        created_at
        rowIndex
        type
        updated_at
        data {
            ... on RoutineNodeEnd {
                id
                wasSuccessful
            }
            ... on RoutineNodeRoutineList {
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