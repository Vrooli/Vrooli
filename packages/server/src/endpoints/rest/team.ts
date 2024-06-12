import { TeamCreateInput, TeamUpdateInput } from "@local/shared";
import { bannerImageConfig, profileImageConfig } from "../../utils";
import { team_create, team_findMany, team_findOne, team_update } from "../generated";
import { TeamEndpoints } from "../logic/team";
import { UploadConfig, setupRoutes } from "./base";

const imagesConfig: UploadConfig<TeamCreateInput | TeamUpdateInput> = {
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

export const TeamRest = setupRoutes({
    "/team/:id": {
        get: [TeamEndpoints.Query.team, team_findOne],
        put: [TeamEndpoints.Mutation.teamUpdate, team_update, imagesConfig],
    },
    "/teams": {
        get: [TeamEndpoints.Query.teams, team_findMany],
    },
    "/team": {
        post: [TeamEndpoints.Mutation.teamCreate, team_create, imagesConfig],
    },
});
