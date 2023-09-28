import { MaxObjects, ReportFor, ReportSortBy, reportValidation } from "@local/shared";
import { Prisma, ReportStatus } from "@prisma/client";
import { CustomError } from "../../events";
import { getSingleTypePermissions } from "../../validators";
import { ReportFormat } from "../formats";
import { ModelLogic } from "../types";
import { ApiVersionModel } from "./apiVersion";
import { ChatMessageModel } from "./chatMessage";
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
import { ApiVersionModelLogic, ChatMessageModelLogic, CommentModelLogic, IssueModelLogic, NoteVersionModelLogic, OrganizationModelLogic, PostModelLogic, ProjectVersionModelLogic, ReportModelLogic, RoutineVersionModelLogic, SmartContractVersionModelLogic, StandardVersionModelLogic, TagModelLogic, UserModelLogic } from "./types";
import { UserModel } from "./user";

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
const suppFields = ["you"] as const;
export const ReportModel: ModelLogic<ReportModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.report,
    display: {
        label: {
            select: () => ({
                id: true,
                apiVersion: { select: ApiVersionModel.display.label.select() },
                chatMessage: { select: ChatMessageModel.display.label.select() },
                comment: { select: CommentModel.display.label.select() },
                issue: { select: IssueModel.display.label.select() },
                noteVersion: { select: NoteVersionModel.display.label.select() },
                organization: { select: OrganizationModel.display.label.select() },
                post: { select: PostModel.display.label.select() },
                projectVersion: { select: ProjectVersionModel.display.label.select() },
                routineVersion: { select: RoutineVersionModel.display.label.select() },
                smartContractVersion: { select: SmartContractVersionModel.display.label.select() },
                standardVersion: { select: StandardVersionModel.display.label.select() },
                tag: { select: TagModel.display.label.select() },
                user: { select: UserModel.display.label.select() },
            }),
            get: (select, languages) => {
                if (select.apiVersion) return ApiVersionModel.display.label.get(select.apiVersion as ApiVersionModelLogic["PrismaModel"], languages);
                if (select.chatMessage) return ChatMessageModel.display.label.get(select.chatMessage as ChatMessageModelLogic["PrismaModel"], languages);
                if (select.comment) return CommentModel.display.label.get(select.comment as CommentModelLogic["PrismaModel"], languages);
                if (select.issue) return IssueModel.display.label.get(select.issue as IssueModelLogic["PrismaModel"], languages);
                if (select.noteVersion) return NoteVersionModel.display.label.get(select.noteVersion as NoteVersionModelLogic["PrismaModel"], languages);
                if (select.organization) return OrganizationModel.display.label.get(select.organization as OrganizationModelLogic["PrismaModel"], languages);
                if (select.post) return PostModel.display.label.get(select.post as PostModelLogic["PrismaModel"], languages);
                if (select.projectVersion) return ProjectVersionModel.display.label.get(select.projectVersion as ProjectVersionModelLogic["PrismaModel"], languages);
                if (select.routineVersion) return RoutineVersionModel.display.label.get(select.routineVersion as RoutineVersionModelLogic["PrismaModel"], languages);
                if (select.smartContractVersion) return SmartContractVersionModel.display.label.get(select.smartContractVersion as SmartContractVersionModelLogic["PrismaModel"], languages);
                if (select.standardVersion) return StandardVersionModel.display.label.get(select.standardVersion as StandardVersionModelLogic["PrismaModel"], languages);
                if (select.tag) return TagModel.display.label.get(select.tag as TagModelLogic["PrismaModel"], languages);
                if (select.user) return UserModel.display.label.get(select.user as UserModelLogic["PrismaModel"], languages);
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
