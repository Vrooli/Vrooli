import { PrismaType, RecursivePartial } from "../types";
import { ProfileEmailUpdateInput, SessionUser, Success, UserDeleteInput } from "../endpoints/types";
import { EmailModel } from "./email";
import { Request } from "express";
import { CustomError } from "../events";
import { readOneHelper } from "../actions";
import { Formatter, GraphQLModelType } from "./types";
import { assertRequestFrom } from "../auth/request";
import { GraphQLInfo, PartialGraphQLInfo } from "../builders/types";
import { hashPassword, setupVerificationCode, validatePassword } from "../auth";
import { addSupplementalFields, modelToGraphQL, padSelect, selectHelper, toPartialGraphQLInfo } from "../builders";

export {}

// const tagSelect = {
//     id: true,
//     created_at: true,
//     tag: true,
//     stars: true,
//     translations: {
//         id: true,
//         language: true,
//         description: true,
//     }
// }

// type SupplementalFields = 'starredTags' | 'hiddenTags';
// const formatter = (): Formatter<Profile, SupplementalFields> => ({
//     relationshipMap: {
//         __typename: 'Profile',
//         comments: 'Comment',
//         roles: 'Role',
//         emails: 'Email',
//         wallets: 'Wallet',
//         resourceLists: 'ResourceList',
//         projects: 'Project',
//         projectsCreated: 'Project',
//         routines: 'Routine',
//         routinesCreated: 'Routine',
//         starredBy: 'User',
//         stars: 'Star',
//         starredTags: 'Tag',
//         hiddenTags: 'TagHidden',
//         sentReports: 'Report',
//         reports: 'Report',
//         votes: 'Vote',
//     },
//     joinMap: { hiddenTags: 'tag', roles: 'role', starredBy: 'user' },
//     supplemental: {
//         toGraphQL: ({ ids, objects, partial, prisma, userData }) => [
//             ['starredTags', async () => {
//                 if (!userData) return new Array(objects.length).fill([]);
//                 // Query starred tags
//                 let data = (await prisma.star.findMany({
//                     where: {
//                         AND: [
//                             { byId: userData.id },
//                             { NOT: { tagId: null } }
//                         ]
//                     },
//                     select: { tag: padSelect(tagSelect) }
//                 })).map((star: any) => star.tag)
//                 // Format to GraphQL
//                 data = data.map(r => modelToGraphQL(r, partial.starredTags as PartialGraphQLInfo));
//                 // Add supplemental fields
//                 data = await addSupplementalFields(prisma, userData, data, partial.starredTags as PartialGraphQLInfo);
//                 // Split by id
//                 const result = ids.map((id) => data.filter(r => r.routineId === id));
//                 return result;
//             }],
//             ['hiddenTags', async () => {
//                 if (!userData) return new Array(objects.length).fill([]);
//                 // // Query hidden tags
//                 // let data = (await prisma.user_tag_hidden.findMany({
//                 //     where: { userId: userData.id },
//                 //     select: {
//                 //         id: true,
//                 //         isBlur: true,
//                 //         tag: padSelect(tagSelect)
//                 //     }
//                 // }));
//                 // // Format to GraphQL
//                 // data = data.map(r => modelToGraphQL(r, partial.starredTags as PartialGraphQLInfo));
//                 // // Add supplemental fields
//                 // data = await addSupplementalFields(prisma, userData, data, partial.starredTags as PartialGraphQLInfo);
//                 // return ids.map((d: any) => ({
//                 //     id: d.id,
//                 //     isBlur: d.isBlur,
//                 //     tag: data.find(t => t.id === d.tag.id)
//                 // }))
//                 return new Array(objects.length).fill([]);
//             }],
//         ],
//     },
// })

// /**
//  * Custom component for importing/exporting data from Vrooli
//  * @param state 
//  * @returns 
//  */
// const porter = () => ({
//     /**
//      * Import JSON data to Vrooli. Useful if uploading data created offline, or if
//      * you're switching from a competitor to Vrooli. :)
//      * @param id 
//      */
//     async importData(prisma: PrismaType, data: string): Promise<Success> {
//         throw new CustomError('0323', 'NotImplemented', ['en']);
//     },
//     /**
//      * Export data to JSON. Useful if you want to use Vrooli data on your own,
//      * or switch to a competitor :(
//      * @param id 
//      */
//     async exportData(prisma: PrismaType, id: string): Promise<string> {
//         // Find user
//         const user = await prisma.user.findUnique({ where: { id }, select: { numExports: true, lastExport: true } });
//         if (!user) throw new CustomError('0065', 'NoUser', ['en']);
//         throw new CustomError('0324', 'NotImplemented', ['en'])
//     },
// })

