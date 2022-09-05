import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment routineTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment routineInputFields on InputItem {
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
            translations {
                id
                language
                description
                jsonVariable
            }
            version
            versionGroupId
        }
    }
    fragment routineOutputFields on OutputItem {
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
            translations {
                id
                language
                description
                jsonVariable
            }
            version
            versionGroupId
        }
    }
    fragment routineNodeFields on Node {
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
                    routine {
                        id
                        complexity
                        isComplete
                        isDeleted
                        isInternal
                        isPrivate
                        inputs {
                            ...routineInputFields
                        }
                        nodesCount
                        outputs {
                            ...routineOutputFields
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
                            canComment
                            canDelete
                            canEdit
                            canFork
                            canStar
                            canReport
                            canRun
                            canVote
                        }
                        resourceLists {
                            ...routineResourceListFields
                        }
                        simplicity
                        tags {
                            ...routineTagFields
                        }
                        translations {
                            id
                            language
                            title
                            description
                            instructions
                        }
                        version
                        versionGroupId
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
    fragment routineNodeLinkFields on NodeLink {
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
    fragment routineResourceListFields on ResourceList {
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
    fragment routineStepFields on RunStep {
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
    fragment routineRunFields on Run {
        id
        completedComplexity
        contextSwitches
        inputs {
            id
            data
            input {
                id
            }
        }
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        steps {
            ...routineStepFields
        }
    }
    fragment routineFields on Routine {
        id
        completedAt
        complexity
        created_at
        inputs {
            ...routineInputFields
        }
        isAutomatable
        isComplete
        isDeleted
        isInternal
        isPrivate
        isStarred
        isUpvoted
        nodeLinks {
            ...routineNodeLinkFields
        }
        nodes {
            ...routineNodeFields
        }
        outputs {
            ...routineOutputFields
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
        reportsCount
        resourceLists {
            ...routineResourceListFields
        }
        runs {
            ...routineRunFields
        }
        score
        simplicity
        stars
        permissionsRoutine {
            canComment
            canDelete
            canEdit
            canFork
            canStar
            canReport
            canRun
            canVote
        }
        tags {
            ...routineTagFields
        }
        translations {
            id
            language
            description
            instructions
            title
        }
        updated_at
        version
        versionGroupId
    }
`