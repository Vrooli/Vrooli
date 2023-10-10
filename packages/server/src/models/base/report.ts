import { MaxObjects, ReportFor, ReportSortBy, reportValidation } from "@local/shared";
import { Prisma, ReportStatus } from "@prisma/client";
import { ModelMap } from ".";
import { CustomError } from "../../events/error";
import { getSingleTypePermissions } from "../../validators";
import { ReportFormat } from "../formats";
import { SuppFields } from "../suppFields";
import { ApiVersionModelInfo, ApiVersionModelLogic, ChatMessageModelInfo, ChatMessageModelLogic, CommentModelInfo, CommentModelLogic, IssueModelInfo, IssueModelLogic, NoteVersionModelInfo, NoteVersionModelLogic, OrganizationModelInfo, OrganizationModelLogic, PostModelInfo, PostModelLogic, ProjectVersionModelInfo, ProjectVersionModelLogic, ReportModelLogic, RoutineVersionModelInfo, RoutineVersionModelLogic, SmartContractVersionModelInfo, SmartContractVersionModelLogic, StandardVersionModelInfo, StandardVersionModelLogic, TagModelInfo, TagModelLogic, UserModelInfo, UserModelLogic } from "./types";

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
    display: {
        label: {
            select: () => ({
                id: true,
                apiVersion: { select: ModelMap.get<ApiVersionModelLogic>("ApiVersion").display.label.select() },
                chatMessage: { select: ModelMap.get<ChatMessageModelLogic>("ChatMessage").display.label.select() },
                comment: { select: ModelMap.get<CommentModelLogic>("Comment").display.label.select() },
                issue: { select: ModelMap.get<IssueModelLogic>("Issue").display.label.select() },
                noteVersion: { select: ModelMap.get<NoteVersionModelLogic>("NoteVersion").display.label.select() },
                organization: { select: ModelMap.get<OrganizationModelLogic>("Organization").display.label.select() },
                post: { select: ModelMap.get<PostModelLogic>("Post").display.label.select() },
                projectVersion: { select: ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").display.label.select() },
                routineVersion: { select: ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.select() },
                smartContractVersion: { select: ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion").display.label.select() },
                standardVersion: { select: ModelMap.get<StandardVersionModelLogic>("StandardVersion").display.label.select() },
                tag: { select: ModelMap.get<TagModelLogic>("Tag").display.label.select() },
                user: { select: ModelMap.get<UserModelLogic>("User").display.label.select() },
            }),
            get: (select, languages) => {
                if (select.apiVersion) return ModelMap.get<ApiVersionModelLogic>("ApiVersion").display.label.get(select.apiVersion as ApiVersionModelInfo["PrismaModel"], languages);
                if (select.chatMessage) return ModelMap.get<ChatMessageModelLogic>("ChatMessage").display.label.get(select.chatMessage as ChatMessageModelInfo["PrismaModel"], languages);
                if (select.comment) return ModelMap.get<CommentModelLogic>("Comment").display.label.get(select.comment as CommentModelInfo["PrismaModel"], languages);
                if (select.issue) return ModelMap.get<IssueModelLogic>("Issue").display.label.get(select.issue as IssueModelInfo["PrismaModel"], languages);
                if (select.noteVersion) return ModelMap.get<NoteVersionModelLogic>("NoteVersion").display.label.get(select.noteVersion as NoteVersionModelInfo["PrismaModel"], languages);
                if (select.organization) return ModelMap.get<OrganizationModelLogic>("Organization").display.label.get(select.organization as OrganizationModelInfo["PrismaModel"], languages);
                if (select.post) return ModelMap.get<PostModelLogic>("Post").display.label.get(select.post as PostModelInfo["PrismaModel"], languages);
                if (select.projectVersion) return ModelMap.get<ProjectVersionModelLogic>("ProjectVersion").display.label.get(select.projectVersion as ProjectVersionModelInfo["PrismaModel"], languages);
                if (select.routineVersion) return ModelMap.get<RoutineVersionModelLogic>("RoutineVersion").display.label.get(select.routineVersion as RoutineVersionModelInfo["PrismaModel"], languages);
                if (select.smartContractVersion) return ModelMap.get<SmartContractVersionModelLogic>("SmartContractVersion").display.label.get(select.smartContractVersion as SmartContractVersionModelInfo["PrismaModel"], languages);
                if (select.standardVersion) return ModelMap.get<StandardVersionModelLogic>("StandardVersion").display.label.get(select.standardVersion as StandardVersionModelInfo["PrismaModel"], languages);
                if (select.tag) return ModelMap.get<TagModelLogic>("Tag").display.label.get(select.tag as TagModelInfo["PrismaModel"], languages);
                if (select.user) return ModelMap.get<UserModelLogic>("User").display.label.get(select.user as UserModelInfo["PrismaModel"], languages);
                return "";
            },
        },
    },
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
    },
});
