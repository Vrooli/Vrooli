export const RunRoutineSchedule_list = `fragment RunRoutineSchedule_list on RunRoutineSchedule {
labels {
    ...Label_list
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
}`;