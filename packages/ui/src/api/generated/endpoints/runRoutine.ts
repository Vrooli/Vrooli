import gql from 'graphql-tag';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Label_full } from '../fragments/Label_full';

export const runRoutineFindOne = gql`${Organization_nav}
${User_nav}
${Label_full}

query runRoutine($input: FindByIdInput!) {
  runRoutine(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    inputsCount
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    runRoutineSchedule {
        labels {
            ...Label_full
        }
        id
        attemptAutomatic
        maxAutomaticAttempts
        timeZone
        windowStart
        windowEnd
        recurrStart
        recurrEnd
        translations {
            id
            language
            description
            name
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runRoutineFindMany = gql`${Organization_nav}
${User_nav}
${Label_full}

query runRoutines($input: RunRoutineSearchInput!) {
  runRoutines(input: $input) {
    edges {
        cursor
        node {
            id
            isPrivate
            completedComplexity
            contextSwitches
            startedAt
            timeElapsed
            completedAt
            name
            status
            stepsCount
            inputsCount
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            runRoutineSchedule {
                labels {
                    ...Label_full
                }
                id
                attemptAutomatic
                maxAutomaticAttempts
                timeZone
                windowStart
                windowEnd
                recurrStart
                recurrEnd
                translations {
                    id
                    language
                    description
                    name
                }
            }
            user {
                ...User_nav
            }
            you {
                canDelete
                canEdit
                canView
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const runRoutineCreate = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runRoutineCreate($input: RunRoutineCreateInput!) {
  runRoutineCreate(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    inputsCount
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    runRoutineSchedule {
        labels {
            ...Label_full
        }
        id
        attemptAutomatic
        maxAutomaticAttempts
        timeZone
        windowStart
        windowEnd
        recurrStart
        recurrEnd
        translations {
            id
            language
            description
            name
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runRoutineUpdate = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runRoutineUpdate($input: RunRoutineUpdateInput!) {
  runRoutineUpdate(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    inputsCount
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    runRoutineSchedule {
        labels {
            ...Label_full
        }
        id
        attemptAutomatic
        maxAutomaticAttempts
        timeZone
        windowStart
        windowEnd
        recurrStart
        recurrEnd
        translations {
            id
            language
            description
            name
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runRoutineDeleteAll = gql`
mutation runRoutineDeleteAll {
  runRoutineDeleteAll {
    count
  }
}`;

export const runRoutineComplete = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runRoutineComplete($input: RunRoutineCompleteInput!) {
  runRoutineComplete(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    inputsCount
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    runRoutineSchedule {
        labels {
            ...Label_full
        }
        id
        attemptAutomatic
        maxAutomaticAttempts
        timeZone
        windowStart
        windowEnd
        recurrStart
        recurrEnd
        translations {
            id
            language
            description
            name
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

export const runRoutineCancel = gql`${Organization_nav}
${User_nav}
${Label_full}

mutation runRoutineCancel($input: RunRoutineCancelInput!) {
  runRoutineCancel(input: $input) {
    inputs {
        id
        data
        input {
            id
            index
            isRequired
            name
            standardVersion {
                translations {
                    id
                    language
                    description
                    jsonVariable
                }
                id
                created_at
                updated_at
                isComplete
                isFile
                isLatest
                isPrivate
                default
                standardType
                props
                yup
                versionIndex
                versionLabel
                commentsCount
                directoryListingsCount
                forksCount
                reportsCount
                you {
                    canComment
                    canCopy
                    canDelete
                    canEdit
                    canReport
                    canUse
                    canView
                }
            }
        }
    }
    steps {
        id
        order
        contextSwitches
        startedAt
        timeElapsed
        completedAt
        name
        status
        step
        subroutine {
            id
            isAutomatable
            isComplete
            isDeleted
            isLatest
            isPrivate
            root {
                id
                isInternal
                isPrivate
            }
            translations {
                id
                language
                description
                instructions
                name
            }
            versionIndex
            versionLabel
        }
    }
    id
    isPrivate
    completedComplexity
    contextSwitches
    startedAt
    timeElapsed
    completedAt
    name
    status
    stepsCount
    inputsCount
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    runRoutineSchedule {
        labels {
            ...Label_full
        }
        id
        attemptAutomatic
        maxAutomaticAttempts
        timeZone
        windowStart
        windowEnd
        recurrStart
        recurrEnd
        translations {
            id
            language
            description
            name
        }
    }
    user {
        ...User_nav
    }
    you {
        canDelete
        canEdit
        canView
    }
  }
}`;

