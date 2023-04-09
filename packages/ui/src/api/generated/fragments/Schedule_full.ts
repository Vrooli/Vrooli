export const Schedule_full = `fragment Schedule_full on Schedule {
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
}`;