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

