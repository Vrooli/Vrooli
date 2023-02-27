import gql from 'graphql-tag';

export const reportResponseCreate = gql`
mutation reportResponseCreate($input: ReportResponseCreateInput!) {
  reportResponseCreate(input: $input) {
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

