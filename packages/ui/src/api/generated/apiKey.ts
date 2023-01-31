import gql from 'graphql-tag';

export const create = gql`
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

export const update = gql`
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

export const deleteOne = gql`
mutation apiKeyDeleteOne($input: ApiKeyDeleteOneInput!) {
  apiKeyDeleteOne(input: $input) {
    success
  }
}`;

export const validate = gql`
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

