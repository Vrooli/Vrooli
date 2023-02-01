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

export const reputationHistoryFindMany = gql`
query reputationHistories($input: ReputationHistorySearchInput!) {
  reputationHistories(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            amount
            event
            objectId1
            objectId2
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

