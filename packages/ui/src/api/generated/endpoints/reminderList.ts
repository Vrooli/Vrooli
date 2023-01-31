import gql from 'graphql-tag';

export const reminderListFindOne = gql`
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
  }
}`;

export const reminderListFindMany = gql`
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
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const reminderListCreate = gql`
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
  }
}`;

export const reminderListUpdate = gql`
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
  }
}`;

