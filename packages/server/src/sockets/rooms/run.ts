import { JOIN_RUN_ROOM_ERRORS, LEAVE_RUN_ROOM_ERRORS } from "@local/shared";
import { Socket } from "socket.io";
import { AuthTokensService } from "../../auth/auth.js";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { RunProjectModelInfo, RunRoutineModelInfo } from "../../models/base/types.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { onSocketEvent } from "../events.js";

/** Socket room for run events */
export function runSocketRoomHandlers(socket: Socket) {
    onSocketEvent(socket, "joinRunRoom", async ({ runId, runType }, callback) => {
        try {
            if (AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: JOIN_RUN_ROOM_ERRORS.SessionExpired });
            }
            const rateLimitError = await RequestService.get().rateLimitSocket({ maxUser: 1000, socket });
            if (rateLimitError) {
                callback({ success: false, error: rateLimitError });
                return;
            }
            // Check if user is authenticated
            const userData = RequestService.assertRequestFrom(socket, { isUser: true });
            // Find run only if permitted
            const { canDelete: canRun } = await getSingleTypePermissions<(RunRoutineModelInfo | RunProjectModelInfo)["ApiPermission"]>(runType, [runId], userData);
            if (!Array.isArray(canRun) || !canRun.every(Boolean)) {
                const message = JOIN_RUN_ROOM_ERRORS.RunNotFoundOrUnauthorized;
                logger.error(message, { trace: "0623" });
                callback({ success: false, error: message });
                return;
            }
            // Otherwise, join the room
            socket.join(runId);
            callback({ success: true });
        } catch (error) {
            const message = JOIN_RUN_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0491", error, runId });
            callback({ success: false, error: message });
        }
    });

    onSocketEvent(socket, "leaveRunRoom", async ({ runId }, callback) => {
        try {
            socket.leave(runId);
            callback({ success: true });
        } catch (error) {
            const message = LEAVE_RUN_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0582", error, runId });
            callback({ success: false, error: message });
        }
    });
}
