import gql from 'graphql-tag';

export const reportFindMany = gql`
query reports($input: ReportSearchInput!) {
  reports(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            details
            language
            reason
            responsesCount
            you {
                canDelete
                canRespond
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

