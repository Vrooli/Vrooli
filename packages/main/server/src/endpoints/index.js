import { makeExecutableSchema } from "@graphql-tools/schema";
import pkg from "lodash";
import * as Api from "./api";
import * as ApiKey from "./apiKey";
import * as ApiVersion from "./apiVersion";
import * as Auth from "./auth";
import * as Award from "./award";
import * as Bookmark from "./bookmark";
import * as BookmarkList from "./bookmarkList";
import * as Chat from "./chat";
import * as ChatInvite from "./chatInvite";
import * as ChatMessage from "./chatMessage";
import * as ChatParticipant from "./chatParticipant";
import * as Comment from "./comment";
import * as Duplicate from "./copy";
import * as DeleteOneOrMany from "./deleteOneOrMany";
import * as Email from "./email";
import * as Feed from "./feed";
import * as FocusMode from "./focusMode";
import * as FocusModeFilter from "./focusModeFilter";
import * as Issue from "./issue";
import * as Label from "./label";
import * as Meeting from "./meeting";
import * as MeetingInvite from "./meetingInvite";
import * as Member from "./member";
import * as MemberInvite from "./memberInvite";
import * as Node from "./node";
import * as NodeEnd from "./nodeEnd";
import * as NodeLink from "./nodeLink";
import * as NodeLinkWhen from "./nodeLinkWhen";
import * as NodeLoop from "./nodeLoop";
import * as NodeLoopWhile from "./nodeLoopWhile";
import * as NodeRoutineList from "./nodeRoutineList";
import * as NodeRoutineListItem from "./nodeRoutineListItem";
import * as Note from "./note";
import * as NoteVersion from "./noteVersion";
import * as Notification from "./notification";
import * as NotificationSubscription from "./notificationSubscription";
import * as Organization from "./organization";
import * as Payment from "./payment";
import * as Phone from "./phone";
import * as Post from "./post";
import * as Premium from "./premium";
import * as Project from "./project";
import * as ProjectVersion from "./projectVersion";
import * as ProjectVersionDirectory from "./projectVersionDirectory";
import * as PullRequest from "./pullRequest";
import * as PushDevice from "./pushDevice";
import * as Question from "./question";
import * as QuestionAnswer from "./questionAnswer";
import * as Quiz from "./quiz";
import * as QuizAttempt from "./quizAttempt";
import * as QuizQuestion from "./quizQuestion";
import * as QuizQuestionResponse from "./quizQuestionResponse";
import * as Reaction from "./reaction";
import * as Reminder from "./reminder";
import * as ReminderItem from "./reminderItem";
import * as ReminderList from "./reminderList";
import * as Report from "./report";
import * as ReportResponse from "./reportResponse";
import * as ReputationHistory from "./reputationHistory";
import * as Resource from "./resource";
import * as ResourceList from "./resourceList";
import * as Role from "./role";
import * as Root from "./root";
import * as Routine from "./routine";
import * as RoutineVersion from "./routineVersion";
import * as RoutineVersionInput from "./routineVersionInput";
import * as RoutineVersionOutput from "./routineVersionOutput";
import * as RunProject from "./runProject";
import * as RunProjectStep from "./runProjectStep";
import * as RunRoutine from "./runRoutine";
import * as RunRoutineInput from "./runRoutineInput";
import * as RunRoutineStep from "./runRoutineStep";
import * as Schedule from "./schedule";
import * as ScheduleException from "./scheduleException";
import * as ScheduleRecurrence from "./scheduleRecurrence";
import * as SmartContract from "./smartContract";
import * as SmartContractVersion from "./smartContractVersion";
import * as Standard from "./standard";
import * as StandardVersion from "./standardVersion";
import * as StatsApi from "./statsApi";
import * as StatsOrganization from "./statsOrganization";
import * as StatsProject from "./statsProject";
import * as StatsQuiz from "./statsQuiz";
import * as StatsRoutine from "./statsRoutine";
import * as StatsSite from "./statsSite";
import * as StatsSmartContract from "./statsSmartContract";
import * as StatsStandard from "./statsStandard";
import * as StatsUser from "./statsUser";
import * as Tag from "./tag";
import * as Transfer from "./transfer";
import * as Translate from "./translate";
import * as Unions from "./unions";
import * as User from "./user";
import * as View from "./view";
import * as Wallet from "./wallet";
const { merge } = pkg;
const schemas = [
    Root,
    Api,
    ApiKey,
    ApiVersion,
    Auth,
    Award,
    Bookmark,
    BookmarkList,
    Chat,
    ChatInvite,
    ChatMessage,
    ChatParticipant,
    Comment,
    DeleteOneOrMany,
    Duplicate,
    Email,
    Feed,
    FocusMode,
    FocusModeFilter,
    Issue,
    Label,
    Meeting,
    MeetingInvite,
    Member,
    MemberInvite,
    Node,
    NodeEnd,
    NodeLink,
    NodeLinkWhen,
    NodeLoop,
    NodeLoopWhile,
    NodeRoutineList,
    NodeRoutineListItem,
    Note,
    NoteVersion,
    Notification,
    NotificationSubscription,
    Organization,
    Payment,
    Phone,
    Post,
    Premium,
    Project,
    ProjectVersion,
    ProjectVersionDirectory,
    PullRequest,
    PushDevice,
    Question,
    QuestionAnswer,
    Quiz,
    QuizAttempt,
    QuizQuestion,
    QuizQuestionResponse,
    Reaction,
    Reminder,
    ReminderItem,
    ReminderList,
    Report,
    ReportResponse,
    ReputationHistory,
    Resource,
    ResourceList,
    Role,
    Routine,
    RoutineVersion,
    RoutineVersionInput,
    RoutineVersionOutput,
    RunProject,
    RunProjectStep,
    RunRoutine,
    RunRoutineInput,
    RunRoutineStep,
    Schedule,
    ScheduleException,
    ScheduleRecurrence,
    SmartContract,
    SmartContractVersion,
    Standard,
    StandardVersion,
    StatsApi,
    StatsOrganization,
    StatsProject,
    StatsQuiz,
    StatsRoutine,
    StatsSite,
    StatsSmartContract,
    StatsStandard,
    StatsUser,
    Tag,
    Transfer,
    Translate,
    Unions,
    User,
    View,
    Wallet,
];
export const schema = makeExecutableSchema({
    typeDefs: schemas.map(m => m.typeDef),
    resolvers: merge(schemas.map(m => m.resolvers)),
});
//# sourceMappingURL=index.js.map