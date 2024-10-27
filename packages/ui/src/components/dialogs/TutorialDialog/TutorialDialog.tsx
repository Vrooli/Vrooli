import { LINKS, parseSearchParams } from "@local/shared";
import { Box, Button, Dialog, IconButton, MobileStepper, Paper, PaperProps, Stack, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useHotkeys } from "hooks/useHotkeys";
import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Draggable from "react-draggable";
import { addSearchParams, removeSearchParams, useLocation } from "route";
import { TUTORIAL_HIGHLIGHT, removeHighlights } from "utils/display/documentTools";
import { PubSub, SIDE_MENU_ID } from "utils/pubsub";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { TutorialDialogProps } from "../types";

type TutorialStep = {
    /**
     * Action triggered when step is reached. 
     * Useful for opening side menus, putting components in 
     * a certain state, etc.
     */
    action?: () => unknown;
    content: {
        /**
         * Main text. Markdown is supported.
         */
        text: string;
        /** 
         * Optional image under text
         */
        imageUrl?: string;
        /**
         * Optional YouTube video under text
         */
        videoUrl?: string;
    };
    location?: {
        /**
         * ID of the element to anchor the tutorial step to.
         */
        element?: string;
        /**
         * What page to navigate to when the step is reached.
         */
        page?: LINKS;
    }
}

export type TutorialSection = {
    title: string;
    steps: TutorialStep[];
}

// Sections of the tutorial
const sections: TutorialSection[] = [
    {
        title: "Welcome to Vrooli!",
        steps: [
            {
                content: {
                    text: "This tutorial will show you how to use Vrooli to assist your personal and professional life.\n\nIt will only take a few minutes, and you can skip it at any time.\n\nTo open the tutorial again, press the **Tutorial** option in the side menu.",
                },
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "The Home Page",
        steps: [
            {
                content: {
                    text: "The home page shows the most important information for you.",
                },
                location: {
                    page: LINKS.Home,
                },
            },
            {
                content: {
                    text: "The first thing you'll see are the focus mode tabs. These allow you to customize the resources and reminders shown on the home page.\n\nBy default, these are **Work** and **Study**. These can be customized in the settings page.\n\nMore focus mode-related features will be released in the future.",
                },
                location: {
                    element: "home-tabs",
                    page: LINKS.Home,
                },
            },
            {
                content: {
                    text: "Next is a customizable list of resources.\n\nThese can be anything you want, such as links to your favorite websites, or objects on Vrooli.",
                },
                location: {
                    element: "main-resource-list",
                    page: LINKS.Home,
                },
            },
            {
                content: {
                    text: "Next is a list of upcoming events. These can be for meetings, focus mode sessions, or scheduled tasks (more on this later).\n\nOpening the schedule will show you a calendar view of your events.",
                },
                location: {
                    element: "main-event-list",
                    page: LINKS.Home,
                },
            },
            {
                content: {
                    text: "Then there's a list of reminders.\n\nThese are associated with the current focus mode. Press **See All** to view all reminders, regardless of focus mode.",
                },
                location: {
                    element: "main-reminder-list",
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "The Side Menu",
        steps: [
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false }); },
                content: {
                    text: "The side menu has many useful features. Open it by pressing on your profile picture.",
                },
                location: {
                    element: "side-menu-profile-icon",
                },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); },
                content: {
                    text: "The first section lists all logged-in accounts.\n\nPress on an account to switch to it, or press on your current account to open your profile.",
                },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); },
                content: {
                    text: "The second section allows you to control your display settings. This includes:\n\n- **Theme**: Choose between light and dark mode.\n- **Text size**: Grow or shrink the text on all pages.\n- **Left handed?**: Move various elements, such as the side menu, to the left side of the screen.\n- **Language**: Change the language of the app.\n- **Focus mode**: Switch between focus modes.",
                },
            },
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: true }); },
                content: {
                    text: "The third section displays a list of pages not listed in the main navigation bar.",
                },
            },
        ],
    },
    {
        title: "Searching Objects",
        steps: [
            {
                action: () => { PubSub.get().publish("sideMenu", { id: SIDE_MENU_ID, isOpen: false }); },
                content: {
                    text: "This page allows you to search public objects on Vrooli.",
                },
                location: {
                    page: LINKS.Search,
                },
            },
            {
                content: {
                    text: "Use these tabs to switch between different types of objects.\n\nThe first, default tab is for **routines**. These allow you to complete and automate various tasks.\n\nLet's look at some routines now.",
                },
                location: {
                    element: "search-tabs",
                    page: LINKS.Search,
                },
            },
        ],
    },
    // {
    //     title: "Routines",
    //     steps: [
    //         {
    //             text: "There are an endless number of tasks which can be completed with routines, and an endless way to design them\n\nLet's look at increasingly complex routines, all designed for a specific task - researching a topic.",
    //         },
    //         {
    //             text: "This routine is as simple as they come. It has no steps or automation features - it's just a simple guide with resources.\n\nWhen automating a task, it can be helpful to start with a simple routine like this, and then add steps and automation features as you go.",
    //         },
    //         {
    //             text: "This routine is a bit more complex. While still having no automation features, you'll notice that it has multiple steps.\n\nSteps are a great way to break down a task into smaller, more manageable chunks. Additionally, you can reuse steps across other routines, saving you time and effort.",
    //         },
    //         {
    //             text: "Finally, we have a routine which automates the entire task.\n\nInstead of viewing resources manually and typing in each step's inputs yourself, an AI bot will do it for you.",
    //         },
    //         {
    //             text: "This is just one example of how routines can be used. There are many other ways to use them, and many other features you can explore.",
    //         },
    //     ],
    // },
    {
        title: "Creating Objects",
        steps: [
            {
                content: {
                    text: "This page allows you to create new objects on Vrooli.",
                },
                location: {
                    page: LINKS.Create,
                },
            },
        ],
    },
    {
        title: "Your Stuff",
        steps: [
            {
                content: {
                    text: "After you've created an object, you can find it here.",
                },
                location: {
                    page: LINKS.MyStuff,
                },
            },
            {
                content: {
                    text: "Just like the search page, you can use these tabs to switch between different types of objects.",
                },
                location: {
                    element: "my-stuff-tabs",
                    page: LINKS.MyStuff,
                },
            },
        ],
    },
    {
        title: "Inbox",
        steps: [
            {
                content: {
                    text: "This page allows you to view your messages and notifications.\n\nIf you have a premium account, you can message bots and have them run tasks and perform other actions for you.",
                },
                location: {
                    page: LINKS.Inbox,
                },
            },
        ],
    },
    {
        title: "That's it!",
        steps: [
            {
                content: {
                    text: "Now you know the basics of Vrooli. Have fun!",
                },
            },
        ],
    },
];

