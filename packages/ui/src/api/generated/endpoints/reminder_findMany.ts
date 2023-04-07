import gql from 'graphql-tag';
import { Schedule_common } from '../fragments/Schedule_common';

export const reminderFindMany = gql`${Schedule_common}

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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

