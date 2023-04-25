import gql from "graphql-tag";

export const chatParticipantUpdate = gql`
mutation chatParticipantUpdate($input: ChatParticipantUpdateInput!) {
  chatParticipantUpdate(input: $input) {
    user {
        id
        name
        handle
    }
    id
    created_at
    updated_at
  }
}`;

