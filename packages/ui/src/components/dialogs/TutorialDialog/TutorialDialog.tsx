import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon, LINKS } from "@local/shared";
import { Box, Dialog, IconButton, MobileStepper, Paper, useTheme } from "@mui/material";
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
    page: LINKS;
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
                text: "The first thing you'll see is a search bar. Use this to filter the home page, or to enter quick commands.",
                page: LINKS.Home,
                element: "main-search",
            },
        ],
    },
    {
        title: "The Side Menu",
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
        title: "Searching Objects",
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
        console.log("user", user);
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
            setStep(step + 1);
        } else if (!isFinalSection) {
            setSection(section + 1);
            setStep(0);
        } else {
            onClose();
        }
    }, [isFinalSection, isFinalStepInSection, onClose, section, step]);

    const handlePrev = useCallback(() => {
        if (!isFirstStepInSection) {
            setStep(step - 1);
        } else if (!isFirstSection) {
            setSection(section - 1);
            setStep(sections[section - 1].steps.length - 1);
        }
    }, [isFirstSection, isFirstStepInSection, section, step]);

    const getCurrentElement = useCallback(() => {
        const elementId = sections[section].steps[step].element;
        return elementId ? document.getElementById(elementId) : null;
    }, [section, step]);

    const content = useMemo(() => (
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
    ), [getCurrentElement, handleNext, handlePrev, isFinalStep, isFinalStepInSection, onClose, section, step]);

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
