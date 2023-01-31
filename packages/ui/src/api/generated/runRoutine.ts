import gql from 'graphql-tag';

export const findOne = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

export const findMany = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

export const create = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

export const update = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

export const deleteAll = gql`
mutation runRoutineDeleteAll {
  runRoutineDeleteAll {
    count
  }
}`;

export const complete = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

export const cancel = gql`fragment Organization_nav on Organization {
    id
    handle
    you {
        canAddMembers
        canDelete
        canEdit
        canStar
        canReport
        canView
        isStarred
        isViewed
        yourMembership {
            id
            created_at
            updated_at
            isAdmin
            permissions
        }
    }
}

fragment User_nav on User {
    id
    name
    handle
}

fragment Label_full on Label {
    apisCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    runProjectSchedulesCount
    runRoutineSchedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canEdit
    }
}


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

