import gql from 'graphql-tag';

export const findOne = gql`
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

export const findMany = gql`
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

export const create = gql`
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

export const update = gql`
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

