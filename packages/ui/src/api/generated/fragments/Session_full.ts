export const Session_full = `fragment Session_full on Session {
isLoggedIn
timeZone
users {
    handle
    hasPremium
    id
    languages
    name
    schedules {
        filters {
            id
            filterType
            tag {
                id
                created_at
                tag
                stars
                translations {
                    id
                    language
                    description
                }
                you {
                    isOwn
                    isStarred
                }
            }
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
        labels {
            ...Label_full
        }
        reminderList {
            id
            created_at
            updated_at
            reminders {
                id
                created_at
                updated_at
                name
                description
                dueDate
                completed
                index
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                }
            }
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
    theme
}
}`;