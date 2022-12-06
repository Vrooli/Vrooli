import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as Api from './api';
import * as ApiKey from './apiKey';
import * as ApiVersion from './apiVersion';
import * as Auth from './auth';
import * as Award from './award';
import * as Comment from './comment';
import * as DeleteOneOrMany from './deleteOneOrMany';
import * as Duplicate from './copy';
import * as Email from './email';
import * as Feed from './feed';
import * as Feedback from './feedback';
import * as History from './history';
import * as Issue from './issue';
import * as Label from './label';
import * as Meeting from './meeting';
import * as MeetingInvite from './meetingInvite';
import * as Node from './node';
import * as Note from './note';
import * as NoteVersion from './noteVersion';
import * as Notification from './notification';
import * as NotificationSubscription from './notificationSubscription';
import * as Organization from './organization';
import * as Payment from './payment';
import * as Post from './post';
import * as Phone from './phone';
import * as Premium from './premium';
import * as Project from './project';
import * as ProjectVersion from './projectVersion';
import * as PullRequest from './pullRequest';
import * as PushDevice from './pushDevice';
import * as Question from './question';
import * as QuestionAnswer from './questionAnswer';
import * as Quiz from './quiz';
import * as Reminder from './reminder';
import * as ReminderList from './reminderList';
import * as Report from './report';
import * as ReportResponse from './reportResponse';
import * as ReputationHistory from './reputationHistory';
import * as Resource from './resource';
import * as ResourceList from './resourceList';
import * as Role from './role';
import * as Routine from './routine';
import * as RoutineVersion from './routineVersion';
import * as RunProject from './runProject';
import * as RunProjectSchedule from './runProjectSchedule';
import * as RunProjectStep from './runProjectStep';
import * as RunRoutine from './runRoutine';
import * as RunRoutineInput from './runRoutineInput';
import * as RunRoutineSchedule from './runRoutineSchedule';
import * as RunRoutineStep from './runRoutineStep';
import * as SmartContract from './smartContract';
import * as SmartContractVersion from './smartContractVersion';
import * as Standard from './standard';
import * as StandardVersion from './standardVersion';
import * as Star from './star';
import * as Stats from './stats';
import * as Tag from './tag';
import * as Transfer from './transfer';
import * as Translate from './translate';
import * as Unions from './unions';
import * as User from './user';
import * as UserSchedule from './userSchedule';
import * as View from './view';
import * as Vote from './vote';
import * as Wallet from './wallet';
import pkg from 'lodash';
import { DocumentNode } from 'graphql';
const { merge } = pkg;

export type SchemaType = {
    typeDef: DocumentNode;
    resolvers: { [key: string]: any };
}

const schemas: SchemaType[] = [
    Root,
    Api,
    ApiKey,
    ApiVersion,
    Auth,
    Award,
    Comment,
    DeleteOneOrMany,
    Duplicate,
    Email,
    Feed,
    Feedback,
    History,
    Issue,
    Label,
    Meeting,
    MeetingInvite,
    Node,
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
    PullRequest,
    PushDevice,
    Question,
    QuestionAnswer,
    Quiz,
    Reminder,
    ReminderList,
    Report,
    ReportResponse,
    ReputationHistory,
    Resource,
    ResourceList,
    Role,
    Routine,
    RoutineVersion,
    RunProject,
    RunProjectSchedule,
    RunProjectStep,
    RunRoutine,
    RunRoutineInput,
    RunRoutineSchedule,
    RunRoutineStep,
    SmartContract,
    SmartContractVersion,
    Standard,
    StandardVersion,
    Star,
    Stats,
    Tag,
    Transfer,
    Translate,
    Unions,
    User,
    UserSchedule,
    View,
    Vote,
    Wallet
]

export const schema = makeExecutableSchema({
    typeDefs: schemas.map(m => m.typeDef),
    resolvers: merge(schemas.map(m => m.resolvers)),
})