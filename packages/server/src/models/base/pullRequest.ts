import { generatePublicId, MaxObjects, type ModelType, type PullRequestFromObjectType, PullRequestSortBy, PullRequestStatus, type PullRequestToObjectType, pullRequestValidation } from "@vrooli/shared";
import type { Prisma } from "@prisma/client";
import { findFirstRel } from "../../builders/findFirstRel.js";
import { noNull } from "../../builders/noNull.js";
import { useVisibility } from "../../builders/visibilityBuilder.js";
import { translationShapeHelper } from "../../utils/shapes/translationShapeHelper.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { PullRequestFormat } from "../formats.js";
import { SuppFields } from "../suppFields.js";
import { ModelMap } from "./index.js";
import { type PullRequestModelInfo, type PullRequestModelLogic } from "./types.js";

const fromMapper: { [key in PullRequestFromObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    ResourceVersion: "fromResourceVersion",
};

const toMapper: { [key in PullRequestToObjectType]: keyof Prisma.pull_requestUpsertArgs["create"] } = {
    Resource: "toResource",
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
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
                ...Object.fromEntries(Object.entries(toMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                let from = "";
                let to = "";
                for (const [key, value] of Object.entries(fromMapper)) {
                    if (select[value]) {
                        from = ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                        break;
                    }
                }
                for (const [key, value] of Object.entries(toMapper)) {
                    if (select[value]) {
                        to = ModelMap.get(key as ModelType).display().label.get(select[value], languages);
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
                id: BigInt(data.id),
                publicId: generatePublicId(),
                createdBy: { connect: { id: BigInt(rest.userData.id) } },
                [fromMapper[data.fromObjectType]]: { connect: { id: BigInt(data.fromConnect) } },
                [toMapper[data.toObjectType]]: { connect: { id: BigInt(data.toConnect) } },
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
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<PullRequestModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    // NOTE: Combines owner/permissions for creator of pull request and owner 
    // of object that has the pull request
    validate: () => ({
        isDeleted: () => false,
        isPublic: (_data, _getParentInfo?) => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data, userId) => {
            if (!data) return {};
            // If you are the creator, return that
            if (data?.createdBy?.id.toString() === userId) return ({
                User: data.createdBy,
            });
            // Otherwise, find owner from the object that has the pull request
            const [onField, onData] = findFirstRel(data, Object.values(toMapper));
            if (!onField || !onData) return {};
            // Object type is field without the 'to' prefix
            const onType = onField.slice(2) as ModelType;
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
            ...Object.fromEntries(Object.entries(toMapper).map(([key, value]) => [value, key as ModelType])),
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    OR: [ // Either you created it or you are the owner of the object that has the pull request
                        {
                            OR: [
                                { fromResourceVersion: useVisibility("ResourceVersion", "Own", data) },
                            ],
                        },
                        {
                            status: { not: PullRequestStatus.Draft }, // If you didn't create it, is cannot be a draft
                            OR: [
                                { toResource: useVisibility("Resource", "Own", data) },
                            ],
                        },
                    ],
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
                        { toResource: useVisibility("Resource", "Public", data) },
                    ],
                };
            },
        },
    }),
});
