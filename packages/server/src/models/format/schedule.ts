import { MaxObjects, Schedule, ScheduleCreateInput, ScheduleSearchInput, ScheduleSortBy, ScheduleUpdateInput, scheduleValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, noNull, selPad, shapeHelper } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { getLogic } from "../../getters";
import { PrismaType } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { FocusModeModel } from "./focusMode";
import { MeetingModel } from "./meeting";
import { RunProjectModel } from "./runProject";
import { RunRoutineModel } from "./runRoutine";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "Schedule" as const;
export const ScheduleFormat: Formatter<ModelScheduleLogic> = {
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
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                // TODO should check if schedule is starting soon (i.e. before cron job runs), and handle accordingly
};
