import type { SubroutineIOMapping } from "@local/shared";

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
 * It allows for the inclusion of contextual elements such as the AI's persona, participants in the conversation,
 * the goal of the interaction, team information, routine details, custom fields, and required tools. Additionally, 
 * it supports a configurable system message that can include information about the application.
 */
export class WorldModel {
    private config: WorldModelConfig;

    private static readonly defaultConfig: WorldModelConfig = {
        appName: "Vrooli",
        appDescription: "a polymorphic, collaborative, and self-improving automation platform that helps you stay organized and achieve your goals.",
        systemMessage: "",
    };

    /**
     * Creates an instance of WorldModel.
     * @param initialConfig - Optional initial configuration to override default values.
     */
    constructor(initialConfig?: Partial<WorldModelConfig>) {
        this.config = {
            ...WorldModel.defaultConfig,
            ...initialConfig,
        };
    }

    /**
     * Updates the current world model configuration with the provided partial configuration.
     * @param updates - A partial configuration object with fields to update.
     */
    public updateConfig(updates: Partial<WorldModelConfig>): void {
        this.config = {
            ...this.config,
            ...updates,
        };
    }

    /**
     * Serialize the world model into a system message string.
     * 
     * @returns The formatted system message.
     */
    public serialize(): string {
        let message = `Welcome to ${this.config.appName}, ${this.config.appDescription}.`;

        message += `\nThe current date and time is ${new Date().toLocaleString()}.`;

        if (this.config.systemMessage) {
            message += `\n${this.config.systemMessage}`;
        }

        if (this.config.persona) {
            message += `\nYou are ${this.config.persona}.`;
        }

        if (this.config.user) {
            message += `\nYou are talking to ${this.config.user.name}.`;
        }

        if (this.config.team) {
            const membersStr = this.config.team.members.map(m => `${m.id} (${m.role}): ${m.description}`).join(", ");
            message += `\nYou are part of the ${this.config.team.name} team, with members: ${membersStr}.`;
        }

        if (this.config.participants && this.config.participants.length > 0) {
            const participantsStr = this.config.participants.map(p => `${p.id}: ${p.description}`).join(", ");
            message += `\nParticipants: ${participantsStr}.`;
            message += `\nTo mention a participant, use @ followed by their id, e.g., @${this.config.participants[0].id}.`;
        }

        if (this.config.routine) {
            message += `\n\nCurrent Routine: ${this.config.routine.name}\nDescription: ${this.config.routine.description}`;
            if (this.config.routine.currentStep) {
                message += `\nCurrent Step: ${this.config.routine.currentStep}`;
            }
            if (this.config.routine.currentSubroutineIOMapping) {
                message += `\nInputs: ${JSON.stringify(this.config.routine.currentSubroutineIOMapping.inputs)}`;
                message += `\nOutputs: ${JSON.stringify(this.config.routine.currentSubroutineIOMapping.outputs)}`;
            }
        }

        if (this.config.customFields && Object.keys(this.config.customFields).length > 0) {
            message += `\nCustom fields: ${JSON.stringify(this.config.customFields)}.`;
        }

        if (this.config.goal) {
            message += `\nGoal: ${this.config.goal}.`;
        }

        if (this.config.requiredTools && this.config.requiredTools.length > 0) {
            const toolsList = this.config.requiredTools.join(", ");
            if (this.config.goal) {
                message += ` To achieve this goal, you must call one of the following tools: ${toolsList}.`;
            } else {
                message += `\nYou must call one of the following tools: ${toolsList}.`;
            }
        }

        return message.trim();
    }

    /**
     * Get the underlying configuration object for this world model.
     * @returns The WorldModelConfig used by this instance.
     */
    public getConfig(): WorldModelConfig {
        return this.config;
    }
}
