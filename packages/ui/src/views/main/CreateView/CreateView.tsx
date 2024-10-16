import { LINKS, TranslationKeyCommon } from "@local/shared";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { ApiIcon, ArticleIcon, BotIcon, CommentIcon, HelpIcon, NoteIcon, ObjectIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, TeamIcon, TerminalIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ScrollBox } from "styles";
import { SvgComponent } from "types";
import { CreateType, getCookie, setCookie } from "utils/localStorage";
import { CreateViewProps } from "../types";

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

function sortCardsByUsageHistory(cards: CreateInfo[]) {
    const history = getCookie("CreateOrder");
    cards.sort((a, b) => {
        const aIndex = history.indexOf(a.id);
        const bIndex = history.indexOf(b.id);
        if (aIndex === -1) return 1;
        if (bIndex === -1) return -1;
        return aIndex - bIndex;
    });
}

export function CreateView({
    display,
    onClose,
}: CreateViewProps) {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const sortedCards = useMemo(() => {
        const cardsCopy = [...createCards];
        sortCardsByUsageHistory(cardsCopy);
        return cardsCopy;
    }, []);

    const onSelect = useCallback((id: typeof createCards[number]["id"]) => {
        const { objectType } = createCards.find(card => card.id === id) ?? {};
        if (!objectType) return;
        // Update location
        setLocation(`${LINKS[objectType === "Bot" ? "User" : objectType]}/add`);
        // Update usage history
        const history = getCookie("CreateOrder");
        const index = history.indexOf(id);
        if (index !== -1) {
            history.splice(index, 1);
        }
        history.unshift(id);
        setCookie("CreateOrder", history);
    }, [setLocation]);

    return (
        <ScrollBox>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Create")}
                titleBehaviorDesktop="ShowIn"
            />
            <CardGrid minWidth={300}>
                {sortedCards.map(({ objectType, description, Icon, id, incomplete }, index) => (
                    <TIDCard
                        buttonText={t("Create")}
                        description={t(description)}
                        Icon={Icon}
                        id={id}
                        key={index}
                        onClick={() => onSelect(id)}
                        title={t(objectType, { count: 1 })}
                        warning={incomplete ? "Coming soon" : undefined}
                    />
                ))}
            </CardGrid>
        </ScrollBox>
    );
}
