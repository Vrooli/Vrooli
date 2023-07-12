import { PullRequestFromObjectType, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType, pullRequestValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, noNull } from "../../builders";
import { getLogic } from "../../getters";
import { translationShapeHelper } from "../../utils";
import { getSingleTypePermissions } from "../../validators";
import { PullRequestFormat } from "../format/pullRequest";
import { ModelLogic } from "../types";
import { ApiModel } from "./api";
import { ApiVersionModel } from "./apiVersion";
import { NoteModel } from "./note";
import { NoteVersionModel } from "./noteVersion";
import { ProjectModel } from "./project";
import { ProjectVersionModel } from "./projectVersion";
import { RoutineModel } from "./routine";
import { RoutineVersionModel } from "./routineVersion";
import { SmartContractModel } from "./smartContract";
import { SmartContractVersionModel } from "./smartContractVersion";
import { StandardModel } from "./standard";
import { StandardVersionModel } from "./standardVersion";
import { ApiModelLogic, ApiVersionModelLogic, NoteModelLogic, NoteVersionModelLogic, ProjectModelLogic, ProjectVersionModelLogic, PullRequestModelLogic, RoutineModelLogic, RoutineVersionModelLogic, SmartContractModelLogic, SmartContractVersionModelLogic, StandardModelLogic, StandardVersionModelLogic } from "./types";

