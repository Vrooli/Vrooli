import gql from 'graphql-tag';

export const findOne = gql`
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

export const findMany = gql`
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

export const create = gql`
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

export const update = gql`
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

