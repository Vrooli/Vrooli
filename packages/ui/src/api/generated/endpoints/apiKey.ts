import gql from 'graphql-tag';

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

export const apiKeyUpdate = gql`
mutation apiKeyUpdate($input: ApiKeyUpdateInput!) {
  apiKeyUpdate(input: $input) {
    id
    creditsUsed
    creditsUsedBeforeLimit
    stopAtLimit
    absoluteMax
    resetsAt
  }
}`;

export const apiKeyDeleteOne = gql`
mutation apiKeyDeleteOne($input: ApiKeyDeleteOneInput!) {
  apiKeyDeleteOne(input: $input) {
    success
  }
}`;

export const apiKeyValidate = gql`
mutation apiKeyValidate($input: ApiKeyValidateInput!) {
  apiKeyValidate(input: $input) {
    id
    creditsUsed
    creditsUsedBeforeLimit
    stopAtLimit
    absoluteMax
    resetsAt
  }
}`;

