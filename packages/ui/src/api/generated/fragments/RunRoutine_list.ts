export const RunRoutine_list = `fragment RunRoutine_list on RunRoutine {
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