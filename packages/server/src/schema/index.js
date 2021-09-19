import pkg from 'lodash';
const { merge } = pkg;
import { makeExecutableSchema } from 'apollo-server-express';
import * as Root from './root';
import * as Address from './address';
import * as Business from './business';
import * as Customer from './customer';
import * as Discount from './discount';
import * as Email from './email';
import * as Feedback from './feedback';
import * as Image from './image';
import * as Order from './order';
import * as OrderItem from './orderItem';
import * as Phone from './phone';
import * as Product from './product';
import * as ProductTrait from './productTrait';
import * as Role from './role';
import * as Sku from './sku';
import * as Task from './task';

const models = [Root, Address, Business, Customer, Discount, 
                Email, Feedback, Image, Order, OrderItem, 
                Phone, Product, ProductTrait, Role, Sku, Task]

export const schema = makeExecutableSchema({
    typeDefs: models.map(m => m.typeDef),
    resolvers: merge(...models.map(m => m.resolvers))
})