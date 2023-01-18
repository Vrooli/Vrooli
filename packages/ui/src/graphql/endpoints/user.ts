import { userFields as fullFields, listUserFields as listFields, profileFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const userEndpoint = {
    profile: toQuery('profile', null, profileFields[1]),
    findOne: toQuery('user', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('users', 'UserSearchInput', toSearch(listFields)),
    profileUpdate: toMutation('profileUpdate', 'ProfileUpdateInput', profileFields[1]),
    profileEmailUpdate: toMutation('profileEmailUpdate', 'ProfileEmailUpdateInput', profileFields[1]),
    deleteOne: toMutation('userDeleteOne', 'UserDeleteInput', `{ success }`),
    exportData: toMutation('exportData', null, null),
}