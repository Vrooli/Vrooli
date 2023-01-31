import gql from 'graphql-tag';

export const findOne = gql`
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

export const findMany = gql`
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

