import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import LinearProgress from "@mui/material/LinearProgress";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import ListSubheader from "@mui/material/ListSubheader";
import Menu from "@mui/material/Menu";
import MenuItem from "@mui/material/MenuItem";
import MobileStepper from "@mui/material/MobileStepper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material";
import { IconButton } from "../buttons/IconButton.js";
import { LINKS, SEEDED_PUBLIC_IDS, SearchPageTabOption, UrlTools, getObjectUrl, parseSearchParams, type TutorialViewSearchParams } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { FormRunView } from "../../forms/FormView/FormRunView.js";
import { useHotkeys } from "../../hooks/useHotkeys.js";
import { useIsMobile } from "../../hooks/useIsMobile.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { addSearchParams, removeSearchParams } from "../../route/searchParams.js";
import { ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { TUTORIAL_HIGHLIGHT, addHighlight, removeHighlights } from "../../utils/display/documentTools.js";
import { PubSub, type MenuPayloads } from "../../utils/pubsub.js";
import { routineTypes } from "../../utils/search/schemas/resource.js";
import { Dialog, DialogContent, DialogActions } from "./Dialog/Dialog.js";
import { tutorialSections } from "./TutorialDialog/tutorialSections.js";
import type { Place, TutorialDialogProps, TutorialSection, TutorialStep } from "./TutorialDialog/types.js";

// Use sections from external file
const sections: TutorialSection[] = tutorialSections;

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
    if (!place || typeof place.section !== "number" || typeof place.step !== "number") return false;
    const section = sections[place.section];
    if (!section) return false;
    const step = section.steps[place.step];
    return !!step;
}

export function getNextPlace(
    sections: TutorialSection[],
    place: Place,
): Place | null {
    // Check if place object exists and has valid properties
    if (!place || typeof place.section !== "number" || typeof place.step !== "number") {
        return { section: 0, step: 0 };
    }

    const section = sections[place.section];
    if (!section) return null;

    const step = section.steps[place.step];
    if (!step) {
        // If step doesn't exist and we're beyond the section, move to next section
        if (place.step >= section.steps.length) {
            // If this is the last section
            if (place.section === sections.length - 1) {
                return null; // No next place
            }
            // Move to the first step of the next section
            return { section: place.section + 1, step: 0 };
        }
        return null;
    }

    // Check if there's a custom next place
    if (step.next) {
        return step.next;
    }

    // If this is the last step in the section
    if (place.step === section.steps.length - 1) {
        // If this is the last section
        if (place.section === sections.length - 1) {
            return null; // No next place
        }
        // Move to the first step of the next section
        return { section: place.section + 1, step: 0 };
    }

    // Move to the next step in the same section
    return { section: place.section, step: place.step + 1 };
}

export function getPrevPlace(
    sections: TutorialSection[],
    place: Place,
): Place | null {
    // Check if place object exists and has valid properties
    if (!place || typeof place.section !== "number" || typeof place.step !== "number") {
        return null;
    }

    const section = sections[place.section];
    if (!section) return null;

    const step = section.steps[place.step];
    if (!step) {
        // If step doesn't exist and we're before the section, move to prev section
        if (place.step < 0) {
            // If this is the first section
            if (place.section === 0) {
                return null; // No previous place
            }
            // Move to the last step of the previous section
            const prevSection = sections[place.section - 1];
            return { section: place.section - 1, step: prevSection.steps.length - 1 };
        }
        return null;
    }

    // Check if there's a custom previous place
    if (step.previous) {
        return step.previous;
    }

    // If this is the first step in the section
    if (place.step === 0) {
        // If this is the first section
        if (place.section === 0) {
            return null; // No previous place
        }
        // Move to the last step of the previous section
        const prevSection = sections[place.section - 1];
        return { section: place.section - 1, step: prevSection.steps.length - 1 };
    }

    // Move to the previous step in the same section
    return { section: place.section, step: place.step - 1 };
}

export function getCurrentElement(sections: TutorialSection[], place: Place): HTMLElement | null {
    const section = sections[place.section];
    if (!section) return null;
    const step = section.steps[place.step];
    if (!step?.location?.element) return null;
    return document.getElementById(step.location.element);
}

export function getCurrentStep(sections: TutorialSection[], place: Place): TutorialStep | null {
    const section = sections[place.section];
    if (!section) return null;
    const step = section.steps[place.step];
    return step || null;
}

