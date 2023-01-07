import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput, MeetingInviteYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { MeetingModel } from "./meeting";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";

const type = 'MeetingInvite' as const;
type Permissions = Pick<MeetingInviteYou, 'canDelete' | 'canEdit'>;
const suppFields = ['you.canDelete', 'you.canEdit'] as const;
export const MeetingInviteModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingInviteCreateInput,
    GqlUpdate: MeetingInviteUpdateInput,
    GqlModel: MeetingInvite,
    GqlSearch: MeetingInviteSearchInput,
    GqlSort: MeetingInviteSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.meeting_inviteUpsertArgs['create'],
    PrismaUpdate: Prisma.meeting_inviteUpsertArgs['update'],
    PrismaModel: Prisma.meeting_inviteGetPayload<SelectWrap<Prisma.meeting_inviteSelect>>,
    PrismaSelect: Prisma.meeting_inviteSelect,
    PrismaWhere: Prisma.meeting_inviteWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: {
        select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
        // Label is the meeting label
        label: (select, languages) => MeetingModel.display.label(select.meeting as any, languages),
    },
    format: {
        gqlRelMap: {
            type,
            meeting: 'Meeting',
            user: 'User',
        },
        prismaRelMap: {
            type,
            meeting: 'Meeting',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                let permissions = await getSingleTypePermissions<Permissions>(type, ids, prisma, userData);
                return {
                    ...(Object.fromEntries(Object.entries(permissions).map(([k, v]) => [`you.${k}`, v])) as PrependString<typeof permissions, 'you.'>),
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})