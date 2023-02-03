import gql from 'graphql-tag';
import { Label_list } from '../fragments/Label_list';

export const reminderFindOne = gql`${Label_list}

query reminder($input: FindByIdInput!) {
  reminder(input: $input) {
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
}`;

export const reminderFindMany = gql`${Label_list}

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

export const reminderCreate = gql`${Label_list}

mutation reminderCreate($input: ReminderCreateInput!) {
  reminderCreate(input: $input) {
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
}`;

export const reminderUpdate = gql`${Label_list}

mutation reminderUpdate($input: ReminderUpdateInput!) {
  reminderUpdate(input: $input) {
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
}`;

