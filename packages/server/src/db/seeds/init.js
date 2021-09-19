import { ACCOUNT_STATUS } from '@local/shared';
import { TABLES } from '../tables';
import bcrypt from 'bcrypt';
import { HASHING_ROUNDS } from '../../consts';
import { db } from '../db';

export async function seed() {
    console.info('🌱 Starting database intial seed...');

    // Find existing roles
    const role_titles = (await db(TABLES.Role).select('title')).map(r => r.title);
    // Specify roles that should exist
    const role_data = [
        ['Customer', 'This role allows a customer to order products'],
        ['Owner', 'This role grants administrative access. This comes with the ability to \
        approve new customers, change customer information, modify inventory and \
        contact hours, and more.'],
        ['Admin', 'This role grants access to everything. Only for developers']
    ]
    // Add missing roles
    for (const role of role_data) {
        if (!role_titles.includes(role[0])) {
            console.info(`🏗 Creating ${role[0]} role`);
            await db(TABLES.Role).insert({
                title: role[0],
                description: role[1],
            });
        }
    }

    // Determine if admin needs to be added
    const role_admin_id = (await db(TABLES.Role).select('id').where('title', 'Admin').first()).id;
    const has_admin = (await db(TABLES.CustomerRoles).where('roleId', role_admin_id)).length > 0;
    if (!has_admin) {
        console.info(`👩🏼‍💻 Creating admin account`);
        // Insert admin's business
        const business_id = (await db(TABLES.Business).insert([
            {
                name: 'Admin'
            }
        ]).returning('id'))[0];
        // Insert admin
        const customer_admin_id = (await db(TABLES.Customer).insert([
            {
                firstName: 'admin',
                lastName: 'account',
                password: bcrypt.hashSync(process.env.ADMIN_PASSWORD, HASHING_ROUNDS),
                emailVerified: true,
                status: ACCOUNT_STATUS.Unlocked,
                businessId: business_id
            }
        ]).returning('id'))[0];

        // Insert admin email
        await db(TABLES.Email).insert([
            {
                emailAddress: process.env.ADMIN_EMAIL,
                receivesDeliveryUpdates: false,
                customerId: customer_admin_id
            }
        ])

        // Associate the admin account with an admin role
        await db(TABLES.CustomerRoles).insert([
            {
                customerId: customer_admin_id,
                roleId: role_admin_id
            }
        ])
    }
    console.info(`✅ Database seeding complete.`);
}