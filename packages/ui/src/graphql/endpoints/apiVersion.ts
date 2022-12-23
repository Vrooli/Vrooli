import { apiVersionFields as fullFields, listApiVersionFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const apiVersionEndpoint = {
    findOne: toQuery('apiVersion', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('apiVersions', 'ApiVersionSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('apiVersionCreate', 'ApiVersionCreateInput', [fullFields], `...fullFields`),
    update: toMutation('apiVersionUpdate', 'ApiVersionUpdateInput', [fullFields], `...fullFields`)
}
// TODO for morning: change graphql-generate for server to save types in shared 
// package. Then we can use these types in the client. We won't know which fields were 
// selected, but we can at least use the types. Needed because this shorthand syntax
// won't allow graphql-codegen to generate types for us. In future, we can 
// write a custom plugin or script to create gql types for us, which can then 
// be used in the codegen