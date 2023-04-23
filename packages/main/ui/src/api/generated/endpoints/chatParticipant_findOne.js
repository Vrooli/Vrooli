import gql from "graphql-tag";
export const chatParticipantFindOne = gql `
query chatParticipant($input: FindByIdInput!) {
  chatParticipant(input: $input) {
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
//# sourceMappingURL=chatParticipant_findOne.js.map