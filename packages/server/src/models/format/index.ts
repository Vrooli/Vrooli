import { GqlModelType } from "@local/shared";
import { Formatter } from "../types";
import { ApiFormat } from "./api";
import { ApiKeyFormat } from "./apiKey";
import { ApiVersionFormat } from "./apiVersion";
import { AwardFormat } from "./award";
import { BookmarkFormat } from "./bookmark";
import { BookmarkListFormat } from "./bookmarkList";
import { ChatFormat } from "./chat";
import { ChatInviteFormat } from "./chatInvite";
import { ChatMessageFormat } from "./chatMessage";
import { ChatParticipantFormat } from "./chatParticipant";
import { CommentFormat } from "./comment";
import { EmailFormat } from "./email";
import { FocusModeFormat } from "./focusMode";
import { FocusModeFilterFormat } from "./focusModeFilter";
import { IssueFormat } from "./issue";
import { LabelFormat } from "./label";
import { MeetingFormat } from "./meeting";
import { MeetingInviteFormat } from "./meetingInvite";
import { MemberFormat } from "./member";
import { MemberInviteFormat } from "./memberInvite";
import { NodeFormat } from "./node";
import { NodeEndFormat } from "./nodeEnd";
import { NodeLinkFormat } from "./nodeLink";
import { NodeLinkWhenFormat } from "./nodeLinkWhen";
import { NodeLoopFormat } from "./nodeLoop";
import { NodeLoopWhileFormat } from "./nodeLoopWhile";
import { NodeRoutineListFormat } from "./nodeRoutineList";
import { NodeRoutineListItemFormat } from "./nodeRoutineListItem";
import { NoteFormat } from "./note";
import { NoteVersionFormat } from "./noteVersion";
import { NotificationFormat } from "./notification";
import { NotificationSubscriptionFormat } from "./notificationSubscription";
import { OrganizationFormat } from "./organization";
import { PaymentFormat } from "./payment";
import { PhoneFormat } from "./phone";
import { PostFormat } from "./post";
import { PremiumFormat } from "./premium";
import { ProjectFormat } from "./project";
import { ProjectVersionFormat } from "./projectVersion";
import { ProjectVersionDirectoryFormat } from "./projectVersionDirectory";
import { PullRequestFormat } from "./pullRequest";
import { PushDeviceFormat } from "./pushDevice";
import { QuestionFormat } from "./question";
import { QuestionAnswerFormat } from "./questionAnswer";
import { QuizFormat } from "./quiz";
import { QuizAttemptFormat } from "./quizAttempt";
import { QuizQuestionFormat } from "./quizQuestion";
import { QuizQuestionResponseFormat } from "./quizQuestionResponse";
import { ReactionFormat } from "./reaction";
import { ReactionSummaryFormat } from "./reactionSummary";
import { ReminderFormat } from "./reminder";
import { ReminderItemFormat } from "./reminderItem";
import { ReminderListFormat } from "./reminderList";
import { ReportFormat } from "./report";
import { ReportResponseFormat } from "./reportResponse";
import { ResourceFormat } from "./resource";
import { ResourceListFormat } from "./resourceList";
import { RoleFormat } from "./role";
import { RoutineFormat } from "./routine";
import { RoutineVersionFormat } from "./routineVersion";
import { RoutineVersionInputFormat } from "./routineVersionInput";
import { RoutineVersionOutputFormat } from "./routineVersionOutput";
import { RunProjectFormat } from "./runProject";
import { RunProjectStepFormat } from "./runProjectStep";
import { RunRoutineFormat } from "./runRoutine";
import { RunRoutineInputFormat } from "./runRoutineInput";
import { RunRoutineStepFormat } from "./runRoutineStep";
import { ScheduleFormat } from "./schedule";
import { ScheduleExceptionFormat } from "./scheduleException";
import { ScheduleRecurrenceFormat } from "./scheduleRecurrence";
import { SmartContractFormat } from "./smartContract";
import { SmartContractVersionFormat } from "./smartContractVersion";
import { StandardFormat } from "./standard";
import { StandardVersionFormat } from "./standardVersion";
import { StatsApiFormat } from "./statsApi";
import { StatsOrganizationFormat } from "./statsOrganization";
import { StatsProjectFormat } from "./statsProject";
import { StatsQuizFormat } from "./statsQuiz";
import { StatsRoutineFormat } from "./statsRoutine";
import { StatsSiteFormat } from "./statsSite";
import { StatsSmartContractFormat } from "./statsSmartContract";
import { StatsStandardFormat } from "./statsStandard";
import { StatsUserFormat } from "./statsUser";
import { TagFormat } from "./tag";
import { TransferFormat } from "./transfer";
import { UserFormat } from "./user";
import { ViewFormat } from "./view";
import { WalletFormat } from "./wallet";

