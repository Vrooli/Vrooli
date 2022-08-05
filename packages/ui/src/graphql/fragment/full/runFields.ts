import { gql } from 'graphql-tag';

export const runFields = gql`
    fragment runTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment runInputItemFields on InputItem {
        id
        isRequired
        name
        translations {
            id
            language
            description
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
                ...runTagFields
            }
            translations {
                id
                language
                description
            }
            version
            versionGroupId
        }
    }
    fragment runOutputFields on OutputItem {
        id
        name
        translations {
            id
            language
            description
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
                ...runTagFields
            }
            translations {
                id
                language
                description
            }
            version
            versionGroupId
        }
    }
    fragment runNodeFields on Node {
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
                        inputs {
                            ...runInputItemFields
                        }
                        isComplete
                        isDeleted
                        isInternal
                        isPrivate
                        nodesCount
                        outputs {
                            ...runOutputFields
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
                            ...runResourceListFields
                        }
                        simplicity
                        tags {
                            ...runTagFields
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
    fragment runNodeLinkFields on NodeLink {
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
    fragment runResourceListFields on ResourceList {
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
    fragment runRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        inputs {
            ...runInputItemFields
        }
        isAutomatable
        isComplete
        isDeleted
        isInternal
        isPrivate
        isStarred
        isUpvoted
        nodeLinks {
            ...runNodeLinkFields
        }
        nodes {
            ...runNodeFields
        }
        outputs {
            ...runOutputFields
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
            ...runResourceListFields
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
            ...runTagFields
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
    fragment runStepFields on RunStep {
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
    fragment runInputDataFields on RunInput {
        id
        data
        input {
            ...runInputItemFields
        }
    }
    fragment runFields on Run {
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
            ...runInputDataFields
        }
        routine {
            ...runRoutineFields
        }
        steps {
            ...runStepFields
        }
    }
`