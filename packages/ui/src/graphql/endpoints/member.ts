import { memberFields as fullFields, listMemberFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const memberEndpoint = {
    findOne: toQuery('member', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('members', 'MemberSearchInput', [listFields], toSearch(listFields)),
    update: toMutation('memberUpdate', 'MemberUpdateInput', [fullFields], `...fullFields`)
}