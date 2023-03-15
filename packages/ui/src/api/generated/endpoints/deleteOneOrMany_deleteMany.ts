import gql from 'graphql-tag';

export const deleteOneOrManyDeleteMany = gql`
mutation deleteMany($input: DeleteManyInput!) {
  deleteMany(input: $input) {
    count
  }
}`;

