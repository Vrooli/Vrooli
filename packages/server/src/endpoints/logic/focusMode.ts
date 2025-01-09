import { ActiveFocusMode, FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeSearchResult, FocusModeUpdateInput, SetActiveFocusModeInput, VisibilityType, setActiveFocusModeValidation } from "@local/shared";
import { PrismaPromise } from "@prisma/client";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { updateSessionCurrentUser } from "../../auth/auth";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { ApiEndpoint } from "../../types";

export type EndpointsFocusMode = {
    findOne: ApiEndpoint<FindByIdInput, FocusMode>;
    findMany: ApiEndpoint<FocusModeSearchInput, FocusModeSearchResult>;
    createOne: ApiEndpoint<FocusModeCreateInput, FocusMode>;
    updateOne: ApiEndpoint<FocusModeUpdateInput, FocusMode>;
    setActive: ApiEndpoint<SetActiveFocusModeInput, ActiveFocusMode | null>;
}

const objectType = "FocusMode";
export const focusMode: EndpointsFocusMode = {
    findOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readOneHelper({ info, input, objectType, req });
    },
    findMany: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
    },
    createOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 100, req });
        return createOneHelper({ info, input, objectType, req });
    },
    updateOne: async ({ input }, { req }, info) => {
        await RequestService.get().rateLimit({ maxUser: 250, req });
        return updateOneHelper({ info, input, objectType, req });
    },
    setActive: async ({ input }, { req, res }) => {
        const userData = RequestService.assertRequestFrom(req, { isUser: true });
        await RequestService.get().rateLimit({ maxUser: 500, req });
        setActiveFocusModeValidation.validateSync(input, { abortEarly: false });
        const userId = userData.id;
        console.log("yeet in setActiveFocusMode", JSON.stringify(input));

        let activeFocusMode: ActiveFocusMode | null = null;

        // If input has an id, activate focus mode
        if (input.id) {
            // Before activating focus mode, ensure it exists and belongs to the user
            const focusMode = await prismaInstance.focus_mode.findFirst({
                where: { id: input.id, user: { id: userId } },
            });
            if (!focusMode) {
                throw new CustomError("0448", "NotFound");
            }
            const transactions: PrismaPromise<object>[] = [
                // Delete all active focus modes for the user
                prismaInstance.active_focus_mode.deleteMany({
                    where: { user: { id: userId } },
                }),
                // Create new active focus mode
                prismaInstance.active_focus_mode.create({
                    data: {
                        user: { connect: { id: userId } },
                        focusMode: { connect: { id: input.id } },
                        stopCondition: input.stopCondition ?? undefined,
                        stopTime: input.stopTime,
                    },
                    select: {
                        focusMode: {
                            select: {
                                id: true,
                                reminderList: { select: { id: true } },
                            },
                        },
                        stopCondition: true,
                        stopTime: true,
                    },
                }),
            ];
            const transactionResults = await prismaInstance.$transaction(transactions);
            // transactions[0]: active_focus_mode.deleteMany
            // transactions[1]: active_focus_mode.create
            const activeFocusModeData = transactionResults[1] as Record<string, any>;
            activeFocusMode = {
                __typename: "ActiveFocusMode" as const,
                focusMode: {
                    __typename: "ActiveFocusModeFocusMode" as const,
                    id: activeFocusModeData.focusMode.id,
                    reminderListId: activeFocusModeData.focusMode.reminderList?.id,
                },
                stopCondition: activeFocusModeData.stopCondition,
                stopTime: activeFocusModeData.stopTime,
            };
        }
        // Otherwise, deactivate focus mode
        else {
            await prismaInstance.active_focus_mode.deleteMany({
                where: { user: { id: userId } },
            });
        }

        // Set active focus mode in user's session token
        updateSessionCurrentUser(req, res, { activeFocusMode });

        return activeFocusMode;
    },
};
