import { memberInvitePartial } from '../partial';
import { toMutation, toQuery, toSearch } from '../utils';

export const memberInviteEndpoint = {
    findOne: toQuery('memberInvite', 'FindByIdInput', memberInvitePartial, 'full'),
    findMany: toQuery('memberInvites', 'MemberInviteSearchInput', ...toSearch(memberInvitePartial)),
    create: toMutation('memberInviteCreate', 'MemberInviteCreateInput', memberInvitePartial, 'full'),
    update: toMutation('memberInviteUpdate', 'MemberInviteUpdateInput', memberInvitePartial, 'full'),
    accept: toMutation('memberInviteAccept', 'FindByIdInput', memberInvitePartial, 'full'),
    decline: toMutation('memberInviteDecline', 'FindByIdInput', memberInvitePartial, 'full'),
}