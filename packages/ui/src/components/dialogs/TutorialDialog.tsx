import { ChatPageTabOption, FormInformationalType, FormStructureType, LINKS, SEEDED_IDS, SearchPageTabOption, TutorialViewSearchParams, UrlTools, getObjectUrl, uuid } from "@local/shared";
import { Box, Button, Dialog, IconButton, LinearProgress, List, ListItem, ListItemText, ListSubheader, Menu, MenuItem, MobileStepper, Paper, PaperProps, Stack, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Draggable from "react-draggable";
import { useTranslation } from "react-i18next";
import { FormRunView } from "../../forms/FormView/FormView.js";
import { useHotkeys } from "../../hooks/useHotkeys.js";
import { usePopover } from "../../hooks/usePopover.js";
import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon, ExpandLessIcon, ExpandMoreIcon } from "../../icons/common.js";
import { useLocation } from "../../route/router.js";
import { addSearchParams, removeSearchParams } from "../../route/searchParams.js";
import { ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { TUTORIAL_HIGHLIGHT, addHighlight, removeHighlights } from "../../utils/display/documentTools.js";
import { CHAT_SIDE_MENU_ID, PubSub, SIDE_MENU_ID } from "../../utils/pubsub.js";
import { routineTypes } from "../../utils/search/schemas/routine.js";
import { PopoverWithArrow } from "../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { MarkdownDisplay } from "../text/MarkdownDisplay.js";
import { DialogTitle } from "./DialogTitle/DialogTitle.js";
import { TutorialDialogProps } from "./types.js";

type Place = {
    section: number;
    step: number;
}

type TutorialStep = {
    /**
     * Action triggered when step is reached. 
     * Useful for opening side menus, putting components in 
     * a certain state, etc.
     */
    action?: () => unknown;
    content: FormInformationalType[];
    location?: {
        /**
         * ID of the element to anchor the tutorial step to.
         */
        element?: string;
        /**
         * What page to navigate to when the step is reached.
         */
        page?: string;
    };
    /**
     * Overrides the next step to go to.
     */
    next?: Place;
    /**
     * Allows you to specify options for what place to go next. 
     * Useful for giving the user a choice of what to learn about.
     * 
     * NOTE: If only one option is provided, the user will be taken there automatically. 
     * No option will be shown.
     */
    options?: {
        label: string;
        place: Place;
    }[];
    /**
     * Overrides the previous step.
     */
    previous?: Place;
}

export type TutorialSection = {
    hideFromMenu?: boolean;
    title: string;
    steps: TutorialStep[];
}

// Sections of the tutorial
const sections: TutorialSection[] = [
    {
        title: "Welcome to Vrooli!",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "This tutorial will show you how to use Vrooli to assist your personal and professional life.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "It will only take a few minutes, and you can skip it at any time.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        id: uuid(),
                        label: "Watch the tutorial video instead (recommended for mobile users)",
                        link: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1", //TODO: Add video link
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Need this again? Look for \"Tutorial\" in the side menu",
                    },
                ],
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Home Page",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The home page shows the most important information for you.",
                        tag: "body1",
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
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The first thing you'll see are the focus mode tabs. These allow you to customize the resources and reminders shown on the home page.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "More focus mode-related features will be released in the future.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Tabs can be customized in the settings page",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.DashboardFocusModeTabs,
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Next is a customizable list of resource cards.\nThese can be anything you want, such as links to your favorite websites, or objects on Vrooli.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press the last card to add a new resource",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Hold or right-click on a resource to edit or delete it",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.DashboardResourceList,
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Next is a list of upcoming events. These can be for meetings, focus mode sessions, or scheduled tasks (more on that later).",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        isMarkdown: true,
                        label: "Press the **Open** button to see a full calendar view",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.DashboardEventList,
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Then there's a list of reminders.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "These reminders are associated with the current focus mode.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press the **Open** icon to view all reminders, regardless of focus mode",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press the **Add** icon to create a new reminder",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.DashboardReminderList,
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Main Side Menu",
        steps: [
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The side menu has many useful features.\nOpen it by pressing on your profile picture.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.SideMenuProfileIcon,
                },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The first section lists all logged-in accounts.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press on your current account to open your profile",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press on another account to switch to it",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.SideMenuAccountList,
                },
            },
            {
                action: () => {
                    PubSub.get().publish("sideMenu", {
                        id: SIDE_MENU_ID,
                        isOpen: true,
                        data: { isDisplaySettingsCollapsed: false },
                    });
                },
                content: [
                    {
                        type: FormStructureType.Header,
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The second section allows you to control your display settings. This includes:\n- **Theme**: Choose between light and dark mode.\n- **Text size**: Grow or shrink the text on all pages.\n- **Left handed**: Move various elements, such as the side menu, to the left side of the screen.\n- **Language**: Change the language of the app.\n- **Focus mode**: Switch between focus modes.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.SideMenuDisplaySettings,
                },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The third section displays additional pages not listed in the main navigation bar.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.SideMenuQuickLinks,
                },
            },
        ],
    },
    {
        title: "Searching Objects",
        steps: [
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The **Search** page allows you to explore public objects available on Vrooli.",
                        tag: "body1",
                    },
                ],
                location: {
                    page: LINKS.Search,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Use the tabs to switch between different types of objects.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.SearchTabs,
                    page: LINKS.Search,
                },
            },
        ],
    },
    {
        title: "Routines",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Routines enable you to automate and streamline various tasks.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "There are several types of routines, each with different capabilities. Let's explore them.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Routines can be used for anything, from simple tasks to complex workflows.",
                    },
                    {
                        type: FormStructureType.Tip,
                        id: uuid(),
                        label: "Watch an example of a routine in action",
                        link: "https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1", //TODO: Add video link
                    },
                ],
                location: {
                    page: `${LINKS.Search}?type="${SearchPageTabOption.RoutineMultiStep}"`,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here's an overview of each routine type.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: routineTypes.map(type => `\n- **${type.label}**: ${type.description}`).join(""),
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Routines can be created using AI, so you don't need to worry about the details.",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Warning",
                        id: uuid(),
                        label: "Not all routine types are fully implemented yet.",
                    },
                ],
                location: {
                    page: `${LINKS.Search}?type="${SearchPageTabOption.RoutineMultiStep}"`,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Choose a routine to explore, or press the check mark to skip to the next section.",
                        tag: "body1",
                    },
                ],
                location: {
                    page: `${LINKS.Search}?type="${SearchPageTabOption.RoutineMultiStep}"`,
                },
                next: { section: 10, step: 0 },
                options: [
                    {
                        label: "Project kickoff checklist (Basic)",
                        place: { section: 5, step: 0 }, //TODO
                    },
                    {
                        label: "Workout plan generator (Generate)",
                        place: { section: 6, step: 0 }, //TODO
                    },
                    {
                        label: "Fiction book writer (Multi-step)",
                        place: { section: 7, step: 0 }, //TODO
                    },
                    {
                        label: "Create reminder (Action)",
                        place: { section: 8, step: 0 }, //TODO
                    },
                    {
                        label: "Plaintext to JSON converter (Code)",
                        place: { section: 9, step: 0 }, //TODO
                    },
                ],
            },
            // TODO move steps below to other sections
            // {
            //     content: [
            //         {
            //             type: FormStructureType.Header,
            //             id: uuid(),
            //             isCollapsible: false,
            //             isMarkdown: true,
            //             label: "This routine is more complex and includes multiple steps.",
            //             tag: "body1",
            //         },
            //         {
            //             type: FormStructureType.Header,
            //             id: uuid(),
            //             isCollapsible: false,
            //             isMarkdown: true,
            //             label: "Steps help break down tasks into manageable parts and can be reused in other routines.",
            //             tag: "body2",
            //         },
            //     ],
            // },
            // {
            //     content: [
            //         {
            //             type: FormStructureType.Header,
            //             id: uuid(),
            //             isCollapsible: false,
            //             isMarkdown: true,
            //             label: "Finally, we have a fully automated routine.",
            //             tag: "body1",
            //         },
            //         {
            //             type: FormStructureType.Header,
            //             id: uuid(),
            //             isCollapsible: false,
            //             isMarkdown: true,
            //             label: "An AI bot handles the entire task, requiring minimal input from you.",
            //             tag: "body2",
            //         },
            //     ],
            // },
            // {
            //     content: [
            //         {
            //             type: FormStructureType.Header,
            //             id: uuid(),
            //             isCollapsible: false,
            //             isMarkdown: true,
            //             label: "This is just one example of how routines can be utilized.",
            //             tag: "body1",
            //         },
            //         {
            //             type: FormStructureType.Tip,
            //             icon: "Info",
            //             id: uuid(),
            //             label: "Explore and create routines to automate tasks and enhance productivity.",
            //         },
            //     ],
            // },
        ],
    },
    {
        hideFromMenu: true,
        title: "Routine - Project Kickoff Checklist",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "In this example, we're showing off the form-building capabilities of routines.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Typically, basic routines like this one are created as part of a multi-step routine. They're important for collecting information to use in later steps.",
                        tag: "body2",
                    },
                ],
                location: {
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.ProjectKickoffChecklist } as const),
                },
                previous: { section: 4, step: 2 },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here is some general information about the routine, such as its completion status and owner.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press on the owner's name to view their profile",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.RelationshipList,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.ProjectKickoffChecklist } as const),
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here are relevant links and resources for the routine.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.ResourceCards,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.ProjectKickoffChecklist } as const),
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here is the form that is filled out when running the routine.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "All of the text and input components you see here (and more!) can be added to your own routines.",
                        tag: "body2",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.RoutineTypeForm,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.ProjectKickoffChecklist } as const),
                },
                next: { section: 4, step: 2 },
            },
        ],
    },
    {
        hideFromMenu: true,
        title: "Routine - Workout Plan Generator",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "In this example, we're showcasing the AI generation capabilities of routines.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "\"Generate\" routines use the inputs you provide to produce an output.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Warning",
                        id: uuid(),
                        label: "We currently support only text inputs and outputs. This will be expanded in the future.",
                    },

                ],
                location: {
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.WorkoutPlanGenerator } as const),
                },
                previous: { section: 4, step: 2 },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here is some general information about the routine, such as its completion status and owner.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Press on the owner's name to view their profile.",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.RelationshipList,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.WorkoutPlanGenerator } as const),
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "You can choose the AI model and bot style for generating the output.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Selecting different bots or styles can significantly affect the generated output.",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.RoutineGenerateSettings,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.WorkoutPlanGenerator } as const),
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here is the form that is filled out when running the routine.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The inputs you provide here will be used by the AI to generate a personalized workout plan.",
                        tag: "body2",
                    },
                ],
                location: {
                    element: `input-${ELEMENT_IDS.FormRunView}`,
                    page: getObjectUrl({ __typename: "Routine", id: SEEDED_IDS.Routine.WorkoutPlanGenerator } as const),
                },
                next: { section: 4, step: 2 },
            },
        ],
    },
    {
        hideFromMenu: true,
        title: "Routine - Fiction Book Writer",
        steps: [],//TODO
    },
    {
        hideFromMenu: true,
        title: "Routine - Create Reminder",
        steps: [],//TODO
    },
    {
        hideFromMenu: true,
        title: "Routine - Plaintext to JSON Converter",
        steps: [],//TODO
    },
    {
        title: "Secondary Side Menu",
        steps: [
            {
                action: () => { PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: false }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The other side menu is used for AI features.\nOpen it by pressing the list icon.",
                        tag: "body1",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.ChatSideMenuIcon,
                    page: LINKS.Search,
                },
                previous: { section: 4, step: 2 },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: true }); },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "There are 4 tabs in the AI side menu.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "- **Chat View**: The active chat conversation.\n- **Chat History**: Shows your most recent conversations.\n- **Routines**: Both public and owned routines. Can be passed into a chat for a bot to run.\n- **Prompts**: Text prompts for passing instructions to a bot.",
                        tag: "body2",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.ChatSideMenuTabs,
                    page: LINKS.Search,
                },
            },
            {
                action: () => {
                    PubSub.get().publish("sideMenu", {
                        id: CHAT_SIDE_MENU_ID,
                        isOpen: true,
                        data: {
                            tab: ChatPageTabOption.Chat,
                        },
                    });
                },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "The **Chat View** tab is where the magic happens.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Use this tab to:\n- Chat with bots\n- Run routines\n- Use prompts",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Type \"/\" in the message box to bring up shortcuts",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Warning",
                        id: uuid(),
                        label: "To message bots, you will need credits. These come with a premium account, or can be purchased separately.",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.ChatSideMenuMessageTree,
                    page: LINKS.Search,
                },
            },
            {
                action: () => {
                    PubSub.get().publish("sideMenu", {
                        id: CHAT_SIDE_MENU_ID,
                        isOpen: true,
                        data: {
                            tab: ChatPageTabOption.Chat,
                        },
                    });
                },
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Here is where the active and suggested tasks are displayed.",
                        tag: "body1",
                    },
                    {
                        type: FormStructureType.Header,
                        color: "secondary",
                        id: uuid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "These are run by the bot(s) in the chat.",
                        tag: "body2",
                    },
                    {
                        type: FormStructureType.Divider,
                        id: uuid(),
                        label: "",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Tasks include routines, as well as built-in actions like autofill and search.",
                    },
                    {
                        type: FormStructureType.Tip,
                        icon: "Info",
                        id: uuid(),
                        label: "Any information you enter in the message box will be used as context when running tasks.",
                    },
                ],
                location: {
                    element: ELEMENT_IDS.TasksRow,
                    page: LINKS.Search,
                },
            },
        ],
    },
    // {
    //     title: "Informational Routine",
    //     steps: [],//TODO
    // },
    // {
    //     title: "Action Routine",
    //     steps: [],//TODO
    // },
    // {
    //     title: "Code Routine",
    //     steps: [],//TODO
    // },
    // {
    //     title: "Multi-step Routine",
    //     steps: [],//TODO
    // },
    // //TODO add sections for projects and teams maybe
    // {
    //     title: "Other Objects",
    //     steps: [],//TODO should replace "Create Objects"
    // },
    // {
    //     title: "Creating Objects",
    //     steps: [
    //         {
    //             content: {
    //                 text: "This page allows you to create new objects on Vrooli.",
    //             },
    //             location: {
    //                 page: LINKS.Create,
    //             },
    //         },
    //     ],
    // },
    // {
    //     title: "Your Stuff",
    //     steps: [
    //         {
    //             content: {
    //                 text: "After you've created an object, you can find it here.",
    //             },
    //             location: {
    //                 page: LINKS.MyStuff,
    //             },
    //         },
    //         {
    //             content: {
    //                 text: "Just like the search page, you can use these tabs to switch between different types of objects.",
    //             },
    //             location: {
    //                 element: ELEMENT_IDS.MyStuffTabs,
    //                 page: LINKS.MyStuff,
    //             },
    //         },
    //     ],
    // },
    // {
    //     title: "Inbox",
    //     steps: [
    //         {
    //             content: {
    //                 text: "This page allows you to view your messages and notifications.\n\nIf you have a premium account, you can message bots and have them run tasks and perform other actions for you.",
    //             },
    //             location: {
    //                 page: LINKS.Inbox,
    //             },
    //         },
    //     ],
    // },
    // {
    //     title: "That's it!",
    //     steps: [
    //         {
    //             content: {
    //                 text: "Now you know the basics of Vrooli. Have fun!",
    //             },
    //         },
    //         // TODO add suggested next steps, like buying premium/credits, creating a team, etc.
    //     ],
    // },
];

