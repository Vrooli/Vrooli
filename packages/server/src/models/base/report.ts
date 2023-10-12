import { GqlModelType, MaxObjects, ReportFor, ReportSortBy, reportValidation } from "@local/shared";
import { Prisma, ReportStatus } from "@prisma/client";
import i18next from "i18next";
import { ModelMap } from ".";
import { CustomError } from "../../events/error";
import { getSingleTypePermissions } from "../../validators";
import { ReportFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ReportModelLogic } from "./types";

const forMapper: { [key in ReportFor]: keyof Prisma.reportUpsertArgs["create"] } = {
    ApiVersion: "apiVersion",
    ChatMessage: "chatMessage",
    Comment: "comment",
    Issue: "issue",
    Organization: "organization",
    NoteVersion: "noteVersion",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    SmartContractVersion: "smartContractVersion",
    StandardVersion: "standardVersion",
    Tag: "tag",
    User: "user",
};

const __typename = "Report" as const;
export const ReportModel: ReportModelLogic = ({
    __typename,
    delegate: (prisma) => prisma.report,
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(forMapper)) {
                    if (select[value]) return ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                }
                return i18next.t("common:Report", { lng: languages[0], count: 1 });
            },
        },
    }),
    format: ReportFormat,
    mutate: {
        shape: {
            pre: async ({ Create, prisma, userData }) => {
                // Make sure user does not have any open reports on these objects
                if (Create.length) {
                    const existing = await prisma.report.findMany({
                        where: {
                            status: "Open",
                            user: { id: userData.id },
                            OR: Create.map((x) => ({
                                [forMapper[x.input.createdFor]]: { id: x.input.createdForConnect },
                            })),
                        },
                    });
                    if (existing.length > 0)
                        throw new CustomError("0337", "MaxReportsReached", userData.languages);
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
                    [forMapper[data.createdFor]]: { connect: { id: data.createdForConnect } },
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
            afterMutations: async ({ createdIds, prisma, userData }) => {
                for (const objectId of createdIds) {
                    // await Trigger(prisma, userData.languages).reportActivity({
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
            commentId: true,
            createdTimeFrame: true,
            fromId: true,
            issueId: true,
            languageIn: true,
            noteVersionId: true,
            organizationId: true,
            postId: true,
            projectVersionId: true,
            routineVersionId: true,
            smartContractVersionId: true,
            standardVersionId: true,
            tagId: true,
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
    validate: () => ({
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            createdBy: "User",
        }),
        permissionResolvers: ({ data, isAdmin, isLoggedIn }) => ({
            canConnect: () => isLoggedIn && data.status !== "Open",
            canDisconnect: () => isLoggedIn,
            canDelete: () => isLoggedIn && isAdmin && data.status !== "Open",
            canRead: () => true,
            canRespond: () => isLoggedIn && data.status === "Open",
            canUpdate: () => isLoggedIn && isAdmin && data.status !== "Open",
        }),
        owner: (data) => ({
            User: data?.createdBy,
        }),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ["reason", "details"],
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    }),
});
