import gql from "graphql-tag";

export const reportResponseUpdate = gql`
mutation reportResponseUpdate($input: ReportResponseUpdateInput!) {
  reportResponseUpdate(input: $input) {
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
}`;

