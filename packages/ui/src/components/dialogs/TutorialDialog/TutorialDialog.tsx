import { LINKS } from "@local/shared";
import { Box, Button, Dialog, IconButton, MobileStepper, Paper, PaperProps, Stack, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { SessionContext } from "contexts/SessionContext";
import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import Draggable from "react-draggable";
import { useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { TutorialDialogProps } from "../types";

type TutorialStep = {
    text: string;
    page?: LINKS;
    element?: string;
    action?: () => void;
}

type TutorialSection = {
    title: string;
    steps: TutorialStep[];
}

// Sections of the tutorial
const sections: TutorialSection[] = [
    {
        title: "Welcome to Vrooli!",
        steps: [
            {
                text: "This tutorial will show you how to use Vrooli to assist your personal and professional life.\n\nIt will only take a few minutes, and you can skip it at any time.\n\nTo open the tutorial again, type **tutorial** in the Home page's search bar.",
                page: LINKS.Home,
            },
        ],
    },
    {
        title: "The Home Page",
        steps: [
            {
                text: "The home page shows the most important information for you.",
                page: LINKS.Home,
            },
            {
                text: "The first thing you'll see are the focus mode tabs. These allow you to customize the resources and reminders shown on the home page.\n\nBy default, these are **Work** and **Study**. These can be customized in the settings page.\n\nMore focus mode-related features will be released in the future.",
                page: LINKS.Home,
                element: "home-tabs",
            },
            {
                text: "Below the focus mode tabs is a search bar. Use this to filter the home page, or to enter quick commands.",
                page: LINKS.Home,
                element: "main-search",
            },
            {
                text: "Next is a customizable list of resources.\n\nThese can be anything you want, such as links to your favorite websites, or objects on Vrooli.",
                page: LINKS.Home,
                element: "main-resource-list",
            },
            {
                text: "Next is a list of upcoming events. These can be for meetings, focus mode sessions, or scheduled tasks (more on this later).\n\nOpening the schedule will show you a calendar view of your events.",
                page: LINKS.Home,
                element: "main-event-list",
            },
            {
                text: "Then there's a list of reminders.\n\nThese are associated with the current focus mode. Press **See All** to view all reminders, regardless of focus mode.",
                page: LINKS.Home,
                element: "main-reminder-list",
            },
            {
                text: "Finally, there's a list of notes. Use these to jot down quick thoughts, store information, or anything else you want.\n\nPress **See All** to view all notes.",
                page: LINKS.Home,
                element: "main-note-list",
            },
        ],
    },
    {
        title: "The Side Menu",
        steps: [
            {
                text: "The side menu has many useful features. Open it by pressing on your profile picture.",
                element: "side-menu-profile-icon",
                action: () => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: false }); },
            },
            {
                text: "The first section lists all logged-in accounts.\n\nPress on an account to switch to it, or press on your current account to open your profile.",
                action: () => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: true }); },
            },
            {
                text: "The second section allows you to control your display settings. This includes:\n\n- **Theme**: Choose between light and dark mode.\n- **Text size**: Grow or shrink the text on all pages.\n- **Left handed?**: Move various elements, such as the side menu, to the left side of the screen.\n- **Language**: Change the language of the app.\n- **Focus mode**: Switch between focus modes.",
                action: () => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: true }); },
            },
            {
                text: "The third section displays a list of pages not listed in the main navigation bar.",
                action: () => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: true }); },
            },
        ],
    },
    {
        title: "Searching Objects",
        steps: [
            {
                text: "This page allows you to search public objects on Vrooli.",
                page: LINKS.Search,
                action: () => { PubSub.get().publishSideMenu({ id: "side-menu", isOpen: false }); },
            },
            {
                text: "Use these tabs to switch between different types of objects.\n\nThe first, default tab is for **routines**. These allow you to complete and automate various tasks.\n\nLet's look at some routines now.",
                page: LINKS.Search,
                element: "search-tabs",
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
                text: "This page allows you to create new objects on Vrooli.",
                page: LINKS.Create,
            },
        ],
    },
    {
        title: "Your Stuff",
        steps: [
            {
                text: "After you've created an object, you can find it here.",
                page: LINKS.MyStuff,
            },
            {
                text: "Just like the search page, you can use these tabs to switch between different types of objects.",
                page: LINKS.MyStuff,
                element: "my-stuff-tabs",
            },
        ],
    },
    {
        title: "Inbox",
        steps: [
            {
                text: "This page allows you to view your messages and notifications.\n\nIf you have a premium account, you can message bots and have them run tasks and perform other actions for you.",
                page: LINKS.Inbox,
            },
        ],
    },
    {
        title: "That's it!",
        steps: [
            {
                text: "Now you know the basics of Vrooli. Have fun!",
            },
        ],
    },
];

const titleId = "tutorial-dialog-title";
const zIndex = 100000;

const DraggableDialogPaper = (props: PaperProps) => {
    return (
        <Draggable
            handle={`#${titleId}`}
            cancel={"[class*=\"MuiDialogContent-root\"]"}
        >
            <Paper {...props} />
        </Draggable>
    );
};

