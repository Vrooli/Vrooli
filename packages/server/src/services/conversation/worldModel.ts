import { SubroutineIOMapping } from "@local/shared";

type UserInfo = {
    name: string;
}

type ParticipantInfo = {
    id: string;
    name: string;
    description: string;
}

type TeamMemberInfo = {
    id: string;
    role: string;
    description: string;
}

type TeamInfo = {
    name: string;
    members: TeamMemberInfo[];
}

type RoutineInfo = {
    name: string;
    description: string;
    currentStep?: string;
    currentSubroutineIOMapping?: SubroutineIOMapping;
}

export type WorldModelConfig = {
    /**
     * The name of the application.
     */
    appName: string;
    /**
     * The description of the application.
     */
    appDescription: string;
    /**
     * The system message to include in the conversation.
     */
    systemMessage: string;
    /**
     * The persona of the AI assistant.
     */
    persona?: string;
    /**
     * The other participants in the conversation.
     */
    participants?: ParticipantInfo[];
    /**
     * The goal of the conversation.
     */
    goal?: string;
    /**
     * The custom fields to include in the conversation.
     */
    customFields?: Record<string, string>;
    /**
     * The team information to include in the conversation.
     */
    team?: TeamInfo;
    /**
     * The routine information to include in the conversation.
     */
    routine?: RoutineInfo;
    /**
     * The required tools to include in the conversation.
     */
    requiredTools?: string[];
    /**
     * The user the bot is talking to.
     */
    user?: UserInfo;
}

/**
 * The `WorldModel` class standardizes the configuration and serialization of a "world model" for an AI assistant.
 * It allows for the inclusion of contextual elements such as the AI’s persona, participants in the conversation,
 * the goal of the interaction, team information, routine details, custom fields, and required tools. Additionally, 
 * it supports a configurable system message that can include information about the application.
 */
export class WorldModel {
    static defaultWorldModelConfig: WorldModelConfig = {
        appName: "Vrooli",
        appDescription: "a polymorphic, collaborative, and self-improving automation platform that helps you stay organized and achieve your goals.",
        systemMessage: "",
    };

    /**
     * Serialize the world model into a system message string.
     * 
     * @param config - The configuration object to serialize.
     * @returns The formatted system message.
     */
    static serialize(config?: WorldModelConfig): string {
        if (!config) config = WorldModel.defaultWorldModelConfig;

        let message = `Welcome to ${config.appName}, ${config.appDescription}.`;

        message += `\nThe current date and time is ${new Date().toLocaleString()}.`;

        if (config.systemMessage) {
            message += `\n${config.systemMessage}`;
        }

        if (config.persona) {
            message += `\nYou are ${config.persona}.`;
        }

        if (config.user) {
            message += `\nYou are talking to ${config.user.name}.`;
        }

        if (config.team) {
            const membersStr = config.team.members.map(m => `${m.id} (${m.role}): ${m.description}`).join(", ");
            message += `\nYou are part of the ${config.team.name} team, with members: ${membersStr}.`;
        }

        if (config.participants && config.participants.length > 0) {
            const participantsStr = config.participants.map(p => `${p.id}: ${p.description}`).join(", ");
            message += `\nParticipants: ${participantsStr}.`;
            message += `\nTo mention a participant, use @ followed by their id, e.g., @${config.participants[0].id}.`;
        }

        if (config.routine) {
            message += `\n\nCurrent Routine: ${config.routine.name}\nDescription: ${config.routine.description}`;
            if (config.routine.currentStep) {
                message += `\nCurrent Step: ${config.routine.currentStep}`;
            }
            if (config.routine.currentSubroutineIOMapping) {
                message += `\nInputs: ${JSON.stringify(config.routine.currentSubroutineIOMapping.inputs)}`;
                message += `\nOutputs: ${JSON.stringify(config.routine.currentSubroutineIOMapping.outputs)}`;
            }
        }

        if (config.customFields && Object.keys(config.customFields).length > 0) {
            message += `\nCustom fields: ${JSON.stringify(config.customFields)}.`;
        }

        if (config.goal) {
            message += `\nGoal: ${config.goal}.`;
        }

        if (config.requiredTools && config.requiredTools.length > 0) {
            const toolsList = config.requiredTools.join(", ");
            if (config.goal) {
                message += ` To achieve this goal, you must call one of the following tools: ${toolsList}.`;
            } else {
                message += `\nYou must call one of the following tools: ${toolsList}.`;
            }
        }

        return message.trim();
    }
}
