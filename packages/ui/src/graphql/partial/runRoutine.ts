import { RunRoutineYou } from "@shared/consts";
import { GqlPartial } from "types";
import { nodeFields } from "./node";
import { resourceListPartial } from "./resourceList";
import { tagPartial } from "./tag";

export const runRoutineYouPartial: GqlPartial<RunRoutineYou> = {
    __typename: 'RunRoutineYou',
    full: () => ({
        canDelete: true,
        canEdit: true,
        canView: true,
    }),
}

export const listRunRoutineFields = ['RunRoutine', `{
    id
    completedComplexity
    contextSwitches
    isPrivate
    startedAt
    timeElapsed
    completedAt
    title
    status
    routine {
        id
        completedAt
        complexity
        created_at
        isAutomatable
        isDeleted
        isInternal
        isPrivate
        isComplete
        isStarred
        isUpvoted
        score
        simplicity
        stars
        permissionsRoutine {
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canVote
        }
        tags ${tagPartial.list}
        translations {
            id
            language
            description
            title
        }
    }
}`] as const;
export const runRoutineFields = ['RunRoutine', `{
    id
    completedComplexity
    contextSwitches
    isPrivate
    startedAt
    timeElapsed
    completedAt
    title
    status
    inputs {
        id
        data
        input {
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
                tags ${tagPartial.list}
                translations {
                    id
                    language
                    description
                }
            }
        }
    }
    routine {
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
                tags ${tagPartial.list}
                translations {
                    id
                    language
                    description
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
        resourceList ${resourceListPartial.full}
        score
        simplicity
        stars
        permissionsRoutine {
            canCopy
            canDelete
            canEdit
            canStar
            canReport
            canRun
            canVote
        }
        tags ${tagPartial.list}
        translations {
            id
            language
            description
            instructions
            title
        }
        updated_at
    }
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
}`] as const;