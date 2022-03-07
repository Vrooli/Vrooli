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
                        role
                        translations {
                            id
                            language
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
    fragment deepRoutineResourceFields on Resource {
        id
        created_at
        link
        updated_at
        translations {
            id
            language
            description
            title
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
        resources {
            ...deepRoutineResourceFields
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