function getTotalSteps(sections) {
    return sections.reduce((total, section) => total + section.steps.length, 0);
}

function getCurrentStepIndex(sections, place) {
    let stepIndex = 0;
    for (let i = 0; i < place.section; i++) {
        stepIndex += sections[i].steps.length;
    }
    stepIndex += place.step;
    return stepIndex;
}

/**
 * Returns information about the current tutorial step.
 * @param sections - Array of tutorial sections.
 * @param Current section and step index in the tutorial.
 */
export function getTutorialStepInfo(
    sections: TutorialSection[],
    place: Place,
) {
    const section = sections[place.section];
    const nextSection = sections[place.section + 1];

    if (!section || place.step < 0 || place.step >= section.steps.length) {
        return {
            isFinalStep: false,
            isFinalStepInSection: false,
            nextStep: null,
        };
    }

    const isFinalStepInSection = place.step === section.steps.length - 1;
    const isFinalSection = place.section === sections.length - 1;
    const isFinalStep = isFinalStepInSection && isFinalSection;

    const nextStep = isFinalStep
        ? null
        : isFinalStepInSection
            ? nextSection?.steps[0] || null
            : section.steps[place.step + 1];

    return { isFinalStep, isFinalStepInSection, nextStep };
}

export function isValidPlace(sections: TutorialSection[], place: Place) {
    if (typeof place.section !== "number" || typeof place.step !== "number") return false;
    const section = sections[place.section];
    if (!section) return false;
    const step = section.steps[place.step];
    return !!step;
}

