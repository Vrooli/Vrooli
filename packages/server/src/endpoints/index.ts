import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as Api from './api';
import * as ApiKey from './apiKey';
import * as ApiVersion from './apiVersion';
import * as Auth from './auth';
import * as Comment from './comment';
import * as DeleteOneOrMany from './deleteOneOrMany';
import * as Duplicate from './copy';
import * as Email from './email';
import * as Feed from './feed';
import * as Feedback from './feedback';
import * as History from './history';
import * as Issue from './issue';
import * as Label from './label';
import * as Node from './node';
import * as Note from './note';
import * as NoteVersion from './noteVersion';
import * as Organization from './organization';
import * as Post from './post';
import * as Phone from './phone';
import * as Project from './project';
import * as ProjectVersion from './projectVersion';
import * as Question from './question';
import * as QuestionAnswer from './questionAnswer';
import * as Reminder from './reminder';
import * as ReminderList from './reminderList';
import * as Report from './report';
import * as Resource from './resource';
import * as ResourceList from './resourceList';
import * as Role from './role';
import * as Routine from './routine';
import * as RoutineVersion from './routineVersion';
import * as Run from './run';
import * as RunInput from './runInput';
import * as SmartContract from './smartContract';
import * as SmartContractVersion from './smartContractVersion';
import * as Standard from './standard';
import * as StandardVersion from './standardVersion';
import * as Star from './star';
import * as Stats from './stats';
import * as Tag from './tag';
import * as Translate from './translate';
import * as Unions from './unions';
import * as User from './user';
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
    Comment, 
    DeleteOneOrMany,
    Duplicate, 
    Email, 
    Feed,
    Feedback, 
    History,
    Issue,
    Label,
    Node, 
    Note,
    NoteVersion,
    Organization, 
    Phone,
    Post,
    Project, 
    ProjectVersion,
    Question,
    QuestionAnswer,
    Reminder,
    ReminderList,
    Report,
    Resource, 
    ResourceList, 
    Role,
    Routine, 
    RoutineVersion,
    Run, 
    RunInput, 
    SmartContract,
    SmartContractVersion,
    Standard, 
    StandardVersion,
    Star, 
    Stats,
    Tag, 
    Translate, 
    Unions, 
    User, 
    View, 
    Vote, 
    Wallet
]

export const schema = makeExecutableSchema({
    typeDefs: schemas.map(m => m.typeDef),
    resolvers: merge(schemas.map(m => m.resolvers)),
})