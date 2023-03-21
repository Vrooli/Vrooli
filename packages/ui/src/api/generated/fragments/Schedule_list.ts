export const Schedule_list = `fragment Schedule_list on Schedule {
labels {
    ...Label_list
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