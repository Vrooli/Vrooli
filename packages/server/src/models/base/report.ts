import { DEFAULT_LANGUAGE, MaxObjects, ModelType, ReportFor, ReportSearchInput, ReportSortBy, ReportStatus, reportValidation } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next from "i18next";
import { ModelMap } from ".";
import { useVisibility, useVisibilityMapper } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { getSingleTypePermissions } from "../../validators";
import { ReportFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ReportModelInfo, ReportModelLogic } from "./types";

const forMapper: { [key in ReportFor]: keyof Prisma.reportUpsertArgs["create"] } = {
    ApiVersion: "apiVersion",
    ChatMessage: "chatMessage",
    CodeVersion: "codeVersion",
    Comment: "comment",
    Issue: "issue",
    NoteVersion: "noteVersion",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    StandardVersion: "standardVersion",
    Tag: "tag",
    Team: "team",
    User: "user",
};
const reversedForMapper = Object.fromEntries(
    Object.entries(forMapper).map(([key, value]) => [value, key]),
);

const __typename = "Report" as const;
export const ReportModel: ReportModelLogic = ({
    __typename,
    dbTable: "report",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as ModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(forMapper)) {
                    if (select[value]) return ModelMap.get(key as ModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:Report", { lng: languages && languages.length > 0 ? languages[0] : DEFAULT_LANGUAGE, count: 1 });
            },
        },
    }),
    format: ReportFormat,
    mutate: {
        shape: {
            pre: async ({ Create, userData }) => {
                // Make sure user does not have any open reports on these objects
                if (Create.length) {
                    const where = {
                        status: "Open",
                        user: { id: userData.id },
                        OR: Create.map((x) => ({
                            [forMapper[x.input.createdForType]]: { id: x.input.createdForConnect },
                        })),
                    };
                    console.log("report pre where", JSON.stringify(where));
                    const existing = await prismaInstance.report.findMany({
                        where: {
                            status: "Open",
                            user: { id: userData.id },
                            OR: Create.map((x) => ({
                                [forMapper[x.input.createdForType]]: { id: x.input.createdForConnect },
                            })),
                        },
                    });
                    console.log("existing", existing);
                    if (existing.length > 0)
                        throw new CustomError("0337", "MaxReportsReached");
                }
            },
            create: async ({ data, userData }) => {
                return {
                    id: data.id,
                    language: data.language,
                    reason: data.reason,
                    details: data.details,
                    status: ReportStatus.Open,
                    createdBy: { connect: { id: userData.id } },
                    [forMapper[data.createdForType]]: { connect: { id: data.createdForConnect } },
                };
            },
            update: async ({ data }) => {
                return {
                    reason: data.reason ?? undefined,
                    details: data.details,
                };
            },
        },
        trigger: {
            afterMutations: async ({ createdIds, userData }) => {
                for (const objectId of createdIds) {
                    // await Trigger(userData.languages).reportActivity({
                    //     objectId,
                    // });
                }
            },
        },
        yup: reportValidation,
    },
    search: {
        defaultSort: ReportSortBy.DateCreatedDesc,
        sortBy: ReportSortBy,
        searchFields: {
            apiVersionId: true,
            chatMessageId: true,
            codeVersionId: true,
            commentId: true,
            createdTimeFrame: true,
            fromId: true,
            issueId: true,
            languageIn: true,
            noteVersionId: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            standardVersionId: true,
            tagId: true,
            teamId: true,
            updatedTimeFrame: true,
            userId: true,
        },
        searchStringQuery: () => ({
            OR: [
                "detailsWrapped",
                "reasonWrapped",
            ],
        }),
        supplemental: {
            suppFields: SuppFields[__typename],
            getSuppFields: async ({ ids, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<ReportModelInfo["ApiPermission"]>(__typename, ids, userData)),
                    },
                };
            },
        },
    },
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            createdBy: "User",
        }),
        permissionResolvers: ({ data, isAdmin, isLoggedIn, isPublic }) => ({
            canConnect: () => isLoggedIn && data.status !== "Open",
            canDisconnect: () => isLoggedIn,
            canDelete: () => isLoggedIn && isAdmin && data.status !== "Open",
            canRead: () => isPublic,
            canRespond: () => isLoggedIn && data.status === "Open",
            canUpdate: () => isLoggedIn && isAdmin && data.status !== "Open",
            isOwn: () => isAdmin,
        }),
        owner: (data) => ({
            User: data?.createdBy,
        }),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ["reason", "details"],
        visibility: {
            own: function getOwn(data) {
                return {
                    createdBy: { id: data.userId },
                };
            },
            ownOrPublic: function getOwnOrPublic(data) {
                const searchInput = data.searchInput as ReportSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return {
                        OR: [
                            useVisibility("Report", "Own", data),
                            { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "OwnOrPublic", data) },
                        ],
                    };
                }
                // Otherwise, use an OR on all relations
                return {
                    // Can use OR because only one relation will be present
                    OR: [
                        useVisibility("Report", "Own", data),
                        ...useVisibilityMapper("OwnOrPublic", data, forMapper, false),
                    ],
                };
            },
            // Search method not useful for this object because reports are not explicitly set as private, so we'll return "Own"
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("Report", "Own", data);
            },
            ownPublic: function getOwnPublic(data) {
                return useVisibility("Report", "Own", data);
            },
            public: function getPublic(data) {
                const searchInput = data.searchInput as ReportSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as ModelType, "Public", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    // Can use OR because only one relation will be present
                    OR: [
                        ...useVisibilityMapper("Public", data, forMapper, false),
                    ],
                };
            },
        },
    }),
});
