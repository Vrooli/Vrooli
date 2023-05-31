import { ActiveFocusMode, FindByIdInput, FocusMode, FocusModeCreateInput, FocusModeSearchInput, FocusModeUpdateInput, SetActiveFocusModeInput, VisibilityType } from "@local/shared";
import { createHelper, readManyHelper, readOneHelper, updateHelper } from "../../actions";
import { assertRequestFrom, focusModeSelect, updateSessionCurrentUser } from "../../auth";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
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
        focusMode: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        focusModes: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req, visibility: VisibilityType.Own });
        },
    },
    Mutation: {
        focusModeCreate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 100, req });
            return createHelper({ info, input, objectType, prisma, req });
        },
        focusModeUpdate: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 250, req });
            return updateHelper({ info, input, objectType, prisma, req });
        },
        setActiveFocusMode: async (_, { input }, { prisma, req, res }) => {
            // Unlink other objects, active focus mode is only stored in the user's session. 
            const userData = assertRequestFrom(req, { isUser: true });
            // Create time frame to limit focus mode's schedule data
            const now = new Date();
            const startDate = now;
            const endDate = new Date(now.setDate(now.getDate() + 7));
            // Find focus mode data & verify that it belongs to the user
            const focusMode = await prisma.focus_mode.findFirst({
                where: {
                    id: input.id,
                    user: { id: userData.id },
                },
                select: focusModeSelect(startDate, endDate),
            });
            if (!focusMode) throw new CustomError("0448", "NotFound", userData.languages);
            const activeFocusMode = {
                __typename: "ActiveFocusMode" as const,
                mode: focusMode as any as FocusMode,
                stopCondition: input.stopCondition,
                stopTime: input.stopTime,
            };
            // Set active focus mode in user's session token
            updateSessionCurrentUser(req, res, { activeFocusMode });
            return activeFocusMode;
        },
    },
};
