export const RunProjectSchedule_list = `fragment RunProjectSchedule_list on RunProjectSchedule {
labels {
    ...Label_list
}
id
timeZone
windowStart
windowEnd
recurrStart
recurrEnd
runProject {
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
    user {
        ...User_nav
    }
    you {
        canDelete
        canUpdate
        canRead
    }
}
translations {
    id
    language
    description
    name
}
}`;