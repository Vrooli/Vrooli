import gql from 'graphql-tag';

export const findMany = gql`fragment RunProject_list on RunProject {
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

fragment Organization_nav on Organization {
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

fragment RunRoutine_list on RunRoutine {
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


query runProjectOrRunRoutines($input: RunProjectOrRunRoutineSearchInput!) {
  runProjectOrRunRoutines(input: $input) {
    edges {
        cursor
        node {
            ... on RunProject {
            }
            ... on RunRoutine {
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

