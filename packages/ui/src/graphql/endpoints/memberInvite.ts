import { memberInviteFields as fullFields, listMemberInviteFields as listFields } from 'graphql/partial';
import { toMutation, toQuery, toSearch } from 'graphql/utils';

export const memberInviteEndpoint = {
    findOne: toQuery('memberInvite', 'FindByIdInput', [fullFields], `...fullFields`),
    findMany: toQuery('memberInvites', 'MemberInviteSearchInput', [listFields], toSearch(listFields)),
    create: toMutation('memberInviteCreate', 'MemberInviteCreateInput', [fullFields], `...fullFields`),
    update: toMutation('memberInviteUpdate', 'MemberInviteUpdateInput', [fullFields], `...fullFields`),
    accept: toMutation('memberInviteAccept', 'FindByIdInput', [fullFields], `...fullFields`),
    decline: toMutation('memberInviteDecline', 'FindByIdInput', [fullFields], `...fullFields`)
}