import { readFiles, saveFiles } from "../../utils";
// import ogs from "open-graph-scraper";
import { ReadAssetsInput, WriteAssetsInput } from "@local/shared";
import { CustomError } from "../../events";
import { rateLimit } from "../../middleware";
import { GQLEndpoint } from "../../types";

export type EndpointsRoot = {
    Query: {
        readAssets: GQLEndpoint<ReadAssetsInput, (string | null)[]>;
    },
    Mutation: {
        writeAssets: GQLEndpoint<WriteAssetsInput, boolean>;
    }
}

export const RootEndpoints: EndpointsRoot = {
    Query: {
        readAssets: async (_parent, { input }, { req }) => {
            await rateLimit({ maxUser: 1000, req });
            return await readFiles(input.files);
        },
    },
    Mutation: {
        writeAssets: async (_parent, { input }, { req }) => {
            await rateLimit({ maxUser: 500, req });
            throw new CustomError("0327", "NotImplemented", req.languages); // TODO add safety checks before allowing uploads
            const data = await saveFiles(input.files);
            // Any failed writes will return null
            return !data.some(d => d === null);
        },
    },
};