export function getNextPlace(
    sections: TutorialSection[],
    place: Place,
): Place | null {
    if (!isValidPlace(sections, place)) return { section: 0, step: 0 };

    const currentSection = sections[place.section];
    const currentStep = currentSection?.steps[place.step];
    const nextSection = sections[place.section + 1];

    // If the step has a specific next place, use that
    if (currentStep?.next) return currentStep.next;

    // If current step has exactly one option, use that as next place
    if (currentStep?.options?.length === 1) {
        return currentStep.options[0].place;
    }

    // Otherwise follow normal sequential flow
    if (currentSection && currentSection.steps[place.step + 1]) {
        return { section: place.section, step: place.step + 1 };
    } else if (nextSection && nextSection.steps[0]) {
        return { section: place.section + 1, step: 0 };
    } else {
        return null; // No more steps
    }
}

export function getPrevPlace(
    sections: TutorialSection[],
    place: Place,
): Place | null {
    if (!isValidPlace(sections, place)) return { section: 0, step: 0 };

    const currentSection = sections[place.section];
    const currentStep = currentSection?.steps[place.step];
    const previousSection = sections[place.section - 1];

    // If the step has a specific previous place, use that
    if (currentStep?.previous) return currentStep.previous;

    if (currentSection && currentSection.steps[place.step - 1]) {
        return { section: place.section, step: place.step - 1 };
    } else if (previousSection) {
        const prevStepIndex = previousSection.steps.length - 1;
        return { section: place.section - 1, step: prevStepIndex };
    }
    return null; // No previous steps
}

