import { MaxObjects, MeetingInviteSortBy } from "@local/consts";
import { meetingInviteValidation } from "@local/validation";
import { noNull, shapeHelper } from "../builders";
import { defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { MeetingModel } from "./meeting";
const __typename = "MeetingInvite";
const suppFields = ["you"];
export const MeetingInviteModel = ({
    __typename,
    delegate: (prisma) => prisma.meeting_invite,
    display: {
        select: () => ({ id: true, meeting: { select: MeetingModel.display.select() } }),
        label: (select, languages) => MeetingModel.display.label(select.meeting, languages),
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
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { meeting: MeetingModel.search.searchStringQuery() },
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
            Organization: data.meeting.organization,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic(data, [
            ["meeting", "Meeting"],
        ], languages),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                meeting: MeetingModel.validate.visibility.owner(userId),
            }),
        },
    },
});
//# sourceMappingURL=meetingInvite.js.map