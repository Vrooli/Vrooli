import { MaxObjects, ReportSortBy } from "@local/consts";
import { reportValidation } from "@local/validation";
import { ReportStatus } from "@prisma/client";
import { selPad } from "../builders";
import { CustomError } from "../events";
import { getSingleTypePermissions } from "../validators";
import { ApiVersionModel } from "./apiVersion";
import { CommentModel } from "./comment";
import { IssueModel } from "./issue";
import { NoteVersionModel } from "./noteVersion";
import { OrganizationModel } from "./organization";
import { PostModel } from "./post";
import { ProjectVersionModel } from "./projectVersion";
import { RoutineVersionModel } from "./routineVersion";
import { SmartContractVersionModel } from "./smartContractVersion";
import { StandardVersionModel } from "./standardVersion";
import { TagModel } from "./tag";
import { UserModel } from "./user";
const forMapper = {
    ApiVersion: "apiVersion",
    Comment: "comment",
    Issue: "issue",
    Organization: "organization",
    NoteVersion: "noteVersion",
    Post: "post",
    ProjectVersion: "projectVersion",
    RoutineVersion: "routineVersion",
    StandardVersion: "standardVersion",
    Tag: "tag",
    User: "user",
};
const __typename = "Report";
const suppFields = ["you"];
export const ReportModel = ({
    __typename,
    delegate: (prisma) => prisma.report,
    display: {
        select: () => ({
            id: true,
            apiVersion: selPad(ApiVersionModel.display.select),
            comment: selPad(CommentModel.display.select),
            issue: selPad(IssueModel.display.select),
            noteVersion: selPad(NoteVersionModel.display.select),
            organization: selPad(OrganizationModel.display.select),
            post: selPad(PostModel.display.select),
            projectVersion: selPad(ProjectVersionModel.display.select),
            routineVersion: selPad(RoutineVersionModel.display.select),
            smartContractVersion: selPad(SmartContractVersionModel.display.select),
            standardVersion: selPad(StandardVersionModel.display.select),
            tag: selPad(TagModel.display.select),
            user: selPad(UserModel.display.select),
        }),
        label: (select, languages) => {
            if (select.apiVersion)
                return ApiVersionModel.display.label(select.apiVersion, languages);
            if (select.comment)
                return CommentModel.display.label(select.comment, languages);
            if (select.issue)
                return IssueModel.display.label(select.issue, languages);
            if (select.noteVersion)
                return NoteVersionModel.display.label(select.noteVersion, languages);
            if (select.organization)
                return OrganizationModel.display.label(select.organization, languages);
            if (select.post)
                return PostModel.display.label(select.post, languages);
            if (select.projectVersion)
                return ProjectVersionModel.display.label(select.projectVersion, languages);
            if (select.routineVersion)
                return RoutineVersionModel.display.label(select.routineVersion, languages);
            if (select.smartContractVersion)
                return SmartContractVersionModel.display.label(select.smartContractVersion, languages);
            if (select.standardVersion)
                return StandardVersionModel.display.label(select.standardVersion, languages);
            if (select.tag)
                return TagModel.display.label(select.tag, languages);
            if (select.user)
                return UserModel.display.label(select.user, languages);
            return "";
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            responses: "ReportResponse",
        },
        prismaRelMap: {
            __typename,
            responses: "ReportResponse",
        },
        hiddenFields: ["createdById"],
        supplemental: {
            graphqlFields: suppFields,
            dbFields: ["createdById"],
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions(__typename, ids, prisma, userData)),
                    },
                };
            },
        },
        countFields: {
            responsesCount: true,
        },
    },
    mutate: {
        shape: {
            pre: async ({ createList, prisma, userData }) => {
                if (createList.length) {
                    const existing = await prisma.report.findMany({
                        where: {
                            status: "Open",
                            user: { id: userData.id },
                            OR: createList.map((x) => ({
                                [`${forMapper[x.createdFor]}Id`]: { id: x.createdForConnect },
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
            onCreated: ({ created, prisma, userData }) => {
                for (const c of created) {
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
    },
    validate: {
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
            User: data.createdBy,
        }),
        isDeleted: () => false,
        isPublic: () => true,
        profanityFields: ["reason", "details"],
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({ createdBy: { id: userId } }),
        },
    },
});
//# sourceMappingURL=report.js.map