import gql from 'graphql-tag';

export const deleteOne = gql`
mutation deleteOne($input: DeleteOneInput!) {
  deleteOne(input: $input) {
    success
  }
}`;

export const deleteMany = gql`
mutation deleteMany($input: DeleteManyInput!) {
  deleteMany(input: $input) {
    count
  }
}`;

