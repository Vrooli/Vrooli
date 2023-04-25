import { PullRequest, PullRequestCreateInput, PullRequestFromObjectType, PullRequestSearchInput, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType, PullRequestUpdateInput, pullRequestValidation, PullRequestYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel, noNull } from "../builders";
import { SelectWrap } from "../builders/types";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
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
import { ModelLogic } from "./types";

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
type Permissions = Pick<PullRequestYou, "canComment" | "canDelete" | "canUpdate" | "canReport">;
const suppFields = ["you"] as const;
export const PullRequestModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: PullRequestCreateInput,
    GqlUpdate: PullRequestUpdateInput,
    GqlModel: PullRequest,
    GqlSearch: PullRequestSearchInput,
    GqlSort: PullRequestSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.pull_requestUpsertArgs["create"],
    PrismaUpdate: Prisma.pull_requestUpsertArgs["update"],
    PrismaModel: Prisma.pull_requestGetPayload<SelectWrap<Prisma.pull_requestSelect>>,
    PrismaSelect: Prisma.pull_requestSelect,
    PrismaWhere: Prisma.pull_requestWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.pull_request,
    display: {
        select: () => ({
            id: true,
            toApi: { select: ApiModel.display.select() },
            fromApiVersion: { select: ApiVersionModel.display.select() },
            toNote: { select: NoteModel.display.select() },
            fromNoteVersion: { select: NoteVersionModel.display.select() },
            toProject: { select: ProjectModel.display.select() },
            fromProjectVersion: { select: ProjectVersionModel.display.select() },
            toRoutine: { select: RoutineModel.display.select() },
            fromRoutineVersion: { select: RoutineVersionModel.display.select() },
            toSmartContract: { select: SmartContractModel.display.select() },
            fromSmartContractVersion: { select: SmartContractVersionModel.display.select() },
            toStandard: { select: StandardModel.display.select() },
            fromStandardVersion: { select: StandardVersionModel.display.select() },
        }),
        // Label is from -> to
        label: (select, languages) => {
            const from = select.fromApiVersion ? ApiVersionModel.display.label(select.fromApiVersion as any, languages) :
                select.fromNoteVersion ? NoteVersionModel.display.label(select.fromNoteVersion as any, languages) :
                    select.fromProjectVersion ? ProjectVersionModel.display.label(select.fromProjectVersion as any, languages) :
                        select.fromRoutineVersion ? RoutineVersionModel.display.label(select.fromRoutineVersion as any, languages) :
                            select.fromSmartContractVersion ? SmartContractVersionModel.display.label(select.fromSmartContractVersion as any, languages) :
                                select.fromStandardVersion ? StandardVersionModel.display.label(select.fromStandardVersion as any, languages) :
                                    "";
            const to = select.toApi ? ApiModel.display.label(select.toApi as any, languages) :
                select.toNote ? NoteModel.display.label(select.toNote as any, languages) :
                    select.toProject ? ProjectModel.display.label(select.toProject as any, languages) :
                        select.toRoutine ? RoutineModel.display.label(select.toRoutine as any, languages) :
                            select.toSmartContract ? SmartContractModel.display.label(select.toSmartContract as any, languages) :
                                select.toStandard ? StandardModel.display.label(select.toStandard as any, languages) :
                                    "";
            return `${from} -> ${to}`;
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            createdBy: "User",
            comments: "Comment",
            from: {
                fromApiVersion: "ApiVersion",
                fromNoteVersion: "NoteVersion",
                fromProjectVersion: "ProjectVersion",
                fromRoutineVersion: "RoutineVersion",
                fromSmartContractVersion: "SmartContractVersion",
                fromStandardVersion: "StandardVersion",
            },
            to: {
                toApi: "Api",
                toNote: "Note",
                toProject: "Project",
                toRoutine: "Routine",
                toSmartContract: "SmartContract",
                toStandard: "Standard",
            },
        },
        prismaRelMap: {
            __typename,
            fromApiVersion: "ApiVersion",
            fromNoteVersion: "NoteVersion",
            fromProjectVersion: "ProjectVersion",
            fromRoutineVersion: "RoutineVersion",
            fromSmartContractVersion: "SmartContractVersion",
            fromStandardVersion: "StandardVersion",
            toApi: "Api",
            toNote: "Note",
            toProject: "Project",
            toRoutine: "Routine",
            toSmartContract: "SmartContract",
            toStandard: "Standard",
            createdBy: "User",
            comments: "Comment",
        },
        countFields: {
            commentsCount: true,
            translationsCount: true,
        },
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
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                "transTextWrapped",
            ],
        }),
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
