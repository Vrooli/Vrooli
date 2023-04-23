import gql from "graphql-tag";
export const apiKeyValidate = gql `
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
//# sourceMappingURL=apiKey_validate.js.map