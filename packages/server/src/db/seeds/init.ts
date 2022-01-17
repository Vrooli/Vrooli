/**
 * Adds initial data to the database. (i.e. data that should be included in production). 
 * This is written so that it can be called multiple times without duplicating data.
 */

import { ROLES } from '@local/shared';
import { PrismaType } from '../../types';

export async function init(prisma: PrismaType) {
    console.info('🌱 Starting database intial seed...');

    // Find existing roles
    const role_titles = (await prisma.role.findMany({ select: { title: true } })).map((r: any) => r.title);
    // Specify roles that should exist
    const role_data = [
        [ROLES.Actor, 'This role allows a user to create routines and save their progress.'],
    ]
    // Add missing roles
    for (const role of role_data) {
        if (!role_titles.includes(role[0])) {
            console.info(`🏗 Creating ${role[0]} role`);
            await prisma.role.create({ data: { title: role[0], description: role[1] } })
        }
    }

    console.info(`✅ Database seeding complete.`);
}

