import { MaxObjects, MeetingInviteSortBy, meetingInviteValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../../builders";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MeetingInviteFormat } from "../format/meetingInvite";
import { ModelLogic } from "../types";
import { MeetingModel } from "./meeting";
import { MeetingInviteModelLogic } from "./types";

const __typename = "MeetingInvite" as const;
const suppFields = ["you"] as const;
export const MeetingInviteModel: ModelLogic<MeetingInviteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.meeting_invite,
    display: {
        // Label is the meeting label
        label: {
            select: () => ({ id: true, meeting: { select: MeetingModel.display.label.select() } }),
            get: (select, languages) => MeetingModel.display.label.get(select.meeting as any, languages),
        },
    },
    format: MeetingInviteFormat,
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