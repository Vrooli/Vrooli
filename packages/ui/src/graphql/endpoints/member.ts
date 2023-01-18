import { memberFields as fullFields, listMemberFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const memberEndpoint = {
    findOne: toQuery('member', 'FindByIdInput', fullFields[1]),
    findMany: toQuery('members', 'MemberSearchInput', toSearch(listFields)),
    update: toMutation('memberUpdate', 'MemberUpdateInput', fullFields[1])
}