// const querier = () => ({
//     async findProfile(
//         prisma: PrismaType,
//         req: Request,
//         info: GraphQLInfo,
//     ): Promise<RecursivePartial<Profile> | null> {
//         const userData = assertRequestFrom(req, { isUser: true });
//         const partial = toPartialGraphQLInfo(info, formatter().relationshipMap, req.languages, true);
//         // Query profile data and tags
//         const profileData = await readOneHelper<any>({
//             info,
//             input: { id: userData.id },
//             objectType: 'Profile',
//             prisma,
//             req,
//         })
//         // Format for GraphQL
//         let formatted = modelToGraphQL(profileData, partial) as RecursivePartial<Profile>;
//         // Return with supplementalfields added
//         const data = (await addSupplementalFields(prisma, userData, [formatted], partial))[0] as RecursivePartial<Profile>;
//         return data;
//     },
// })

// const mutater = () => ({
//     async updateEmails(
//         prisma: PrismaType,
//         userData: SessionUser,
//         input: ProfileEmailUpdateInput,
//         info: GraphQLInfo,
//     ): Promise<RecursivePartial<Profile>> {
//         throw new CustomError('0069', 'NotImplemented', userData.languages)
//         // // Check for correct password
//         // let user = await prisma.user.findUnique({ where: { id: userData.id } });
//         // if (!user)
//         //     throw new CustomError('0068', 'NoUser', userData.languages);
//         // if (!validatePassword(input.currentPassword, user, userData.languages))
//         //     throw new CustomError('0069', 'BadCredentials', userData.languages);
//         // // Convert input to partial select
//         // const partial = toPartialGraphQLInfo(info, formatter().relationshipMap, userData.languages, true);
//         // // Create user data
//         // let data: { [x: string]: any } = {
//         //     password: input.newPassword ? hashPassword(input.newPassword) : undefined,
//         //     emails: await EmailModel.mutate(prisma).relationshipBuilder!(userId, input, true),
//         // };
//         // // Send verification emails
//         // if (Array.isArray(input.emailsCreate)) {
//         //     for (const email of input.emailsCreate) {
//         //         await setupVerificationCode(email.emailAddress, prisma, userData.languages);
//         //     }
//         // }
//         // // Update user
//         // user = await prisma.user.update({
//         //     where: { id: userData.id },
//         //     data: data,
//         //     ...selectHelper(partial)
//         // });
//         // return modelToGraphQL(user, partial);
//     },
//     //TODO write this function, and make a similar one for organizations and projects
//     // /**
//     //  * Anonymizes or deletes objects belonging to a user (yourself)
//     //  * @param userId The user ID
//     //  * @param input The data to anonymize/delete
//     //  */
//     // async clearUserData(userId: string, input: UserClearDataInput): Promise<Success> {
//     //     // Comments
//     //     // Organizations
//     //     if (input.organization) {
//     //         if (input.organization.anonymizeIds) {

//     //         }
//     //         if (input.organization.deleteIds) {

//     //         }
//     //         if (input.organization.anonymizeAll) {

//     //         }
//     //         if (input.organization.deleteAll) {

//     //         }
//     //     }
//     //     // Projects
//     //     // Routines
//     //     // Runs (can only be deleted, not anonymized)
//     //     // Standards
//     //     // Tags (can only be anonymized, not deleted)
//     // },
//     async deleteProfile(
//         prisma: PrismaType,
//         userData: SessionUser,
//         input: UserDeleteInput
//     ): Promise<Success> {
//         return { success: false }
//         // // Check for correct password
//         // let user = await prisma.user.findUnique({ where: { id: userData.id } });
//         // if (!user)
//         //     throw new CustomError('0071', 'NoUser', userData.languages);
//         // if (!validatePassword(input.password, user, userData.languages))
//         //     throw new CustomError('0072', 'BadCredentials', userData.languages);
//         // // Delete user. User's created objects are deleted separately, with explicit confirmation 
//         // // given by the user. This is to minimize the chance of deleting objects which other users rely on. TODO
//         // await prisma.user.delete({
//         //     where: { id: userData.id }
//         // })
//         // return { success: true };
//     },
// })


// export const ProfileModel = ({
//     delegate: (prisma: PrismaType) => prisma.user,
//     format: formatter(),
//     mutate: mutater(),
//     port: porter(),
//     query: querier(),
// })