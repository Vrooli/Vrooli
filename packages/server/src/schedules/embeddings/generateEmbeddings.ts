import { Configuration, OpenAIApi } from "openai";
import { logger } from "../../events";

const configuration = new Configuration({
    apiKey: process.env.OPENAI_API_KEY,
});
const openai = new OpenAIApi(configuration);

/**
 * Creates text embeddings for all searchable objects, which either:
 * - Don't have embeddings yet
 * - Have been updated since their last embedding was created
 */
export const generateEmbeddings = async () => {
    // Test openai by generating a completion
    const completion = await openai.createCompletion({
        model: "text-davinci-003",
        prompt: "Hello world",
    });
    logger.info("Got OpenAI completion:" + completion.data.choices[0].text);
};
