import { RunStatus } from "@prisma/client";
import { selectHelper } from "../../builders";
import { Formatter } from "../types";

const __typename = "RunRoutine" as const;
export const RunRoutineFormat: Formatter<ModelRunRoutineLogic> = {
    gqlRelMap: {
        __typename,
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    prismaRelMap: {
        __typename,
        inputs: "RunRoutineInput",
        organization: "Organization",
        routineVersion: "RoutineVersion",
        runProject: "RunProject",
        schedule: "Schedule",
        steps: "RunRoutineStep",
        user: "User",
    },
    countFields: {
        inputsCount: true,
        stepsCount: true,
    },
} + (input.completedComplexity ?? 0),
    contextSwitches: contextSwitches + (input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0),
        status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
            completedAt: new Date(),
                timeElapsed: (timeElapsed ?? 0) + (input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0),
                    steps: {
    create: input.finalStepCreate ? {
        order: input.finalStepCreate.order ?? 1,
        name: input.finalStepCreate.name ?? "",
        contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
        timeElapsed: input.finalStepCreate.timeElapsed,
        status: input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed,
    } as any : undefined,
        update: input.finalStepUpdate ? {
            id: input.finalStepUpdate.id,
            contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
            timeElapsed: input.finalStepUpdate.timeElapsed,
            status: input.finalStepUpdate.status ?? (input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed),
        } as any : undefined,
                        },
                        //TODO
                        // inputs: {
                        //     create: input.finalInputCreate ? {
                        // }
                    },
                    ...selectHelper(partial),
    run = await prisma.run_routine.create({
        data: {
            completedComplexity: input.completedComplexity ?? 0,
            startedAt: new Date(),
            completedAt: new Date(),
            timeElapsed: input.finalStepCreate?.timeElapsed ?? input.finalStepUpdate?.timeElapsed ?? 0,
            contextSwitches: input.finalStepCreate?.contextSwitches ?? input.finalStepUpdate?.contextSwitches ?? 0,
            routineVersionId: input.id,
            status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
            name: input.name ?? "",
            userId: userData.id,
            steps: {
                create: input.finalStepCreate ? {
                    order: input.finalStepCreate.order ?? 1,
                    name: input.finalStepCreate.name ?? "",
                    contextSwitches: input.finalStepCreate.contextSwitches ?? 0,
                    timeElapsed: input.finalStepCreate.timeElapsed,
                    status: input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed,
                } as any : input.finalStepUpdate ? {
                    id: input.finalStepUpdate.id,
                    contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                    timeElapsed: input.finalStepUpdate.timeElapsed,
                    status: input.finalStepUpdate?.status ?? (input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed),
                } : undefined,
            },
            //TODO inputs
        },
        ...selectHelper(partial),
        where: {
            AND: [
                { userId: userData.id },
                { id: input.id },
            ],
            data: {
                status: RunStatus.Cancelled,
            };
