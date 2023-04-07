export const Schedule_common = `fragment Schedule_common on Schedule {
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