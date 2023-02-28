import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const reminderFindMany = gql`${Label_list}
${Organization_nav}
${User_nav}

query reminders($input: ReminderSearchInput!) {
  reminders(input: $input) {
    edges {
        cursor
        node {
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;
