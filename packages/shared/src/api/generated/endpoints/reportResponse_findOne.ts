import gql from "graphql-tag";

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
        canUpdate
    }
  }
}`;

