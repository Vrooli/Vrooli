import gql from "graphql-tag";

export const notificationMarkAllAsRead = gql`
mutation notificationMarkAllAsRead {
  notificationMarkAllAsRead {
    count
  }
}`;

