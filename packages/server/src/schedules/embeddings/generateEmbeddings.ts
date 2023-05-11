import https from "https";
import { logger } from "../../events";

/**
 * Function to get embeddings from your new API
 * @param instruction The instruction for what to generate embeddings for
 * @param text The text being embedded
 * @returns A Promise that resolves with the embeddings
 * @throws An Error if the API request fails
 */
async function getEmbeddings(instruction: string, text: string): Promise<any> {
    return new Promise((resolve, reject) => {
        const data = JSON.stringify({ instruction, sentence: text });
        const options = {
            hostname: "embedtext.com",
            port: 443,
            path: "/embed",
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Content-Length": data.length,
            },
        };
        const apiRequest = https.request(options, apiRes => {
            let responseBody = "";
            apiRes.on("data", chunk => {
                responseBody += chunk;
            });
            apiRes.on("end", () => {
                const embeddings = JSON.parse(responseBody);
                resolve(embeddings);
            });
        });
        apiRequest.on("error", error => {
            console.error(`Error: ${error}`);
            reject(error);
        });
        apiRequest.write(data);
        apiRequest.end();
    });
}

/**
 * Calculates the cosine similarity between two vectors
 * @param vecA First vector
 * @param vecB Second vector
 * @returns Cosine similarity
 */
function cosineSimilarity(vecA: number[], vecB: number[]): number {
    if (vecA.length !== vecB.length) {
        throw new Error("Vectors must have the same dimension");
    }

    let dotProduct = 0.0;
    let normA = 0.0;
    let normB = 0.0;
    for (let i = 0; i < vecA.length; i++) {
        dotProduct += vecA[i] * vecB[i];
        normA += Math.pow(vecA[i], 2);
        normB += Math.pow(vecB[i], 2);
    }

    return dotProduct / (Math.sqrt(normA) * Math.sqrt(normB));
}

/**
 * K-Nearest Neighbors function
 * @param embeddings The array of embeddings
 * @param searchEmbedding The search embedding
 * @param K The number of nearest neighbors to return
 * @returns An array of indices of the K nearest neighbors in the embeddings array
 */
function knn(embeddings: number[][], searchEmbedding: number[], K: number): number[] {
    // Compute the similarity with each embedding
    const similarities = embeddings.map(embedding => cosineSimilarity(embedding, searchEmbedding));

    // Create an array of indices and sort it by similarity
    const indices = Array.from({ length: similarities.length }, (_, i) => i);
    indices.sort((a, b) => similarities[b] - similarities[a]);

    // Return the first K indices
    return indices.slice(0, K);
}


/**
 * Creates text embeddings for all searchable objects, which either:
 * - Don't have embeddings yet
 * - Have been updated since their last embedding was created
 */
export const generateEmbeddings = async () => {
    // Test 1: Plain strings
    const instruction1 = "Represent the text for classification";
    const testStrings1 = [
        "The quick brown fox jumps over the lazy dog",
        "The fast brown fox jumps over the lazy dog",
        "The quick brown fox jumps over the lazy dog",
        "This sentence is not similar to the others",
        "It was a dark and stormy night",
        "The pen is mightier than the sword",
        "A stitch in time saves nine",
        "Out of sight, out of mind",
        "Once upon a time in a land far, far away",
        "To be or not to be, that is the question",
        "A",
        "B",
        "C",
    ];
    const embeddings1 = await Promise.all(testStrings1.map(text => getEmbeddings(instruction1, text)));
    const embeddings1Array = embeddings1.map(embed => embed[0]);  // Convert to 2D array for knn
    const searchStrings1 = ["fox", "pen", "dog", "the", "a"];
    const searchEmbeddings1 = await Promise.all(searchStrings1.map(text => getEmbeddings(instruction1, text)));
    for (let i = 0; i < searchStrings1.length; i++) {
        const indices = knn(embeddings1Array, searchEmbeddings1[i][0], 5);  // Find the indices of the 5 nearest neighbors
        const nearestNeighbors = indices.map(index => testStrings1[index]);  // Retrieve the actual strings
        logger.info(`Nearest neighbors for "${searchStrings1[i]}": ${nearestNeighbors}`);
    }

    // Test 2: JSON strings
    const instruction2 = "Represent the title and description for classification";
    const testStrings2 = [
        JSON.stringify({
            title: "All about penguins",
            description: "Penguins are birds. They are black and white. When it is cold, they huddle together.",
        }),
        JSON.stringify({
            title: "The best dinosaurs",
            description: "Here are my top five dinosaurs: T-Rex, Triceratops, Stegosaurus, Velociraptor, and Brontosaurus. I like these dinosaurs because they are big and scary.",
        }),
        JSON.stringify({
            title: "How to bake a cake",
            description: "Hello, here is my backstory that you don't want to hear, but it's in front of the instructions for some reason",
        }),
        JSON.stringify({
            title: "The history of Rome",
            description: "The history of Rome is fascinating. From its mythical beginnings with Romulus and Remus, through the era of the Republic and the powerful Empire, to its fall. Rome has influenced the world in countless ways. Here are some key points to remember...",
        }),
        JSON.stringify({
            title: "Guide to Astrophotography",
            description: "Astrophotography is the art of capturing images of the night sky. It's a challenging but rewarding hobby that can take you from the comfort of your own backyard to the most remote corners of the planet. In this guide, we'll cover everything you need to know to get started, from choosing the right equipment to processing your images.",
        }),
        JSON.stringify({
            title: "Advanced JavaScript Concepts",
            description: "",
        }),
        JSON.stringify({
            title: "A Guide to Yoga Poses",
            description: "This guide will provide a comprehensive list of yoga poses, from beginner to advanced levels. Learn how to execute each pose correctly and the benefits each pose offers. Remember, yoga is not a competition. Always listen to your body, and modify poses as needed. Namaste.",
        }),
        JSON.stringify({
            title: "",
            description: "This is a description without a title. It can be for anything you'd like, but in this case, it's for a description without a title. It's a bit strange, isn't it?",
        }),
    ];
    const embeddings2 = await Promise.all(testStrings2.map(text => getEmbeddings(instruction2, text)));
    const embeddings2Array = embeddings2.map(embed => embed[0]);
    const searchStrings2 = ["penguins", "bake cake", "dinosaurs", "birds", "baking"];
    const searchEmbeddings2 = await Promise.all(searchStrings2.map(text => getEmbeddings(instruction2, text)));
    for (let i = 0; i < searchStrings2.length; i++) {
        const indices = knn(embeddings2Array, searchEmbeddings2[i][0], 5);
        const nearestNeighbors = indices.map(index => JSON.parse(testStrings2[index]).title);
        logger.info(`Nearest neighbors for "${searchStrings2[i]}": ${nearestNeighbors}`);
    }

    // Test 3: Same string multiple times
    const instruction3 = "Represent the text for classification";
    const testString3 = "The quick brown fox jumps over the lazy dog";
    const embeddings3 = await Promise.all([1, 2, 3].map(_ => getEmbeddings(instruction3, testString3)));
    const similarities3 = embeddings3.map((embed, i) => cosineSimilarity(embeddings3[0][0], embed[0]));
    logger.info(`Similarities for test 3: ${JSON.stringify(similarities3)}`);
};