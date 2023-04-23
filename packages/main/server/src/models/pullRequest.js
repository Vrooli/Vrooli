import { PullRequestSortBy, PullRequestStatus } from "@local/consts";
import { pullRequestValidation } from "@local/validation";
import { findFirstRel, noNull } from "../builders";
import { getLogic } from "../getters";
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
const fromMapper = {
    ApiVersion: "fromApiVersion",
    NoteVersion: "fromNoteVersion",
    ProjectVersion: "fromProjectVersion",
    RoutineVersion: "fromRoutineVersion",
    SmartContractVersion: "fromSmartContractVersion",
    StandardVersion: "fromStandardVersion",
};
const toMapper = {
    Api: "toApi",
    Note: "toNote",
    Project: "toProject",
    Routine: "toRoutine",
    SmartContract: "toSmartContract",
    Standard: "toStandard",
};
const __typename = "PullRequest";
const suppFields = ["you"];
export const PullRequestModel = ({
    __typename,
    delegate: (prisma) => prisma.pull_request,
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
        label: (select, languages) => {
            const from = select.fromApiVersion ? ApiVersionModel.display.label(select.fromApiVersion, languages) :
                select.fromNoteVersion ? NoteVersionModel.display.label(select.fromNoteVersion, languages) :
                    select.fromProjectVersion ? ProjectVersionModel.display.label(select.fromProjectVersion, languages) :
                        select.fromRoutineVersion ? RoutineVersionModel.display.label(select.fromRoutineVersion, languages) :
                            select.fromSmartContractVersion ? SmartContractVersionModel.display.label(select.fromSmartContractVersion, languages) :
                                select.fromStandardVersion ? StandardVersionModel.display.label(select.fromStandardVersion, languages) :
                                    "";
            const to = select.toApi ? ApiModel.display.label(select.toApi, languages) :
                select.toNote ? NoteModel.display.label(select.toNote, languages) :
                    select.toProject ? ProjectModel.display.label(select.toProject, languages) :
                        select.toRoutine ? RoutineModel.display.label(select.toRoutine, languages) :
                            select.toSmartContract ? SmartContractModel.display.label(select.toSmartContract, languages) :
                                select.toStandard ? StandardModel.display.label(select.toStandard, languages) :
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
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
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
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: 1000000,
        owner: (data, userId) => {
            if (data.createdBy?.id === userId)
                return ({
                    User: data.createdBy,
                });
            const [onField, onData] = findFirstRel(data, [
                "toApi",
                "toNote",
                "toProject",
                "toRoutine",
                "toSmartContract",
                "toStandard",
            ]);
            const onType = onField.slice(2);
            const { validate } = getLogic(["validate"], onType, ["en"], "ResourceListModel.validate.owner");
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
//# sourceMappingURL=pullRequest.js.map