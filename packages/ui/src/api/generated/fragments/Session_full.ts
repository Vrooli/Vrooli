export const Session_full = `fragment Session_full on Session {
isLoggedIn
timeZone
users {
    activeFocusMode {
        mode {
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
                        id
                        color
                        label
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
                id
                color
                label
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
            resourceList {
                id
                created_at
                translations {
                    id
                    language
                    description
                    name
                }
                resources {
                    id
                    index
                    link
                    usedFor
                    translations {
                        id
                        language
                        description
                        name
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
        stopCondition
        stopTime
    }
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
                    id
                    color
                    label
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
            id
            color
            label
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
        resourceList {
            id
            created_at
            translations {
                id
                language
                description
                name
            }
            resources {
                id
                index
                link
                usedFor
                translations {
                    id
                    language
                    description
                    name
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
    bookmarkLists {
        id
        created_at
        updated_at
        label
        bookmarksCount
    }
    handle
    hasPremium
    id
    languages
    name
    theme
}
}`;