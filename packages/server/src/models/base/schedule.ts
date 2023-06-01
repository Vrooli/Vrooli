import { MaxObjects, ScheduleSortBy, scheduleValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, noNull, selPad, shapeHelper } from "../builders";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic, ScheduleModelLogic } from "./types";

const __typename = "Schedule" as const;
const suppFields = [] as const;
export const ScheduleModel: ModelLogic<ScheduleModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.schedule,
    display: {
        label: {
            select: () => ({
                id: true,
                focusModes: selPad(FocusModeModel.display.label.select),
                meetings: selPad(MeetingModel.display.label.select),
                runProjects: selPad(RunProjectModel.display.label.select),
                runRoutines: selPad(RunRoutineModel.display.label.select),
            }),
            get: (select, languages) => {
                if (select.focusModes) return FocusModeModel.display.label.get(select.focusModes as any, languages);
                if (select.meetings) return MeetingModel.display.label.get(select.meetings as any, languages);
                if (select.runProjects) return RunProjectModel.display.label.get(select.runProjects as any, languages);
                if (select.runRoutines) return RunRoutineModel.display.label.get(select.runRoutines as any, languages);
                return "";
            },
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            exceptions: "ScheduleException",
            labels: "Label",
            recurrences: "ScheduleRecurrence",
            runProjects: "RunProject",
            runRoutines: "RunRoutine",
            focusModes: "FocusMode",
            meetings: "Meeting",
        },
        prismaRelMap: {
            __typename,
            exceptions: "ScheduleException",
            labels: "Label",
            recurrences: "ScheduleRecurrence",
            runProjects: "RunProject",
            runRoutines: "RunRoutine",
            focusModes: "FocusMode",
            meetings: "Meeting",
        },
        joinMap: { labels: "label" },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => {
                return {
                    id: data.id,
                    startTime: noNull(data.startTime),
                    endTime: noNull(data.endTime),
                    timezone: data.timezone,
                    ...(await shapeHelper({ relation: "exceptions", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ScheduleException", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "focusMode", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "FocusMode", parentRelationshipName: "schedule", data, ...rest })),
                    // ...(await labelsShapeHelper({ relation: "labels", relTypes: ["Connect", "Create"], isOneToOne: false, isRequired: false, parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "meeting", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "Meeting", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "recurrences", relTypes: ["Create"], isOneToOne: false, isRequired: false, objectType: "ScheduleRecurrence", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "runProject", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "RunProject", parentRelationshipName: "schedule", data, ...rest })),
                    ...(await shapeHelper({ relation: "runRoutine", relTypes: ["Connect"], isOneToOne: true, isRequired: false, objectType: "RunRoutine", parentRelationshipName: "schedule", data, ...rest })),
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
            onUpdated: ({ prisma, updated, updateInput, userData }) => {
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
                { focusModes: { some: FocusModeModel.search!.searchStringQuery() } },
                { meetings: { some: MeetingModel.search!.searchStringQuery() } },
                { runProjects: { some: RunProjectModel.search!.searchStringQuery() } },
                { runRoutines: { some: RunRoutineModel.search!.searchStringQuery() } },
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
                    { focusModes: { some: FocusModeModel.validate!.visibility.owner(userId) } },
                    { meetings: { some: MeetingModel.validate!.visibility.owner(userId) } },
                    { runProjects: { some: RunProjectModel.validate!.visibility.owner(userId) } },
                    { runRoutines: { some: RunRoutineModel.validate!.visibility.owner(userId) } },
                ],
            }),
        },
    },
});
