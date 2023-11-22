import Bull from "bull";
import { PreMapMessageData, PreMapUserData } from "../../models/base/chatMessage.js";
import { HOST, PORT } from "../../redisConn.js";
import { SessionUserToken } from "../../types.js";
import { llmProcess } from "./process.js";

export type RequestBotResponsePayload = {
    chatId: string;
    messageId: string;
    message: PreMapMessageData;
    respondingBotId: string;
    participantsData: Record<string, PreMapUserData>;
    userData: SessionUserToken;
}

const llmQueue = new Bull<RequestBotResponsePayload>("llm", { redis: { port: PORT, host: HOST } });
llmQueue.process(llmProcess);

/**
 * Responds to a chat message, handling response generation and processing, 
 * websocket events, and any other logic
 */
export function requestBotResponse(props: RequestBotResponsePayload) {
    llmQueue.add(props);
}
