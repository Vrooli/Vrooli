import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';

export const reminderListFindOne = gql`${Label_list}

query reminderList($input: FindByIdInput!) {
  reminderList(input: $input) {
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
        completed
        index
        reminderItems {
            id
            created_at
            updated_at
            name
            description
            dueDate
            completed
            index
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

export const reminderListFindMany = gql`${Label_list}

query reminderLists($input: ReminderListSearchInput!) {
  reminderLists(input: $input) {
    edges {
        cursor
        node {
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
                completed
                index
                reminderItems {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const reminderListCreate = gql`${Label_list}

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
        completed
        index
        reminderItems {
            id
            created_at
            updated_at
            name
            description
            dueDate
            completed
            index
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

export const reminderListUpdate = gql`${Label_list}

mutation reminderListUpdate($input: ReminderListUpdateInput!) {
  reminderListUpdate(input: $input) {
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
        completed
        index
        reminderItems {
            id
            created_at
            updated_at
            name
            description
            dueDate
            completed
            index
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