export function getCurrentElement(
    sections: TutorialSection[],
    place: Place,
): HTMLElement | null {
    if (!isValidPlace(sections, place)) return null;
    const currentSection = sections[place.section];
    const currentStep = currentSection?.steps[place.step];
    const elementId = currentStep?.location?.element;
    return elementId ? document.getElementById(elementId) : null;
}

export function getCurrentStep(
    sections: TutorialSection[],
    place: Place,
): TutorialStep | null {
    if (!isValidPlace(sections, place)) return null;
    const currentSection = sections[place.section];
    return currentSection?.steps[place.step] || null;
}

const titleId = "tutorial-dialog-title";
/** Poll interval for finding anchor element */
const POLL_INTERVAL_MS = 1000;
const INITIAL_RENDER_DELAY_MS = 100;

const wrongPageDialogTitleStyle = { root: { cursor: "move" } } as const;
const popoverWithArrowStyle = {
    root: {
        zIndex: Z_INDEX.TutorialDialog,
        maxWidth: "500px",
    },
    content: { padding: 0 },
} as const;
const sectionMenuSlotProps = {
    root: {
        style: {
            zIndex: Z_INDEX.TutorialDialog + 1,
        },
    },
    paper: {
        style: {
            maxHeight: "60vh",
            width: "250px",
        },
    },
} as const;

