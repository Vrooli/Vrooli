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
                    isOptional
                    routine {
                        id
                        isInternal
                        nodesCount
                        role
                        owner {
                            ... on Organization {
                                id
                                translations {
                                    id
                                    language
                                    name
                                }
                            }
                            ... on User {
                                id
                                username
                            }
                        }
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
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                username
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