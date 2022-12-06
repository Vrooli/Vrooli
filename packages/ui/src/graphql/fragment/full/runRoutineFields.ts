import { gql } from 'graphql-tag';

export const runRoutineFields = gql`
    fragment runRoutineTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment runRoutineInputItemFields on InputItem {
        id
        isRequired
        name
        translations {
            id
            language
            description
            helpText
        }
        standard {
            id
            default
            isDeleted
            isInternal
            isPrivate
            name
            type
            props
            yup
            tags {
                ...runRoutineTagFields
            }
            translations {
                id
                language
                description
            }
        }
    }
    fragment runRoutineOutputFields on OutputItem {
        id
        name
        translations {
            id
            language
            description
            helpText
        }
        standard {
            id
            default
            isDeleted
            isInternal
            isPrivate
            name
            type
            props
            yup
            tags {
                ...runRoutineTagFields
            }
            translations {
                id
                language
                description
            }
        }
    }
    fragment runRoutineNodeFields on Node {
        id
        columnIndex,
        created_at
        rowIndex,
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
                        inputs {
                            ...runRoutineInputItemFields
                        }
                        isComplete
                        isDeleted
                        isInternal
                        isPrivate
                        nodesCount
                        outputs {
                            ...runRoutineOutputFields
                        }
                        owner {
                            ... on Organization {
                                id
                                handle
                                translations {
                                    id
                                    language
                                    name
                                }
                            }
                            ... on User {
                                id
                                name
                                handle
                            }
                        }
                        permissionsRoutine {
                            canDelete
                            canEdit
                            canFork
                            canStar
                            canReport
                            canRun
                            canVote
                        }
                        resourceLists {
                            ...runRoutineResourceListFields
                        }
                        simplicity
                        tags {
                            ...runRoutineTagFields
                        }
                        translations {
                            id
                            language
                            title
                            description
                            instructions
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
    }
    fragment runRoutineNodeLinkFields on NodeLink {
        id
        fromId
        toId
        operation
        whens {
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
    fragment runRoutineResourceListFields on ResourceList {
        id
        created_at
        index
        usedFor
        translations {
            id
            language
            description
            title
        }
        resources {
            id
            created_at
            index
            link
            updated_at
            usedFor
            translations {
                id
                language
                description
                title
            }
        }
    }
    fragment runRoutineRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        inputs {
            ...runRoutineInputItemFields
        }
        isAutomatable
        isComplete
        isDeleted
        isInternal
        isPrivate
        isStarred
        isUpvoted
        nodeLinks {
            ...runRoutineNodeLinkFields
        }
        nodes {
            ...runRoutineNodeFields
        }
        outputs {
            ...runRoutineOutputFields
        }
        owner {
            ... on Organization {
                id
                handle
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                name
                handle
            }
        }
        parent {
            id
            translations {
                id
                language
                title
            }
        }
        resourceLists {
            ...runRoutineResourceListFields
        }
        score
        simplicity
        stars
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
            ...runRoutineTagFields
        }
        translations {
            id
            language
            description
            instructions
            title
        }
        updated_at
    }
    fragment runRoutineStepFields on RunRoutineStep {
        id
        order
        contextSwitches
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        step
        node {
            id
        }
    }
    fragment runRoutineInputDataFields on RunRoutineInput {
        id
        data
        input {
            ...runRoutineInputItemFields
        }
    }
    fragment runRoutineFields on RunRoutine {
        id
        completedComplexity
        contextSwitches
        isPrivate
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        inputs {
            ...runRoutineInputDataFields
        }
        routine {
            ...runRoutineRoutineFields
        }
        steps {
            ...runRoutineStepFields
        }
    }
`