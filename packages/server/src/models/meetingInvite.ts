import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput, MeetingInviteYou, PrependString } from '@shared/consts';
import { PrismaType } from "../types";
import { MeetingModel } from "./meeting";
import { ModelLogic } from "./types";
import { getSingleTypePermissions } from "../validators";
import { defaultPermissions, oneIsPublic } from "../utils";

const __typename = 'MeetingInvite' as const;
type Permissions = Pick<MeetingInviteYou, 'canDelete' | 'canUpdate'>;
const suppFields = ['you'] as const;
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
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: {
        select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
        // Label is the meeting label
        label: (select, languages) => MeetingModel.display.label(select.meeting as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            meeting: 'Meeting',
            user: 'User',
        },
        prismaRelMap: {
            __typename,
            meeting: 'Meeting',
            user: 'User',
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    }
                }
            },
        },
    },
    mutate: {} as any,
    search: {} as any,
    validate: {
        isTransferable: false,
        maxObjects: {
            User: 0,
            Organization: 5000,
        },
        permissionsSelect: () => ({
            id: true,
            status: true,
            meeting: 'Meeting',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: (data.meeting as any).organization,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.meeting_inviteSelect>(data, [
            ['meeting', 'Meeting'],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                meeting: MeetingModel.validate!.visibility.owner(userId),
            }),
        }
    },
})