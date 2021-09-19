import { ACCOUNT_STATUS } from '@local/shared';
import { TABLES } from '../tables';
import bcrypt from 'bcrypt';
import { HASHING_ROUNDS } from '../../consts';
import { db } from '../db';

// Create a user, with business, emails, phones, and roles
async function createUser({ userData, businessData, emailsData, phonesData, roleIds }) {
    let business = await db(TABLES.Business).select('id').where({ name: businessData.name }).first();
    if (!business) {
        console.info(`🏢 Creating business for ${userData.firstName}`);
        business = (await db(TABLES.Business).insert([businessData]).returning('id'))[0];
    }
    let customer = await db(TABLES.Customer).select('id').where({ firstName: userData.firstName, lastName: userData.lastName }).first();
    if (!customer) {
        console.info(`👩🏼‍💻 Creating account for ${userData.firstName}`);
        // Insert account
        const customerId = (await db(TABLES.Customer).insert([{ ...userData, businessId: business.id }]).returning('id'))[0];
        // Insert emails
        for (const email of emailsData) {
            await db(TABLES.Email).insert([{ ...email, customerId }]);
        }
        // Insert phones
        for (const phone of phonesData) {
            await db(TABLES.Phone).insert([{ ...phone, customerId }]);
        }
        // Insert roles
        for (const roleId of roleIds) {
            await db(TABLES.CustomerRoles).insert([{ roleId, customerId }]);
        }
    }
}

export async function seed() {
    console.info('🎭 Creating mock data...');

    // Find existing roles
    const roles = (await db(TABLES.Role).select('id', 'title'));
    const customerRoleId = roles.filter(r => r.title === 'Customer')[0].id;
    const ownerRoleId = roles.filter(r => r.title === 'Owner')[0].id;
    // const adminRoleId = roles.filter(r => r.title === 'Admin')[0].id;

    // Create user with owner role
    await createUser({
        businessData: { name: 'SpaceX' },
        userData: {
            firstName: 'Elon',
            lastName: 'Tusk🦏',
            password: bcrypt.hashSync('Elon', HASHING_ROUNDS),
            emailVerified: true,
            status: ACCOUNT_STATUS.Unlocked,
        },
        emailsData: [
            { emailAddress: 'notarealemail@afakesite.com', receivesDeliveryUpdates: false },
            { emailAddress: 'backupemailaddress@afakesite.com', receivesDeliveryUpdates: false }
        ],
        phonesData: [
            { number: '15558675309', receivesDeliveryUpdates: false },
            { number: '5555555555', receivesDeliveryUpdates: false }
        ],
        roleIds: [ownerRoleId]
    });

    // Create a few customers
    await createUser({
        businessData: { name: 'Rocket supplier A' },
        userData: {
            firstName: 'John',
            lastName: 'Cena',
            password: bcrypt.hashSync('John', HASHING_ROUNDS),
            emailVerified: true,
            status: ACCOUNT_STATUS.Unlocked,
        },
        emailsData: [
            { emailAddress: 'itsjohncena@afakesite.com', receivesDeliveryUpdates: false }
        ],
        phonesData: [],
        roleIds: [customerRoleId]
    });
    await createUser({
        businessData: { name: '🤘🏻A Steel Company' },
        userData: {
            firstName: 'Spongebob',
            lastName: 'Customerpants',
            password: bcrypt.hashSync('Spongebob', HASHING_ROUNDS),
            emailVerified: true,
            status: ACCOUNT_STATUS.Unlocked,
        },
        emailsData: [
            { emailAddress: 'spongebobmeboy@afakesite.com', receivesDeliveryUpdates: false }
        ],
        phonesData: [
            { number: '15553214321', receivesDeliveryUpdates: false },
            { number: '8762342222', receivesDeliveryUpdates: false }
        ],
        roleIds: [customerRoleId]
    });

    console.info(`✅ Database mock complete.`);
}