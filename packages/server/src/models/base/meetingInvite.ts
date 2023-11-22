import { MaxObjects, MeetingInviteSortBy, meetingInviteValidation } from "@local/shared";
import { ModelMap } from ".";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MeetingInviteFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { MeetingInviteModelInfo, MeetingInviteModelLogic, MeetingModelInfo, MeetingModelLogic } from "./types";

const __typename = "MeetingInvite" as const;
export const MeetingInviteModel: MeetingInviteModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.meeting_invite,
    display: () => ({
        // Label is the meeting label
        label: {
            select: () => ({ id: true, meeting: { select: ModelMap.get<MeetingModelLogic>("Meeting").display().label.select() } }),
            get: (select, languages) => ModelMap.get<MeetingModelLogic>("Meeting").display().label.get(select.meeting as MeetingModelInfo["PrismaModel"], languages),
        },
    }),
    format: MeetingInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                message: noNull(data.message),
                ...(await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, objectType: "User", parentRelationshipName: "meetingsInvited", data, ...rest })),
                ...(await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, objectType: "Meeting", parentRelationshipName: "invites", data, ...rest })),
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
            statuses: true,
            meetingId: true,
            userId: true,
            organizationId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { meeting: ModelMap.get<MeetingModelLogic>("Meeting").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            graphqlFields: SuppFields[__typename],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            status: true,
            meeting: "Meeting",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: (data?.meeting as MeetingModelInfo["PrismaModel"])?.organization,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MeetingInviteModelInfo["PrismaSelect"]>([["meeting", "Meeting"]], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                meeting: ModelMap.get<MeetingModelLogic>("Meeting").validate().visibility.owner(userId),
            }),
        },
    }),
});