const StyledStepper = styled(MobileStepper)(({ theme }) => ({
    background: "transparent",
    padding: theme.spacing(0.5, 1),
    "& .MuiMobileStepper-dots": {
        display: "flex",
        gap: theme.spacing(0.5),
    },
    "& .MuiMobileStepper-dot": {
        width: 14,
        height: 14,
        margin: 0,
        backgroundColor: theme.palette.action.disabled,
        cursor: "pointer",
        transition: "all 0.2s ease",
        "&:hover": {
            backgroundColor: theme.palette.primary.light,
            transform: "scale(1.1)",
        },
    },
    "& .MuiMobileStepper-dotActive": {
        backgroundColor: theme.palette.primary.main,
        width: 18,
        height: 18,
        "&:hover": {
            backgroundColor: theme.palette.primary.dark,
        },
    },
}));


const StyledLinearProgress = styled(LinearProgress)(({ theme }) => ({
    height: 6,
    borderRadius: 3,
    backgroundColor: theme.palette.action.hover,
    "& .MuiLinearProgress-bar": {
        borderRadius: 3,
        backgroundColor: theme.palette.secondary.main,
    },
}));

const StyledMenuItem = styled(MenuItem)(({ theme }) => ({
    position: "relative",
    paddingLeft: theme.spacing(6),
    "&::before": {
        content: "''",
        position: "absolute",
        left: theme.spacing(3),
        top: "50%",
        width: 2,
        height: "100%",
        backgroundColor: theme.palette.divider,
        transform: "translateY(-50%)",
        zIndex: 1,
    },
    "&:first-of-type::before": {
        height: "50%",
        top: "50%",
    },
    "&:last-of-type::before": {
        height: "50%",
        top: 0,
    },
    "&:only-of-type::before": {
        display: "none",
    },
}));

const StyledListSubheader = styled(ListSubheader)(({ theme }) => ({
    backgroundColor: theme.palette.background.paper,
    borderBottom: `1px solid ${theme.palette.divider}`,
    fontWeight: theme.typography.fontWeightMedium,
}));

const SectionNumber = styled(Box, {
    shouldForwardProp: (prop) => prop !== "completed",
})<{ completed: boolean }>(({ theme, completed }) => ({
    position: "absolute",
    left: theme.spacing(1.5),
    top: "50%",
    transform: "translateY(-50%)",
    width: 24,
    height: 24,
    borderRadius: "50%",
    backgroundColor: completed ? theme.palette.success.main : theme.palette.primary.main,
    color: theme.palette.primary.contrastText,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: "0.75rem",
    fontWeight: theme.typography.fontWeightBold,
    zIndex: 2,
}));

const StyledDialog = styled(Dialog)(({ theme }) => ({
    "& .MuiDialog-paper": {
        minWidth: "min(90vw, 600px)",
        maxWidth: "min(95vw, 800px)",
        minHeight: "min(90vh, 500px)",
        maxHeight: "95vh",
        margin: theme.spacing(1),
    },
    "& .MuiBackdrop-root": {
        display: "none",
    },
}));

