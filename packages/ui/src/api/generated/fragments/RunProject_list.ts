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
    canUpdate
    canRead
}
}`;