import gql from 'graphql-tag';

export const notificationFindOne = gql`
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

export const notificationFindMany = gql`
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

export const notificationMarkAsRead = gql`
mutation notificationMarkAsRead($input: FindByIdInput!) {
  notificationMarkAsRead(input: $input) {
    success
  }
}`;

export const notificationUpdate = gql`
mutation notificationMarkAllAsRead {
  notificationMarkAllAsRead {
    count
  }
}`;

export const notificationSettingsUpdate = gql`
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

