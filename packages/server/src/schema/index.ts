import merge from 'lodash/merge';
import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as User from './user';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Node from './node';
import * as Organization from './organization';
import * as Project from './project';
import * as Resource from './resource';
import * as Role from './role';
import * as Routine from './routine';
import * as Wallet from './wallet';

const models = [Root, User, 
                Email, Feedback, Node, Organization, Resource,
                Routine, Project, Role, Wallet]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(models.map(m => m.resolvers)),
})