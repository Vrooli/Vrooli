import { AUTH_PROVIDERS, profileEmailUpdateValidation, type BotCreateInput, type BotUpdateInput, type ExportCalendarResult, type FindByPublicIdInput, type ImportCalendarInput, type ProfileEmailUpdateInput, type ProfileUpdateInput, type ScheduleCreateInput, type ScheduleSearchInput, type Success, type User, type UserSearchInput, type UserSearchResult } from "@vrooli/shared";
import { createOneHelper } from "../../actions/creates.js";
import { cudHelper } from "../../actions/cuds.js";
import { readManyHelper, readOneHelper } from "../../actions/reads.js";
import { updateOneHelper } from "../../actions/updates.js";
import { PasswordAuthService } from "../../auth/email.js";
import { RequestService } from "../../auth/request.js";
import { type PartialApiInfo } from "../../builders/types.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { type ApiEndpoint } from "../../types.js";
import { convertICalEventsToSchedules, createICalFromSchedules, parseICalFile } from "../../utils/calendar.js";
import { user_profile } from "../generated/user_profile.js";
import { createStandardCrudEndpoints, PermissionPresets, RateLimitPresets } from "../helpers/endpointFactory.js";

export type EndpointsUser = {
    findOne: ApiEndpoint<FindByPublicIdInput, User>;
    findMany: ApiEndpoint<UserSearchInput, UserSearchResult>;
    botCreateOne: ApiEndpoint<BotCreateInput, User>;
    botUpdateOne: ApiEndpoint<BotUpdateInput, User>;
    profileUpdate: ApiEndpoint<ProfileUpdateInput, User>;
    profileEmailUpdate: ApiEndpoint<ProfileEmailUpdateInput, User>;
    importCalendar: ApiEndpoint<ImportCalendarInput, Success>;
    // importUserData: ApiEndpoint<ImportUserDataInput, Success>;
    exportCalendar: ApiEndpoint<never, ExportCalendarResult>;
    exportData: ApiEndpoint<never, Success>;
}