export function TutorialDialog(props: TutorialDialogProps = {}) {
    const { bypassPageValidation = false } = props;

    const { t } = useTranslation();
    const { palette } = useTheme();
    const [location, setLocation] = useLocation();

    const [anchorElement, setAnchorElement] = useState<HTMLElement | null>(null);
    const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null);
    
    const openMenu = useCallback((event: React.MouseEvent<HTMLElement>) => {
        setMenuAnchor(event.currentTarget);
    }, []);
    
    const closeMenu = useCallback(() => {
        setMenuAnchor(null);
    }, []);

    const isMobile = useIsMobile();

    // Parse tutorial step from URL
    const parsedParams = useMemo(() => {
        const parsed = parseSearchParams(location.search, ["TutorialView"]);
        return parsed?.TutorialView as TutorialViewSearchParams | undefined;
    }, [location.search]);

    const place = useMemo<Place>(() => {
        if (!parsedParams) return { section: 0, step: 0 };
        const section = Math.max(0, parsedParams.section ?? 0);
        const step = Math.max(0, parsedParams.step ?? 0);
        return { section, step };
    }, [parsedParams]);

    const isOpen = useMemo(() => {
        if (!parsedParams) return false;
        // In Storybook, bypass page validation if prop is set
        if (bypassPageValidation) {
            return parsedParams.hasOwnProperty("section") || parsedParams.hasOwnProperty("step");
        }
        return parsedParams.hasOwnProperty("section") || parsedParams.hasOwnProperty("step");
    }, [parsedParams, bypassPageValidation]);

    const stepInfo = useMemo(() => getTutorialStepInfo(sections, place), [place]);
    const currentStep = useMemo(() => getCurrentStep(sections, place), [place]);
    const currentElement = useMemo(() => getCurrentElement(sections, place), [place]);

    const handleStepDotClick = useCallback((index: number) => {
        let currentIndex = 0;
        for (let sectionIndex = 0; sectionIndex < sections.length; sectionIndex++) {
            const section = sections[sectionIndex];
            if (currentIndex + section.steps.length > index) {
                const stepIndex = index - currentIndex;
                const newPlace = { section: sectionIndex, step: stepIndex };
                setLocation({
                    search: addSearchParams(location.search, {
                        "TutorialView": {
                            section: newPlace.section,
                            step: newPlace.step,
                        },
                    }),
                });
                return;
            }
            currentIndex += section.steps.length;
        }
    }, [location.search, setLocation]);

    const handleStepperClick = useCallback((direction: "next" | "back") => {
        const newPlace = direction === "next" 
            ? getNextPlace(sections, place) 
            : getPrevPlace(sections, place);
        
        if (newPlace) {
            setLocation({
                search: addSearchParams(location.search, {
                    "TutorialView": {
                        section: newPlace.section,
                        step: newPlace.step,
                    },
                }),
            });
        }
    }, [location.search, place, setLocation]);

    const closeDialog = useCallback(() => {
        setLocation({
            search: removeSearchParams(location.search, ["TutorialView"]),
        });
    }, [location.search, setLocation]);

    const skipToSection = useCallback((sectionIndex: number) => {
        const newPlace = { section: sectionIndex, step: 0 };
        setLocation({
            search: addSearchParams(location.search, {
                "TutorialView": {
                    section: newPlace.section,
                    step: newPlace.step,
                },
            }),
        });
        closeMenu();
    }, [closeMenu, location.search, setLocation]);

    // Handle hotkeys
    useHotkeys([
        {
            keys: ["ArrowRight"],
            callback: () => {
                if (isOpen) {
                    const nextPlace = getNextPlace(sections, place);
                    if (nextPlace) handleStepperClick("next");
                }
            },
        },
        {
            keys: ["ArrowLeft"],
            callback: () => {
                if (isOpen) {
                    const prevPlace = getPrevPlace(sections, place);
                    if (prevPlace) handleStepperClick("back");
                }
            },
        },
        {
            keys: ["Escape"],
            callback: () => {
                if (isOpen) closeDialog();
            },
        },
    ], isOpen);

    // Handle step actions and page navigation
    useEffect(() => {
        if (!isOpen || !currentStep) return;

        // Run step action if it exists
        if (typeof currentStep.action === "function") {
            try {
                currentStep.action();
            } catch (error) {
                console.error("Error running tutorial step action:", error);
            }
        }

        // Handle page navigation
        if (currentStep.location?.page && currentStep.location.page !== location.pathname) {
            setLocation(currentStep.location.page);
        }
    }, [currentStep, isOpen, location.pathname, setLocation]);

    // Handle element highlighting and anchoring
    useEffect(() => {
        if (!isOpen || !currentElement) {
            setAnchorElement(null);
            return;
        }

        const timeoutId = setTimeout(() => {
            const element = document.getElementById(currentElement);
            if (element) {
                addHighlight(element, TUTORIAL_HIGHLIGHT);
                setAnchorElement(element);
            }
        }, 300);

        return () => {
            clearTimeout(timeoutId);
            removeHighlights(TUTORIAL_HIGHLIGHT);
            setAnchorElement(null);
        };
    }, [currentElement, isOpen]);

    // Subscribe to menu events
    useEffect(() => {
        const handleMenuEvents = (data: MenuPayloads["menu"]) => {
            if (data.id === ELEMENT_IDS.Tutorial) {
                if (data.isOpen) {
                    // Open tutorial by setting URL parameters
                    setLocation({
                        search: addSearchParams(location.search, {
                            "TutorialView": {
                                section: 0,
                                step: 0,
                            },
                        }),
                    });
                } else {
                    // Close tutorial by removing URL parameters
                    setLocation({
                        search: removeSearchParams(location.search, ["TutorialView"]),
                    });
                }
            } else if (data.id === ELEMENT_IDS.UserMenu && !data.isOpen) {
                // Handle any cleanup when user menu closes
            }
        };

        const unsubscribe = PubSub.get().subscribe("menu", handleMenuEvents);
        return unsubscribe;
    }, [location.search, setLocation]);

    // Calculate progress and step info
    const totalSteps = useMemo(() => getTotalSteps(sections), []);
    const currentStepIndex = useMemo(() => getCurrentStepIndex(sections, place), [place]);
    const progress = useMemo(() => ((currentStepIndex + 1) / totalSteps) * 100, [currentStepIndex, totalSteps]);


    const sectionMenu = useMemo(() => (
        <Menu
            anchorEl={menuAnchor}
            open={Boolean(menuAnchor)}
            onClose={closeMenu}
            PaperProps={{
                sx: {
                    maxHeight: 400,
                    minWidth: 300,
                },
            }}
        >
            <StyledListSubheader>Jump to Section</StyledListSubheader>
            {sections.map((section, index) => {
                const isCompleted = index < place.section || (index === place.section && stepInfo.isFinalStepInSection);
                return (
                    <StyledMenuItem
                        key={index}
                        onClick={() => skipToSection(index)}
                        selected={index === place.section}
                    >
                        <SectionNumber completed={isCompleted}>
                            {index + 1}
                        </SectionNumber>
                        <ListItemText
                            primary={section.title}
                            secondary={`${section.steps.length} step${section.steps.length !== 1 ? "s" : ""}`}
                        />
                    </StyledMenuItem>
                );
            })}
        </Menu>
    ), [closeMenu, menuAnchor, place.section, skipToSection, stepInfo.isFinalStepInSection]);

    if (!isOpen || !currentStep) {
        return null;
    }

    return (
        <>
            {sectionMenu}
            <Dialog
                isOpen={isOpen}
                onClose={closeDialog}
                title={
                    <Box 
                        onClick={openMenu} 
                        data-no-drag="true"
                        sx={{ 
                            cursor: "pointer",
                            display: "flex",
                            alignItems: "center",
                            gap: 1.5,
                            "&:hover": {
                                opacity: 0.8,
                            },
                        }}
                    >
                        <SectionNumber completed={false}>
                            {place.section + 1}
                        </SectionNumber>
                        {sections[place.section]?.title}
                    </Box>
                }
                size={isMobile ? "full" : "md"}
                draggable={!isMobile}
                anchorEl={anchorElement}
                anchorPlacement="auto"
                highlightAnchor={true}
                closeOnEscape={true}
                closeOnOverlayClick={false}
                showCloseButton={true}
                contentClassName="tw-relative tw-flex tw-flex-col"
            >
                <DialogContent className="tw-flex-1 tw-overflow-auto" style={{ paddingBottom: "100px" }}>
                    <FormRunView 
                        schema={{ elements: currentStep.content }}
                        disabled={true}
                    />
                </DialogContent>

                {/* Custom navigation area outside of DialogActions */}
                <div className="tw-sticky tw-bottom-0 tw-bg-background-paper tw-border-t tw-border-gray-200 dark:tw-border-gray-700">
                    {/* Navigation buttons */}
                    <div className="tw-flex tw-justify-center tw-items-center tw-p-4">
                        <IconButton
                            size="small"
                            onClick={() => handleStepperClick("back")}
                            disabled={place.section === 0 && place.step === 0}
                        >
                            <IconCommon name="ArrowLeft" />
                        </IconButton>
                        
                        <StyledStepper
                            variant="dots"
                            steps={totalSteps}
                            position="static"
                            activeStep={currentStepIndex}
                            nextButton={<div />}
                            backButton={<div />}
                            sx={{
                                "& .MuiMobileStepper-dot": {
                                    cursor: "pointer",
                                },
                            }}
                            onClick={(event) => {
                                const target = event.target as HTMLElement;
                                if (target.classList.contains("MuiMobileStepper-dot")) {
                                    const dots = Array.from(target.parentElement?.children || []);
                                    const index = dots.indexOf(target);
                                    if (index >= 0) {
                                        handleStepDotClick(index);
                                    }
                                }
                            }}
                        />
                        
                        <IconButton
                            size="small"
                            onClick={() => handleStepperClick("next")}
                            disabled={stepInfo.isFinalStep}
                        >
                            <IconCommon name="ArrowRight" />
                        </IconButton>
                    </div>
                    
                    {/* Progress bar - at the very bottom with no padding */}
                    <div className="tw-w-full tw-h-1 tw-bg-gray-200" style={{ marginTop: "-1px" }}>
                        <div 
                            className="tw-h-full tw-bg-secondary tw-transition-all tw-duration-300"
                            style={{ width: `${progress}%` }}
                        />
                    </div>
                </div>
            </Dialog>
        </>
    );
}