function DraggableDialogPaper(props: PaperProps) {
    return (
        <Draggable
            handle={`#${titleId}`}
            cancel={"[class*=\"MuiDialogContent-root\"]"}
        >
            <Paper {...props} />
        </Draggable>
    );
}

const ContentWrapper = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    height: "100%",
    "& .content": {
        flex: 1,
        overflowY: "auto",
        padding: theme.spacing(2),
    },
    "& .stepper": {
        flexShrink: 0,
        borderTop: `1px solid ${theme.palette.divider}`,
    },
}));
const SectionTitleBox = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    gap: 0,
    paddingBottom: theme.spacing(2),
    cursor: "pointer",
}));
const StyledStepper = styled(MobileStepper)(({ theme }) => ({
    background: theme.palette.background.paper,
    borderTop: `1px solid ${theme.palette.divider}`,
    position: "relative",
}));

const StyledDialog = styled(Dialog)(({ theme }) => ({
    zIndex: Z_INDEX.TutorialDialog,
    pointerEvents: "none",
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            zIndex: Z_INDEX.TutorialDialog,
            pointerEvents: "auto",
            borderRadius: theme.spacing(2),
            margin: theme.spacing(2),
            maxWidth: "500px",
            background: theme.palette.background.paper,
            position: "absolute",
            top: "0",
            left: "0",
        },
    },
    "& .MuiBackdrop-root": {
        display: "none",
    },
}));

