import gql from "graphql-tag";
import { Schedule_common } from "../fragments/Schedule_common";

export const reminderCreate = gql`${Schedule_common}

mutation reminderCreate($input: ReminderCreateInput!) {
  reminderCreate(input: $input) {
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
}`;

