import merge from 'lodash/merge';
import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as Auth from './auth';
import * as Comment from './comment';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Node from './node';
import * as Organization from './organization';
import * as Project from './project';
import * as Report from './report';
import * as Resource from './resource';
import * as Role from './role';
import * as Routine from './routine';
import * as Standard from './standard';
import * as Tag from './tag';
import * as User from './user';

const models = [
    Root, Auth, Comment, Email, Feedback, Node,
    Organization, Project, Report, Resource, Role,
    Routine, Standard, Tag, User
]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(models.map(m => m.resolvers)),
})