import gql from "graphql-tag";

export const apiKeyDeleteOne = gql`
mutation apiKeyDeleteOne($input: ApiKeyDeleteOneInput!) {
  apiKeyDeleteOne(input: $input) {
    success
  }
}`;

