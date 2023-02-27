import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const reminderListCreate = gql`${Label_list}
${Organization_nav}
${User_nav}

mutation reminderListCreate($input: ReminderListCreateInput!) {
  reminderListCreate(input: $input) {
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
}`;

