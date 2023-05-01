import gql from "graphql-tag";

export const notificationUpdate = gql`
mutation notificationMarkAllAsRead {
  notificationMarkAllAsRead {
    count
  }
}`;

