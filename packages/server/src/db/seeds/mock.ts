import { PrismaType } from '../../types';
import { AccountStatus, ROLES } from '@local/shared';
import { OrganizationModel, UserModel } from '../../models';

export async function mock(prisma: PrismaType) {
    console.info('🎭 Creating mock data...');

    // Find existing roles
    const roles = await prisma.role.findMany({ select: { id: true, title: true } });
    const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;

    // Create a few users
    const userModel = UserModel(prisma);
    const user1 = await userModel.create({
        username: 'Elon Tusk🦏',
        password: userModel.hashPassword('Elon'),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: 'notarealemail@afakesite.com', verified: true },
                { emailAddress: 'backupemailaddress@afakesite.com', verified: false }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    })
    const user2 = await userModel.create({
        username: 'JohnCena87',
        password: userModel.hashPassword('John'),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: 'itsjohncena@afakesite.com', verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user3 = await userModel.create({
        username: 'Spongebob Userpants',
        password: userModel.hashPassword('Spongebob'),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: 'spongebobmeboy@afakesite.com', verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });

    // // Create a few organizations
    // const organizationModel = OrganizationModel(prisma);
    // await organizationModel.create({
    //     name: 'The Organization',
    //     bio: 'This is a description of the organization',
    //     members: [
    //         user1 ? { user: { id: user1.id } } : undefined,
    //         user2 ? { user: { id: user2.id } } : undefined,
    //         user3 ? { user: { id: user3.id } } : undefined,
    //     ]
    // })

    console.info(`✅ Database mock complete.`);
}