import gql from 'graphql-tag';

export const findOne = gql`
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

export const findMany = gql`
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

export const create = gql`
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

export const update = gql`
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

