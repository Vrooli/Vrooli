import gql from 'graphql-tag';

export const reputationHistoryFindOne = gql`
query reputationHistory($input: FindByIdInput!) {
  reputationHistory(input: $input) {
    id
    created_at
    updated_at
    amount
    event
    objectId1
    objectId2
  }
}`;

