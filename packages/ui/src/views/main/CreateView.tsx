import { LINKS, RoutineType, TranslationKeyCommon } from "@local/shared";
import { Box, Dialog, DialogProps, IconButton, Stack, Typography, styled } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../components/Page/Page.js";
import { CardGrid } from "../../components/lists/CardGrid/CardGrid.js";
import { TIDCard } from "../../components/lists/TIDCard/TIDCard.js";
import { Navbar } from "../../components/navigation/Navbar.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { IconCommon, IconInfo } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { ViewProps } from "../../types.js";
import { ELEMENT_IDS, Z_INDEX } from "../../utils/consts.js";
import { CreateType, getCookie, setCookie } from "../../utils/localStorage.js";
import { RoutineTypeOption, routineTypes } from "../../utils/search/schemas/resource.js";

type CreateInfo = {
    objectType: CreateType;
    description: TranslationKeyCommon,
    iconInfo: IconInfo,
    id: string,
    incomplete?: boolean;
}

export const createCards: CreateInfo[] = [
    {
        objectType: "Reminder",
        description: "CreateReminderDescription",
        iconInfo: { name: "Reminder", type: "Common" },
        id: "create-reminder-card",
    },
    {
        objectType: "Note",
        description: "CreateNoteDescription",
        iconInfo: { name: "Note", type: "Common" },
        id: "create-note-card",
    },
    {
        objectType: "Routine",
        description: "CreateRoutineDescription",
        iconInfo: { name: "Routine", type: "Routine" },
        id: "create-routine-card",
    },
    {
        objectType: "Project",
        description: "CreateProjectDescription",
        iconInfo: { name: "Project", type: "Common" },
        id: "create-project-card",
    },
    {
        objectType: "Team",
        description: "CreateTeamDescription",
        iconInfo: { name: "Team", type: "Common" },
        id: "create-team-card",
    },
    {
        objectType: "Bot",
        description: "CreateBotDescription",
        iconInfo: { name: "Bot", type: "Common" },
        id: "create-bot-card",
    },
    {
        objectType: "Chat",
        description: "CreateChatDescription",
        iconInfo: { name: "Comment", type: "Common" },
        id: "create-chat-card",
    },
    {
        objectType: "Prompt",
        description: "CreatePromptDescription",
        iconInfo: { name: "Article", type: "Common" },
        id: "create-prompt-card",
    },
    {
        objectType: "DataStructure",
        description: "CreateDataStructureDescription",
        iconInfo: { name: "Object", type: "Common" },
        id: "create-data-structure-card",
    },
    {
        objectType: "DataConverter",
        description: "CreateDataConverterDescription",
        iconInfo: { name: "Terminal", type: "Common" },
        id: "create-code-card",
    },
    // {
    //     objectType: "SmartContract",
    //     description: "CreateSmartContractDescription",
    //     iconInfo: { name: "SmartContract", type: "Common" },
    //     id: "create-smart-contract-card",
    //     incomplete: true,
    // },
    // {
    //     objectType: "Api",
    //     description: "CreateApiDescription",
    //     iconInfo: { name: "Api", type: "Common" },
    //     id: "create-api-card",
    //     incomplete: true,
    // },
] as const;

const singleStepRoutineIconInfo: IconInfo = { name: "Action", type: "Common" };
const multiStepRoutineIconInfo: IconInfo = { name: "Routine", type: "Routine" };

function sortByUsageHistory<T>(
    items: T[],
    getId: (item: T) => string,
    historyKey: "CreateOrder" | "SingleStepRoutineOrder",
): void {
    const history = getCookie(historyKey);
    items.sort((a, b) => {
        const aIndex = history.indexOf(getId(a));
        const bIndex = history.indexOf(getId(b));

        // If both are not in history, maintain their current order
        if (aIndex === -1 && bIndex === -1) return 0;
        if (aIndex === -1) return 1;
        if (bIndex === -1) return -1;
        return aIndex - bIndex;
    });
}

function updateUsageHistory(
    id: string,
    historyKey: "CreateOrder" | "SingleStepRoutineOrder",
): void {
    const history = getCookie(historyKey);
    const index = history.indexOf(id);
    if (index !== -1) {
        history.splice(index, 1);
    }
    history.unshift(id);
    setCookie(historyKey, history);
}

const INITIAL_WIZARD_STEP = "complexity";

interface StyledDialogProps extends Omit<DialogProps, "zIndex"> {
    zIndex: number;
}
const StyledDialog = styled(Dialog, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<StyledDialogProps>(({ theme, zIndex }) => ({
    zIndex,
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            zIndex,
            borderRadius: 4,
            background: theme.palette.background.default,
        },
    },
}));

type RoutineWizardDialogProps = {
    isOpen: boolean,
    onClose: () => unknown,
}

