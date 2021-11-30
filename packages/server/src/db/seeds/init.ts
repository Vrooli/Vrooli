import { PrismaType } from '../../types';

export async function init(prisma: PrismaType) {
    console.info('🌱 Starting database intial seed...');

    // Find existing roles
    const role_titles = (await prisma.role.findMany({ select: { title: true } })).map(r => r.title);
    // Specify roles that should exist
    const role_data = [
        ['Customer', 'This role allows a customer to create routines and save their progress.'],
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

