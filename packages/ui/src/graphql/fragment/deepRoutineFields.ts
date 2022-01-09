import { gql } from 'graphql-tag';

export const deepRoutineFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment ioFields on RoutineInputItem {
        id
        standard {
            id
            default
            description
            isFile
            name
            schema
            tags {
                ...tagFields
            }
        }
    }
    fragment nodeFields on Node {
        id
        created_at
        description
        next
        previous
        title
        type
        updated_at
        data {
            ... on NodeCombine {
                id
                from
                to
            }
            ... on NodeDecision {
                id
                decisions {
                    id
                    description
                    title
                    toId
                    when {
                        id
                        condition
                    }
                }
            }
            ... on NodeEnd {
                id
                wasSuccessful
            }
            ... on NodeLoop {
                id
            }
            ... on NodeRoutineList {
                id
                isOptional
                isOrdered
                routines {
                    ...routineFields
                }
            }
            ... on NodeRedirect {
                id
            }
            ... on NodeStart {
                id
            }
        }
    }
    fragment resourceFields on Resource {
        id
        created_at
        description
        link
        title
        updated_at
    }
    fragment deepRoutineFields on Routine {
        id
        created_at
        instructions
        isAutomatable
        title
        updated_at
        version
        stars
        inputs {
            ...ioFields
        }
        nodes {
            ...nodeFields
        }
        organizations {
            id
            name
        }
        outputs {
            ...ioFields
        }
        parent {
            id
            title
        }
        contextualResources {
            ...resourceFields
        }
        externalResources {
            ...resourceFields
        }
        tags {
            ...tagFields
        }
        users {
            id
            username
        }
    }
`