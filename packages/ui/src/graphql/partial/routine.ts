import { nodeFields } from "./node";
import { resourceListFields } from "./resourceList";
import { tagFields } from "./tag";

export const routineNameFields = ['Routine', `{
    id
    translatedName
}`] as const;
export const listRoutineFields = ['Routine', `{
    id
    commentsCount
    completedAt
    complexity
    created_at
    isAutomatable
    isDeleted
    isInternal
    isComplete
    isPrivate
    isStarred
    isUpvoted
    reportsCount
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
    tags ${tagFields[1]}
    translations {
        id
        language
        description
        title
    }
}`] as const;
export const routineFields = ['Routine', `{
    id
    completedAt
    complexity
    created_at
    inputs {
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
        }
    }
    isAutomatable
    isComplete
    isDeleted
    isInternal
    isPrivate
    isStarred
    isUpvoted
    nodeLinks {
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
    nodes ${nodeFields[1]}
    outputs {
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
        }
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
    resourceList ${resourceListFields[1]}
    runs {
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
        startedAt
        timeElapsed
        completedAt
        title
        status
        steps {
            id
            order
            contextSwitches
            startedAt
            timeElapsed
            completedAt
            title
            status
            step
            node {
                id
            }
        }
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
    tags ${tagFields[1]}
    translations {
        id
        language
        description
        instructions
        title
    }
    updated_at
}`] as const;