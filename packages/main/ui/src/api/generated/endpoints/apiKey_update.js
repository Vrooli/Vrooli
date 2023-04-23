import gql from "graphql-tag";
export const apiKeyUpdate = gql `
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
//# sourceMappingURL=apiKey_update.js.map