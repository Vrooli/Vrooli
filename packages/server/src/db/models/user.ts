import { PrismaSelect } from "@paljs/plugins";
import { onlyPrimitives } from "../../utils/objectTools";
import { CustomError } from "../../error";
import { CODE } from '@local/shared';
import { BaseModel } from "./base";

// export class UserModel extends BaseModel<any> {
//     create(data: any): Promise<any> {
//         throw new Error("Method not implemented.");
//     }
//     update(id: string, data: any): Promise<any> {
//         throw new Error("Method not implemented.");
//     }
//     delete(id: string): Promise<void> {
//         throw new Error("Method not implemented.");
//     }
//     deleteMany(ids: string[]): Promise<number> {
//         throw new Error("Method not implemented.");
//     }
    
// }

// Validates email address, and returns user data
export async function userFromEmail(email: string, prisma: any) {
    if (!email) throw new CustomError(CODE.BadCredentials);
    // Validate email address
    const emailRow = await prisma.email.findUnique({ where: { emailAddress: email } });
    if (!emailRow) throw new CustomError(CODE.BadCredentials);
    // Find user
    let user = await prisma.user.findUnique({ where: { id: emailRow.userId } });
    if (!user) throw new CustomError(CODE.ErrorUnknown);
    return user;
}

// Upsert a user, with emails and roles
export async function upsertUser({ prisma, info, data }: any) {
    // Remove relationship data, as they are handled on a 
    // case-by-case basis
    let cleanedData = onlyPrimitives(data);
    // Upsert user
    let user;
    if (!data.id) {
        // Check for valid username
        //TODO
        // Make sure username isn't in use
        if (await prisma.user.findUnique({ where: { username: data.username }})) throw new CustomError(CODE.UsernameInUse);
        user = await prisma.user.create({ data: cleanedData })
    } else {
        user = await prisma.user.update({ 
            where: { id: data.id },
            data: cleanedData
        })
    }
    // Upsert emails
    for (const email of (data.emails ?? [])) {
        const emailExists = await prisma.email.findUnique({ where: { emailAddress: email.emailAddress } });
        if (emailExists && emailExists.id !== email.id) throw new CustomError(CODE.EmailInUse);
        if (!email.id) {
            await prisma.email.create({ data: { ...email, id: undefined, user: user.id } })
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
        const roleData = { userId: user.id, roleId: role.id };
        await prisma.user_roles.upsert({
            where: { user_roles_userid_roleid_unique: roleData },
            create: roleData,
            update: roleData
        })
    }
    if (info) {
        const prismaInfo = new PrismaSelect(info).value;
        return await prisma.user.findUnique({ where: { id: user.id }, ...prismaInfo });
    }
    return true;
}