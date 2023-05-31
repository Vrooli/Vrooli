import gql from "graphql-tag";
import { Schedule_common } from "../fragments/Schedule_common";

export const reminderListFindMany = gql`${Schedule_common}

query reminderLists($input: ReminderListSearchInput!) {
  reminderLists(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

