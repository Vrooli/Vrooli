import { GqlModelType, MaxObjects, ScheduleSortBy, scheduleValidation, uppercaseFirstLetter } from "@local/shared";
import i18next from "i18next";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { noNull } from "../../builders/noNull";
import { shapeHelper } from "../../builders/shapeHelper";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleFormat } from "../formats";
import { FocusModeModelInfo, FocusModeModelLogic, MeetingModelInfo, MeetingModelLogic, RunProjectModelInfo, RunProjectModelLogic, RunRoutineModelInfo, RunRoutineModelLogic, ScheduleModelInfo, ScheduleModelLogic } from "./types";

const __typename = "Schedule" as const;
export const ScheduleModel: ScheduleModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.schedule,
    display: {
        label: {
            select: () => ({
                id: true,
                focusModes: { select: ModelMap.get<FocusModeModelLogic>("FocusMode").display.label.select() },
                meetings: { select: ModelMap.get<MeetingModelLogic>("Meeting").display.label.select() },
                runProjects: { select: ModelMap.get<RunProjectModelLogic>("RunProject").display.label.select() },
                runRoutines: { select: ModelMap.get<RunRoutineModelLogic>("RunRoutine").display.label.select() },
            }),
            get: (select, languages) => {
                if (select.focusModes && select.focusModes.length > 0) return ModelMap.get<FocusModeModelLogic>("FocusMode").display.label.get(select.focusModes[0] as FocusModeModelInfo["PrismaModel"], languages);
                if (select.meetings && select.meetings.length > 0) return ModelMap.get<MeetingModelLogic>("Meeting").display.label.get(select.meetings[0] as MeetingModelInfo["PrismaModel"], languages);
                if (select.runProjects && select.runProjects.length > 0) return ModelMap.get<RunProjectModelLogic>("RunProject").display.label.get(select.runProjects[0] as RunProjectModelInfo["PrismaModel"], languages);
                if (select.runRoutines && select.runRoutines.length > 0) return ModelMap.get<RunRoutineModelLogic>("RunRoutine").display.label.get(select.runRoutines[0] as RunRoutineModelInfo["PrismaModel"], languages);
                return i18next.t("common:Schedule", { lng: languages[0] });
            },
        },
    },
    format: ScheduleFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                // These relations are treated as one-to-one in the API, but not in the database.
                const difficultOnes = {
                    ...(await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "FocusMode", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Meeting", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "RunProject", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "RunRoutine", parentRelationshipName: "schedule", data, ...rest })),
                };
                return {
                    id: data.id,
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: data.timezone,
                    ...(await shapeHelper({ relation: "exceptions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, ...rest })),
                    // ...(await labelsShapeHelper({ relation: "labels", relTypes: ["Connect", "Create"], isOneToOne: false, isRequired: false, parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "recurrences", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, ...rest })),
                    // Replace each difficult key with plural version
                    ...Object.fromEntries(Object.entries(difficultOnes).map(([k, v]) => [`${k}s`, v])),
                };
            },
            update: async ({ data, ...rest }) => {
                return {
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: noNull(data.timezone),
                    ...(await shapeHelper({ relation: "exceptions", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, ...rest })),
                    // ...(await labelsShapeHelper({ relation: "labels", relTypes: ["Connect", "Create", "Disconnect"], isOneToOne: false, isRequired: false, parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "recurrences", relTypes: ["Create", "Update", "Delete"], isOneToOne: false, isRequired: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, ...rest })),
                };
            },
        },
        trigger: {
            afterMutations: ({ createdIds, updatedIds, prisma, userData }) => {
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
            scheduleForUserId: true,
            startTimeFrame: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                { focusModes: { some: ModelMap.get<FocusModeModelLogic>("FocusMode").search.searchStringQuery() } },
                { meetings: { some: ModelMap.get<MeetingModelLogic>("Meeting").search.searchStringQuery() } },
                { runProjects: { some: ModelMap.get<RunProjectModelLogic>("RunProject").search.searchStringQuery() } },
                { runRoutines: { some: ModelMap.get<RunRoutineModelLogic>("RunRoutine").search.searchStringQuery() } },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (...rest) => oneIsPublic<ScheduleModelInfo["PrismaSelect"]>([
            ["focusModes", "FocusMode"],
            ["meetings", "Meeting"],
            ["runProjects", "RunProject"],
            ["runRoutines", "RunRoutine"],
        ], ...rest),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            if (!data) return {};
            // Find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, [
                "focusModes",
                "meetings",
                "runProjects",
                "runRoutines",
            ]);
            if (!onField || !onData) return {};
            const onType = uppercaseFirstLetter(onField.slice(0, -1)) as GqlModelType;
            const canValidate = Array.isArray(onData) && onData.length > 0;
            if (!canValidate) return {};
            const validate = ModelMap.get(onType).validate;
            return validate.owner(onData[0], userId);
        },
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            focusModes: "FocusMode",
            meetings: "Meeting",
            runProjects: "RunProject",
            runRoutines: "RunRoutine",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                OR: [
                    { focusModes: { some: ModelMap.get<FocusModeModelLogic>("FocusMode").validate.visibility.owner(userId) } },
                    { meetings: { some: ModelMap.get<MeetingModelLogic>("Meeting").validate.visibility.owner(userId) } },
                    { runProjects: { some: ModelMap.get<RunProjectModelLogic>("RunProject").validate.visibility.owner(userId) } },
                    { runRoutines: { some: ModelMap.get<RunRoutineModelLogic>("RunRoutine").validate.visibility.owner(userId) } },
                ],
            }),
        },
    },
});