export const TutorialDialog = ({
    isOpen,
    onClose,
}: TutorialDialogProps) => {
    const { palette } = useTheme();
    const [{ pathname }, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);

    const [place, setPlace] = useState({ section: 0, step: 0 });

    // Reset when dialog is closed
    useEffect(() => { !isOpen && setPlace({ section: 0, step: 0 }); }, [isOpen]);

    // Close if user logs out
    useEffect(() => { !user.id && onClose(); }, [onClose, user]);

    // Call action if it exists
    useEffect(() => { isOpen && sections[place.section]?.steps[place.step]?.action?.(); }, [isOpen, place]);

    // Find information about our position in the tutorial
    const {
        isFinalStep,
        isFinalStepInSection,
        nextStep,
    } = useMemo(() => {
        const section = sections[place.section];
        const nextSection = sections[place.section + 1];

        // If no current section is found, or the current step is invalid, default to initial values
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
                ? nextSection ? nextSection.steps[0] : null
                : section.steps[place.step + 1];

        return { isFinalStep, isFinalStepInSection, nextStep };
    }, [place]);

    /** Move to the next step */
    const handleNext = useCallback(() => {
        const currentSection = sections[place.section];
        const nextSection = sections[place.section + 1];

        // ensure the current section and next step exists
        if (currentSection && currentSection.steps[place.step + 1]) {
            const nextPage = currentSection.steps[place.step + 1].page;
            if (nextPage && nextPage !== pathname) {
                setLocation(nextPage);
            }
            setPlace({ section: place.section, step: place.step + 1 });
        } else if (nextSection && nextSection.steps[0]) {
            const nextPage = nextSection.steps[0].page;
            if (nextPage && nextPage !== pathname) {
                setLocation(nextPage);
            }
            setPlace({ section: place.section + 1, step: 0 });
        } else {
            onClose();
        }
    }, [pathname, onClose, place, setLocation]);

    /** Move to the previous step */
    const handlePrev = useCallback(() => {
        const currentSection = sections[place.section];
        const previousSection = sections[place.section - 1];

        // ensure the current section and previous step exists
        if (currentSection && currentSection.steps[place.step - 1]) {
            const prevPage = currentSection.steps[place.step - 1].page;
            if (prevPage && prevPage !== pathname) {
                setLocation(prevPage);
            }
            setPlace({ section: place.section, step: place.step - 1 });
        } else if (previousSection) {
            const prevStepIndex = previousSection.steps.length - 1;
            if (previousSection.steps[prevStepIndex]) {
                const prevPage = previousSection.steps[prevStepIndex].page;
                if (prevPage && prevPage !== pathname) {
                    setLocation(prevPage);
                }
                setPlace({ section: place.section - 1, step: prevStepIndex });
            }
        }
    }, [pathname, place, setLocation]);

    const getCurrentElement = useCallback(() => {
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            return null;
        }
        const elementId = currentSection.steps[place.step].element;
        return elementId ? document.getElementById(elementId) : null;
    }, [place]);

    const content = useMemo(() => {
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            PubSub.get().publishSnack({ message: "Failed to load tutorial step", severity: "Error" });
            return null;
        }
        const currentStep = currentSection.steps[place.step];

        // Guide user to correct page if they're on the wrong one
        const correctPage = currentStep.page;
        if (correctPage && correctPage !== pathname) {
            return (
                <>
                    <DialogTitle
                        id={titleId}
                        title={"Wrong Page"}
                        onClose={onClose}
                        variant="subheader"
                        sxs={{ root: { cursor: "move" } }}
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
                            onClick={() => setLocation(correctPage)}
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
                    onClose={onClose}
                    variant="subheader"
                    // Can only drag dialogs, not popovers
                    sxs={{ root: { cursor: getCurrentElement() ? "auto" : "move" } }}
                />
                <Box sx={{ padding: "16px" }}>
                    <MarkdownDisplay
                        variant="body1"
                        content={currentStep.text}
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
                    sx={{ background: "transparent" }}
                />
            </>
        );
    }, [getCurrentElement, handleNext, handlePrev, isFinalStep, isFinalStepInSection, pathname, onClose, place, setLocation]);

    // If the user navigates to the page for the next step, automatically advance. 
    // This is temporarily disabled after the previous/next buttons are pressed.
    useEffect(() => {
        const currentSection = sections[place.section];
        if (!currentSection || !currentSection.steps[place.step]) {
            PubSub.get().publishSnack({ message: "Failed to load tutorial", severity: "Error" });
            return;
        }
        const currentStep = currentSection.steps[place.step];
        // Find current step's page
        const currPage = currentStep.page;
        // If already on the correct page, return
        if (currPage && currPage === pathname) return;

        // Find next step's page
        const nextPage = nextStep?.page;

        // If next step has a page and it's the current page, advance
        if (currPage && nextPage && nextPage === pathname) {
            console.log("handling next", currPage, nextPage);
            handleNext();
        }
    }, [handleNext, pathname, nextStep?.page, place, setLocation]);


    // If there's an anchor, use a popper
    if (getCurrentElement()) {
        return (
            <PopoverWithArrow
                anchorEl={getCurrentElement()}
                disableScrollLock={true}
                sxs={{
                    root: {
                        zIndex,
                        maxWidth: "500px",
                    },
                    content: { padding: 0 },
                }}
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
};
