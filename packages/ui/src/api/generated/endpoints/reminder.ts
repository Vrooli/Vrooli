import gql from 'graphql-tag';

export const reminderFindOne = gql`
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
  }
}`;

export const reminderFindMany = gql`
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const reminderCreate = gql`
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
  }
}`;

export const reminderUpdate = gql`
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
  }
}`;

