import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE, ORDER_STATUS } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Order;

export const typeDef = gql`
    enum OrderStatus {
        CanceledByAdmin
        CanceledByCustomer
        PendingCancel
        Rejected
        Draft
        Pending
        Approved
        Scheduled
        InTransit
        Delivered
    }

    input OrderInput {
        id: ID
        status: OrderStatus
        specialInstructions: String
        desiredDeliveryDate: Date
        isDelivery: Boolean
        items: [OrderItemInput!]
    }

    type Order {
        id: ID!
        status: OrderStatus!
        specialInstructions: String
        desiredDeliveryDate: Date
        expectedDeliveryDate: Date
        isDelivery: Boolean
        address: Address
        customer: Customer!
        items: [OrderItem!]!
    }

    extend type Query {
        orders(ids: [ID!], status: OrderStatus, searchString: String): [Order!]!
    }

    extend type Mutation {
        updateOrder(input: OrderInput): Order!
        submitOrder(id: ID!): Boolean
        cancelOrder(id: ID!): OrderStatus
        deleteOrders(ids: [ID!]!): Count!
    }
`

const STATUS_TO_SORT = {
    [ORDER_STATUS.CanceledByAdmin]: { updated_at: 'desc' },
    [ORDER_STATUS.CanceledByCustomer]: { updated_at: 'desc' },
    [ORDER_STATUS.PendingCancel]: { updated_at: 'desc' },
    [ORDER_STATUS.Rejected]: { updated_at: 'desc' },
    [ORDER_STATUS.Draft]: { updated_at: 'desc' },
    [ORDER_STATUS.Pending]: { updated_at: 'desc' },
    [ORDER_STATUS.Approved]: { expectedDeliveryDate: 'desc' },
    [ORDER_STATUS.Scheduled]: { expectedDeliveryDate: 'desc' },
    [ORDER_STATUS.InTransit]: { expectedDeliveryDate: 'desc' },
    [ORDER_STATUS.Delivered]: { expectedDeliveryDate: 'desc' },
}

export const resolvers = {
    OrderStatus: ORDER_STATUS,
    Query: {
        orders: async (_, args, context, info) => {
            // Must be admin (customers query SKUs)
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            let idQuery;
            if (Array.isArray(args.ids)) { idQuery = { id: { in: args.ids } } }
            // Determine sort order
            let sortQuery = { updated_at: 'desc' };
            if (args.status) sortQuery = STATUS_TO_SORT[args.status];
            // If search string provided, match it with customer or business name.
            // Maybe in the future, this could also be matched to sku names and such
            let searchQuery;
            if (args.searchString !== undefined && args.searchString.length > 0) {
                searchQuery = {
                    OR: [
                        { customer: { fullName: { contains: args.searchString.trim(), mode: 'insensitive' } } },
                        { customer: { business: { name: { contains: args.searchString.trim(), mode: 'insensitive' } } } }
                    ]
                }
            }
            return await context.prisma[_model].findMany({
                where: {
                    ...idQuery,
                    ...searchQuery,
                    status: args.status ?? undefined,
                },
                orderBy: sortQuery,
                ...(new PrismaSelect(info).value)
            });
        },
    },
    Mutation: {
        updateOrder: async (_, args, context, info) => {
            // Must be admin, or updating your own
            const curr = await context.prisma[_model].findUnique({
                where: { id: args.input.id },
                select: { id: true, customerId: true, status: true, items: { select: { id: true } } }
            });
            if (!context.req.isAdmin && context.req.customerId !== curr.customerId) return new CustomError(CODE.Unauthorized);
            if (!context.req.isAdmin) {
                // Customers can only update their own orders in certain states
                const editable_order_statuses = [ORDER_STATUS.Draft, ORDER_STATUS.Pending];
                if (!editable_order_statuses.includes(curr.status))return new CustomError(CODE.Unauthorized);
                // Customers cannot edit order status
                delete args.input.status;
            }
            // Determine which order item rows need to be updated, and which will be deleted
            if (Array.isArray(args.input.items)) {
                const updatedItemIds = args.input.items.map(i => i.id);
                const deletingItemIds = curr.items.filter(i => !updatedItemIds.includes(i.id)).map(i => i.id);
                if (updatedItemIds.length > 0) {
                    const updateMany = args.input.items.map(d => context.prisma[TABLES.OrderItem].updateMany({
                        where: { id: d.id },
                        data: { ...d }
                    }))
                    Promise.all(updateMany)
                }
                if (deletingItemIds.length > 0) {
                    await context.prisma[TABLES.OrderItem].deleteMany({ where: { id: { in: deletingItemIds } } })
                }
            }
            return await context.prisma[_model].update({
                where: { id: curr.id },
                data: { ...args.input, items: undefined },
                ...(new PrismaSelect(info).value)
            })
        },
        submitOrder: async (_, args, context) => {
            // Must be admin, or submitting your own
            const curr = await context.prisma[_model].findUnique({ where: { id: args.id } });
            if (!context.req.isAdmin && context.req.customerId !== curr.customerId) return new CustomError(CODE.Unauthorized);
            // Only orders in the draft state can be submitted
            if (curr.status !== ORDER_STATUS.Draft) return new CustomError(CODE.ErrorUnknown);
            await context.prisma[_model].update({
                where: { id: curr.id },
                data: { status: ORDER_STATUS.Pending }
            });
            return true;
        },
        cancelOrder: async (_, args, context) => {
            // Must be admin, or canceling your own
            const curr = await context.prisma[_model].findUnique({ where: { id: args.id } });
            if (!context.req.isAdmin && context.req.customerId !== curr.customerId) return new CustomError(CODE.Unauthorized);
            let order_status = curr.status;
            // Only pending orders can be fully cancelled by customer
            if (curr.status === ORDER_STATUS.Pending) {
                order_status = ORDER_STATUS.CanceledByCustomer;
            }
            const pending_order_statuses = [ORDER_STATUS.Approved, ORDER_STATUS.Scheduled];
            if (curr.status in pending_order_statuses) {
                order_status = ORDER_STATUS.PendingCancel;
            }
            await context.prisma[_model].update({
                where: { id: curr.id },
                data: { status: order_status }
            })
            return order_status;
        },
        deleteOrders: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}