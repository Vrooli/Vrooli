import { MaxObjects, MeetingInviteSortBy, meetingInviteValidation } from "@local/shared";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { MeetingInviteFormat } from "../formats";
import { ModelLogic } from "../types";
import { MeetingModel } from "./meeting";
import { MeetingInviteModelLogic, MeetingModelLogic } from "./types";

const __typename = "MeetingInvite" as const;
const suppFields = ["you"] as const;
export const MeetingInviteModel: ModelLogic<MeetingInviteModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.meeting_invite,
    display: {
        // Label is the meeting label
        label: {
            select: () => ({ id: true, meeting: { select: MeetingModel.display.label.select() } }),
            get: (select, languages) => MeetingModel.display.label.get(select.meeting as MeetingModelLogic["PrismaModel"], languages),
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
                { meeting: MeetingModel.search.searchStringQuery() },
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
            Organization: (data?.meeting as MeetingModelLogic["PrismaModel"])?.organization,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MeetingInviteModelLogic["PrismaSelect"]>([["meeting", "Meeting"]], ...rest),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                meeting: MeetingModel.validate.visibility.owner(userId),
            }),
        },
    },
});
