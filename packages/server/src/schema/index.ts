import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as Auth from './auth';
import * as Comment from './comment';
import * as DeleteOne from './deleteOne';
import * as Duplicate from './duplicate';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Log from './log';
import * as Node from './node';
import * as Organization from './organization';
import * as Pages from './pages';
import * as Project from './project';
import * as Report from './report';
import * as Resource from './resource';
import * as ResourceList from './resourceList';
import * as Role from './role';
import * as Routine from './routine';
import * as Run from './run';
import * as RunInput from './runInput';
import * as Standard from './standard';
import * as Star from './star';
import * as Tag from './tag';
import * as User from './user';
import * as View from './view';
import * as Vote from './vote';
import * as Wallet from './wallet';
import pkg from 'lodash';
const { merge } = pkg;

const models = [
    Root, Auth, Comment, DeleteOne, Duplicate, Email, Feedback, Log, Node,
    Organization, Pages, Project, Report, Resource, ResourceList, Role,
    Routine, Run, RunInput, Standard, Star, Tag, User, View, Vote, Wallet
]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(models.map(m => m.resolvers)),
})