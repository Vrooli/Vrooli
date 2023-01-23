import { memberPartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const memberEndpoint = {
    findOne: toQuery('member', 'FindByIdInput', memberPartial, 'full'),
    findMany: toQuery('members', 'MemberSearchInput', ...toSearch(memberPartial)),
    update: toMutation('memberUpdate', 'MemberUpdateInput', memberPartial, 'full')
}