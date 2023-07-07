import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon, LINKS, useLocation } from "@local/shared";
import { Box, Button, Dialog, IconButton, MobileStepper, Paper, Stack, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import Draggable from "react-draggable";
import { getCurrentUser } from "utils/authentication/session";
import { SessionContext } from "utils/SessionContext";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { TutorialDialogProps } from "../types";

type TutorialStep = {
    text: string;
    page?: LINKS;
    element?: string;
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
                text: "This tutorial will show you how to use Vrooli to assist your personal and professional life.\n\nIt will only take a few minutes, and you can skip it at any time.",
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
            },
            {
                text: "The first section lists all logged-in accounts.\n\nPress on an account to switch to it, or press on your current account to open your profile.",
            },
            {
                text: "The second section allows you to control your display settings. This includes:\n\n- **Theme**: Choose between light and dark mode.\n- **Text size**: Grow or shrink the text on all pages.\n- **Left handed?**: Move various elements, such as the side menu, to the left side of the screen.\n- **Language**: Change the language of the app.\n- **Focus mode**: Switch between focus modes.",
            },
            {
                text: "The third section displays a list of pages not listed in the main navigation bar.",
            },
        ],
    },
    {
        title: "Searching Objects",
        steps: [
            {
                text: "Step 2.1",
                page: LINKS.Search,
            },
            {
                text: "Step 2.2",
                page: "Page 2",
            },
        ],
    },
    {
        title: "Creating Objects",
        steps: [
            {
                text: "Step 2.1",
                page: "Page 2",
                element: "create-project-card",
            },
            {
                text: "Step 2.2",
                page: "Page 2",
            },
        ],
    },
    {
        title: "Messages and Notifications",
        steps: [
            {
                text: "Step 2.1",
                page: "Page 2",
                element: "create-project-card",
            },
            {
                text: "Step 2.2",
                page: "Page 2",
            },
        ],
    },
    {
        title: "Your Stuff",
        steps: [
            {
                text: "Step 2.1",
                page: "Page 2",
                element: "create-project-card",
            },
            {
                text: "Step 2.2",
                page: "Page 2",
            },
        ],
    },
    {
        title: "That's it!",
        steps: [
            {
                text: "Step 2.1",
                page: "Page 2",
                element: "create-project-card",
            },
            {
                text: "Step 2.2",
                page: "Page 2",
            },
        ],
    },
];

const titleId = "tutorial-dialog-title";

/** Draggable paper for dialog */
const PaperComponent = (props) => {
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
    const [location, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const user = useMemo(() => getCurrentUser(session), [session]);

    const [step, setStep] = useState(0);
    const [section, setSection] = useState(0);

    // Reset when dialog is closed
    useEffect(() => {
        if (!isOpen) {
            setStep(0);
            setSection(0);
        }
    }, [isOpen]);

    // Close if user logs out
    useEffect(() => {
        if (!user.id) {
            onClose();
        }
    }, [onClose, user]);

    const {
        isFinalStep,
        isFinalSection,
        isFinalStepInSection,
        isFirstSection,
        isFirstStepInSection,
    } = useMemo(() => {
        const isFinalStepInSection = step === sections[section].steps.length - 1;
        const isFinalSection = section === sections.length - 1;
        const isFinalStep = isFinalStepInSection && isFinalSection;

        const isFirstStepInSection = step === 0;
        const isFirstSection = section === 0;

        return { isFinalStep, isFinalSection, isFinalStepInSection, isFirstSection, isFirstStepInSection };
    }, [section, step]);

    const handleNext = useCallback(() => {
        if (!isFinalStepInSection) {
            const nextPage = sections[section].steps[step + 1].page;
            if (nextPage && nextPage !== location) {
                setLocation(nextPage);
            }
            setStep(step + 1);
        } else if (!isFinalSection) {
            const nextPage = sections[section + 1].steps[0].page;
            if (nextPage && nextPage !== location) {
                setLocation(nextPage);
            }
            setSection(section + 1);
            setStep(0);
        } else {
            onClose();
        }
    }, [isFinalSection, isFinalStepInSection, location, onClose, section, setLocation, step]);

    const handlePrev = useCallback(() => {
        if (!isFirstStepInSection) {
            const prevPage = sections[section].steps[step - 1].page;
            if (prevPage && prevPage !== location) {
                setLocation(prevPage);
            }
            setStep(step - 1);
        } else if (!isFirstSection) {
            const prevPage = sections[section - 1].steps[sections[section - 1].steps.length - 1].page;
            if (prevPage && prevPage !== location) {
                setLocation(prevPage);
            }
            setSection(section - 1);
            setStep(sections[section - 1].steps.length - 1);
        }
    }, [isFirstSection, isFirstStepInSection, location, section, setLocation, step]);

    const getCurrentElement = useCallback(() => {
        const elementId = sections[section].steps[step].element;
        return elementId ? document.getElementById(elementId) : null;
    }, [section, step]);

    const content = useMemo(() => {
        // Guide user to correct page if they're on the wrong one
        const correctPage = sections[section].steps[step].page;
        if (correctPage && correctPage !== location) {
            return (
                <>
                    <DialogTitle
                        id={titleId}
                        title={"Wrong Page"}
                        onClose={onClose}
                        variant="subheader"
                    />
                    <Stack direction="column" spacing={2} p={2}>
                        <MarkdownDisplay variant="body1" content={"Please return to the correct page to continue the tutorial."} />
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
                    title={`${sections[section].title} (${section + 1} of ${sections.length})`}
                    onClose={onClose}
                    variant="subheader"
                    sxs={{
                        // Can move dialog, but not popper
                        root: { cursor: getCurrentElement() ? "auto" : "move" },
                    }}
                />
                <Box sx={{ padding: "16px", paddingBottom: 0 }}>
                    <MarkdownDisplay variant="body1" content={sections[section].steps[step].text} />
                </Box>
                <MobileStepper
                    variant="dots"
                    steps={sections[section].steps.length}
                    position="static"
                    activeStep={step}
                    backButton={
                        <IconButton
                            onClick={handlePrev}
                            disabled={section === 0 && step === 0}
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
            </>
        );
    }, [getCurrentElement, handleNext, handlePrev, isFinalStep, isFinalStepInSection, location, onClose, section, setLocation, step]);

    // If there's an anchor, use a popper
    if (getCurrentElement()) {
        return (
            <PopoverWithArrow
                anchorEl={getCurrentElement()}
                disableScrollLock={true}
                sxs={{
                    root: { zIndex: 100000 },
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
            PaperComponent={PaperComponent}
            sx={{
                zIndex: 100000,
                pointerEvents: "none",
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex: 100000,
                        pointerEvents: "auto",
                        borderRadius: 2,
                        margin: 2,
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
