/**
 * Adds mock data to the database. (i.e. should not be used in production). 
 * This should only be called once, or you may run into problems
 */

import { PrismaType } from "../../types";
import { AccountStatus, ROLES } from "@local/shared";
import { OrganizationModel, ProjectModel, UserModel } from "../../models";

export async function mock(prisma: PrismaType) {
    console.info("🎭 Creating mock data...");

    // Find existing roles
    const roles = await prisma.role.findMany({ select: { id: true, title: true } });
    const actorRoleId = roles.filter((r: any) => r.title === ROLES.Actor)[0].id;

    //==============================================================
    /* #region Create users */
    //==============================================================
    // Make sure to test the following:
    // - Deleted users do not show up in queries
    // - Bios and lack of bios are displayed correctly
    // - Organizations, projects, standards, and routines are displayed correctly
    //==============================================================
    const userModel = UserModel(prisma);
    const user1 = await userModel.create({
        username: "Elon Tusk🦏",
        password: userModel.hashPassword("Elon"),
        status: AccountStatus.Unlocked,
        bio: "I am the greatest",
        emails: {
            create: [
                { emailAddress: "elontusk@afakesite.com", verified: true },
                { emailAddress: "memelord420@afakesite.com", verified: false }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    })
    const user2 = await userModel.create({
        username: "HAL 9000🤖",
        password: userModel.hashPassword("HAL"),
        status: AccountStatus.Deleted,
        bio: "Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et dolore magnam aliquam quaerat voluptatem. Ut enim ad minima veniam, quis nostrum exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex ea commodi consequatur? Quis autem vel eum iure reprehenderit qui in ea voluptate velit esse quam nihil molestiae consequatur, vel illum qui dolorem eum fugiat quo voluptas nulla pariatur?",
        emails: {
            create: [
                { emailAddress: "notarobot@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user3 = await userModel.create({
        username: "Spongebob Squarepants",
        password: userModel.hashPassword("Spongebob"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "spongebobmeboy@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user4 = await userModel.create({
        username: "Patrick Star",
        password: userModel.hashPassword("Patrick"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "rocklover@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user5 = await userModel.create({
        username: "Mr. Krabs",
        password: userModel.hashPassword("Mr"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "agagagaga@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user6 = await userModel.create({
        username: "Plankton (Sheldon)",
        password: userModel.hashPassword("Plankton"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "eatatchumbucket@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user7 = await userModel.create({
        username: "Daisy Buchanan",
        password: userModel.hashPassword("Daisy"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "itsmedaisy@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user8 = await userModel.create({
        username: "John Galt",
        password: userModel.hashPassword("John"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "whoami@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user9 = await userModel.create({
        username: "Fancisco d'Anconia",
        password: userModel.hashPassword("Francisco"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "ilovedagny@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user10 = await userModel.create({
        username: "James Taggart",
        password: userModel.hashPassword("James"),
        status: AccountStatus.Deleted,
        emails: {
            create: [
                { emailAddress: "bestboss@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user11 = await userModel.create({
        username: "Gregor Samsa",
        password: userModel.hashPassword("Gregor"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "sadbug@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user12 = await userModel.create({
        username: "Aang",
        password: userModel.hashPassword("Aang"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "ilovemomo@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    const user13 = await userModel.create({
        username: "Zuko",
        password: userModel.hashPassword("Zuko"),
        status: AccountStatus.Unlocked,
        emails: {
            create: [
                { emailAddress: "angstyfireman@afakesite.com", verified: true }
            ]
        },
        roles: {
           create: [{ role: { connect: { id: actorRoleId } } }]
        }
    });
    //==============================================================
    /* #endregion Create users*/
    //==============================================================

    //==============================================================
    /* #region Create organizations */
    //==============================================================
    // Make sure to test the following:
    // - Bios and lack of bios display correctly
    // - Members/lack of members display correctly
    //==============================================================
    const organizationModel = OrganizationModel(prisma);
    const organization1 = await organizationModel.create({
        name: "Not Tesla",
        bio: "Building a sustainable future for all😎🔋",
        members: {
            create: [
                { user: { connect: { id: user1?.id } } },
                { user: { connect: { id: user2?.id } } },
                { user: { connect: { id: user3?.id } } },
            ]
        }
    })
    const organization2 = await organizationModel.create({
        name: "Not Apple",
        members: {
            create: [
                { user: { connect: { id: user2?.id } } },
                { user: { connect: { id: user3?.id } } },
                { user: { connect: { id: user4?.id } } },
            ]
        }
    })
    const organization3 = await organizationModel.create({
        name: "Vrooli 2",
        bio: "Competitor to Vrooli, now on Vrooli!",
        members: {
            create: [
                { user: { connect: { id: user10?.id } } },
            ]
        }
    })
    const organization4 = await organizationModel.create({
        name: "Empty Organization",
    })
    const organization5 = await organizationModel.create({
        name: "Vrooli 2",
        bio: "Competitor to Vrooli, now on Vrooli!",
        members: {
            create: [
                { user: { connect: { id: user10?.id } } },
            ]
        }
    })
    //==============================================================
    /* #endregion Create organizations */
    //==============================================================

    //==============================================================
    /* #region Create projects */
    //==============================================================
    // Make sure to test the following:
    // - Descriptions and lack of descriptions display correctly
    // - Organizations and users display correctly
    const projectModel = ProjectModel(prisma);
    const project1 = await projectModel.create({
        name: "Vrooli",
        description: "This is getting meta!",
        organizations: { 
            create: [{ organization: { connect: { id: organization1?.id } } }]
        },
    })
    const project2 = await projectModel.create({
        name: "Vrooli 2",
        description: "This is getting meta!",
        organizations: { 
            create: [{ organization: { connect: { id: organization1?.id } } }]
        },
    })
    //==============================================================
    /* #endregion Create projects */
    //==============================================================

    console.info(`✅ Database mock complete.`);
}