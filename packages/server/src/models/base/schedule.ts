import type { Prisma } from "@prisma/client";
import { DEFAULT_LANGUAGE, MaxObjects, ScheduleSortBy, generatePublicId, scheduleValidation, type ModelType, type ScheduleFor } from "@vrooli/shared";
import i18next from "i18next";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { defaultPermissions } from "../../validators/permissions.js";
import { ScheduleFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { type MeetingModelLogic, type ScheduleModelInfo, type ScheduleModelLogic } from "./types.js";

const forMapper: { [key in ScheduleFor]: keyof Prisma.scheduleUpsertArgs["create"] } = {
    Meeting: "meetings",
    Run: "runs",
};

const __typename = "Schedule" as const;
export const ScheduleModel: ScheduleModelLogic = ({
    __typename,
    dbTable: "schedule",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(forMapper)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:Schedule", { lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE, count: 1 });
            },
        },
    }),
    format: ScheduleFormat,
    mutate: {
        shape: {
            create: async ({ additionalData, data, idsCreateToConnect, preMap, userData }) => {
                return {
                    id: BigInt(data.id),
                    publicId: generatePublicId(),
                    startTime: data.startTime,
                    endTime: data.endTime,
                    timezone: data.timezone,
                    user: { connect: { id: BigInt(userData.id) } },
                    exceptions: await shapeHelper({ relation: "exceptions", relTypes: ["Create"], isOneToOne: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                    recurrences: await shapeHelper({ relation: "recurrences", relTypes: ["Create"], isOneToOne: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                    // These relations are treated as one-to-one in the API, but not in the database.
                    // Therefore, the key is pural, but the "relation" passed to shapeHelper is singular.
                    meetings: await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, objectType: "Meeting", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                    runs: await shapeHelper({ relation: "run", relTypes: ["Connect"], isOneToOne: true, objectType: "Run", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                };
            },
            update: async ({ additionalData, data, idsCreateToConnect, preMap, userData }) => {
                return {
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: noNull(data.timezone),
                    exceptions: await shapeHelper({ relation: "exceptions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                    recurrences: await shapeHelper({ relation: "recurrences", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, additionalData, idsCreateToConnect, preMap, userData }),
                };
            },
        },
        trigger: {
            afterMutations: ({ createdIds: _createdIds, updatedIds: _updatedIds, userData: _userData }) => {
                // TODO should check both creates and updates if schedule is starting soon (i.e. before cron job runs), and handle accordingly
            },
        },
        yup: scheduleValidation,
    },
    search: {
        defaultSort: ScheduleSortBy.StartTimeAsc,
        sortBy: ScheduleSortBy,
        searchFields: {
            createdTimeFrame: true,
            endTimeFrame: true,
            scheduleFor: true,
            scheduleForUserId: true,
            startTimeFrame: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                ...Object.entries(forMapper).map(([key, value]) => ({ [value]: { some: ModelMap.getLogic(["search"], key as ModelType).search.searchStringQuery() } })),
            ],
        }),
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: (data, getParentInfo?) => oneIsPublic<ScheduleModelInfo["DbSelect"]>(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType]), data, getParentInfo),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, _userId) => {
            // Extract team ownership from meetings
            const firstMeeting = data?.meetings?.[0] as any;
            const teamOwner = firstMeeting?.team ? { id: firstMeeting.team.id } : null;

            // Extract user ownership from direct user or runs
            const directUser = (data as any)?.user;
            const runUser = (data as any)?.runs?.[0]?.user;
            const userOwner = directUser ? { id: directUser.id } : (runUser ? { id: runUser.id } : null);

            return {
                Team: teamOwner,
                User: userOwner,
            };
        },
        permissionResolvers: (params) => defaultPermissions(params),
        permissionsSelect: () => ({
            id: true,
            user: { select: { id: true } },
            meetings: {
                select: {
                    id: true,
                    team: { select: { id: true } },
                },
                take: 1, // Only need first meeting for ownership
            },
            runs: {
                select: {
                    id: true,
                    user: { select: { id: true } },
                },
                take: 1, // Only need first run for ownership
            },
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
                        // Direct user ownership
                        { user: useVisibility("User", "Own", data) },
                        // Ownership through related objects
                        ...Object.entries(forMapper).map(([key, value]) => ({
                            [value]: { some: useVisibility(key as ModelType, "Own", data) },
                        })),
                    ],
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("Schedule", "Own", data),
                        useVisibility("Schedule", "Public", data),
                    ],
                };
            },
            // Search method not useful for this object because schedules are not explicitly set as private, so we'll return "Own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Schedule", "Own", data);
            },
            ownPublic: function getOwnPublic(data) {
                return useVisibility("Schedule", "Own", data);
            },
            public: function getPublic(data) {
                return {
                    // Can use OR because only one relation will be present
                    OR: [
                        ...Object.entries(forMapper).map(([key, value]) => ({
                            // Custom validation for meetings
                            [value]: key === "Meeting"
                                ? (ModelMap.get<MeetingModelLogic>("Meeting").validate().visibility as any).attendingOrInvited(data)
                                : useVisibility(key as ModelType, "Public", data),
                        })),
                    ],
                };
            },
        },
    }),
});
