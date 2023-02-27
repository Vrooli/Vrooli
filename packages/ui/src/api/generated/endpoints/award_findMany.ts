import gql from 'graphql-tag';

export const awardFindMany = gql`
query awards($input: AwardSearchInput!) {
  awards(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            timeCurrentTierCompleted
            category
            progress
            title
            description
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

