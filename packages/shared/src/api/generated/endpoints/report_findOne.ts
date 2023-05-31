import gql from "graphql-tag";

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

