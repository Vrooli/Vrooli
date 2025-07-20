import type { TutorialSection } from "./types.js";
import { PubSub } from "../../../utils/pubsub.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";

// Constants to avoid import issues
const FormStructureType = {
    Header: "Header",
    Tip: "Tip",
    Divider: "Divider",
} as const;

const LINKS = {
    Home: "/",
} as const;

// Simple ID generator
const generateId = () => Math.random().toString(36).substr(2, 9);

// New AI-first tutorial sections based on docs/user-guide/tutorial
export const tutorialSections: TutorialSection[] = [
    {
        title: "Welcome to AI-Powered Productivity",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Welcome to the future of productivity",
                        tag: "h1",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Vrooli uses **AI agent swarms** - teams of specialized AI that collaborate to accomplish your goals. Think of it as having a brilliant team that never sleeps, working 24/7 to make your ideas reality.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "AI agent swarms coordinate automatically - no micromanagement needed!",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: generateId(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        id: generateId(),
                        label: "Need this tutorial again? Look for \"Tutorial\" in the user menu",
                        icon: "Info",
                    },
                ],
                location: {
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Meet your new workspace",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "This chat interface is your primary workspace. Instead of navigating menus or filling forms, you simply **type what you want to accomplish**.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Example**: Try typing 'Help me plan my weekly schedule'",
                    },
                ],
                location: {
                    element: "chat-input-area",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Valyxa - AI Assistant",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Hello! I'm **Valyxa**, your AI assistant. I coordinate with specialized AI agents to help you accomplish any task. What would you like to work on today?",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: generateId(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "Valyxa can help with everything from simple reminders to complex project planning. Just describe what you need in **natural language**.",
                    },
                ],
                location: {
                    element: "chat-bubble-tree",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Interface Overview",
                        tag: "h3",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "**Left Panel**: Navigation & History\n**Center**: Main Conversation\n**Right Panel**: Context & Tools",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        label: "The layout automatically adjusts to your screen size and preferences",
                    },
                ],
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Your First AI Conversation",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Start your first conversation",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Click in the message area below and type what you want to work on. Here are some ideas to get started:",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "**Examples to try:**\n• Help me organize my daily tasks\n• I need to plan a project presentation\n• Create a workout routine for busy professionals\n• Help me write a professional email",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        label: "Simply type what you want to accomplish - no special syntax needed",
                    },
                ],
                location: {
                    element: "chat-input-area",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        label: "Great! You sent your first message naturally",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Notice how you wrote that like you were talking to a person? That's exactly right. Valyxa understands natural language, so you can:",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "• **Ask questions**: 'How should I approach this?'\n• **Request actions**: 'Create a schedule for my week'\n• **Brainstorm**: 'What are some creative solutions for...'\n• **Problem-solve**: 'I'm stuck with this issue...'",
                        tag: "body1",
                    },
                ],
                location: {
                    element: "chat-bubble-tree",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        label: "Look how Valyxa structured the response",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        label: "AI responses typically include:",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "• Clear understanding of your request\n• Specific suggestions or solutions\n• Follow-up questions for clarification\n• Action buttons for immediate next steps",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "The AI builds context from everything you've said. **No need to repeat yourself!**",
                    },
                ],
                location: {
                    element: "chat-bubble-tree",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Continue the conversation",
                        tag: "h3",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        label: "Now respond to Valyxa's suggestions. You can:",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Ask for more details**: 'Can you elaborate on the second suggestion?'",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Request changes**: 'That's good, but make it simpler'",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Try something different**: 'Actually, let's focus on time management instead'",
                    },
                ],
                location: {
                    element: "chat-input-area",
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "AI Agents Working Together",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Understanding Agent Swarms",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Think of agent swarms like a specialized team where each member has different expertise:",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Research Agent**: Gathers information and analyzes data",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Planning Agent**: Creates strategies and organizes workflows",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Execution Agent**: Completes tasks and uses tools",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Coordination Agent**: Manages the team and synthesizes results",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "**The magic**: They work simultaneously, share knowledge instantly, and produce results no single AI could achieve alone.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: "chat-bubble-tree",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Watch agents collaborate",
                        tag: "h3",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "When you request something complex, you'll see multiple agents working together in real-time. Each agent contributes their expertise to create a comprehensive solution.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Real-time collaboration**: Agents share information and build on each other's work",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Quality improvement**: Multiple perspectives lead to better solutions",
                    },
                ],
                location: {
                    element: "chat-bubble-tree",
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Watching Tasks Execute",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Watch your task come to life",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "When your request requires multiple steps, agents automatically create and execute a routine. You'll see the execution begin immediately.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Execution Starting**: Agents are analyzing your request and preparing the routine",
                    },
                ],
                location: {
                    element: "routine-executor",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Monitor progress",
                        tag: "h3",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "You can monitor task progress in real-time. Agents will ask for your input when needed and keep you updated on their progress.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Real-time updates**: See progress as agents work",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Interactive decisions**: Agents will ask for your input when needed",
                    },
                ],
                location: {
                    element: "routine-executor",
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Your Profile and Settings",
        steps: [
            {
                action: () => { 
                    // Ensure UserMenu is closed first
                    PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: false }); 
                },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Access your profile",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Click on your profile picture to access account settings, preferences, and personalization options.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: "user-menu-profile-icon",
                    page: LINKS.Home,
                },
            },
            {
                action: () => { 
                    // Open UserMenu
                    PubSub.get().publish("menu", { id: ELEMENT_IDS.UserMenu, isOpen: true }); 
                },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Profile and settings",
                        tag: "h3",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here you can manage your account, switch between accounts, access settings, and personalize your AI experience.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Account switching**: Click on different accounts to switch between them",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Settings**: Customize your AI behavior and preferences",
                    },
                ],
                location: {
                    element: "user-menu-account-list",
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Personalizing Your AI Experience",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "Customize your AI assistant",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "You can personalize how Valyxa and other AI agents interact with you. Access settings to adjust communication style, expertise areas, and behavior preferences.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Communication style**: Adjust formality, verbosity, and tone",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Expertise areas**: Focus AI assistance on your specific domains",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Behavior preferences**: Set how proactive or conservative the AI should be",
                    },
                ],
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Getting Help and Next Steps",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: generateId(),
                        isCollapsible: false,
                        label: "You're ready to be productive!",
                        tag: "h2",
                    },
                    {
                        type: FormStructureType.Header,
                        id: generateId(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Congratulations! You now understand the basics of AI-powered productivity with Vrooli. Start by typing what you want to accomplish in the chat.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: generateId(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Need help?** Type 'help' in the chat or access the Tutorial again from your profile menu",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Start simple**: Begin with basic requests and gradually try more complex tasks",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: generateId(),
                        isMarkdown: true,
                        label: "**Experiment**: The AI learns from your interactions and gets better over time",
                    },
                ],
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
];
