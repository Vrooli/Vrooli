import gql from 'graphql-tag';

export const reportResponseFindOne = gql`
query reportResponse($input: FindByIdInput!) {
  reportResponse(input: $input) {
    id
    created_at
    updated_at
    actionSuggested
    details
    language
    you {
        canDelete
        canEdit
    }
  }
}`;

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
                canEdit
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

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
        canEdit
    }
  }
}`;

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
        canEdit
    }
  }
}`;

