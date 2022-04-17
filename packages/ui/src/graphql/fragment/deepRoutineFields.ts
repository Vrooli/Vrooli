import { gql } from 'graphql-tag';

export const deepRoutineFields = gql`
    fragment deepRoutineTagFields on Tag {
        id
        tag
        translations {
            id
            language
            description
        }
    }
    fragment deepRoutineInputFields on InputItem {
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
            isFile
            name
            schema
            tags {
                ...deepRoutineTagFields
            }
            translations {
                id
                language
                description
            }
        }
    }
    fragment deepRoutineOutputFields on OutputItem {
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
            isFile
            name
            schema
            tags {
                ...deepRoutineTagFields
            }
            translations {
                id
                language
                description
            }
        }
    }
    fragment deepRoutineNodeFields on Node {
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
    fragment deepRoutineNodeLinkFields on NodeLink {
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
    fragment deepRoutineResourceListFields on ResourceList {
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
    fragment deepRoutineFields on Routine {
        id
        completedAt
        complexity
        created_at
        inputs {
            ...deepRoutineInputFields
        }
        isAutomatable
        isComplete
        isInternal
        isStarred
        isUpvoted
        nodeLinks {
            ...deepRoutineNodeLinkFields
        }
        nodes {
            ...deepRoutineNodeFields
        }
        outputs {
            ...deepRoutineOutputFields
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
            ...deepRoutineResourceListFields
        }
        score
        simplicity
        stars
        role
        tags {
            ...deepRoutineTagFields
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
`