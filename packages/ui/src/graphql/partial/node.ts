import { tagFields } from "./tag";

export const nodeFields = ['Node', `{
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
                    id
                    complexity
                    created_at
                    isAutomatable
                    isInternal
                    simplicity
                    permissionsRoutine {
                        canCopy
                        canDelete
                        canEdit
                        canStar
                        canReport
                        canRun
                        canVote
                    }
                    tags ${tagFields[1]}
                    translations {
                        id
                        language
                        description
                        instructions
                        title
                    }
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
}`] as const;