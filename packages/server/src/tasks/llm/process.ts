import { ChatMessage } from "@local/shared";
import { Job } from "bull";
import { addSupplementalFields } from "../../builders/addSupplementalFields";
import { modelToGql } from "../../builders/modelToGql";
import { selectHelper } from "../../builders/selectHelper";
import { chatMessage_findOne } from "../../endpoints";
import { Trigger } from "../../events/trigger";
import { io } from "../../io";
import { withPrisma } from "../../utils/withPrisma";
import { RequestBotResponsePayload } from "./queue";
import { BotSettings, getLanguageModelService } from "./service";

/**
 * Converts db bot info to BotSettings type
 */
const toBotSettings = (bot: { name: string, botSettings: string | undefined }): BotSettings => {
    let result: BotSettings = { name: bot.name };
    if (typeof bot.botSettings !== "string") return result;
    try {
        const botSettings = JSON.parse(bot.botSettings);
        if (typeof botSettings !== "object") return result;
        result = { ...botSettings, ...result };
        // eslint-disable-next-line no-empty
    } catch { }
    return result;
};

// TODO should query messages all at once if there are less than k in the chat, otherwise batch. Should select 
// fork data, fork's fork data, etc. a few times, as a way to locate relevant messages
// TODO to handle race conditions where multiple messages set the same forkId, we should batch both upwards (see the previous TODO), 
// and downwards. To query downwards, get all forkIds from upwards query and query for all messages with those forkIds. Then,
// combine upwards and downwards.
// TODO create cron job to fix messages with duplicate forkIds, and to fix messages with forkIds that don't exist
export async function llmProcess({ data }: Job<RequestBotResponsePayload>) {
    await withPrisma({
        process: async (prisma) => {
            const { chatId, messageId, message, respondingBotId, participantsData, userData } = data;
            const bot = participantsData[respondingBotId];
            if (!bot) throw new Error("Bot data not found in participants data");
            const botSettings = toBotSettings(bot);
            const service = getLanguageModelService(botSettings);
            // Send typing message while bot is responding
            io.to(message.chatId as string).emit("typing", { starting: [respondingBotId] });
            const text = await service.generateResponse(chatId, messageId, message.content, respondingBotId, botSettings, userData);
            const select = selectHelper(chatMessage_findOne);
            const createdData = await prisma.chat_message.create({
                data: {
                    chat: { connect: { id: message.chatId as string } },
                    parent: { connect: { id: messageId } },
                    user: { connect: { id: respondingBotId } },
                    translations: {
                        create: {
                            language: message.language,
                            text,
                        },
                    },
                },
                ...select,
            });
            const formatted = modelToGql(createdData, chatMessage_findOne);
            const fullMessageData = (await addSupplementalFields(prisma, userData, [formatted], chatMessage_findOne))[0] as ChatMessage;
            await Trigger(prisma, [message.language]).objectCreated({
                createdById: respondingBotId,
                hasCompleteAndPublic: true, // N/A
                hasParent: true, // N/A
                owner: { id: respondingBotId, __typename: "User" },
                objectId: fullMessageData.id,
                objectType: "ChatMessage",
            });
            await Trigger(prisma, [message.language]).chatMessageCreated({
                createdById: respondingBotId,
                data: message,
                message: fullMessageData,
            });
        },
        trace: "0081",
    });
    if (data.message?.chatId && data?.respondingBotId) io.to(data.message.chatId).emit("typing", { stopping: [data.respondingBotId] });
}
