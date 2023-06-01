import { MaxObjects, MeetingInvite, MeetingInviteCreateInput, MeetingInviteSearchInput, MeetingInviteSortBy, MeetingInviteUpdateInput, meetingInviteValidation, MeetingInviteYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { MeetingModel } from "./meeting";
import { ModelLogic } from "./types";

const __typename = "MeetingInvite" as const;
type Permissions = Pick<MeetingInviteYou, "canDelete" | "canUpdate">;
const suppFields = ["you"] as const;
export const MeetingInviteModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: MeetingInviteCreateInput,
    GqlUpdate: MeetingInviteUpdateInput,
    GqlModel: MeetingInvite,
    GqlSearch: MeetingInviteSearchInput,
    GqlSort: MeetingInviteSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.meeting_inviteUpsertArgs["create"],
    PrismaUpdate: Prisma.meeting_inviteUpsertArgs["update"],
    PrismaModel: Prisma.meeting_inviteGetPayload<SelectWrap<Prisma.meeting_inviteSelect>>,
    PrismaSelect: Prisma.meeting_inviteSelect,
    PrismaWhere: Prisma.meeting_inviteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.meeting_invite,
    display: {
        // Label is the meeting label
        label: {
            select: () => ({ id: true, meeting: { select: MeetingModel.display.label.select() } }),
            get: (select, languages) => MeetingModel.display.label.get(select.meeting as any, languages),
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            meeting: "Meeting",
            user: "User",
        },
        prismaRelMap: {
            __typename,
            meeting: "Meeting",
            user: "User",
        },
        countFields: {},
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                message: noNull(data.message),
                ...(await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "User", parentRelationshipName: "meetingsInvited", data, ...rest })),
                ...(await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, isRequired: true, objectType: "Meeting", parentRelationshipName: "invites", data, ...rest })),
            }),
            update: async ({ data }) => ({
                message: noNull(data.message),
            }),
        },
        yup: meetingInviteValidation,
    },
    search: {
        defaultSort: MeetingInviteSortBy.DateUpdatedDesc,
        sortBy: MeetingInviteSortBy,
        searchFields: {
            createdTimeFrame: true,
            status: true,
            meetingId: true,
            userId: true,
            organizationId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { meeting: MeetingModel.search!.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            status: true,
            meeting: "Meeting",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: (data.meeting as any).organization,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.meeting_inviteSelect>(data, [
            ["meeting", "Meeting"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                meeting: MeetingModel.validate!.visibility.owner(userId),
            }),
        },
    },
});
