import { JOIN_RUN_ROOM_ERRORS, LEAVE_RUN_ROOM_ERRORS } from "@vrooli/shared";
// AI_CHECK: TYPE_SAFETY=server-phase2-1 | LAST: 2025-07-03 - Added explicit return type annotation
import { type Socket } from "socket.io";
import { AuthTokensService } from "../../auth/auth.js";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { type RunModelInfo } from "../../models/base/types.js";
import { getSingleTypePermissions } from "../../validators/permissions.js";
import { SocketService } from "../io.js";

/** Socket room for run events */
export function runSocketRoomHandlers(socket: Socket): void {
    SocketService.get().onSocketEvent(socket, "joinRunRoom", async ({ runId }, callback) => {
        try {
            if (!("session" in socket) || !socket.session || AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: JOIN_RUN_ROOM_ERRORS.SessionExpired });
                return;
            }
            const rateLimitError = await RequestService.get().rateLimitSocket({ maxUser: 1000, socket });
            if (rateLimitError) {
                callback({ success: false, error: rateLimitError });
                return;
            }
            // Check if user is authenticated
            const userData = RequestService.assertRequestFrom(socket, { isUser: true });
            // Find run only if permitted
            const { canDelete: canRun } = await getSingleTypePermissions<RunModelInfo["ApiPermission"]>("Run", [runId], userData);
            if (!Array.isArray(canRun) || !canRun.every(Boolean)) {
                const message = JOIN_RUN_ROOM_ERRORS.RunNotFoundOrUnauthorized;
                logger.error(message, { trace: "0623" });
                callback({ success: false, error: message });
                return;
            }
            // Otherwise, join the room
            socket.join(runId);
            callback({ success: true });
        } catch (error: unknown) {
            const message = JOIN_RUN_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0491", error, runId });
            callback({ success: false, error: message });
        }
    });

    SocketService.get().onSocketEvent(socket, "leaveRunRoom", async ({ runId }, callback) => {
        try {
            socket.leave(runId);
            callback({ success: true });
        } catch (error: unknown) {
            const message = LEAVE_RUN_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0582", error, runId });
            callback({ success: false, error: message });
        }
    });
}
