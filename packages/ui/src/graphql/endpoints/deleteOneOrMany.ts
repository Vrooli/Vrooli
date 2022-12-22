import { toGql } from 'graphql/utils';

export const deleteOneOrManyEndpoint = {
    deleteOne: toGql('mutation', 'deleteOne', 'DeleteOneInput', [], `success`),
    deleteMany: toGql('mutation', 'deleteMany', 'DeleteManyInput', [], `count`),
}