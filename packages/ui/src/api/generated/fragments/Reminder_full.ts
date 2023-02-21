export const Reminder_full = `fragment Reminder_full on Reminder {
id
created_at
updated_at
name
description
dueDate
index
isComplete
reminderItems {
    id
    created_at
    updated_at
    name
    description
    dueDate
    index
    isComplete
}
reminderList {
    id
    created_at
    updated_at
    userSchedule {
        labels {
            ...Label_list
        }
        id
        name
        description
        timeZone
        eventStart
        eventEnd
        recurring
        recurrStart
        recurrEnd
    }
}
}`;