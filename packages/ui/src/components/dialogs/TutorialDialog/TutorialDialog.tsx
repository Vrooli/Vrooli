import { ArrowLeftIcon, ArrowRightIcon, CompleteAllIcon, CompleteIcon } from "@local/shared";
import { Dialog, IconButton, MobileStepper, Typography, useTheme } from "@mui/material";
import { PopoverWithArrow } from "components/dialogs/PopoverWithArrow/PopoverWithArrow";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { TutorialDialogProps } from "../types";

type TutorialStep = {
    text: string;
    page: string;
    element?: string;
}

type TutorialSection = {
    title: string;
    steps: TutorialStep[];
}

// Sections of the tutorial
const sections: TutorialSection[] = [
    {
        title: "Section 1",
        steps: [
            {
                text: "Step 1.1",
                page: "Page 1",
                element: "Element 1.1",
            },
            {
                text: "Step 1.2",
                page: "Page 1",
            },
        ],
    },
    {
        title: "Section 2",
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

export const TutorialDialog = ({
    isOpen,
    onClose,
}: TutorialDialogProps) => {
    const { palette } = useTheme();

    const [step, setStep] = useState(0);
    const [section, setSection] = useState(0);
    const anchorEl = useRef<HTMLElement | null>(null);

    useEffect(() => {
        if (!isOpen) {
            setStep(0);
            setSection(0);
        }
    }, [isOpen]);

    // Get current element to anchor to, if any
    useEffect(() => {
        const elementId = sections[section].steps[step].element;
        if (elementId) {
            anchorEl.current = document.getElementById(elementId);
        } else {
            anchorEl.current = null;
        }
    }, [section, step]);

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

    const content = useMemo(() => (
        <>
            <DialogTitle
                id={titleId}
                title={`${sections[section].title} (${section + 1} of ${sections.length})`}
                onClose={onClose}
                variant="subheader"
            />
            <Typography variant="body1" sx={{ padding: 2, paddingBottom: 0 }}>
                {sections[section].steps[step].text}
            </Typography>
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
    ), [handleNext, handlePrev, isFinalStep, isFinalStepInSection, onClose, section, step]);

    // If there's an anchor, use a popper
    if (anchorEl.current) {
        return (
            <PopoverWithArrow
                anchorEl={anchorEl.current}
                disableScrollLock={true}
                sxs={{
                    paper: {
                        padding: 0,
                    },
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
            aria-labelledby={titleId}
            sx={{
                zIndex: 100000,
                "& > .MuiDialog-container": {
                    "& > .MuiPaper-root": {
                        zIndex: 100000,
                        borderRadius: 2,
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
