import { MaxObjects, MeetingInviteSortBy, MeetingInviteStatus, meetingInviteValidation, uuidValidate } from "@local/shared";
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
                status: MeetingInviteStatus.Pending,
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
            user: "User",
        }),
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic, userId }) => {
            const inviteUserId = data.userId ?? data.user?.id;
            const isYourInvite = uuidValidate(userId) && inviteUserId === userId;
            const basePermissions = defaultPermissions({ isAdmin, isDeleted, isLoggedIn, isPublic });
            return {
                ...basePermissions,
                canAccept: () => isYourInvite,
                canDecline: () => isYourInvite,
                canRead: () => basePermissions.canRead() || isYourInvite,
            };
        },
        owner: (data) => ({
            Team: (data?.meeting as MeetingModelInfo["DbModel"])?.team,
        }),
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<MeetingInviteModelInfo["DbSelect"]>([["meeting", "Meeting"]], ...rest),
        visibility: {
            own: function getOwn(data) {
                return { // If you created the invite or were invited
                    OR: [
                        { meeting: useVisibility("Meeting", "Own", data) },
                        { user: { id: data.userId } },
                    ],
                };
            },
            ownOrPublic: function getOwnPrivate(data) {
                return useVisibility("MeetingInvite", "Own", data);
            },
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("MeetingInvite", "Own", data);
            },
            ownPublic: null, // Search method disabled
            public: null, // Search method disabled
        },
    }),
});
