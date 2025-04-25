import { DEFAULT_LANGUAGE, MaxObjects, ModelType, ScheduleFor, ScheduleSortBy, scheduleValidation, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { findFirstRel } from "../../builders/findFirstRel.js";
import { noNull } from "../../builders/noNull.js";
import { shapeHelper } from "../../builders/shapeHelper.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { defaultPermissions } from "../../utils/defaultPermissions.js";
import { oneIsPublic } from "../../utils/oneIsPublic.js";
import { labelShapeHelper } from "../../utils/shapes/labelShapeHelper.js";
import { ScheduleFormat } from "../formats.js";
import { ModelMap } from "./index.js";
import { MeetingModelLogic, ScheduleModelInfo, ScheduleModelLogic } from "./types.js";

const forMapper: { [key in ScheduleFor]: keyof Prisma.scheduleUpsertArgs["create"] } = {
    Meeting: "meetings",
    RunProject: "runProjects",
    RunRoutine: "runRoutines",
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
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: data.timezone,
                    exceptions: await shapeHelper({ relation: "exceptions", relTypes: ["Create"], isOneToOne: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, ...rest }),
                    labels: await labelShapeHelper({ relation: "labels", relTypes: ["Connect", "Create"], parentType: "Schedule", data, ...rest }),
                    recurrences: await shapeHelper({ relation: "recurrences", relTypes: ["Create"], isOneToOne: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, ...rest }),
                    // These relations are treated as one-to-one in the API, but not in the database.
                    // Therefore, the key is pural, but the "relation" passed to shapeHelper is singular.
                    meetings: await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, objectType: "Meeting", parentRelationshipName: "schedule", data, ...rest }),
                    runProjects: await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, objectType: "RunProject", parentRelationshipName: "schedule", data, ...rest }),
                    runRoutines: await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, objectType: "RunRoutine", parentRelationshipName: "schedule", data, ...rest }),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: noNull(data.timezone),
                    exceptions: await shapeHelper({ relation: "exceptions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, ...rest }),
                    labels: await labelShapeHelper({ relation: "labels", relTypes: ["Connect", "Create", "Disconnect"], parentType: "Schedule", data, ...rest }),
                    recurrences: await shapeHelper({ relation: "recurrences", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, ...rest }),
                };
            },
        },
        trigger: {
            afterMutations: ({ createdIds, updatedIds, userData }) => {
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
        isPublic: (...rest) => oneIsPublic<ScheduleModelInfo["DbSelect"]>(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType]), ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            if (!data) return {};
            // Find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, Object.values(forMapper));
            if (!onField || !onData) return {};
            const onType = uppercaseFirstLetter(onField.slice(0, -1)) as ModelType;
            const canValidate = Array.isArray(onData) && onData.length > 0;
            if (!canValidate) return {};
            return ModelMap.get(onType).validate().owner(onData[0], userId);
        },
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) => [value, key as ModelType])),
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [
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