const fromMapper: { [key in PullRequestFromObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    ApiVersion: "fromApiVersion",
    NoteVersion: "fromNoteVersion",
    ProjectVersion: "fromProjectVersion",
    RoutineVersion: "fromRoutineVersion",
    SmartContractVersion: "fromSmartContractVersion",
    StandardVersion: "fromStandardVersion",
};

const toMapper: { [key in PullRequestToObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    Api: "toApi",
    Note: "toNote",
    Project: "toProject",
    Routine: "toRoutine",
    SmartContract: "toSmartContract",
    Standard: "toStandard",
};

const __typename = "PullRequest" as const;
const suppFields = ["you"] as const;
export const PullRequestModel: ModelLogic<PullRequestModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.pull_request,
    display: {
        label: {
            select: () => ({
                id: true,
                toApi: { select: ApiModel.display.label.select() },
                fromApiVersion: { select: ApiVersionModel.display.label.select() },
                toNote: { select: NoteModel.display.label.select() },
                fromNoteVersion: { select: NoteVersionModel.display.label.select() },
                toProject: { select: ProjectModel.display.label.select() },
                fromProjectVersion: { select: ProjectVersionModel.display.label.select() },
                toRoutine: { select: RoutineModel.display.label.select() },
                fromRoutineVersion: { select: RoutineVersionModel.display.label.select() },
                toSmartContract: { select: SmartContractModel.display.label.select() },
                fromSmartContractVersion: { select: SmartContractVersionModel.display.label.select() },
                toStandard: { select: StandardModel.display.label.select() },
                fromStandardVersion: { select: StandardVersionModel.display.label.select() },
            }),
            // Label is from -> to
            get: (select, languages) => {
                const from = select.fromApiVersion ? ApiVersionModel.display.label.get(select.fromApiVersion as ApiVersionModelLogic["PrismaModel"], languages) :
                    select.fromNoteVersion ? NoteVersionModel.display.label.get(select.fromNoteVersion as NoteVersionModelLogic["PrismaModel"], languages) :
                        select.fromProjectVersion ? ProjectVersionModel.display.label.get(select.fromProjectVersion as ProjectVersionModelLogic["PrismaModel"], languages) :
                            select.fromRoutineVersion ? RoutineVersionModel.display.label.get(select.fromRoutineVersion as RoutineVersionModelLogic["PrismaModel"], languages) :
                                select.fromSmartContractVersion ? SmartContractVersionModel.display.label.get(select.fromSmartContractVersion as SmartContractVersionModelLogic["PrismaModel"], languages) :
                                    select.fromStandardVersion ? StandardVersionModel.display.label.get(select.fromStandardVersion as StandardVersionModelLogic["PrismaModel"], languages) :
                                        "";
                const to = select.toApi ? ApiModel.display.label.get(select.toApi as ApiModelLogic["PrismaModel"], languages) :
                    select.toNote ? NoteModel.display.label.get(select.toNote as NoteModelLogic["PrismaModel"], languages) :
                        select.toProject ? ProjectModel.display.label.get(select.toProject as ProjectModelLogic["PrismaModel"], languages) :
                            select.toRoutine ? RoutineModel.display.label.get(select.toRoutine as RoutineModelLogic["PrismaModel"], languages) :
                                select.toSmartContract ? SmartContractModel.display.label.get(select.toSmartContract as SmartContractModelLogic["PrismaModel"], languages) :
                                    select.toStandard ? StandardModel.display.label.get(select.toStandard as StandardModelLogic["PrismaModel"], languages) :
                                        "";
                return `${from} -> ${to}`;
            },
        },
    },
    format: PullRequestFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                createdBy: { connect: { id: rest.userData.id } },
                [fromMapper[data.fromObjectType]]: { connect: { id: data.fromConnect } },
                [toMapper[data.toObjectType]]: { connect: { id: data.toConnect } },
                ...(await translationShapeHelper({ relTypes: ["Create"], isRequired: false, data, ...rest })),
            }),
            // NOTE: Pull request creator can only set status to 'Canceled'. 
            // Owner of object that pull request is on can set status to anything but 'Canceled'
            // TODO need to update params for shape to account for this (probably). Then need to update this function
            update: async ({ data, ...rest }) => ({
                status: noNull(data.status),
                ...(await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], isRequired: false, data, ...rest })),
            }),
        },
        yup: pullRequestValidation,
    },
    search: {
        defaultSort: PullRequestSortBy.DateUpdatedDesc,
        sortBy: PullRequestSortBy,
        searchFields: {
            createdTimeFrame: true,
            isMergedOrRejected: true,
            translationLanguages: true,
            toId: true,
            createdById: true,
            tags: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transTextWrapped",
            ],
        }),
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
    },
    // NOTE: Combines owner/permissions for creator of pull request and owner 
    // of object that has the pull request
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: 1000000,
        owner: (data, userId) => {
            // If you are the creator, return that
            if (data.createdBy?.id === userId) return ({
                User: data.createdBy,
            });
            // Otherwise, find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, [
                "toApi",
                "toNote",
                "toProject",
                "toRoutine",
                "toSmartContract",
                "toStandard",
            ]);
            // Object type is field without the 'to' prefix
            const onType = onField!.slice(2);
            const { validate } = getLogic(["validate"], onType as any, ["en"], "ResourceListModel.validate.owner");
            return validate.owner(onData, userId);
        },
        permissionResolvers: ({ data, isAdmin, isDeleted, isLoggedIn, isPublic }) => ({
            canComment: () => isLoggedIn && (isAdmin || isPublic) && !isDeleted && data.status === PullRequestStatus.Open,
            canConnect: () => isLoggedIn,
            canDelete: () => isLoggedIn && isAdmin && !isDeleted,
            canDisconnect: () => isLoggedIn,
            canRead: () => isAdmin || isPublic,
            canReport: () => isLoggedIn && !isAdmin && !isDeleted && isPublic && data.status === PullRequestStatus.Open,
            canUpdate: () => isLoggedIn && isAdmin && !isDeleted,
        }),
        permissionsSelect: () => ({
            id: true,
            createdBy: "User",
            status: true,
            toApi: "Api",
            toNote: "Note",
            toProject: "Project",
            toRoutine: "Routine",
            toSmartContract: "SmartContract",
            toStandard: "Standard",
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                createdBy: { id: userId },
            }),
        },
    },
});
