import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const authSwitchCurrentAccount = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation switchCurrentAccount($input: SwitchCurrentAccountInput!) {
  switchCurrentAccount(input: $input) {
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
  }
}`;

