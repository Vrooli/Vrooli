import merge from 'lodash/merge';
import { makeExecutableSchema } from '@graphql-tools/schema';
import * as Root from './root';
import * as Customer from './customer';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Image from './image';
import * as Role from './role';
import * as Task from './task';

const models = [Root, Customer, 
                Email, Feedback, Image, 
                Role, Task]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(...models.map(m => m.resolvers))
})