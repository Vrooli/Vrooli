export const Session_full = `fragment Session_full on Session {
isLoggedIn
timeZone
users {
    focusModes {
        filters {
            id
            filterType
            tag {
                id
                created_at
                tag
                bookmarks
                translations {
                    id
                    language
                    description
                }
                you {
                    isOwn
                    isBookmarked
                }
            }
            focusMode {
                labels {
                    ...Label_list
                }
                schedule {
                    ...Schedule_common
                }
                id
                name
                description
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
            }
        }
        schedule {
            ...Schedule_common
        }
        id
        name
        description
    }
    handle
    hasPremium
    id
    languages
    name
    theme
}
}`;