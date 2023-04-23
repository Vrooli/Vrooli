import gql from "graphql-tag";
export const emailCreate = gql `
mutation emailCreate($input: EmailCreateInput!) {
  emailCreate(input: $input) {
    id
    emailAddress
    verified
  }
}`;
//# sourceMappingURL=email_create.js.map