import { BotCreateInput, BotUpdateInput, HttpStatus, MB_10_BYTES, SERVER_VERSION, ServerError, SessionUser, TeamCreateInput, TeamUpdateInput, decodeValue, endpointsActions, endpointsApi, endpointsApiKey, endpointsApiVersion, endpointsAuth, endpointsAward, endpointsBookmark, endpointsBookmarkList, endpointsChat, endpointsChatInvite, endpointsChatMessage, endpointsChatParticipant, endpointsCode, endpointsCodeVersion, endpointsComment, endpointsEmail, endpointsFeed, endpointsFocusMode, endpointsIssue, endpointsLabel, endpointsMeeting, endpointsMeetingInvite, endpointsMember, endpointsMemberInvite, endpointsNote, endpointsNoteVersion, endpointsNotification, endpointsNotificationSubscription, endpointsPhone, endpointsPost, endpointsProject, endpointsProjectVersion, endpointsProjectVersionDirectory, endpointsPullRequest, endpointsPushDevice, endpointsQuestion, endpointsQuestionAnswer, endpointsQuiz, endpointsQuizAttempt, endpointsQuizQuestionResponse, endpointsReaction, endpointsReminder, endpointsReminderList, endpointsReport, endpointsReportResponse, endpointsReputationHistory, endpointsResourceList, endpointsRole, endpointsRoutine, endpointsRoutineVersion, endpointsRunProject, endpointsRunRoutine, endpointsRunRoutineInput, endpointsRunRoutineOutput, endpointsSchedule, endpointsStandard, endpointsStandardVersion, endpointsStatsApi, endpointsStatsCode, endpointsStatsProject, endpointsStatsQuiz, endpointsStatsRoutine, endpointsStatsSite, endpointsStatsStandard, endpointsStatsTeam, endpointsStatsUser, endpointsTag, endpointsTask, endpointsTeam, endpointsTransfer, endpointsTranslate, endpointsUnions, endpointsUser, endpointsView, endpointsWallet } from "@local/shared";
import Busboy from "busboy";
import { Express, NextFunction, Request, Response, Router } from "express";
import { SessionService } from "../auth/session";
import { ApiEndpointInfo } from "../builders/types";
import { CustomError } from "../events/error";
import { Context, context } from "../middleware";
import { processAndStoreFiles } from "../utils/fileStorage";
import { ResponseService } from "../utils/response";

const DEFAULT_MAX_FILES = 1;
const DEFAULT_MAX_FILE_SIZE = MB_10_BYTES;

export type EndpointDef = {
    endpoint: string; // e.g. "/bookmark/:id"
    method: "GET" | "POST" | "PUT" | "DELETE";
}
export type EndpointFunction = (
    parent: undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data: any,
    context: Context,
    info: ApiEndpointInfo,
) => Promise<unknown>;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type FileConfig<TInput = any> = {
    readonly allowedExtensions?: Array<"txt" | "pdf" | "doc" | "docx" | "xls" | "xlsx" | "ppt" | "pptx" | "heic" | "heif" | "png" | "jpg" | "jpeg" | "gif" | "webp" | "tiff" | "bmp" | string>;
    fieldName: string;
    /** Creates the base name for the file */
    fileNameBase?: (input: TInput, currentUser: SessionUser) => string;
    /** Max file size, in bytes */
    maxFileSize?: number;
    maxFiles?: number;
    /** For image files, what they should be resized to */
    imageSizes?: { width: number; height: number }[];
}
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type UploadConfig<TInput = any> = {
    acceptsFiles?: boolean;
    fields: FileConfig<TInput>[];
}
export type EndpointTuple = readonly [
    EndpointDef,
    EndpointFunction,
    ApiEndpointInfo | Record<string, unknown>,
    UploadConfig?
];
export type EndpointType = "get" | "post" | "put" | "delete";
export type EndpointGroup = { [key in EndpointType]?: EndpointTuple };

/**
 * Converts request search params to their correct types. 
 * For example, "true" becomes true, ""true"" becomes "true", and "1" becomes 1.
 * @param input Search parameters from request
 * @returns Object with key/value pairs, or empty object if no params
 */
function parseInput(input: Record<string, unknown>): Record<string, unknown> {
    const parsed: Record<string, unknown> = {};
    Object.entries(input).forEach(([key, value]) => {
        try {
            parsed[key] = decodeValue(JSON.parse(value as string));
        } catch (error) {
            parsed[key] = value;
        }
    });
    return parsed;
}

async function handleEndpoint(
    endpoint: EndpointFunction,
    selection: ApiEndpointInfo,
    input: unknown,
    req: Request,
    res: Response,
) {
    try {
        const data = await endpoint(
            undefined,
            (input ? { input } : undefined),
            context({ req, res }),
            selection,
        );
        ResponseService.sendSuccess(res, data);
    } catch (error: unknown) {
        let formattedError: ServerError;

        if (error instanceof CustomError) {
            formattedError = error.toServerError();
        } else if ((error as Error).name === "ValidationError") {
            formattedError = { trace: "0455", code: "ValidationFailed" };
        } else {
            formattedError = { trace: "0460", code: "InternalError" };
        }

        ResponseService.sendError(res, formattedError, HttpStatus.InternalServerError);
    }
}

/**
 * Middleware to conditionally setup middleware for file uploads.
 */
