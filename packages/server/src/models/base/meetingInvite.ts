import { MaxObjects, MeetingInviteSortBy, meetingInviteValidation } from "@local/shared";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { MeetingInviteFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { MeetingInviteModelInfo, MeetingInviteModelLogic, MeetingModelInfo, MeetingModelLogic } from "./types.js";

const __typename = "MeetingInvite" as const;
export const MeetingInviteModel: MeetingInviteModelLogic = ({
    __typename,
    dbTable: "meeting_invite",
    display: () => ({
        // Label is the meeting label
        label: {
            select: () => ({ id: true, meeting: { select: ModelMap.get<MeetingModelLogic>("Meeting").display().label.select() } }),
            get: (select, languages) => ModelMap.get<MeetingModelLogic>("Meeting").display().label.get(select.meeting as MeetingModelInfo["DbModel"], languages),
        },
    }),
    format: MeetingInviteFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                message: noNull(data.message),
                user: await shapeHelper({ relation: "user", relTypes: ["Connect"], isOneToOne: true, objectType: "User", parentRelationshipName: "meetingsInvited", data, ...rest }),
                meeting: await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, objectType: "Meeting", parentRelationshipName: "invites", data, ...rest }),
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
            teamId: true,
            userId: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                "messageWrapped",
                { meeting: ModelMap.get<MeetingModelLogic>("Meeting").search.searchStringQuery() },
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<MeetingInviteModelInfo["ApiPermission"]>(__typename, ids, userData)),
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
            Team: (data?.meeting as MeetingModelInfo["DbModel"])?.team,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MeetingInviteModelInfo["DbSelect"]>([["meeting", "Meeting"]], ...rest),
        // Not sure which search methods are needed, so we'll add them as needed
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        { meeting: useVisibility("Meeting", "OwnOrPublic", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: null,
            ownPrivate: null,
            ownPublic: null,
            public: null,
        },
    }),
});
