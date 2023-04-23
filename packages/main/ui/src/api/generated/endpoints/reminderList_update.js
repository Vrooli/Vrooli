import gql from "graphql-tag";
import { Schedule_common } from "../fragments/Schedule_common";
export const reminderListUpdate = gql `${Schedule_common}

mutation reminderListUpdate($input: ReminderListUpdateInput!) {
  reminderListUpdate(input: $input) {
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
}`;
//# sourceMappingURL=reminderList_update.js.map