function maybeFileUploads(config?: UploadConfig) {
    // Return fily upload middleware if the endpoint accepts files.
    return (req: Request, res: Response, next: NextFunction) => {
        if (!config || !config.acceptsFiles) {
            return next();  // No file uploading needed, proceed to next middleware.
        }

        // Find the combined max file size and max number of files, since we don't know which 
        // fields the files are associated with yet.
        // Later on we'll do these checks again, but for each field.
        const totalMaxFileSize = config.fields
            .map(field => field.maxFileSize ?? DEFAULT_MAX_FILE_SIZE)
            .reduce((acc, value) => acc + value, 0);
        const totalMaxFiles = config.fields
            .map(field => field.maxFiles ?? DEFAULT_MAX_FILES)
            .reduce((acc, value) => acc + value, 0);

        const busboy = Busboy({
            headers: req.headers,
            limits: {
                fileSize: totalMaxFileSize,
                files: totalMaxFiles,
            },
        });

        req.files = [];
        req.body = {};

        busboy.on("file", (fieldname, file, info) => {
            const { filename, encoding, mimeType } = info;
            const buffers: Buffer[] = [];

            file.on("data", (data) => buffers.push(data));

            file.on("end", () => {
                if (!req.files) {
                    req.files = [];
                }
                const buffer = Buffer.concat(buffers);
                req.files.push({
                    fieldname,
                    originalname: filename,
                    encoding,
                    mimetype: mimeType,
                    buffer,
                    size: buffer.length,
                });
            });
        });

        busboy.on("field", (fieldname, value) => {
            req.body[fieldname] = value;
        });

        busboy.on("finish", () => next());

        req.pipe(busboy);
    };
}

/**
 * Creates router with endpoints from given object.
 * @param restEndpoints List of endpoint tuples, where each tuple contains 
 * the information needed to create the endpoint.
 */
function setupRoutes(endpointTuples: EndpointTuple[]): Router {
    const router = Router();
    // Loop through each endpoint
    endpointTuples.forEach(([endpointDef, endpointHandler, selection, uploadConfig]) => {
        const { endpoint, method } = endpointDef;
        const methodInLowerCase = method.toLowerCase() as keyof Router;

        // e.g. router.route("/bookmark/:id").get(...)
        const routeChain = router.route(endpoint);

        async function handleEndpointHelper(req: Request, res: Response) {
            // Find non-file data
            let input: Record<string, unknown> | unknown[] = {}; // default to an empty object
            // If it's a GET method, combine params and parsed query
            if (methodInLowerCase === "get") {
                input = Object.assign({}, req.params, parseInput(req.query));
            } else {
                // For non-GET methods, handle object and array bodies
                if (Array.isArray(req.body)) {
                    // Use slice to create a shallow copy of the array
                    input = req.body.slice();
                } else if (typeof req.body === "object" && req.body !== null) {
                    input = Object.assign({}, req.params, req.body);
                }
            }
            // Get files from request
            const files = req.files || [];
            let fileNames: { [x: string]: string[] } = {};
            // Upload files if necessary
            const canUploadFiles =
                uploadConfig?.acceptsFiles
                && Array.isArray(files)
                && files.length > 0
                && (methodInLowerCase === "post" || methodInLowerCase === "put");
            if (canUploadFiles) {
                try {
                    // Must be logged in to upload files
                    const userData = SessionService.getUser(req.session);
                    if (!userData) {
                        ResponseService.sendError(res, { trace: "0509", code: "NotLoggedIn" }, HttpStatus.Unauthorized);
                        return;
                    }
                    fileNames = await processAndStoreFiles(files, input, userData, uploadConfig);
                } catch (error) {
                    ResponseService.sendError(res, { trace: "0506", code: "InternalError" }, HttpStatus.InternalServerError);
                    return;
                }
            }
            // Add files to input
            for (const { fieldname, originalname } of files) {
                // If there are no files, upload must have failed or been rejected. So skip this file.
                const currFileNames = fileNames[originalname];
                if (!currFileNames || currFileNames.length === 0) continue;
                // We're going to store image files differently than other files. 
                // For normal files, there should only be one file in the array. So we'll just use that.
                if (currFileNames.length === 1) {
                    input[fieldname] = currFileNames[0] as string;
                }
                // For image files, there may be multiple sizes. We'll store them like this: 
                // { sizes: ["1024x1024", "512x512"], file: "https://bucket-name.s3.region.amazonaws.com/image-hash_*.jpg }
                // Note how there is an asterisk in the file name. You can replace this with any of the sizes to create a valid URL.
                else {
                    // Find the size string of each file
                    const sizes = currFileNames.map((file: string) => {
                        const url = new URL(file);
                        const size = url.pathname.split("/").pop()?.split("_")[1]?.split(".")[0];
                        return size;
                    }).filter((size: string | undefined) => size !== undefined) as string[];
                    // Find the file name without the size string
                    const file = (currFileNames[0] as string).replace(`_${sizes[0]}`, "_*");
                    input[fieldname] = { sizes, file };
                }
            }
            handleEndpoint(endpointHandler, selection as ApiEndpointInfo, input, req, res);
        }

        // Add method to route
        routeChain[methodInLowerCase](
            maybeFileUploads(uploadConfig),
            handleEndpointHelper,
        );
    });
    return router;
}

/**
 * Creates a router with all the API endpoints.
 */
