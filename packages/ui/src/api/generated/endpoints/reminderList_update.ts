import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { Schedule_common } from '../fragments/Schedule_common';
import { User_nav } from '../fragments/User_nav';

export const reminderListUpdate = gql`${Label_list}
${Organization_nav}
${Schedule_common}
${User_nav}

mutation reminderListUpdate($input: ReminderListUpdateInput!) {
  reminderListUpdate(input: $input) {
    id
    created_at
    updated_at
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

