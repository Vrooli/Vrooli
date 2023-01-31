import gql from 'graphql-tag';

export const findOne = gql`
query notification($input: FindByIdInput!) {
  notification(input: $input) {
    id
    created_at
    category
    isRead
    title
    description
    link
    imgLink
  }
}`;

export const findMany = gql`
query notifications($input: NotificationSearchInput!) {
  notifications(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            category
            isRead
            title
            description
            link
            imgLink
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const markAsRead = gql`
mutation notificationMarkAsRead($input: FindByIdInput!) {
  notificationMarkAsRead(input: $input) {
    success
  }
}`;

export const update = gql`
mutation notificationMarkAllAsRead {
  notificationMarkAllAsRead {
    count
  }
}`;

export const settingsUpdate = gql`
mutation notificationSettingsUpdate($input: NotificationSettingsUpdateInput!) {
  notificationSettingsUpdate(input: $input) {
    includedEmails
    includedSms
    includedPush
    toEmails
    toSms
    toPush
    dailyLimit
    enabled
    categories {
        category
        enabled
        dailyLimit
        toEmails
        toSms
        toPush
    }
  }
}`;

