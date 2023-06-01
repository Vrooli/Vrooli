import { MaxObjects, NotificationSubscription, NotificationSubscriptionCreateInput, NotificationSubscriptionSearchInput, NotificationSubscriptionSortBy, NotificationSubscriptionUpdateInput, notificationSubscriptionValidation, SubscribableObject } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull } from "../../builders";
import { SelectWrap } from "../../builders/types";
import { PrismaType } from "../../types";
import { defaultPermissions } from "../../utils";
import { ApiModel } from "./api";
import { CommentModel } from "./comment";
import { IssueModel } from "./issue";
import { MeetingModel } from "./meeting";
import { NoteModel } from "./note";
import { OrganizationModel } from "./organization";
import { ProjectModel } from "./project";
import { PullRequestModel } from "./pullRequest";
import { QuestionModel } from "./question";
import { QuizModel } from "./quiz";
import { ReportModel } from "./report";
import { RoutineModel } from "./routine";
import { ScheduleModel } from "./schedule";
import { SmartContractModel } from "./smartContract";
import { StandardModel } from "./standard";
import { ModelLogic } from "./types";
import { Formatter } from "../types";

const __typename = "NotificationSubscription" as const;
export const NotificationSubscriptionFormat: Formatter<ModelNotificationSubscriptionLogic> = {
        gqlRelMap: {
            __typename,
            object: {
                api: "Api",
                comment: "Comment",
                issue: "Issue",
                meeting: "Meeting",
                note: "Note",
                organization: "Organization",
                project: "Project",
                pullRequest: "PullRequest",
                question: "Question",
                quiz: "Quiz",
                report: "Report",
                routine: "Routine",
                schedule: "Schedule",
                smartContract: "SmartContract",
                standard: "Standard",
            },
        },
        prismaRelMap: {
            __typename,
            api: "Api",
            comment: "Comment",
            issue: "Issue",
            meeting: "Meeting",
            note: "Note",
            organization: "Organization",
            project: "Project",
            pullRequest: "PullRequest",
            question: "Question",
            quiz: "Quiz",
            report: "Report",
            routine: "Routine",
            schedule: "Schedule",
            smartContract: "SmartContract",
            standard: "Standard",
            subscriber: "User",
        },
        countFields: {},
    },
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                silent: noNull(data.silent),
                subscriber: { connect: { id: rest.userData.id } },
                [subscribableMapper[data.objectType]]: { connect: { id: data.objectConnect } },
            }),
            update: async ({ data }) => ({
                silent: noNull(data.silent),
            }),
        searchFields: {
            createdTimeFrame: true,
            silent: true,
            objectType: true,
            objectId: true,
            updatedTimeFrame: true,
        searchStringQuery: () => ({
            OR: [
                "descriptionWrapped",
                "titleWrapped",
                { api: ApiModel.search!.searchStringQuery() },
                { comment: CommentModel.search!.searchStringQuery() },
                { issue: IssueModel.search!.searchStringQuery() },
                { meeting: MeetingModel.search!.searchStringQuery() },
                { note: NoteModel.search!.searchStringQuery() },
                { organization: OrganizationModel.search!.searchStringQuery() },
                { project: ProjectModel.search!.searchStringQuery() },
                { pullRequest: PullRequestModel.search!.searchStringQuery() },
                { question: QuestionModel.search!.searchStringQuery() },
                { quiz: QuizModel.search!.searchStringQuery() },
                { report: ReportModel.search!.searchStringQuery() },
                { routine: RoutineModel.search!.searchStringQuery() },
                { schedule: ScheduleModel.search!.searchStringQuery() },
                { smartContract: SmartContractModel.search!.searchStringQuery() },
                { standard: StandardModel.search!.searchStringQuery() },
            ],
        owner: (data) => ({
            User: data.subscriber,
        permissionsSelect: () => ({
            id: true,
            subscriber: "User",
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                subscriber: { id: userId },
            }),
};
