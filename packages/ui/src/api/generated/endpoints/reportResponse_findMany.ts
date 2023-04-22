import gql from "graphql-tag";

export const reportResponseFindMany = gql`
query reportResponses($input: ReportResponseSearchInput!) {
  reportResponses(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            actionSuggested
            details
            language
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

