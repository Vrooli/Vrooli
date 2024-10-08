import { GqlModelType, MaxObjects, PullRequestFromObjectType, PullRequestSortBy, PullRequestStatus, PullRequestToObjectType, pullRequestValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { findFirstRel } from "../../builders/findFirstRel";
import { noNull } from "../../builders/noNull";
import { useVisibility } from "../../builders/visibilityBuilder";
import { translationShapeHelper } from "../../utils/shapes";
import { getSingleTypePermissions } from "../../validators";
import { PullRequestFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { PullRequestModelInfo, PullRequestModelLogic } from "./types";

const fromMapper: { [key in PullRequestFromObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    ApiVersion: "fromApiVersion",
    CodeVersion: "fromCodeVersion",
    NoteVersion: "fromNoteVersion",
    ProjectVersion: "fromProjectVersion",
    RoutineVersion: "fromRoutineVersion",
    StandardVersion: "fromStandardVersion",
};

const toMapper: { [key in PullRequestToObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    Api: "toApi",
    Code: "toCode",
    Note: "toNote",
    Project: "toProject",
    Routine: "toRoutine",
    Standard: "toStandard",
};

const __typename = "PullRequest" as const;
export const PullRequestModel: PullRequestModelLogic = ({
    __typename,
    dbTable: "pull_request",
    dbTranslationTable: "pull_request_translation",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(fromMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
                ...Object.fromEntries(Object.entries(toMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                let from = "";
                let to = "";
                for (const [key, value] of Object.entries(fromMapper)) {
                    if (select[value]) {
                        from = ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                        break;
                    }
                }
                for (const [key, value] of Object.entries(toMapper)) {
                    if (select[value]) {
                        to = ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                        break;
                    }
                }
                return `${from} -> ${to}`;
            },
        },
    }),
    format: PullRequestFormat,
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                createdBy: { connect: { id: rest.userData.id } },
                [fromMapper[data.fromObjectType]]: { connect: { id: data.fromConnect } },
                [toMapper[data.toObjectType]]: { connect: { id: data.toConnect } },
                translations: await translationShapeHelper({ relTypes: ["Create"], data, ...rest }),
            }),
            // NOTE: Pull request creator can only set status to 'Canceled'. 
            // Owner of object that pull request is on can set status to anything but 'Canceled'.
            // Once out of 'Draft' status, status cannot be set back to 'Draft'.
            // TODO need to update params for shape to account for this (probably). Then need to update this function
            update: async ({ data, ...rest }) => ({
                status: noNull(data.status),
                translations: await translationShapeHelper({ relTypes: ["Create", "Update", "Delete"], data, ...rest }),
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
            toGraphQL: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<PullRequestModelInfo["GqlPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    // NOTE: Combines owner/permissions for creator of pull request and owner 
    // of object that has the pull request
    validate: () => ({
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            if (!data) return {};
            // If you are the creator, return that
            if (data?.createdBy?.id === userId) return ({
                User: data.createdBy,
            });
            // Otherwise, find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, Object.values(toMapper));
            if (!onField || !onData) return {};
            // Object type is field without the 'to' prefix
            const onType = onField.slice(2) as GqlModelType;
            const validate = ModelMap.get(onType).validate;
            return validate().owner(onData, userId);
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
            ...Object.fromEntries(Object.entries(toMapper).map(([key, value]) => [value, key as GqlModelType])),
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [ // Either you created it or you are the owner of the object that has the pull request
                        {
                            OR: [
                                { fromApiVersion: useVisibility("ApiVersion", "Own", data) },
                                { fromCodeVersion: useVisibility("CodeVersion", "Own", data) },
                                { fromNoteVersion: useVisibility("NoteVersion", "Own", data) },
                                { fromProjectVersion: useVisibility("ProjectVersion", "Own", data) },
                                { fromRoutineVersion: useVisibility("RoutineVersion", "Own", data) },
                                { fromStandardVersion: useVisibility("StandardVersion", "Own", data) },
                            ]
                        },
                        {
                            status: { not: PullRequestStatus.Draft }, // If you didn't create it, is cannot be a draft
                            OR: [
                                { toApi: useVisibility("Api", "Own", data) },
                                { toCode: useVisibility("Code", "Own", data) },
                                { toNote: useVisibility("Note", "Own", data) },
                                { toProject: useVisibility("Project", "Own", data) },
                                { toRoutine: useVisibility("Routine", "Own", data) },
                                { toStandard: useVisibility("Standard", "Own", data) },
                            ]
                        }
                    ]
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                return {
                    OR: [
                        useVisibility("PullRequest", "Own", data),
                        useVisibility("PullRequest", "Public", data),
                    ],
                };
            },
            ownPrivate: null,
            ownPublic: function getOwnPublic(data) {
                return useVisibility("PullRequest", "Own", data);
            },
            public: function getPublic(data) {
                return {
                    status: { not: PullRequestStatus.Draft },
                    OR: [
                        { toApi: useVisibility("Api", "Public", data) },
                        { toCode: useVisibility("Code", "Public", data) },
                        { toNote: useVisibility("Note", "Public", data) },
                        { toProject: useVisibility("Project", "Public", data) },
                        { toRoutine: useVisibility("Routine", "Public", data) },
                        { toStandard: useVisibility("Standard", "Public", data) },
                    ]
                };
            },
        },
    }),
});
