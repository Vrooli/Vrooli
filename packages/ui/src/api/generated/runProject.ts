import gql from 'graphql-tag';

export const runProjectFindOne = gql`fragment Organization_nav on Organization {
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


query runProject($input: FindByIdInput!) {
  runProject(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
        labels {
            ...Label_full
        }
        id
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

export const runProjectFindMany = gql`fragment Organization_nav on Organization {
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


query runProjects($input: RunProjectSearchInput!) {
  runProjects(input: $input) {
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
            wasRunAutomaticaly
            organization {
                ...Organization_nav
            }
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
            runProjectSchedule {
                labels {
                    ...Label_full
                }
                id
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

export const runProjectCreate = gql`fragment Organization_nav on Organization {
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


mutation runProjectCreate($input: RunProjectCreateInput!) {
  runProjectCreate(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
        labels {
            ...Label_full
        }
        id
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

export const runProjectUpdate = gql`fragment Organization_nav on Organization {
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


mutation runProjectUpdate($input: RunProjectUpdateInput!) {
  runProjectUpdate(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
        labels {
            ...Label_full
        }
        id
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

export const runProjectDeleteAll = gql`
mutation runProjectDeleteAll {
  runProjectDeleteAll {
    count
  }
}`;

export const runProjectComplete = gql`fragment Organization_nav on Organization {
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


mutation runProjectComplete($input: RunProjectCompleteInput!) {
  runProjectComplete(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
        labels {
            ...Label_full
        }
        id
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

export const runProjectCancel = gql`fragment Organization_nav on Organization {
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


mutation runProjectCancel($input: RunProjectCancelInput!) {
  runProjectCancel(input: $input) {
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
        directory {
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
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
    wasRunAutomaticaly
    organization {
        ...Organization_nav
    }
    projectVersion {
        id
        isLatest
        isPrivate
        versionIndex
        versionLabel
        root {
            id
            isPrivate
        }
        translations {
            id
            language
            description
            name
        }
    }
    runProjectSchedule {
        labels {
            ...Label_full
        }
        id
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

