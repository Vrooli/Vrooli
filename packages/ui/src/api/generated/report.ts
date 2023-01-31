import gql from 'graphql-tag';

export const findOne = gql`
query report($input: FindByIdInput!) {
  report(input: $input) {
    responses {
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
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
  }
}`;

export const findMany = gql`
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
                canEdit
                canRespond
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
mutation reportCreate($input: ReportCreateInput!) {
  reportCreate(input: $input) {
    responses {
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
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
  }
}`;

export const update = gql`
mutation reportUpdate($input: ReportUpdateInput!) {
  reportUpdate(input: $input) {
    responses {
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
    id
    created_at
    updated_at
    details
    language
    reason
    responsesCount
    you {
        canDelete
        canEdit
        canRespond
    }
  }
}`;

