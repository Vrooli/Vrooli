import { GqlModelType, PullRequestFromObjectType, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType, pullRequestValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { noNull } from "../../builders/noNull";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { PullRequestFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ApiModelInfo, ApiModelLogic, ApiVersionModelInfo, ApiVersionModelLogic, NoteModelInfo, NoteModelLogic, NoteVersionModelInfo, NoteVersionModelLogic, ProjectModelInfo, ProjectModelLogic, ProjectVersionModelInfo, ProjectVersionModelLogic, PullRequestModelLogic, RoutineModelInfo, RoutineModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic, SmartContractModelInfo, SmartContractModelLogic, SmartContractVersionModelInfo, SmartContractVersionModelLogic, StandardModelInfo, StandardModelLogic, StandardVersionModelInfo, StandardVersionModelLogic } from "./types";

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
export const PullRequestModel: PullRequestModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.pull_request,
    display: {
        label: {
            select: () => ({
                id: true,
                toApi: { select: ModelMap.get<ApiModelLogic>("Api").display.label.select() },
                fromApiVersion: { select: ModelMap.get<ApiVersionModelLogic>("ApiVersion").display.label.select() },
                toNote: { select: ModelMap.get<NoteModelLogic>("Note").display.label.select() },
                fromNoteVersion: { select: ModelMap.get<NoteVersionModelLogic>("NoteVersion").display.label.select() },
                toProject: { select: ModelMap.get<ProjectModelLogic>("Project").display.label.select() },
                fromProjectVersion: { select: ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").display.label.select() },
                toRoutine: { select: ModelMap.get<RoutineModelLogic>("Routine").display.label.select() },
                fromRoutineVersion: { select: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.select() },
                toSmartContract: { select: ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.select() },
                fromSmartContractVersion: { select: ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion").display.label.select() },
                toStandard: { select: ModelMap.get<StandardModelLogic>("Standard").display.label.select() },
                fromStandardVersion: { select: ModelMap.get<StandardVersionModelLogic>("StandardVersion").display.label.select() },
            }),
            // Label is from -> to
            get: (select, languages) => {
                const from = select.fromApiVersion ? ModelMap.get<ApiVersionModelLogic>("ApiVersion").display.label.get(select.fromApiVersion as ApiVersionModelInfo["PrismaModel"], languages) :
                    select.fromNoteVersion ? ModelMap.get<NoteVersionModelLogic>("NoteVersion").display.label.get(select.fromNoteVersion as NoteVersionModelInfo["PrismaModel"], languages) :
                        select.fromProjectVersion ? ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").display.label.get(select.fromProjectVersion as ProjectVersionModelInfo["PrismaModel"], languages) :
                            select.fromRoutineVersion ? ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.get(select.fromRoutineVersion as RoutineVersionModelInfo["PrismaModel"], languages) :
                                select.fromSmartContractVersion ? ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion").display.label.get(select.fromSmartContractVersion as SmartContractVersionModelInfo["PrismaModel"], languages) :
                                    select.fromStandardVersion ? ModelMap.get<StandardVersionModelLogic>("StandardVersion").display.label.get(select.fromStandardVersion as StandardVersionModelInfo["PrismaModel"], languages) :
                                        "";
                const to = select.toApi ? ModelMap.get<ApiModelLogic>("Api").display.label.get(select.toApi as ApiModelInfo["PrismaModel"], languages) :
                    select.toNote ? ModelMap.get<NoteModelLogic>("Note").display.label.get(select.toNote as NoteModelInfo["PrismaModel"], languages) :
                        select.toProject ? ModelMap.get<ProjectModelLogic>("Project").display.label.get(select.toProject as ProjectModelInfo["PrismaModel"], languages) :
                            select.toRoutine ? ModelMap.get<RoutineModelLogic>("Routine").display.label.get(select.toRoutine as RoutineModelInfo["PrismaModel"], languages) :
                                select.toSmartContract ? ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.get(select.toSmartContract as SmartContractModelInfo["PrismaModel"], languages) :
                                    select.toStandard ? ModelMap.get<StandardModelLogic>("Standard").display.label.get(select.toStandard as StandardModelInfo["PrismaModel"], languages) :
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
            graphqlFields: SuppFields[__typename],
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
            if (!data) return {};
            // If you are the creator, return that
            if (data?.createdBy?.id === userId) return ({
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
            if (!onField || !onData) return {};
            // Object type is field without the 'to' prefix
            const onType = onField.slice(2) as GqlModelType;
            const validate = ModelMap.get(onType).validate;
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
