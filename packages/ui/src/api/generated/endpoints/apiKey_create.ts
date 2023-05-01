import gql from "graphql-tag";

export const apiKeyCreate = gql`
mutation apiKeyCreate($input: ApiKeyCreateInput!) {
  apiKeyCreate(input: $input) {
    id
    creditsUsed
    creditsUsedBeforeLimit
    stopAtLimit
    absoluteMax
    resetsAt
  }
}`;

