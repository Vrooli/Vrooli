import { PrismaSelect } from "@paljs/plugins";
import { onlyPrimitives } from "../../utils/objectTools";
import { CustomError } from "../../error";
import { CODE } from '@local/shared';

// Validates email address, and returns customer data
export async function customerFromEmail(email: string, prisma: any) {
    if (!email) throw new CustomError(CODE.BadCredentials);
    // Validate email address
    const emailRow = await prisma.email.findUnique({ where: { emailAddress: email } });
    if (!emailRow) throw new CustomError(CODE.BadCredentials);
    // Find customer
    let customer = await prisma.customer.findUnique({ where: { id: emailRow.customerId } });
    if (!customer) throw new CustomError(CODE.ErrorUnknown);
    return customer;
}

// Upsert a customer, with emails and roles
export async function upsertCustomer({ prisma, info, data }: any) {
    // Remove relationship data, as they are handled on a 
    // case-by-case basis
    let cleanedData = onlyPrimitives(data);
    // Upsert customer
    let customer;
    if (!data.id) {
        // Check for valid username
        //TODO
        // Make sure username isn't in use
        if (await prisma.customer.findUnique({ where: { username: data.username }})) throw new CustomError(CODE.UsernameInUse);
        customer = await prisma.customer.create({ data: cleanedData })
    } else {
        customer = await prisma.customer.update({ 
            where: { id: data.id },
            data: cleanedData
        })
    }
    // Upsert emails
    for (const email of (data.emails ?? [])) {
        const emailExists = await prisma.email.findUnique({ where: { emailAddress: email.emailAddress } });
        if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
        if (!email.id) {
            await prisma.email.create({ data: { ...email, id: undefined, customerId: customer.id } })
        } else {
            await prisma.email.update({
                where: { id: email.id },
                data: email
            })
        }
    }
    // Upsert roles
    for (const role of (data.roles ?? [])) {
        if (!role.id) continue;
        const roleData = { customerId: customer.id, roleId: role.id };
        await prisma.customer_roles.upsert({
            where: { customer_roles_customerid_roleid_unique: roleData },
            create: roleData,
            update: roleData
        })
    }
    if (info) {
        const prismaInfo = new PrismaSelect(info).value;
        return await prisma.customer.findUnique({ where: { id: customer.id }, ...prismaInfo });
    }
    return true;
}