const objectType = "User";
export const user: EndpointsUser = createStandardCrudEndpoints({
    objectType,
    endpoints: {
        findOne: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
            customImplementation: async ({ input, req, info }) => {
                // If the id is "me", use the current user's id
                if (input.publicId === "me") {
                    const { publicId: userPublicId } = RequestService.assertRequestFrom(req, { hasReadPrivatePermissions: true });
                    input.publicId = userPublicId;
                    info = user_profile;
                }
                return readOneHelper({ info, input, objectType, req });
            },
        },
        findMany: {
            rateLimit: RateLimitPresets.HIGH,
            permissions: PermissionPresets.READ_PUBLIC,
        },
    },
    customEndpoints: {
        botCreateOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 500, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const additionalData = { isBot: true };
            return createOneHelper({ additionalData, info, input, objectType, req });
        },
        botUpdateOne: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            const additionalData = { isBot: true };
            return updateOneHelper({ additionalData, info, input, objectType, req });
        },
        profileUpdate: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 250, req });
            // Force usage of your own id
            const { id } = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });
            return updateOneHelper({ info, input: { ...input, id }, objectType, req });
        },
        profileEmailUpdate: async (wrapped, { req }, info) => {
            const input = wrapped?.input;
            const userData = RequestService.assertRequestFrom(req, { hasWriteAuthPermissions: true });
            await RequestService.get().rateLimit({ maxUser: 100, req });
            // Validate input
            profileEmailUpdateValidation.update({ env: process.env.NODE_ENV as "development" | "production" }).validateSync(input, { abortEarly: false });
            // Find user
            const user = await DbProvider.get().user.findUnique({
                where: { id: BigInt(userData.id) },
                select: PasswordAuthService.selectUserForPasswordAuth(),
            });
            if (!user)
                throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
            // If user doesn't have a password, they must reset it first
            const passwordAuth = user.auths.find((auth) => auth.provider === AUTH_PROVIDERS.Password);
            const passwordHash = passwordAuth?.hashed_password ?? null;
            if (!passwordAuth || !passwordHash) {
                await PasswordAuthService.setupPasswordReset(user);
                throw new CustomError("0125", "MustResetPassword");
            }
            // If new password is provided
            if (input.newPassword) {
                // Validate current password
                const session = await PasswordAuthService.logIn(input.currentPassword, user, req);
                if (!session) {
                    throw new CustomError("0062", "InvalidCredentials"); // Purposefully vague with duplicate code for security
                }
                // Update password
                await DbProvider.get().user_auth.update({
                    where: {
                        id: passwordAuth.id,
                    },
                    data: {
                        hashed_password: PasswordAuthService.hashPassword(input.newPassword),
                    },
                });
            }
            // Create new emails
            if (input.emailsCreate) {
                await cudHelper({
                    info: { __typename: "Email", id: true, emailAddress: true },
                    inputData: input.emailsCreate.map(email => ({ action: "Create", input: email, objectType: "Email" })),
                    userData,
                });
            }
            // Delete emails
            if (input.emailsDelete) {
                // Make sure you have at least one authentication method remaining
                const emailsCount = await DbProvider.get().email.count({ where: { userId: BigInt(userData.id) } });
                const walletsCount = await DbProvider.get().wallet.count({ where: { userId: BigInt(userData.id) } });
                if (emailsCount - input.emailsDelete.length <= 0 && walletsCount <= 0)
                    throw new CustomError("0126", "MustLeaveVerificationMethod");
                // Delete emails
                await DbProvider.get().email.deleteMany({ where: { id: { in: input.emailsDelete.map(id => BigInt(id)) } } });
            }
            // Return updated user
            return readOneHelper({ info, input: { id: userData.id }, objectType, req });
        },
        importCalendar: async (wrapped, { req }) => {
            const input = wrapped?.input;
            await RequestService.get().rateLimit({ maxUser: 25, req });
            const userData = RequestService.assertRequestFrom(req, { hasWritePrivatePermissions: true });

            try {
                // Parse the iCal file
                const events = await parseICalFile(input.file);

                // Convert iCal events to schedule creation inputs
                const scheduleInputs = convertICalEventsToSchedules(events, userData.id);

                if (scheduleInputs.length === 0) {
                    return { __typename: "Success" as const, success: true, message: "No valid events found in the calendar file." };
                }

                // Create schedules in the database
                let createdCount = 0;
                for (const scheduleInput of scheduleInputs) {
                    if (!scheduleInput.startTime || !scheduleInput.endTime) continue;

                    try {
                        const scheduleCreateInput: ScheduleCreateInput = {
                            id: scheduleInput.id,
                            startTime: scheduleInput.startTime,
                            endTime: scheduleInput.endTime,
                            timezone: scheduleInput.timezone,
                            recurrencesCreate: scheduleInput.recurrences?.map(rec => ({
                                id: rec.id,
                                recurrenceType: rec.recurrenceType,
                                interval: rec.interval,
                                dayOfWeek: rec.dayOfWeek,
                                dayOfMonth: rec.dayOfMonth,
                                month: rec.month,
                                endDate: rec.endDate,
                                duration: rec.duration,
                            })),
                        };

                        await createOneHelper({
                            input: scheduleCreateInput,
                            objectType: "Schedule",
                            req,
                            info: { GQLResolveInfo: {} } as PartialApiInfo,
                        });

                        createdCount++;
                    } catch (error) {
                        console.error(`Failed to create run with schedule ${scheduleInput.id}:`, error);
                        // Continue with other schedules even if one fails
                    }
                }

                return {
                    __typename: "Success" as const,
                    success: true,
                    message: `Successfully imported ${createdCount} calendar events as scheduled runs.`,
                };
            } catch (error) {
                console.error("Error importing calendar:", error);
                throw new CustomError("0523", "InternalError", { error: (error as Error).message });
            }
        },
        exportCalendar: async (_, { req }) => {
            await RequestService.get().rateLimit({ maxUser: 25, req });
            const userData = RequestService.assertRequestFrom(req, { hasReadPrivatePermissions: true });
            const input: ScheduleSearchInput = {
                searchString: "",
                take: 1000,
                scheduleForUserId: userData.id,
            };

            try {
                // Fetch user's schedules with related meetings and runs
                const schedules = await readManyHelper({
                    input,
                    objectType: "Schedule",
                    req,
                    info: {
                        GQLResolveInfo: {},
                        select: {
                            edges: {
                                node: {
                                    id: true,
                                    startTime: true,
                                    endTime: true,
                                    timezone: true,
                                    recurrences: {
                                        id: true,
                                        recurrenceType: true,
                                        interval: true,
                                        dayOfWeek: true,
                                        dayOfMonth: true,
                                        month: true,
                                        endDate: true,
                                        duration: true,
                                    },
                                    meetings: {
                                        id: true,
                                        translations: {
                                            id: true,
                                            language: true,
                                            name: true,
                                            description: true,
                                        },
                                    },
                                    runs: {
                                        id: true,
                                        name: true,
                                        resourceVersion: {
                                            id: true,
                                            translations: {
                                                id: true,
                                                language: true,
                                                name: true,
                                                description: true,
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    } as PartialApiInfo,
                });

                // Convert schedules to the format expected by createICalFromSchedules
                const scheduleData = schedules.edges.map((edge: any) => ({
                    schedule: edge.node,
                    meeting: edge.node.meetings && edge.node.meetings.length > 0 ? edge.node.meetings[0] : undefined,
                    run: edge.node.runs && edge.node.runs.length > 0 ? edge.node.runs[0] : undefined,
                }));

                // Generate iCal content
                const icalContent = createICalFromSchedules(scheduleData);

                return {
                    __typename: "ExportCalendarResult" as const,
                    calendar: icalContent,
                };
            } catch (error) {
                console.error("Error exporting calendar:", error);
                throw new CustomError("0524", "InternalError", { error: (error as Error).message });
            }
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, teams, etc.).
         * @returns JSON of all user data
         */
        exportData: async (_d) => {
            throw new CustomError("0999", "NotImplemented");
            // const userData = RequestService.assertRequestFrom(req, { isUser: true });
            // await RequestService.get().rateLimit({ maxUser: 5, req });
            // return await ProfileModel.port().exportData(userData.id);
        },
    },
});