/**
 * Returns information about the current tutorial step.
 * @param sections - Array of tutorial sections.
 * @param Current section and step index in the tutorial.
 */
export function getTutorialStepInfo(
    sections: TutorialSection[],
    place: { section: number; step: number },
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

export function getNextPlace(
    sections: TutorialSection[],
    place: { section: number; step: number },
) {
    const currentSection = sections[place.section];
    const nextSection = sections[place.section + 1];

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
    place: { section: number; step: number },
) {
    const currentSection = sections[place.section];
    const previousSection = sections[place.section - 1];

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
    place: { section: number; step: number },
) {
    const currentSection = sections[place.section];
    const currentStep = currentSection?.steps[place.step];
    const elementId = currentStep?.location?.element;
    return elementId ? document.getElementById(elementId) : null;
}

export function getCurrentStep(
    sections: TutorialSection[],
    place: { section: number; step: number },
): TutorialStep | null {
    const currentSection = sections[place.section];
    return currentSection?.steps[place.step] || null;
}

const titleId = "tutorial-dialog-title";
const zIndex = 100000;
/** Poll interval for finding anchor element */
const POLL_INTERVAL_MS = 1000;
const INITIAL_RENDER_DELAY_MS = 100;

const wrongPageDialogTitleStyle = { root: { cursor: "move" } } as const;
const mobileStepperStyle = { background: "transparent" } as const;
const popoverWithArrowStyle = {
    root: {
        zIndex,
        maxWidth: "500px",
    },
    content: { padding: 0 },
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

export function TutorialDialog({
    isOpen,
    onClose,
    onOpen,
}: TutorialDialogProps) {
    const { palette } = useTheme();
    const [{ pathname }, setLocation] = useLocation();

    const [openImageUrl, setOpenImageUrl] = useState("");
    const [openVideoUrl, setOpenVideoUrl] = useState("");

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

    const [place, setPlace] = useState({ section: 0, step: 0 });
    useEffect(function initializePlace() {
        const searchParams = parseSearchParams();
        const section = parseInt(searchParams.tutorial_section as string, 10);
        const step = parseInt(searchParams.tutorial_step as string, 10);
        if (!isNaN(section) && !isNaN(step)) {
            setPlace({ section, step });
            onOpen();
        }
    }, [onOpen]);
    useEffect(function updatePlaceInUrl() {
        if (!isOpen) return;
        addSearchParams(setLocation, {
            tutorial_section: place.section,
            tutorial_step: place.step,
        });
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
        if (!isOpen) return;
        sections[place.section]?.steps[place.step]?.action?.();
    }, [isOpen, place]);

    // Find information about our position in the tutorial
    const {
        isFinalStep,
        isFinalStepInSection,
        nextStep,
    } = useMemo(() => getTutorialStepInfo(sections, place), [place]);

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
    console.log("qqqq is initial render", isInitialRender);

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
            anchorElement.classList.add(TUTORIAL_HIGHLIGHT);
        }
        anchorElementRef.current = anchorElement;
    }, [anchorElement, palette.action.hover]);

    const content = useMemo(() => {
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            PubSub.get().publish("snack", { message: "Failed to load tutorial step", severity: "Error" });
            return null;
        }
        const currentStep = currentSection.steps[place.step];

        // Guide user to correct page if they're on the wrong one
        const correctPage = currentStep.location?.page;
        const isOnWrongPage = correctPage && correctPage !== pathname;
        function toCorrectPage() {
            if (!isOnWrongPage) return;
            setLocation(correctPage);
        }
        if (isOnWrongPage) {
            return (
                <>
                    <DialogTitle
                        id={titleId}
                        title={"Wrong Page"}
                        onClose={handleClose}
                        variant="subheader"
                        sxs={wrongPageDialogTitleStyle}
                    />
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
                            Go to {correctPage}
                        </Button>
                    </Stack>
                </>
            );
        }
        // Otherwise, show the current step
        return (
            <>
                <DialogTitle
                    id={titleId}
                    title={`${currentSection.title} (${place.section + 1} of ${sections.length})`}
                    onClose={handleClose}
                    variant="subheader"
                    // Can only drag dialogs, not popovers
                    sxs={{ root: { cursor: anchorElement ? "auto" : "move" } }}
                />
                <Box p={2}>
                    <MarkdownDisplay
                        variant="body1"
                        content={currentStep.content.text}
                    />
                </Box>
                <MobileStepper
                    variant="dots"
                    steps={currentSection.steps.length}
                    position="static"
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
                    sx={mobileStepperStyle}
                />
            </>
        );
    }, [place.section, place.step, pathname, handleClose, anchorElement, handlePrev, handleNext, isFinalStep, isFinalStepInSection, setLocation]);

    useEffect(function autoAdvanceOnCorrectNavigationEffect() {
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


    // If there's an anchor, use a popper
    if (anchorElement) {
        return (
            <PopoverWithArrow
                anchorEl={anchorElement}
                disableScrollLock={true}
                sxs={popoverWithArrowStyle}
            >
                {content}
            </PopoverWithArrow>
        );
    }
    // Otherwise, use a dialog
    return (
        <Dialog
            open={isOpen}
            scroll="paper"
            disableScrollLock={true}
            aria-labelledby={titleId}
            PaperComponent={DraggableDialogPaper}
            sx={{
                zIndex,
                pointerEvents: "none",
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex,
                        pointerEvents: "auto",
                        borderRadius: 2,
                        margin: 2,
                        maxWidth: "500px",
                        background: palette.background.paper,
                        position: "absolute",
                        top: "0",
                        left: "0",
                    },
                },
                "& .MuiBackdrop-root": {
                    display: "none",
                },
            }}
        >
            {content}
        </Dialog>
    );
}
