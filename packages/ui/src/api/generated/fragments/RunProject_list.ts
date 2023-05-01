export const RunProject_list = `fragment RunProject_list on RunProject {
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
schedule {
    labels {
        ...Label_full
    }
    id
    created_at
    updated_at
    startTime
    endTime
    timezone
    exceptions {
        id
        originalStartTime
        newStartTime
        newEndTime
    }
    recurrences {
        id
        recurrenceType
        interval
        dayOfWeek
        dayOfMonth
        month
        endDate
    }
}
user {
    ...User_nav
}
you {
    canDelete
    canUpdate
    canRead
}
}`;
