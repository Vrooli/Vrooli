import { gql } from 'graphql-tag';

export const runFields = gql`
    fragment runTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment runInputFields on InputItem {
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
                        isInternal
                        nodesCount
                        role
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
                        simplicity
                        translations {
                            id
                            language
                            title
                            description
                            instructions
                        }
                        version
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
            ...runInputFields
        }
        isAutomatable
        isComplete
        isInternal
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
        role
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
    }
    fragment runStepFields on RunStep {
        id
        order
        pickups
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
    fragment runFields on Run {
        id
        completedComplexity
        pickups
        timeStarted
        timeElapsed
        timeCompleted
        title
        status
        routine {
            ...runRoutineFields
        }
        steps {
            ...runStepFields
        }
    }
`