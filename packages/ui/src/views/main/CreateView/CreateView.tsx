import { LINKS, RoutineType, TranslationKeyCommon } from "@local/shared";
import { Box, Dialog, DialogProps, IconButton, Stack, Typography, styled } from "@mui/material";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar.js";
import { useZIndex } from "hooks/useZIndex.js";
import { ApiIcon, ArrowLeftIcon, ArticleIcon, BotIcon, CommentIcon, HelpIcon, NoteIcon, ObjectIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, TeamIcon, TerminalIcon } from "icons";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ScrollBox } from "styles";
import { SvgComponent } from "types";
import { ELEMENT_IDS } from "utils/consts.js";
import { CreateType, getCookie, setCookie } from "utils/localStorage.js";
import { RoutineTypeOption, routineTypes } from "utils/search/schemas/routine";
import { CreateViewProps } from "../types.js";

type CreateInfo = {
    objectType: CreateType;
    description: TranslationKeyCommon,
    Icon: SvgComponent,
    id: string,
    incomplete?: boolean;
}

export const createCards: CreateInfo[] = [
    {
        objectType: "Reminder",
        description: "CreateReminderDescription",
        Icon: ReminderIcon,
        id: "create-reminder-card",
    },
    {
        objectType: "Note",
        description: "CreateNoteDescription",
        Icon: NoteIcon,
        id: "create-note-card",
    },
    {
        objectType: "Routine",
        description: "CreateRoutineDescription",
        Icon: RoutineIcon,
        id: "create-routine-card",
    },
    {
        objectType: "Project",
        description: "CreateProjectDescription",
        Icon: ProjectIcon,
        id: "create-project-card",
    },
    {
        objectType: "Team",
        description: "CreateTeamDescription",
        Icon: TeamIcon,
        id: "create-team-card",
    },
    {
        objectType: "Bot",
        description: "CreateBotDescription",
        Icon: BotIcon,
        id: "create-bot-card",
    },
    {
        objectType: "Chat",
        description: "CreateChatDescription",
        Icon: CommentIcon,
        id: "create-chat-card",
    },
    {
        objectType: "Question",
        description: "CreateQuestionDescription",
        Icon: HelpIcon,
        id: "create-question-card",
        incomplete: true,
    },
    {
        objectType: "Prompt",
        description: "CreatePromptDescription",
        Icon: ArticleIcon,
        id: "create-prompt-card",
    },
    {
        objectType: "DataStructure",
        description: "CreateDataStructureDescription",
        Icon: ObjectIcon,
        id: "create-data-structure-card",
    },
    {
        objectType: "DataConverter",
        description: "CreateDataConverterDescription",
        Icon: TerminalIcon,
        id: "create-code-card",
    },
    {
        objectType: "SmartContract",
        description: "CreateSmartContractDescription",
        Icon: SmartContractIcon,
        id: "create-smart-contract-card",
        incomplete: true,
    },
    {
        objectType: "Api",
        description: "CreateApiDescription",
        Icon: ApiIcon,
        id: "create-api-card",
        incomplete: true,
    },
] as const;

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


const Z_INDEX_OFFSET = 1000;
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
    const zIndex = useZIndex(isOpen, false, Z_INDEX_OFFSET);
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
            zIndex={zIndex}
        >
            <TopBar
                display="dialog"
                onClose={handleClose}
            />
            {step === "complexity" && (
                <Box p={2}>
                    <Typography variant="h5" mb={4}>What kind of routine would you like to create?</Typography>
                    <Stack direction="column" spacing={2}>
                        <TIDCard
                            buttonText={t("Select")}
                            description={"Ideal for quick, one-off tasks or simple transformations."}
                            key="singleStep"
                            onClick={selectSingleStep}
                            size="small"
                            title={"Single-Step (Simple)"}
                        />
                        <TIDCard
                            buttonText={t("Select")}
                            description={"Use a graph of steps for complex, automated workflows."}
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
                        <IconButton onClick={selectComplexity}><ArrowLeftIcon /></IconButton>
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
                                    Icon={typeOption.Icon}
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

export function CreateView({
    display,
    onClose,
}: CreateViewProps) {
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
        <ScrollBox>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Create")}
                titleBehaviorDesktop="ShowIn"
            />
            <RoutineWizardDialog isOpen={showRoutineWizard} onClose={closeRoutineWizard} />
            <CardGrid minWidth={300}>
                {sortedCards.map(({ objectType, description, Icon, id, incomplete }, index) => {
                    function handleClick() {
                        onSelect(id);
                    }

                    return (
                        <TIDCard
                            buttonText={t("Create")}
                            description={t(description)}
                            Icon={Icon}
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
    );
}