/**
 * Maps model types to their respective formatter logic.
 */
export const FormatMap: { [key in GqlModelType]?: Formatter<any> } = {
    Api: ApiFormat,
    ApiKey: ApiKeyFormat,
    ApiVersion: ApiVersionFormat,
    Award: AwardFormat,
    Bookmark: BookmarkFormat,
    BookmarkList: BookmarkListFormat,
    Chat: ChatFormat,
    ChatInvite: ChatInviteFormat,
    ChatMessage: ChatMessageFormat,
    ChatParticipant: ChatParticipantFormat,
    Comment: CommentFormat,
    Email: EmailFormat,
    FocusMode: FocusModeFormat,
    FocusModeFilter: FocusModeFilterFormat,
    Issue: IssueFormat,
    Label: LabelFormat,
    Meeting: MeetingFormat,
    MeetingInvite: MeetingInviteFormat,
    Member: MemberFormat,
    MemberInvite: MemberInviteFormat,
    Node: NodeFormat,
    NodeEnd: NodeEndFormat,
    NodeLink: NodeLinkFormat,
    NodeLinkWhen: NodeLinkWhenFormat,
    NodeLoop: NodeLoopFormat,
    NodeLoopWhile: NodeLoopWhileFormat,
    NodeRoutineList: NodeRoutineListFormat,
    NodeRoutineListItem: NodeRoutineListItemFormat,
    Note: NoteFormat,
    NoteVersion: NoteVersionFormat,
    Notification: NotificationFormat,
    NotificationSubscription: NotificationSubscriptionFormat,
    Organization: OrganizationFormat,
    Payment: PaymentFormat,
    Phone: PhoneFormat,
    Post: PostFormat,
    Premium: PremiumFormat,
    Project: ProjectFormat,
    ProjectVersion: ProjectVersionFormat,
    ProjectVersionDirectory: ProjectVersionDirectoryFormat,
    PullRequest: PullRequestFormat,
    PushDevice: PushDeviceFormat,
    Question: QuestionFormat,
    QuestionAnswer: QuestionAnswerFormat,
    Quiz: QuizFormat,
    QuizAttempt: QuizAttemptFormat,
    QuizQuestion: QuizQuestionFormat,
    QuizQuestionResponse: QuizQuestionResponseFormat,
    Reaction: ReactionFormat,
    ReactionSummary: ReactionSummaryFormat,
    Reminder: ReminderFormat,
    ReminderItem: ReminderItemFormat,
    ReminderList: ReminderListFormat,
    Report: ReportFormat,
    ReportResponse: ReportResponseFormat,
    Resource: ResourceFormat,
    ResourceList: ResourceListFormat,
    Role: RoleFormat,
    Routine: RoutineFormat,
    RoutineVersion: RoutineVersionFormat,
    RoutineVersionInput: RoutineVersionInputFormat,
    RoutineVersionOutput: RoutineVersionOutputFormat,
    RunProject: RunProjectFormat,
    RunProjectStep: RunProjectStepFormat,
    RunRoutine: RunRoutineFormat,
    RunRoutineInput: RunRoutineInputFormat,
    RunRoutineStep: RunRoutineStepFormat,
    Schedule: ScheduleFormat,
    ScheduleException: ScheduleExceptionFormat,
    ScheduleRecurrence: ScheduleRecurrenceFormat,
    SmartContract: SmartContractFormat,
    SmartContractVersion: SmartContractVersionFormat,
    Standard: StandardFormat,
    StandardVersion: StandardVersionFormat,
    StatsApi: StatsApiFormat,
    StatsOrganization: StatsOrganizationFormat,
    StatsProject: StatsProjectFormat,
    StatsQuiz: StatsQuizFormat,
    StatsRoutine: StatsRoutineFormat,
    StatsSite: StatsSiteFormat,
    StatsSmartContract: StatsSmartContractFormat,
    StatsStandard: StatsStandardFormat,
    StatsUser: StatsUserFormat,
    Tag: TagFormat,
    Transfer: TransferFormat,
    User: UserFormat,
    View: ViewFormat,
    Wallet: WalletFormat,
};
