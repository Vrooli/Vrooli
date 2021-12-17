import bcrypt from 'bcrypt';
import pkg from '@prisma/client';
import { PrismaType } from '../../types';
import { ROLES } from '@local/shared';
const { AccountStatus } = pkg;
const HASHING_ROUNDS = 8;

// Create a user with emails and roles
async function createUser({ prisma, userData, emailsData, roleIds }: any) {
    let user = await prisma.user.findFirst({ where: { username: userData.username } });
    if (!user) {
        console.info(`👩🏼‍💻 Creating account for ${userData.username}`);
        // Insert account
        const user = await prisma.user.create({ data: { ...userData } });
        // Insert emails
        for (const email of emailsData) {
            await prisma.email.create({ data: { ...email, userId: user.id }})
        }
        // Insert roles
        for (const roleId of roleIds) {
            await prisma.user_roles.create({ data: { roleId, userId: user.id }})
        }
    }
}

export async function mock(prisma: PrismaType) {
    console.info('🎭 Creating mock data...');

    // Find existing roles
    const roles = await prisma.role.findMany({ select: { id: true, title: true } });
    const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;

    // Create user with owner role
    await createUser({
        prisma,
        userData: {
            username: 'Elon Tusk🦏',
            password: bcrypt.hashSync('Elon', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'notarealemail@afakesite.com', verified: true },
            { emailAddress: 'backupemailaddress@afakesite.com', verified: false }
        ],
        roleIds: [actorRoleId]
    });

    // Create a few users
    await createUser({
        prisma,
        userData: {
            username: 'JohnCena87',
            password: bcrypt.hashSync('John', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'itsjohncena@afakesite.com', verified: true }
        ],
        roleIds: [actorRoleId]
    });
    await createUser({
        prisma,
        userData: {
            username: 'Spongebob Userpants',
            password: bcrypt.hashSync('Spongebob', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'spongebobmeboy@afakesite.com', verified: true }
        ],
        roleIds: [actorRoleId]
    });

    console.info(`✅ Database mock complete.`);
}