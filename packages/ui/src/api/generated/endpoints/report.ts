import gql from 'graphql-tag';

export const reportFindOne = gql`
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
            canUpdate
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
        canRespond
        canUpdate
    }
  }
}`;

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

export const reportCreate = gql`
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
            canUpdate
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
        canRespond
        canUpdate
    }
  }
}`;

export const reportUpdate = gql`
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
            canUpdate
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
        canRespond
        canUpdate
    }
  }
}`;