function RoutineWizardDialog({
    isOpen,
    onClose,
}: RoutineWizardDialogProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const [step, setStep] = useState<"complexity" | "singleStepType">(INITIAL_WIZARD_STEP);

    function handleClose() {
        onClose();
        setStep(INITIAL_WIZARD_STEP);
    }

    function selectComplexity() {
        setStep("complexity");
    }
    function selectSingleStep() {
        setStep("singleStepType");
    }
    function selectMultiStep() {
        setLocation(`${LINKS.RoutineMultiStep}/add`);
        handleClose();
    }

    function handleSelectSingleStepType(type?: RoutineTypeOption) {
        let url = `${LINKS.RoutineSingleStep}/add`;
        if (type) {
            url += `?type=${type.type}`;
        }
        setLocation(url);
        handleClose();
    }

    const singleStepRoutines = useMemo(() => {
        const result = routineTypes.filter(rt => rt.type !== RoutineType.MultiStep);
        sortByUsageHistory(result, (rt) => rt.type, "SingleStepRoutineOrder");
        return result;
    }, []);

    return (
        <StyledDialog
            id={ELEMENT_IDS.RoutineWizardDialog}
            open={isOpen}
            onClose={handleClose}
            zIndex={Z_INDEX.Dialog}
        >
            <TopBar
                display="Dialog"
                onClose={handleClose}
            />
            {step === "complexity" && (
                <Box p={2}>
                    <Typography variant="h5" mb={4}>What kind of routine would you like to create?</Typography>
                    <Stack direction="column" spacing={2}>
                        <TIDCard
                            buttonText={t("Select")}
                            description={"Ideal for quick, one-off tasks or simple transformations."}
                            iconInfo={singleStepRoutineIconInfo}
                            key="singleStep"
                            onClick={selectSingleStep}
                            size="small"
                            title={"Single-Step (Simple)"}
                        />
                        <TIDCard
                            buttonText={t("Select")}
                            description={"Use a graph of steps for complex, automated workflows."}
                            iconInfo={multiStepRoutineIconInfo}
                            key="multiStep"
                            onClick={selectMultiStep}
                            size="small"
                            title={"Multi-Step (Complex)"}
                        />
                    </Stack>
                </Box>
            )}
            {step === "singleStepType" && (
                <Box p={2}>
                    <Box display="flex" alignItems="center" mb={4}>
                        <IconButton onClick={selectComplexity}>
                            <IconCommon
                                decorative
                                name="ArrowLeft"
                            />
                        </IconButton>
                        <Typography variant="h5" ml={1}>Select a Single-Step Routine Type</Typography>
                    </Box>
                    <Stack spacing={2}>
                        {singleStepRoutines.map(typeOption => {
                            function handleClick() {
                                updateUsageHistory(typeOption.type, "SingleStepRoutineOrder");
                                handleSelectSingleStepType(typeOption);
                            }

                            return (
                                <TIDCard
                                    buttonText={t("Select")}
                                    description={typeOption.description}
                                    iconInfo={typeOption.iconInfo}
                                    id={typeOption.type}
                                    key={typeOption.type}
                                    onClick={handleClick}
                                    size="small"
                                    title={typeOption.label}
                                />
                            );
                        })}
                    </Stack>
                </Box>
            )}
        </StyledDialog>
    );
}

export function CreateView(_props: ViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const [showRoutineWizard, setShowRoutineWizard] = useState(false);
    const closeRoutineWizard = useCallback(() => setShowRoutineWizard(false), []);

    const sortedCards = useMemo(() => {
        const cardsCopy = [...createCards];
        sortByUsageHistory(cardsCopy, (card) => card.id, "CreateOrder");
        return cardsCopy;
    }, []);

    const onSelect = useCallback((id: typeof createCards[number]["id"]) => {
        const { objectType } = createCards.find(card => card.id === id) ?? {};
        if (!objectType) return;

        updateUsageHistory(id, "CreateOrder");

        // Handle routines differently
        if (objectType === "Routine") {
            setShowRoutineWizard(true);
        } else {
            setLocation(`${LINKS[objectType === "Bot" ? "User" : objectType]}/add`);
        }
    }, [setLocation]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <Navbar title={t("Create")} />
                <RoutineWizardDialog isOpen={showRoutineWizard} onClose={closeRoutineWizard} />
                <CardGrid minWidth={300}>
                    {sortedCards.map(({ objectType, description, iconInfo, id, incomplete }, index) => {
                        function handleClick() {
                            onSelect(id);
                        }

                        return (
                            <TIDCard
                                buttonText={t("Create")}
                                description={t(description)}
                                iconInfo={iconInfo}
                                id={id}
                                key={index}
                                onClick={handleClick}
                                title={t(objectType, { count: 1 })}
                                warning={incomplete ? "Coming soon" : undefined}
                            />
                        );
                    })}
                </CardGrid>
            </ScrollBox>
        </PageContainer>
    );
}
