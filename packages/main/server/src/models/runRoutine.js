import { MaxObjects, RunRoutineSortBy } from "@local/consts";
import { runRoutineValidation } from "@local/validation";
import { RunStatus } from "@prisma/client";
import { addSupplementalFields, modelToGql, selectHelper, toPartialGqlInfo } from "../builders";
import { CustomError, Trigger } from "../events";
import { defaultPermissions, oneIsPublic } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { OrganizationModel } from "./organization";
const __typename = "RunRoutine";
const suppFields = ["you"];
export const RunRoutineModel = ({
    __typename,
    danger: {
        async anonymize(prisma, owner) {
            await prisma.run_routine.updateMany({
                where: {
                    userId: owner.__typename === "User" ? owner.id : undefined,
                    organizationId: owner.__typename === "Organization" ? owner.id : undefined,
                    isPrivate: false,
                },
                data: {
                    userId: null,
                    organizationId: null,
                },
            });
        },
        async deleteAll(prisma, owner) {
            return prisma.run_routine.deleteMany({
                where: {
                    userId: owner.__typename === "User" ? owner.id : undefined,
                    organizationId: owner.__typename === "Organization" ? owner.id : undefined,
                },
            }).then(({ count }) => ({ __typename: "Count", count }));
        },
    },
    delegate: (prisma) => prisma.run_routine,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name,
    },
    format: {
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
            routineVersion: "Routine",
            runProject: "RunProject",
            schedule: "Schedule",
            steps: "RunRoutineStep",
            user: "User",
        },
        countFields: {
            inputsCount: true,
            stepsCount: true,
        },
        supplemental: {
            dbFields: ["name"],
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    mutate: {
        shape: {
            pre: async ({ updateList }) => {
                if (updateList.length) {
                }
            },
            create: async ({ data, prisma, userData }) => {
                return {};
            },
            update: async ({ data, prisma, userData }) => {
                return {};
            },
        },
        trigger: {
            onCreated: ({ created, prisma, userData }) => {
                for (const c of created) {
                    if (c.status === RunStatus.InProgress) {
                        Trigger(prisma, userData.languages).runRoutineStart(c.id, userData.id, false);
                    }
                }
            },
            onUpdated: ({ prisma, updated, updateInput, userData }) => {
                for (let i = 0; i < updated.length; i++) {
                    if (updated[i].status === RunStatus.InProgress && updateInput[i].hasOwnProperty("status")) {
                        Trigger(prisma, userData.languages).runRoutineStart(updated[i].id, userData.id, false);
                    }
                    if (updated[i].status === RunStatus.Completed && updateInput[i].hasOwnProperty("status")) {
                        Trigger(prisma, userData.languages).runRoutineComplete(updated[i].id, userData.id, false);
                    }
                    if (updated[i].status === RunStatus.Failed && updateInput[i].hasOwnProperty("status")) {
                        Trigger(prisma, userData.languages).runRoutineFail(updated[i].id, userData.id, false);
                    }
                }
            },
        },
        yup: runRoutineValidation,
    },
    run: {
        async complete(prisma, userData, input, info) {
            const partial = toPartialGqlInfo(info, RunRoutineModel.format.gqlRelMap, userData.languages, true);
            let run;
            if (input.exists) {
                run = await prisma.run_routine.findFirst({
                    where: {
                        AND: [
                            { userId: userData.id },
                            { id: input.id },
                        ],
                    },
                });
                if (!run)
                    throw new CustomError("0180", "NotFound", userData.languages);
                const { timeElapsed, contextSwitches, completedComplexity } = run;
                run = await prisma.run_routine.update({
                    where: { id: input.id },
                    data: {
                        completedComplexity: completedComplexity + (input.completedComplexity ?? 0),
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
                            } : undefined,
                            update: input.finalStepUpdate ? {
                                id: input.finalStepUpdate.id,
                                contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                                timeElapsed: input.finalStepUpdate.timeElapsed,
                                status: input.finalStepUpdate.status ?? (input.wasSuccessful === false ? RunStatus.Failed : RunStatus.Completed),
                            } : undefined,
                        },
                    },
                    ...selectHelper(partial),
                });
            }
            else {
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
                            } : input.finalStepUpdate ? {
                                id: input.finalStepUpdate.id,
                                contextSwitches: input.finalStepUpdate.contextSwitches ?? 0,
                                timeElapsed: input.finalStepUpdate.timeElapsed,
                                status: input.finalStepUpdate?.status ?? (input.wasSuccessful ? RunStatus.Completed : RunStatus.Failed),
                            } : undefined,
                        },
                    },
                    ...selectHelper(partial),
                });
            }
            let converted = modelToGql(run, partial);
            converted = (await addSupplementalFields(prisma, userData, [converted], partial))[0];
            if (input.wasSuccessful)
                await Trigger(prisma, userData.languages).runRoutineComplete(input.id, userData.id, false);
            else
                await Trigger(prisma, userData.languages).runRoutineFail(input.id, userData.id, false);
            return converted;
        },
        async cancel(prisma, userData, input, info) {
            const partial = toPartialGqlInfo(info, RunRoutineModel.format.gqlRelMap, userData.languages, true);
            const object = await prisma.run_routine.findFirst({
                where: {
                    AND: [
                        { userId: userData.id },
                        { id: input.id },
                    ],
                },
            });
            if (!object)
                throw new CustomError("0182", "NotFound", userData.languages);
            const updated = await prisma.run_routine.update({
                where: { id: input.id },
                data: {
                    status: RunStatus.Cancelled,
                },
                ...selectHelper(partial),
            });
            let converted = modelToGql(updated, partial);
            converted = (await addSupplementalFields(prisma, userData, [converted], partial))[0];
            return converted;
        },
    },
    search: {
        defaultSort: RunRoutineSortBy.DateUpdatedDesc,
        sortBy: RunRoutineSortBy,
        searchFields: {
            completedTimeFrame: true,
            createdTimeFrame: true,
            excludeIds: true,
            routineVersionId: true,
            startedTimeFrame: true,
            status: true,
            updatedTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "nameWrapped",
                { routineVersion: RunRoutineModel.search.searchStringQuery() },
            ],
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            organization: "Organization",
            routineVersion: "Routine",
            user: "User",
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({
            Organization: data.organization,
            User: data.user,
        }),
        isDeleted: () => false,
        isPublic: (data, languages) => data.isPrivate === false && oneIsPublic(data, [
            ["organization", "Organization"],
            ["user", "User"],
        ], languages),
        profanityFields: ["name"],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({
                OR: [
                    { user: { id: userId } },
                    { organization: OrganizationModel.query.hasRoleQuery(userId) },
                ],
            }),
        },
    },
});
//# sourceMappingURL=runRoutine.js.map