export async function initRestApi(app: Express) {
    const Select = await import("./generated");
    const Logic = await import("./logic");
    const { bannerImageConfig, profileImageConfig } = await import("../utils/fileStorage");

    const botImagesConfig: UploadConfig<BotCreateInput | BotUpdateInput> = {
        acceptsFiles: true,
        fields: [{
            ...profileImageConfig,
            fieldName: "profileImage",
            fileNameBase: (input) => `${input.id}-profile`,
        }, {
            ...bannerImageConfig,
            fieldName: "bannerImage",
            fileNameBase: (input) => `${input.id}-banner`,
        }],
    };

    const userImagesConfig: UploadConfig<undefined> = {
        acceptsFiles: true,
        fields: [{
            ...profileImageConfig,
            fieldName: "profileImage",
            fileNameBase: (_, currentUser) => `${currentUser.id}-profile`,
        }, {
            ...bannerImageConfig,
            fieldName: "bannerImage",
            fileNameBase: (_, currentUser) => `${currentUser.id}-banner`,
        }],
    };

    const icalConfig: UploadConfig<undefined> = {
        acceptsFiles: true,
        fields: [{
            allowedExtensions: ["ics"],
            fieldName: "file",
            fileNameBase: (_, currentUser) => `${currentUser.id}-import`,
            maxFileSize: 1024 * 1024 * 2, // 2MB
        }],
    };

    const teamImagesConfig: UploadConfig<TeamCreateInput | TeamUpdateInput> = {
        acceptsFiles: true,
        fields: [{
            ...profileImageConfig,
            fieldName: "profileImage",
            fileNameBase: (input) => `${input.id}-profile`,
        }, {
            ...bannerImageConfig,
            fieldName: "bannerImage",
            fileNameBase: (input) => `${input.id}-banner`,
        }],
    };

    const routes = setupRoutes([
        // Actions
        [endpointsActions.copy, Logic.actions.copy, Select.copy_copy],
        [endpointsActions.deleteOne, Logic.actions.deleteOne, Select.deleteOneOrMany_deleteOne],
        [endpointsActions.deleteMany, Logic.actions.deleteMany, Select.deleteOneOrMany_deleteMany],
        [endpointsActions.deleteAll, Logic.actions.deleteAll, Select.deleteOneOrMany_deleteAll],
        [endpointsActions.deleteAccount, Logic.actions.deleteAccount, Select.deleteAccount],
        // API
        [endpointsApi.findOne, Logic.api.findOne, Select.api_findOne],
        [endpointsApi.findMany, Logic.api.findMany, Select.api_findMany],
        [endpointsApi.createOne, Logic.api.createOne, Select.api_create],
        [endpointsApi.updateOne, Logic.api.updateOne, Select.api_update],
        // API key
        [endpointsApiKey.createOne, Logic.apiKey.createOne, Select.apiKey_create],
        [endpointsApiKey.updateOne, Logic.apiKey.updateOne, Select.apiKey_update],
        [endpointsApiKey.deleteOne, Logic.apiKey.deleteOne, Select.apiKey_deleteOne],
        [endpointsApiKey.validateOne, Logic.apiKey.validate, Select.apiKey_validate],
        // API version
        [endpointsApiVersion.findOne, Logic.apiVersion.findOne, Select.apiVersion_findOne],
        [endpointsApiVersion.findMany, Logic.apiVersion.findMany, Select.apiVersion_findMany],
        [endpointsApiVersion.createOne, Logic.apiVersion.createOne, Select.apiVersion_create],
        [endpointsApiVersion.updateOne, Logic.apiVersion.updateOne, Select.apiVersion_update],
        // Auth
        [endpointsAuth.emailLogin, Logic.auth.emailLogIn, Select.auth_emailLogIn],
        [endpointsAuth.emailSignup, Logic.auth.emailSignUp, Select.auth_emailSignUp],
        [endpointsAuth.emailRequestPasswordChange, Logic.auth.emailRequestPasswordChange, Select.auth_emailRequestPasswordChange],
        [endpointsAuth.emailResetPassword, Logic.auth.emailResetPassword, Select.auth_emailResetPassword],
        [endpointsAuth.emailLogin, Logic.auth.guestLogIn, Select.auth_guestLogIn],
        [endpointsAuth.logout, Logic.auth.logOut, Select.auth_logOut],
        [endpointsAuth.logoutAll, Logic.auth.logOutAll, Select.auth_logOut],
        [endpointsAuth.validateSession, Logic.auth.validateSession, Select.auth_validateSession],
        [endpointsAuth.switchCurrentAccount, Logic.auth.switchCurrentAccount, Select.auth_switchCurrentAccount],
        [endpointsAuth.walletInit, Logic.auth.walletInit, Select.auth_walletInit],
        [endpointsAuth.walletComplete, Logic.auth.walletComplete, Select.auth_walletComplete],
        // Award
        [endpointsAward.findMany, Logic.award.findMany, Select.award_findMany],
        // Bookmark
        [endpointsBookmark.findOne, Logic.bookmark.findOne, Select.bookmark_findOne],
        [endpointsBookmark.findMany, Logic.bookmark.findMany, Select.bookmark_findMany],
        [endpointsBookmark.createOne, Logic.bookmark.createOne, Select.bookmark_create],
        [endpointsBookmark.updateOne, Logic.bookmark.updateOne, Select.bookmark_update],
        // Bookmark list
        [endpointsBookmarkList.findOne, Logic.bookmarkList.findOne, Select.bookmarkList_findOne],
        [endpointsBookmarkList.findMany, Logic.bookmarkList.findMany, Select.bookmarkList_findMany],
        [endpointsBookmarkList.createOne, Logic.bookmarkList.createOne, Select.bookmarkList_create],
        [endpointsBookmarkList.updateOne, Logic.bookmarkList.updateOne, Select.bookmarkList_update],
        // Chat
        [endpointsChat.findOne, Logic.chat.findOne, Select.chat_findOne],
        [endpointsChat.findMany, Logic.chat.findMany, Select.chat_findMany],
        [endpointsChat.createOne, Logic.chat.createOne, Select.chat_create],
        [endpointsChat.updateOne, Logic.chat.updateOne, Select.chat_update],
        // Chat invite
        [endpointsChatInvite.findOne, Logic.chatInvite.findOne, Select.chatInvite_findOne],
        [endpointsChatInvite.findMany, Logic.chatInvite.findMany, Select.chatInvite_findMany],
        [endpointsChatInvite.createOne, Logic.chatInvite.createOne, Select.chatInvite_createOne],
        [endpointsChatInvite.createMany, Logic.chatInvite.createMany, Select.chatInvite_createMany],
        [endpointsChatInvite.updateOne, Logic.chatInvite.updateOne, Select.chatInvite_updateOne],
        [endpointsChatInvite.updateMany, Logic.chatInvite.updateMany, Select.chatInvite_updateMany],
        [endpointsChatInvite.acceptOne, Logic.chatInvite.acceptOne, Select.chatInvite_accept],
        [endpointsChatInvite.declineOne, Logic.chatInvite.declineOne, Select.chatInvite_decline],
        // Chat message
        [endpointsChatMessage.findOne, Logic.chatMessage.findOne, Select.chatMessage_findOne],
        [endpointsChatMessage.findMany, Logic.chatMessage.findMany, Select.chatMessage_findMany],
        [endpointsChatMessage.findTree, Logic.chatMessage.findTree, Select.chatMessage_findTree],
        [endpointsChatMessage.createOne, Logic.chatMessage.createOne, Select.chatMessage_create],
        [endpointsChatMessage.updateOne, Logic.chatMessage.updateOne, Select.chatMessage_update],
        [endpointsChatMessage.regenerateResponse, Logic.chatMessage.regenerateResponse, Select.chatMessage_regenerateResponse],
        // Chat participant
        [endpointsChatParticipant.findOne, Logic.chatParticipant.findOne, Select.chatParticipant_findOne],
        [endpointsChatParticipant.findMany, Logic.chatParticipant.findMany, Select.chatParticipant_findMany],
        [endpointsChatParticipant.updateOne, Logic.chatParticipant.updateOne, Select.chatParticipant_update],
        // Code
        [endpointsCode.findOne, Logic.code.findOne, Select.code_findOne],
        [endpointsCode.findMany, Logic.code.findMany, Select.code_findMany],
        [endpointsCode.createOne, Logic.code.createOne, Select.code_create],
        [endpointsCode.updateOne, Logic.code.updateOne, Select.code_update],
        // Code version
        [endpointsCodeVersion.findOne, Logic.codeVersion.findOne, Select.codeVersion_findOne],
        [endpointsCodeVersion.findMany, Logic.codeVersion.findMany, Select.codeVersion_findMany],
        [endpointsCodeVersion.createOne, Logic.codeVersion.createOne, Select.codeVersion_create],
        [endpointsCodeVersion.updateOne, Logic.codeVersion.updateOne, Select.codeVersion_update],
        // Comment
        [endpointsComment.findOne, Logic.comment.findOne, Select.comment_findOne],
        [endpointsComment.findMany, Logic.comment.findMany, Select.comment_findMany],
        [endpointsComment.createOne, Logic.comment.createOne, Select.comment_create],
        [endpointsComment.updateOne, Logic.comment.updateOne, Select.comment_update],
        // Email
        [endpointsEmail.createOne, Logic.email.createOne, Select.email_create],
        [endpointsEmail.verify, Logic.email.verify, Select.email_verify],
        // Feed
        [endpointsFeed.home, Logic.feed.home, Select.feed_home],
        [endpointsFeed.popular, Logic.feed.popular, Select.popular_findMany],
        // Focus mode
        [endpointsFocusMode.findOne, Logic.focusMode.findOne, Select.focusMode_findOne],
        [endpointsFocusMode.findMany, Logic.focusMode.findMany, Select.focusMode_findMany],
        [endpointsFocusMode.createOne, Logic.focusMode.createOne, Select.focusMode_create],
        [endpointsFocusMode.updateOne, Logic.focusMode.updateOne, Select.focusMode_update],
        [endpointsFocusMode.setActive, Logic.focusMode.setActive, Select.focusMode_setActive],
        // Issue
        [endpointsIssue.findOne, Logic.issue.findOne, Select.issue_findOne],
        [endpointsIssue.findMany, Logic.issue.findMany, Select.issue_findMany],
        [endpointsIssue.createOne, Logic.issue.createOne, Select.issue_create],
        [endpointsIssue.updateOne, Logic.issue.updateOne, Select.issue_update],
        [endpointsIssue.closeOne, Logic.issue.closeOne, Select.issue_close],
        // Label
        [endpointsLabel.findOne, Logic.label.findOne, Select.label_findOne],
        [endpointsLabel.findMany, Logic.label.findMany, Select.label_findMany],
        [endpointsLabel.createOne, Logic.label.createOne, Select.label_create],
        [endpointsLabel.updateOne, Logic.label.updateOne, Select.label_update],
        // Meeting
        [endpointsMeeting.findOne, Logic.meeting.findOne, Select.meeting_findOne],
        [endpointsMeeting.findMany, Logic.meeting.findMany, Select.meeting_findMany],
        [endpointsMeeting.createOne, Logic.meeting.createOne, Select.meeting_create],
        [endpointsMeeting.updateOne, Logic.meeting.updateOne, Select.meeting_update],
        // Meeting invite
        [endpointsMeetingInvite.findOne, Logic.meetingInvite.findOne, Select.meetingInvite_findOne],
        [endpointsMeetingInvite.findMany, Logic.meetingInvite.findMany, Select.meetingInvite_findMany],
        [endpointsMeetingInvite.createOne, Logic.meetingInvite.createOne, Select.meetingInvite_createOne],
        [endpointsMeetingInvite.createMany, Logic.meetingInvite.createMany, Select.meetingInvite_createMany],
        [endpointsMeetingInvite.updateOne, Logic.meetingInvite.updateOne, Select.meetingInvite_updateOne],
        [endpointsMeetingInvite.updateMany, Logic.meetingInvite.updateMany, Select.meetingInvite_updateMany],
        [endpointsMeetingInvite.acceptOne, Logic.meetingInvite.acceptOne, Select.meetingInvite_accept],
        [endpointsMeetingInvite.declineOne, Logic.meetingInvite.declineOne, Select.meetingInvite_decline],
        // Member
        [endpointsMember.findOne, Logic.member.findOne, Select.member_findOne],
        [endpointsMember.findMany, Logic.member.findMany, Select.member_findMany],
        [endpointsMember.updateOne, Logic.member.updateOne, Select.member_update],
        // Member invite
        [endpointsMemberInvite.findOne, Logic.memberInvite.findOne, Select.memberInvite_findOne],
        [endpointsMemberInvite.findMany, Logic.memberInvite.findMany, Select.memberInvite_findMany],
        [endpointsMemberInvite.createOne, Logic.memberInvite.createOne, Select.memberInvite_createOne],
        [endpointsMemberInvite.createMany, Logic.memberInvite.createMany, Select.memberInvite_createMany],
        [endpointsMemberInvite.updateOne, Logic.memberInvite.updateOne, Select.memberInvite_updateOne],
        [endpointsMemberInvite.updateMany, Logic.memberInvite.updateMany, Select.memberInvite_updateMany],
        [endpointsMemberInvite.acceptOne, Logic.memberInvite.acceptOne, Select.memberInvite_accept],
        [endpointsMemberInvite.declineOne, Logic.memberInvite.declineOne, Select.memberInvite_decline],
        // Note
        [endpointsNote.findOne, Logic.note.findOne, Select.note_findOne],
        [endpointsNote.findMany, Logic.note.findMany, Select.note_findMany],
        [endpointsNote.createOne, Logic.note.createOne, Select.note_create],
        [endpointsNote.updateOne, Logic.note.updateOne, Select.note_update],
        // Note version
        [endpointsNoteVersion.findOne, Logic.noteVersion.findOne, Select.noteVersion_findOne],
        [endpointsNoteVersion.findMany, Logic.noteVersion.findMany, Select.noteVersion_findMany],
        [endpointsNoteVersion.createOne, Logic.noteVersion.createOne, Select.noteVersion_create],
        [endpointsNoteVersion.updateOne, Logic.noteVersion.updateOne, Select.noteVersion_update],
        // Notification
        [endpointsNotification.findOne, Logic.notification.findOne, Select.notification_findOne],
        [endpointsNotification.findMany, Logic.notification.findMany, Select.notification_findMany],
        [endpointsNotification.markAsRead, Logic.notification.markAsRead, Select.notification_markAsRead],
        [endpointsNotification.markAllAsRead, Logic.notification.markAllAsRead, Select.notification_markAllAsRead],
        [endpointsNotification.getSettings, Logic.notification.getSettings, Select.notification_settings],
        [endpointsNotification.updateSettings, Logic.notification.updateSettings, Select.notification_settingsUpdate],
        // Notification subscription
        [endpointsNotificationSubscription.findOne, Logic.notificationSubscription.findOne, Select.notificationSubscription_findOne],
        [endpointsNotificationSubscription.findMany, Logic.notificationSubscription.findMany, Select.notificationSubscription_findMany],
        [endpointsNotificationSubscription.createOne, Logic.notificationSubscription.createOne, Select.notificationSubscription_create],
        [endpointsNotificationSubscription.updateOne, Logic.notificationSubscription.updateOne, Select.notificationSubscription_update],
        // Phone
        [endpointsPhone.createOne, Logic.phone.createOne, Select.phone_create],
        [endpointsPhone.verify, Logic.phone.verify, Select.phone_verify],
        [endpointsPhone.validate, Logic.phone.validate, Select.phone_validate],
        // Post
        [endpointsPost.findOne, Logic.post.findOne, Select.post_findOne],
        [endpointsPost.findMany, Logic.post.findMany, Select.post_findMany],
        [endpointsPost.createOne, Logic.post.createOne, Select.post_create],
        [endpointsPost.updateOne, Logic.post.updateOne, Select.post_update],
        // Project
        [endpointsProject.findOne, Logic.project.findOne, Select.project_findOne],
        [endpointsProject.findMany, Logic.project.findMany, Select.project_findMany],
        [endpointsProject.createOne, Logic.project.createOne, Select.project_create],
        [endpointsProject.updateOne, Logic.project.updateOne, Select.project_update],
        // Project version
        [endpointsProjectVersion.findOne, Logic.projectVersion.findOne, Select.projectVersion_findOne],
        [endpointsProjectVersion.findMany, Logic.projectVersion.findMany, Select.projectVersion_findMany],
        [endpointsProjectVersion.createOne, Logic.projectVersion.createOne, Select.projectVersion_create],
        [endpointsProjectVersion.updateOne, Logic.projectVersion.updateOne, Select.projectVersion_update],
        [endpointsProjectVersion.contents, Logic.projectVersion.contents, Select.projectVersion_findMany], //TODO selection not right
        // Project version directory
        [endpointsProjectVersionDirectory.findOne, Logic.projectVersionDirectory.findOne, Select.projectVersionDirectory_findOne],
        [endpointsProjectVersionDirectory.findMany, Logic.projectVersionDirectory.findMany, Select.projectVersionDirectory_findMany],
        [endpointsProjectVersionDirectory.createOne, Logic.projectVersionDirectory.createOne, Select.projectVersionDirectory_create],
        [endpointsProjectVersionDirectory.updateOne, Logic.projectVersionDirectory.updateOne, Select.projectVersionDirectory_update],
        // Pull request
        [endpointsPullRequest.findOne, Logic.pullRequest.findOne, Select.pullRequest_findOne],
        [endpointsPullRequest.findMany, Logic.pullRequest.findMany, Select.pullRequest_findMany],
        [endpointsPullRequest.createOne, Logic.pullRequest.createOne, Select.pullRequest_create],
        [endpointsPullRequest.updateOne, Logic.pullRequest.updateOne, Select.pullRequest_update],
        // Push device
        [endpointsPushDevice.findMany, Logic.pushDevice.findMany, Select.pushDevice_findMany],
        [endpointsPushDevice.createOne, Logic.pushDevice.createOne, Select.pushDevice_create],
        [endpointsPushDevice.updateOne, Logic.pushDevice.updateOne, Select.pushDevice_update],
        [endpointsPushDevice.testOne, Logic.pushDevice.testOne, Select.pushDevice_test],
        // Question
        [endpointsQuestion.findOne, Logic.question.findOne, Select.question_findOne],
        [endpointsQuestion.findMany, Logic.question.findMany, Select.question_findMany],
        [endpointsQuestion.createOne, Logic.question.createOne, Select.question_create],
        [endpointsQuestion.updateOne, Logic.question.updateOne, Select.question_update],
        // Question answer
        [endpointsQuestionAnswer.findOne, Logic.questionAnswer.findOne, Select.questionAnswer_findOne],
        [endpointsQuestionAnswer.findMany, Logic.questionAnswer.findMany, Select.questionAnswer_findMany],
        [endpointsQuestionAnswer.createOne, Logic.questionAnswer.createOne, Select.questionAnswer_create],
        [endpointsQuestionAnswer.updateOne, Logic.questionAnswer.updateOne, Select.questionAnswer_update],
        [endpointsQuestionAnswer.acceptOne, Logic.questionAnswer.acceptOne, Select.questionAnswer_accept],
        // Quiz
        [endpointsQuiz.findOne, Logic.quiz.findOne, Select.quiz_findOne],
        [endpointsQuiz.findMany, Logic.quiz.findMany, Select.quiz_findMany],
        [endpointsQuiz.createOne, Logic.quiz.createOne, Select.quiz_create],
        [endpointsQuiz.updateOne, Logic.quiz.updateOne, Select.quiz_update],
        // Quiz attempt
        [endpointsQuizAttempt.findOne, Logic.quizAttempt.findOne, Select.quizAttempt_findOne],
        [endpointsQuizAttempt.findMany, Logic.quizAttempt.findMany, Select.quizAttempt_findMany],
        [endpointsQuizAttempt.createOne, Logic.quizAttempt.createOne, Select.quizAttempt_create],
        [endpointsQuizAttempt.updateOne, Logic.quizAttempt.updateOne, Select.quizAttempt_update],
        // Quiz question response
        [endpointsQuizQuestionResponse.findOne, Logic.quizQuestionResponse.findOne, Select.quizQuestionResponse_findOne],
        [endpointsQuizQuestionResponse.findMany, Logic.quizQuestionResponse.findMany, Select.quizQuestionResponse_findMany],
        // React
        [endpointsReaction.findMany, Logic.reaction.findMany, Select.reaction_findMany],
        [endpointsReaction.createOne, Logic.reaction.createOne, Select.reaction_react],
        // Reminder
        [endpointsReminder.findOne, Logic.reminder.findOne, Select.reminder_findOne],
        [endpointsReminder.findMany, Logic.reminder.findMany, Select.reminder_findMany],
        [endpointsReminder.createOne, Logic.reminder.createOne, Select.reminder_create],
        [endpointsReminder.updateOne, Logic.reminder.updateOne, Select.reminder_update],
        // Reminder list
        [endpointsReminderList.createOne, Logic.reminderList.createOne, Select.reminderList_create],
        [endpointsReminderList.updateOne, Logic.reminderList.updateOne, Select.reminderList_update],
        // Report
        [endpointsReport.findOne, Logic.report.findOne, Select.report_findOne],
        [endpointsReport.findMany, Logic.report.findMany, Select.report_findMany],
        [endpointsReport.createOne, Logic.report.createOne, Select.report_create],
        [endpointsReport.updateOne, Logic.report.updateOne, Select.report_update],
        // Report response
        [endpointsReportResponse.findOne, Logic.reportResponse.findOne, Select.reportResponse_findOne],
        [endpointsReportResponse.findMany, Logic.reportResponse.findMany, Select.reportResponse_findMany],
        [endpointsReportResponse.createOne, Logic.reportResponse.createOne, Select.reportResponse_create],
        [endpointsReportResponse.updateOne, Logic.reportResponse.updateOne, Select.reportResponse_update],
        // Reputation history
        [endpointsReputationHistory.findOne, Logic.reputationHistory.findOne, Select.reputationHistory_findOne],
        [endpointsReputationHistory.findMany, Logic.reputationHistory.findMany, Select.reputationHistory_findMany],
        // Resource list
        [endpointsResourceList.findOne, Logic.resourceList.findOne, Select.resourceList_findOne],
        [endpointsResourceList.findMany, Logic.resourceList.findMany, Select.resourceList_findMany],
        [endpointsResourceList.createOne, Logic.resourceList.createOne, Select.resourceList_create],
        [endpointsResourceList.updateOne, Logic.resourceList.updateOne, Select.resourceList_update],
        // Role
        [endpointsRole.findOne, Logic.role.findOne, Select.role_findOne],
        [endpointsRole.findMany, Logic.role.findMany, Select.role_findMany],
        [endpointsRole.createOne, Logic.role.createOne, Select.role_create],
        [endpointsRole.updateOne, Logic.role.updateOne, Select.role_update],
        // Routine
        [endpointsRoutine.findOne, Logic.routine.findOne, Select.routine_findOne],
        [endpointsRoutine.findMany, Logic.routine.findMany, Select.routine_findMany],
        [endpointsRoutine.createOne, Logic.routine.createOne, Select.routine_create],
        [endpointsRoutine.updateOne, Logic.routine.updateOne, Select.routine_update],
        // Routine version
        [endpointsRoutineVersion.findOne, Logic.routineVersion.findOne, Select.routineVersion_findOne],
        [endpointsRoutineVersion.findMany, Logic.routineVersion.findMany, Select.routineVersion_findMany],
        [endpointsRoutineVersion.createOne, Logic.routineVersion.createOne, Select.routineVersion_create],
        [endpointsRoutineVersion.updateOne, Logic.routineVersion.updateOne, Select.routineVersion_update],
        // Run project
        [endpointsRunProject.findOne, Logic.runProject.findOne, Select.runProject_findOne],
        [endpointsRunProject.findMany, Logic.runProject.findMany, Select.runProject_findMany],
        [endpointsRunProject.createOne, Logic.runProject.createOne, Select.runProject_create],
        [endpointsRunProject.updateOne, Logic.runProject.updateOne, Select.runProject_update],
        // Run routine
        [endpointsRunRoutine.findOne, Logic.runRoutine.findOne, Select.runRoutine_findOne],
        [endpointsRunRoutine.findMany, Logic.runRoutine.findMany, Select.runRoutine_findMany],
        [endpointsRunRoutine.createOne, Logic.runRoutine.createOne, Select.runRoutine_create],
        [endpointsRunRoutine.updateOne, Logic.runRoutine.updateOne, Select.runRoutine_update],
        // Run routine input
        [endpointsRunRoutineInput.findMany, Logic.runRoutineInput.findMany, Select.runRoutineInput_findMany],
        // Run routine output
        [endpointsRunRoutineOutput.findMany, Logic.runRoutineOutput.findMany, Select.runRoutineOutput_findMany],
        // Schedule
        [endpointsSchedule.findOne, Logic.schedule.findOne, Select.schedule_findOne],
        [endpointsSchedule.findMany, Logic.schedule.findMany, Select.schedule_findMany],
        [endpointsSchedule.createOne, Logic.schedule.createOne, Select.schedule_create],
        [endpointsSchedule.updateOne, Logic.schedule.updateOne, Select.schedule_update],
        // Standard
        [endpointsStandard.findOne, Logic.standard.findOne, Select.standard_findOne],
        [endpointsStandard.findMany, Logic.standard.findMany, Select.standard_findMany],
        [endpointsStandard.createOne, Logic.standard.createOne, Select.standard_create],
        [endpointsStandard.updateOne, Logic.standard.updateOne, Select.standard_update],
        // Standard version
        [endpointsStandardVersion.findOne, Logic.standardVersion.findOne, Select.standardVersion_findOne],
        [endpointsStandardVersion.findMany, Logic.standardVersion.findMany, Select.standardVersion_findMany],
        [endpointsStandardVersion.createOne, Logic.standardVersion.createOne, Select.standardVersion_create],
        [endpointsStandardVersion.updateOne, Logic.standardVersion.updateOne, Select.standardVersion_update],
        // Stats API
        [endpointsStatsApi.findMany, Logic.statsApi.findMany, Select.statsApi_findMany],
        // Stats code
        [endpointsStatsCode.findMany, Logic.statsCode.findMany, Select.statsCode_findMany],
        // Stats project
        [endpointsStatsProject.findMany, Logic.statsProject.findMany, Select.statsProject_findMany],
        // Stats quiz
        [endpointsStatsQuiz.findMany, Logic.statsQuiz.findMany, Select.statsQuiz_findMany],
        // Stats routine
        [endpointsStatsRoutine.findMany, Logic.statsRoutine.findMany, Select.statsRoutine_findMany],
        // Stats site
        [endpointsStatsSite.findMany, Logic.statsSite.findMany, Select.statsSite_findMany],
        // Stats standard
        [endpointsStatsStandard.findMany, Logic.statsStandard.findMany, Select.statsStandard_findMany],
        // Stats team
        [endpointsStatsTeam.findMany, Logic.statsTeam.findMany, Select.statsTeam_findMany],
        // Stats user
        [endpointsStatsUser.findMany, Logic.statsUser.findMany, Select.statsUser_findMany],
        // Tag
        [endpointsTag.findOne, Logic.tag.findOne, Select.tag_findOne],
        [endpointsTag.findMany, Logic.tag.findMany, Select.tag_findMany],
        [endpointsTag.createOne, Logic.tag.createOne, Select.tag_create],
        [endpointsTag.updateOne, Logic.tag.updateOne, Select.tag_update],
        // Task
        [endpointsTask.startLlmTask, Logic.task.startLlmTask, Select.task_startLlmTask],
        [endpointsTask.startRunTask, Logic.task.startRunTask, Select.task_startRunTask],
        [endpointsTask.cancelTask, Logic.task.cancelTask, Select.task_cancelTask],
        [endpointsTask.checkStatuses, Logic.task.checkStatuses, Select.task_checkTaskStatuses],
        // Team
        [endpointsTeam.findOne, Logic.team.findOne, Select.team_findOne],
        [endpointsTeam.findMany, Logic.team.findMany, Select.team_findMany],
        [endpointsTeam.createOne, Logic.team.createOne, Select.team_create, teamImagesConfig],
        [endpointsTeam.updateOne, Logic.team.updateOne, Select.team_update, teamImagesConfig],
        // Transfer
        [endpointsTransfer.findOne, Logic.transfer.findOne, Select.transfer_findOne],
        [endpointsTransfer.findMany, Logic.transfer.findMany, Select.transfer_findMany],
        [endpointsTransfer.updateOne, Logic.transfer.updateOne, Select.transfer_update],
        [endpointsTransfer.requestSendOne, Logic.transfer.requestSendOne, Select.transfer_requestSend],
        [endpointsTransfer.requestReceiveOne, Logic.transfer.requestReceiveOne, Select.transfer_requestReceive],
        [endpointsTransfer.cancelOne, Logic.transfer.cancelOne, Select.transfer_cancel],
        [endpointsTransfer.acceptOne, Logic.transfer.acceptOne, Select.transfer_accept],
        [endpointsTransfer.denyOne, Logic.transfer.denyOne, Select.transfer_deny],
        // Translate
        [endpointsTranslate.translate, Logic.translate.translate, Select.translate_translate],
        // Unions
        [endpointsUnions.projectOrRoutines, Logic.unions.projectOrRoutines, Select.projectOrRoutine_findMany],
        [endpointsUnions.projectOrTeams, Logic.unions.projectOrTeams, Select.projectOrTeam_findMany],
        [endpointsUnions.runProjectOrRunRoutines, Logic.unions.runProjectOrRunRoutines, Select.runProjectOrRunRoutine_findMany],
        // User
        [endpointsUser.botUpdateOne, Logic.user.botUpdateOne, Select.user_botUpdate, botImagesConfig],
        [endpointsUser.botCreateOne, Logic.user.botCreateOne, Select.user_botCreate, botImagesConfig],
        [endpointsUser.profile, Logic.user.profile, Select.user_profile],
        [endpointsUser.profileUpdate, Logic.user.profileUpdate, Select.user_profileUpdate, userImagesConfig],
        [endpointsUser.findOne, Logic.user.findOne, Select.user_findOne],
        [endpointsUser.findMany, Logic.user.findMany, Select.user_findMany],
        [endpointsUser.profileEmailUpdate, Logic.user.profileEmailUpdate, Select.user_profileEmailUpdate],
        //[endpointsUser.importCalendar, Logic.user.importCalendar, Select.user_importCalendar, icalConfig],
        //[endpointsUser.importUserData, Logic.user.importUserData, Select.user_importUserData],
        //[endpointsUser.exportCalendar, Logic.user.exportCalendar, Select.user_exportCalendar],
        //[endpointsUser.exportUserData, Logic.user.exportUserData, Select.user_exportUserData],
        // Views
        [endpointsView.findMany, Logic.view.findMany, Select.view_findMany],
        // Wallet
        [endpointsWallet.updateOne, Logic.wallet.updateOne, Select.wallet_update],
    ]);
    app.use(`/api/${SERVER_VERSION}/rest`, routes);
}

