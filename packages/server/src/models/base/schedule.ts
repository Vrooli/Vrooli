import { MaxObjects, ScheduleSortBy, scheduleValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { findFirstRel, noNull, shapeHelper } from "../../builders";
import { getLogic } from "../../getters";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { ScheduleFormat } from "../format/schedule";
import { ModelLogic } from "../types";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
import { FocusModeModelLogic, MeetingModelLogic, RunProjectModelLogic, RunRoutineModelLogic, ScheduleModelLogic } from "./types";

const __typename = "Schedule" as const;
const suppFields = [] as const;
export const ScheduleModel: ModelLogic<ScheduleModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.schedule,
    display: {
        label: {
            select: () => ({
                id: true,
                focusModes: { select: FocusModeModel.display.label.select() },
                meetings: { select: MeetingModel.display.label.select() },
                runProjects: { select: RunProjectModel.display.label.select() },
                runRoutines: { select: RunRoutineModel.display.label.select() },
            }),
            get: (select, languages) => {
                if (select.focusModes && select.focusModes.length > 0) return FocusModeModel.display.label.get(select.focusModes[0] as FocusModeModelLogic["PrismaModel"], languages);
                if (select.meetings && select.meetings.length > 0) return MeetingModel.display.label.get(select.meetings[0] as MeetingModelLogic["PrismaModel"], languages);
                if (select.runProjects && select.runProjects.length > 0) return RunProjectModel.display.label.get(select.runProjects[0] as RunProjectModelLogic["PrismaModel"], languages);
                if (select.runRoutines && select.runRoutines.length > 0) return RunRoutineModel.display.label.get(select.runRoutines[0] as RunRoutineModelLogic["PrismaModel"], languages);
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
            onCreated: ({ created, prisma, userData }) => {
                // TODO should check if schedule is starting soon (i.e. before cron job runs), and handle accordingly
            },
            onUpdated: ({ prisma, updated, updateInputs, userData }) => {
                // TODO should check if schedule is starting soon (i.e. before cron job runs), and handle accordingly
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
                { focusModes: { some: FocusModeModel.search.searchStringQuery() } },
                { meetings: { some: MeetingModel.search.searchStringQuery() } },
                { runProjects: { some: RunProjectModel.search.searchStringQuery() } },
                { runRoutines: { some: RunRoutineModel.search.searchStringQuery() } },
            ],
        }),
    },
    validate: {
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.scheduleSelect>(data, [
            ["focusModes", "FocusMode"],
            ["meetings", "Meeting"],
            ["runProjects", "RunProject"],
            ["runRoutines", "RunRoutine"],
        ], languages),
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            // Find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, [
                "focusModes",
                "meetings",
                "runProjects",
                "runRoutines",
            ]);
            const { validate } = getLogic(["validate"], onField as any, ["en"], "ScheduleModel.validate.owner");
            return Array.isArray(onData) && onData.length > 0 ?
                validate.owner(onData[0], userId)
                : {};
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
                    { focusModes: { some: FocusModeModel.validate.visibility.owner(userId) } },
                    { meetings: { some: MeetingModel.validate.visibility.owner(userId) } },
                    { runProjects: { some: RunProjectModel.validate.visibility.owner(userId) } },
                    { runRoutines: { some: RunRoutineModel.validate.visibility.owner(userId) } },
                ],
            }),
        },
    },
});
