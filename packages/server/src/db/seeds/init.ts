import bcrypt from 'bcrypt';
import pkg from '@prisma/client';
import { envVariableExists } from '../../utils/envVariableExists.js';
import { PrismaType } from '../../types';
const { AccountStatus } = pkg;
const HASHING_ROUNDS = 8;

export async function init(prisma: PrismaType) {
    console.info('🌱 Starting database intial seed...');

    // First, check if required environment variables are set
    if (['ADMIN_PASSWORD', 'ADMIN_EMAIL'].some(name => !envVariableExists(name))) process.exit(1);

    // Find existing roles
    const role_titles = (await prisma.role.findMany({ select: { title: true } })).map(r => r.title);
    // Specify roles that should exist
    const role_data = [
        ['Customer', 'This role allows a customer to create routines and save their progress.'],
        ['Owner', 'This role grants administrative access. This comes with the ability to \
        approve new customers, change customer information, modify inventory and \
        contact hours, and more.'],
        ['Admin', 'This role grants access to everything. Only for developers']
    ]
    // Add missing roles
    for (const role of role_data) {
        if (!role_titles.includes(role[0])) {
            console.info(`🏗 Creating ${role[0]} role`);
            await prisma.role.create({ data: { title: role[0], description: role[1] } })
        }
    }

    // Determine if admin needs to be added
    const role_admin: any = await prisma.role.findUnique({ where: { title: 'Admin' }, select: { id: true } });
    const has_admin = (await prisma.customer_roles.findMany({ where: { roleId: role_admin.id }})).length > 0;
    if (!has_admin) {
        console.info(`👤 Creating admin account`);
        // Insert admin
        const customer_admin = await prisma.customer.create({ data: {
            username: 'admin account',
            password: bcrypt.hashSync(process.env.ADMIN_PASSWORD || '', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        }});
        // Insert admin email
        await prisma.email.create({ data: {
            emailAddress: process.env.ADMIN_EMAIL || '',
            receivesDeliveryUpdates: false,
            verified: true,
            customerId: customer_admin.id
        }})
        // Associate the admin account with an admin role
        await prisma.customer_roles.create({ data: {
            customerId: customer_admin.id,
            roleId: role_admin.id
        }})
    }
    console.info(`✅ Database seeding complete.`);
}

