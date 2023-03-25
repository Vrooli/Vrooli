import gql from 'graphql-tag';
import { Schedule_common } from '../fragments/Schedule_common';

export const authLogOut = gql`${Schedule_common}

mutation logOut($input: LogOutInput!) {
  logOut(input: $input) {
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
  }
}`;

