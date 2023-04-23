import gql from "graphql-tag";

export const notificationMarkAsRead = gql`
mutation notificationMarkAsRead($input: FindByIdInput!) {
  notificationMarkAsRead(input: $input) {
    success
  }
}`;

