import bcrypt from 'bcrypt';
import pkg from '@prisma/client';
const { PrismaClient, AccountStatus } = pkg;
const prisma = new PrismaClient();
const HASHING_ROUNDS = 8;

// Create a user, with emails, and roles
async function createUser({ userData, emailsData, roleIds }) {
    let customer = await prisma.customer.findFirst({ username: userData.username });
    if (!customer) {
        console.info(`👩🏼‍💻 Creating account for ${userData.username}`);
        // Insert account
        const customer = await prisma.customer.create({ data: { ...userData } });
        // Insert emails
        for (const email of emailsData) {
            await prisma.email.create({ data: { ...email, customerId: customer.id }})
        }
        // Insert roles
        for (const roleId of roleIds) {
            await prisma.phone.create({ data: { ...phone, customerId: customer.id }})
        }
    }
}

async function main() {
    console.info('🎭 Creating mock data...');

    // Find existing roles
    const roles = await prisma.role.findMany({ select: { id: true, title: true } });
    const customerRoleId = roles.filter(r => r.title === 'Customer')[0].id;
    const ownerRoleId = roles.filter(r => r.title === 'Owner')[0].id;
    // const adminRoleId = roles.filter(r => r.title === 'Admin')[0].id;

    // Create user with owner role
    await createUser({
        userData: {
            username: 'Elon Tusk🦏',
            password: bcrypt.hashSync('Elon', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'notarealemail@afakesite.com', receivesDeliveryUpdates: false, verified: true },
            { emailAddress: 'backupemailaddress@afakesite.com', receivesDeliveryUpdates: false, verified: false }
        ],
        roleIds: [ownerRoleId]
    });

    // Create a few customers
    await createUser({
        userData: {
            username: 'JohnCena87',
            password: bcrypt.hashSync('John', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'itsjohncena@afakesite.com', receivesDeliveryUpdates: false, verified: true }
        ],
        roleIds: [customerRoleId]
    });
    await createUser({
        userData: {
            username: 'Spongebob Customerpants',
            password: bcrypt.hashSync('Spongebob', HASHING_ROUNDS),
            status: AccountStatus.UNLOCKED,
        },
        emailsData: [
            { emailAddress: 'spongebobmeboy@afakesite.com', receivesDeliveryUpdates: false, verified: true }
        ],
        roleIds: [customerRoleId]
    });

    console.info(`✅ Database mock complete.`);
}

main().catch((error) => {
    console.error(error);
    process.exit(1);
}).finally(async () => {
    await prisma.$disconnect();
})