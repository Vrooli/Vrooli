import { ActiveFocusMode, FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeUpdateInput, SetActiveFocusModeInput, VisibilityType } from "@local/shared";
import { createOneHelper } from "../../actions/creates";
import { readManyHelper, readOneHelper } from "../../actions/reads";
import { updateOneHelper } from "../../actions/updates";
import { updateSessionCurrentUser } from "../../auth/auth";
import { RequestService } from "../../auth/request";
import { focusModeSelect } from "../../auth/session";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { CreateOneResult, FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from "../../types";

export type EndpointsFocusMode = {
    Query: {
        focusMode: GQLEndpoint<FindByIdInput, FindOneResult<FocusMode>>;
        focusModes: GQLEndpoint<FocusModeSearchInput, FindManyResult<FocusMode>>;
    },
    Mutation: {
        focusModeCreate: GQLEndpoint<FocusModeCreateInput, CreateOneResult<FocusMode>>;
        focusModeUpdate: GQLEndpoint<FocusModeUpdateInput, UpdateOneResult<FocusMode>>;
        setActiveFocusMode: GQLEndpoint<SetActiveFocusModeInput, ActiveFocusMode>;
    }
}

const objectType = "FocusMode";
export const FocusModeEndpoints: EndpointsFocusMode = {
    Query: {
        focusMode: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, req });
        },
        focusModes: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        focusModeCreate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 100, req });
            return createOneHelper({ info, input, objectType, req });
        },
        focusModeUpdate: async (_, { input }, { req }, info) => {
            await RequestService.get().rateLimit({ maxUser: 250, req });
            return updateOneHelper({ info, input, objectType, req });
        },
        setActiveFocusMode: async (_, { input }, { req, res }) => {
            // Unlink other objects, active focus mode is only stored in the user's session. 
            const userData = RequestService.assertRequestFrom(req, { isUser: true });
            await RequestService.get().rateLimit({ maxUser: 500, req });
            // Create time frame to limit focus mode's schedule data
            const now = new Date();
            const startDate = now;
            const endDate = new Date(now.setDate(now.getDate() + 7));
            // Find focus mode data & verify that it belongs to the user
            const focusMode = await prismaInstance.focus_mode.findFirst({
                where: {
                    id: input.id,
                    user: { id: userData.id },
                },
                select: focusModeSelect(startDate, endDate),
            });
            if (!focusMode) throw new CustomError("0448", "NotFound");
            // Set active focus mode in user's session token
            updateSessionCurrentUser(req, res, {
                activeFocusMode: {
                    id: focusMode.id,
                    reminderListId: focusMode.reminderList?.id,
                }
            });
            return {
                __typename: "ActiveFocusMode" as const,
                mode: focusMode as unknown as FocusMode,
                stopCondition: input.stopCondition,
                stopTime: input.stopTime,
            };
        },
    },
};
