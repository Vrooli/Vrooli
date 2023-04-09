import gql from 'graphql-tag';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';

export const labelFindMany = gql`${Organization_nav}
${User_nav}

query labels($input: LabelSearchInput!) {
  labels(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            color
            label
            owner {
                ... on Organization {
                    ...Organization_nav
                }
                ... on User {
                    ...User_nav
                }
            }
            you {
                canDelete
                canUpdate
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

