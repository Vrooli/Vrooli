import { userFields as fullFields, listUserFields as listFields, profileFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const userEndpoint = {
    profile: toQuery('profile', null, [profileFields], `...profileFields`),
    findOne: toQuery('user', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('users', 'UserSearchInput', [listFields], toSearch(listFields)),
    profileUpdate: toMutation('profileUpdate', 'ProfileUpdateInput', [profileFields], `...profileFields`),
    profileEmailUpdate: toMutation('profileEmailUpdate', 'ProfileEmailUpdateInput', [profileFields], `...profileFields`),
    deleteOne: toMutation('userDeleteOne', 'UserDeleteInput', [], `success`),
    exportData: toMutation('exportData', null, [], null),
}