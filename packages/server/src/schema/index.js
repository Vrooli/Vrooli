import pkg from 'lodash';
const { merge } = pkg;
import { makeExecutableSchema } from 'apollo-server-express';
import * as Root from './root';
import * as Address from './address';
import * as Business from './business';
import * as Customer from './customer';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Image from './image';
import * as Phone from './phone';
import * as Role from './role';
import * as Task from './task';

const models = [Root, Address, Business, Customer, 
                Email, Feedback, Image, 
                Phone, Role, Task]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(...models.map(m => m.resolvers))
})