export function TutorialDialog({
    isOpen,
    onClose,
    onOpen,
}: TutorialDialogProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [{ pathname, search }, setLocation] = useLocation();

    const pollIntervalRef = useRef<number>();
    // Add a ref to track initial render. Added to prevent hotkeys 
    // from triggering on initial render, if tutorial is selected using 
    // hotkeys in the command palette.
    const initialRenderRef = useRef(true);
    const [isInitialRender, setIsInitialRender] = useState(true);

    useEffect(function clearInitialRenderEffect() {
        // Clear the initial render flag after a short delay
        if (isOpen && initialRenderRef.current) {
            const timer = setTimeout(() => {
                initialRenderRef.current = false;
                setIsInitialRender(false);
            }, INITIAL_RENDER_DELAY_MS); // Small delay to avoid the initial Enter keypress
            return () => clearTimeout(timer);
        }
    }, [isOpen]);

    const [place, setPlace] = useState<Place>({ section: 0, step: 0 });
    useEffect(function initializePlace() {
        const searchParams = UrlTools.parseSearchParams("Tutorial");
        const section = searchParams.tutorial_section ? parseInt(searchParams.tutorial_section + "", 10) : undefined;
        const step = searchParams.tutorial_step ? parseInt(searchParams.tutorial_step + "", 10) : undefined;
        if (section !== undefined && step !== undefined && isValidPlace(sections, { section, step })) {
            setPlace({ section, step });
            onOpen();
        }
    }, [onOpen]);
    useEffect(function updatePlaceInUrl() {
        if (!isOpen) return;
        addSearchParams(setLocation, {
            tutorial_section: place.section,
            tutorial_step: place.step,
        } as TutorialViewSearchParams);
    }, [place.section, place.step, isOpen, setLocation]);

    const handleClose = useCallback(function handleCloseCallback() {
        setPlace({ section: 0, step: 0 });
        removeHighlights(TUTORIAL_HIGHLIGHT);
        removeSearchParams(setLocation, ["tutorial_section", "tutorial_step"]);
        onClose();
        setTimeout(() => {
            setIsInitialRender(true);
            initialRenderRef.current = true;
        }, INITIAL_RENDER_DELAY_MS);
    }, [onClose, setLocation]);

    useEffect(function triggerStepLoadAction() {
        if (!isOpen || !isValidPlace(sections, place)) return;
        sections[place.section]?.steps[place.step]?.action?.();
    }, [isOpen, place]);

    // Find information about our position in the tutorial
    const {
        isFinalStep,
        isFinalStepInSection,
        nextStep,
    } = useMemo(() => getTutorialStepInfo(sections, place), [place]);

    const totalSteps = getTotalSteps(sections);
    const currentStepIndex = getCurrentStepIndex(sections, place);
    const percentageComplete = Math.round(((currentStepIndex + 1) / totalSteps) * 100);

    const handleNext = useCallback(function handleNextCallback() {
        const nextPlace = getNextPlace(sections, place);
        if (nextPlace) {
            const nextStep = getCurrentStep(sections, nextPlace);
            const nextPage = nextStep?.location?.page;
            if (nextPage && nextPage !== pathname) {
                setLocation(nextPage);
            }
            setPlace(nextPlace);
        } else {
            handleClose();
        }
    }, [pathname, handleClose, place, setLocation]);

    const handlePrev = useCallback(function handlePreviousCallback() {
        const prevPlace = getPrevPlace(sections, place);
        if (prevPlace) {
            const prevStep = getCurrentStep(sections, prevPlace);
            const prevPage = prevStep?.location?.page;
            if (prevPage && prevPage !== pathname) {
                setLocation(prevPage);
            }
            setPlace(prevPlace);
        }
    }, [pathname, place, setLocation]);

    useHotkeys([
        {
            keys: ["ArrowRight", "l"],
            callback: handleNext,
        },
        {
            keys: ["ArrowLeft", "h"],
            callback: handlePrev,
        },
        {
            keys: ["Enter", " "],
            callback: handleNext,
        },
    ], isOpen && !isInitialRender);

    const anchorElement = useMemo(
        () => getCurrentElement(sections, place),
        [place],
    );
    // Poll for anchor element, since it might not be available yet on initial render
    useEffect(function pollForAnchorElementEffect() {
        if (!isOpen || anchorElement) {
            // Clear interval if dialog is closed, element is found, or polling is disabled
            if (pollIntervalRef.current) {
                window.clearInterval(pollIntervalRef.current);
                pollIntervalRef.current = undefined;
            }
            return;
        }

        // Start polling if element is not found
        pollIntervalRef.current = window.setInterval(() => {
            const element = getCurrentElement(sections, place);
            if (element) {
                // Force a re-render to update the anchor element
                setPlace(prev => ({ ...prev }));
            }
        }, POLL_INTERVAL_MS);

        return () => {
            if (pollIntervalRef.current) {
                window.clearInterval(pollIntervalRef.current);
                pollIntervalRef.current = undefined;
            }
        };
    }, [isOpen, anchorElement, place]);
    const anchorElementRef = useRef(anchorElement);

    useEffect(function highlightElementEffect() {
        // Remove highlight from previous element
        if (anchorElementRef.current) {
            removeHighlights(TUTORIAL_HIGHLIGHT, anchorElementRef.current);
        }
        // Highlight current element
        if (anchorElement) {
            addHighlight(TUTORIAL_HIGHLIGHT, anchorElement);
        }
        anchorElementRef.current = anchorElement;
    }, [anchorElement, palette.action.hover]);

    const [sectionMenuAnchorEl, handleSectionMenuOpen, handleSectionMenuClose, isSectionMenuOpen] = usePopover();
    const handleSectionSelect = useCallback(function handleSectionSelectCallback(sectionIndex: number) {
        const updatedPlace = { section: sectionIndex, step: 0 };
        if (!isValidPlace(sections, updatedPlace)) return;
        setPlace(updatedPlace);
        handleSectionMenuClose();
        const section = sections[updatedPlace.section];
        const firstStep = section.steps[0];
        const page = firstStep?.location?.page;
        if (page && page !== pathname) {
            setLocation(page);
        }
    }, [handleSectionMenuClose, pathname, setLocation]);

    const content = useMemo(() => {
        if (!isValidPlace(sections, place)) {
            PubSub.get().publish("snack", { message: "Failed to load tutorial", severity: "Error" });
            return null;
        }
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            PubSub.get().publish("snack", { message: "Failed to load tutorial step", severity: "Error" });
            return null;
        }
        const currentStep = currentSection.steps[place.step];

        // Guide user to correct page if they're on the wrong one
        const correctPage = currentStep.location?.page;
        function toCorrectPage() {
            if (!correctPage) return;
            setLocation(correctPage);
        }
        if (correctPage) {
            const correctURL = new URL(correctPage, window.location.origin);

            // Compare pathnames
            const isPathnameDifferent = pathname !== correctURL.pathname;

            // Compare required query parameters
            const correctParams = new URLSearchParams(correctURL.search);
            const currentParams = new URLSearchParams(search);

            let isQueryParamsDifferent = false;
            for (const [key, value] of correctParams.entries()) {
                if (currentParams.get(key) !== value) {
                    isQueryParamsDifferent = true;
                    break;
                }
            }

            const isOnWrongPage = isPathnameDifferent || isQueryParamsDifferent;

            if (isOnWrongPage) {
                return (
                    <ContentWrapper>
                        <DialogTitle
                            id={titleId}
                            title={"Wrong Page"}
                            onClose={handleClose}
                            variant="subheader"
                            sxs={wrongPageDialogTitleStyle}
                        />
                        <Box className="content">
                            <Stack direction="column" spacing={2} p={2}>
                                <MarkdownDisplay
                                    variant="body1"
                                    content={"Please return to the correct page to continue the tutorial."}
                                />
                                <Button
                                    fullWidth
                                    variant="contained"
                                    color="primary"
                                    onClick={toCorrectPage}
                                >
                                    Continue
                                </Button>
                            </Stack>
                        </Box>
                    </ContentWrapper>
                );
            }
        }

        // Otherwise, show the current step
        const dialogTitleStyle = { root: { cursor: anchorElement ? "auto" : "move" } } as const; // Can only drag dialogs, not popovers
        const stepFormSchema = {
            containers: [{
                direction: "column",
                disableCollapse: true,
                totalItems: currentStep.content.length,
            }],
            elements: currentStep.content,
        } as const;
        return (
            <ContentWrapper>
                <DialogTitle
                    id={titleId}
                    title={t("Tutorial")}
                    onClose={handleClose}
                    variant="subheader"
                    sxs={dialogTitleStyle}
                />
                <Box className="content" p={2}>
                    <SectionTitleBox onClick={handleSectionMenuOpen}>
                        <Typography variant="h5" component="h2" lineHeight={2}>
                            {currentSection.title}
                        </Typography>
                        <IconButton edge="end" color="inherit">
                            {
                                isSectionMenuOpen ?
                                    <ExpandLessIcon fill={palette.background.textPrimary} /> :
                                    <ExpandMoreIcon fill={palette.background.textPrimary} />
                            }
                        </IconButton>
                    </SectionTitleBox>
                    <Menu
                        id="section-menu"
                        anchorEl={sectionMenuAnchorEl}
                        open={isSectionMenuOpen}
                        onClose={handleSectionMenuClose}
                        slotProps={sectionMenuSlotProps}
                    >
                        <ListSubheader>{"Sections"}</ListSubheader>
                        {sections.filter((section) => !section.hideFromMenu).map((section, index) => {
                            function handleSectionSelectCallback() {
                                handleSectionSelect(index);
                            }

                            return (
                                <MenuItem
                                    key={index}
                                    selected={index === place.section}
                                    onClick={handleSectionSelectCallback}
                                >
                                    {`${index + 1}. ${section.title}`}
                                </MenuItem>
                            );
                        })}
                    </Menu>
                    <FormRunView
                        disabled={true}
                        schema={stepFormSchema}
                    />
                    {/* Display redirect options if available */}
                    {currentStep.options && currentStep.options.length > 1 && (
                        <Box mt={2}>
                            <List>
                                {currentStep.options.map((option, index) => {
                                    function handleOptionSelect() {
                                        setPlace(option.place);
                                        const page = sections[option.place.section].steps[0].location?.page;
                                        if (page && page !== pathname) {
                                            setLocation(page);
                                        }
                                    }

                                    return (
                                        <ListItem
                                            key={index}
                                            button
                                            onClick={handleOptionSelect}
                                        >
                                            <ListItemText
                                                primary={
                                                    <Typography color={palette.mode === "light" ? "#001cd3" : "#dd86db"}>
                                                        ◦ {option.label}
                                                    </Typography>
                                                }
                                            />
                                        </ListItem>
                                    );
                                })}
                            </List>
                        </Box>
                    )}
                </Box>
                <StyledStepper
                    className="stepper"
                    variant="dots"
                    steps={currentSection.steps.length}
                    activeStep={place.step}
                    backButton={
                        <IconButton
                            onClick={handlePrev}
                            disabled={place.section === 0 && place.step === 0}
                        >
                            <ArrowLeftIcon />
                        </IconButton>
                    }
                    nextButton={
                        <IconButton
                            onClick={handleNext}
                        >
                            {isFinalStep ? <CompleteAllIcon /> : isFinalStepInSection ? <CompleteIcon /> : <ArrowRightIcon />}
                        </IconButton>
                    }
                />
                <LinearProgress variant="determinate" value={percentageComplete} />
            </ContentWrapper>
        );
    }, [place, anchorElement, t, handleClose, handleSectionMenuOpen, isSectionMenuOpen, palette.background.textPrimary, palette.mode, sectionMenuAnchorEl, handleSectionMenuClose, handlePrev, handleNext, isFinalStep, isFinalStepInSection, percentageComplete, setLocation, pathname, search, handleSectionSelect]);

    useEffect(function autoAdvanceOnCorrectNavigationEffect() {
        if (!isValidPlace(sections, place)) return;
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            PubSub.get().publish("snack", { message: "Failed to load tutorial", severity: "Error" });
            return;
        }
        const currentStep = currentSection.steps[place.step];
        // Find current step's page
        const currPage = currentStep.location?.page;
        // If already on the correct page, return
        if (currPage && currPage === pathname) return;

        // Find next step's page
        const nextPage = nextStep?.location?.page;

        // If next step has a page and it's the current page, advance
        if (currPage && nextPage && nextPage === pathname) {
            handleNext();
        }
    }, [handleNext, pathname, nextStep?.location?.page, place, setLocation]);

    return (
        <>
            {anchorElement ? (
                <PopoverWithArrow
                    anchorEl={anchorElement}
                    disableScrollLock={true}
                    sxs={popoverWithArrowStyle}
                >
                    {content}
                </PopoverWithArrow>
            ) : (
                <StyledDialog
                    open={isOpen}
                    scroll="paper"
                    disableScrollLock={true}
                    aria-labelledby={titleId}
                    PaperComponent={DraggableDialogPaper}
                >
                    {content}
                </StyledDialog>
            )}
        </>
    );
}
