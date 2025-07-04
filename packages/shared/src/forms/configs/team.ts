/**
 * Team Form Configuration
 * 
 * Unified configuration for Team forms that can be used by
 * UI components, validation, and integration testing.
 */
import { endpointsTeam } from "../../api/pairs.js";
import type {
    Team,
    TeamCreateInput,
    TeamUpdateInput,
    Session,
} from "../../api/types.js";
import { DUMMY_ID } from "../../id/snowflake.js";
import type { TeamShape } from "../../shape/models/models.js";
import { shapeTeam } from "../../shape/models/models.js";
import { orDefault } from "../../utils/orDefault.js";
import { teamTranslationValidation, teamValidation } from "../../validation/models/team.js";
import type { FormConfig } from "./types.js";

/**
 * Core Team form configuration
 */
export const teamFormConfig: FormConfig<
    TeamShape,
    TeamCreateInput,
    TeamUpdateInput,
    Team
> = {
    objectType: "Team",

    validation: {
        schema: teamValidation,
        translationSchema: teamTranslationValidation,
    },

    transformations: {
        /** Convert form data to shape - for Team, form data is already in shape format */
        formToShape: (formData: TeamShape) => formData,

        /** Shape transformation object - use the shape model directly */
        shapeToInput: shapeTeam,

        /** Convert API response back to shape format */
        apiResultToShape: (apiResult: Team): TeamShape => ({
            __typename: "Team" as const,
            id: apiResult.id,
            handle: apiResult.handle || null,
            isOpenToNewMembers: apiResult.isOpenToNewMembers || false,
            isPrivate: apiResult.isPrivate || false,
            bannerImage: apiResult.bannerImage || null,
            profileImage: apiResult.profileImage || null,
            tags: apiResult.tags?.map(tag => ({
                __typename: "Tag" as const,
                id: tag.id,
                tag: tag.tag,
                translations: tag.translations,
            })) || [],
            translations: apiResult.translations?.map(t => ({
                __typename: "TeamTranslation" as const,
                id: t.id,
                language: t.language,
                name: t.name,
                bio: t.bio || "",
            })) || [],
        }),

        /** Generate initial values for the form */
        getInitialValues: (session?: Session, existing?: Partial<TeamShape>) => {
            const languages = session?.users?.[0]?.languages || ["en"];
            
            return {
                __typename: "Team" as const,
                id: existing?.id || DUMMY_ID,
                handle: existing?.handle || null,
                isOpenToNewMembers: existing?.isOpenToNewMembers || false,
                isPrivate: existing?.isPrivate || false,
                bannerImage: existing?.bannerImage || null,
                profileImage: existing?.profileImage || null,
                tags: existing?.tags || [],
                translations: orDefault(existing?.translations, [{
                    __typename: "TeamTranslation" as const,
                    id: DUMMY_ID,
                    language: languages[0],
                    name: "",
                    bio: "",
                }]),
            };
        },
    },

    endpoints: endpointsTeam,
};
