import { PrismaSelect } from "@paljs/plugins";
import { TABLES } from "../tables";
import { onlyPrimitives } from "../../utils/objectTools";
import { CustomError } from "../../error";
import { CODE } from '@local/shared';

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

// Upsert a customer, with emails and roles
export async function upsertCustomer({ prisma, info, data }) {
    // Remove relationship data, as they are handled on a 
    // case-by-case basis
    let cleanedData = onlyPrimitives(data);
    // Upsert customer
    let customer;
    if (!data.id) {
        // Make sure username isn't in use
        if (await prisma[TABLES.Customer].findUnique({ where: { username: data.username }})) throw new CustomError(CODE.UsernameInUse);
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
    if (info) {
        const prismaInfo = getCustomerSelect(info);
        return await prisma[TABLES.Customer].findUnique({ where: { id: customer.id }, ...prismaInfo });
    }
    return true;
}