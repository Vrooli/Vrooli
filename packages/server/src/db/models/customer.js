import { PrismaSelect } from "@paljs/plugins";
import { TABLES } from "../tables";
import { onlyPrimitives } from "../../utils/objectTools";
import { CustomError } from "../../error";
import { CODE, ORDER_STATUS } from '@local/shared';

// Validates email address, and returns customer data
export async function customerFromEmail(email, prisma) {
    if (!email) throw new CustomError(CODE.BadCredentials);
    // Validate email address
    const emailRow = await prisma[TABLES.Email].findUnique({ where: { emailAddress: email } });
    if (!emailRow) throw new CustomError(CODE.BadCredentials);
    // Find customer
    let customer = await prisma[TABLES.Customer].findUnique({ where: { id: emailRow.customerId } });
    if (!customer) throw new CustomError(CODE.ErrorUnknown);
    return customer;
}

// 'cart' is not a field or relationship in the database,
// so it must be removed from the select
export function getCustomerSelect(info) {
    let prismaInfo = new PrismaSelect(info).value;
    delete prismaInfo.select.cart;
    return prismaInfo;
}

// 'cart' is not a field or relationship in the database,
// so it must be manually queried
export async function getCart(prisma, info, customerId) {
    const selectInfo = new PrismaSelect(info).value.select.cart;
    let results;
    if (selectInfo) {
        results = await prisma[TABLES.Order].findMany({ 
            where: { customerId: customerId, status: ORDER_STATUS.Draft },
            ...selectInfo
        });
    }
    return results?.length > 0 ? results[0] : null;
}

// Upsert a customer, with business, emails, phones, and roles
export async function upsertCustomer({ prisma, info, data }) {
    // Remove relationship data, as they are handled on a 
    // case-by-case basis
    let cleanedData = onlyPrimitives(data);
    // If user already exists, try to find their business
    let business;
    if (data.id) {
        const businessResults = await prisma[TABLES.Business].findMany({
            where: { employees: { some: { id: data.id } } },
        });
        if (businessResults.length > 0) business = businessResults[0];
    }
    // Upsert business
    if (data.business) {
        if (data.business.id) {
            business = await prisma[TABLES.Business].update({ where: { id: data.business.id }, data: data.business });
        } else {
            business = await prisma[TABLES.Business].create({ data: data.business });
        }
        cleanedData.business = { connect: { id: business.id } }
    }
    // Upsert customer
    let customer;
    if (!data.id) {
        customer = await prisma[TABLES.Customer].create({ data: cleanedData })
    } else {
        customer = await prisma[TABLES.Customer].update({ 
            where: { id: data.id },
            data: cleanedData
        })
    }
    // Upsert emails
    for (const email of (data.emails ?? [])) {
        const emailExists = await prisma[TABLES.Email].findUnique({ where: { emailAddress: email.emailAddress } });
        if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
        if (!email.id) {
            await prisma[TABLES.Email].create({ data: { ...email, id: undefined, customerId: customer.id } })
        } else {
            await prisma[TABLES.Email].update({
                where: { id: email.id },
                data: email
            })
        }
    }
    // Upsert phones
    for (const phone of (data.phones ?? [])) {
        const phoneExists = await prisma[TABLES.Phone].findUnique({ where: { number: phone.number }});
        if (phoneExists && phoneExists.id !== phone.id) throw new CustomError(CODE.PhoneInUse)
        if (!phone.id) {
            await prisma[TABLES.Phone].create({ data: { ...phone, id: undefined, customerId: customer.id } })
        } else {
            await prisma[TABLES.Phone].update({
                where: { id: phone.id },
                data: phone
            })
        }
    }
    // Upsert roles
    for (const role of (data.roles ?? [])) {
        if (!role.id) continue;
        const roleData = { customerId: customer.id, roleId: role.id };
        await prisma[TABLES.CustomerRoles].upsert({
            where: { customer_roles_customerid_roleid_unique: roleData },
            create: roleData,
            update: roleData
        })
    }
    const prismaInfo = getCustomerSelect(info);
    const cart = await getCart(prisma, info, customer.id);
    const customerData = await prisma[TABLES.Customer].findUnique({ where: { id: customer.id }, ...prismaInfo });
    if (cart) customerData.cart = cart;
    return customerData;
}