import { CommonKey, LINKS } from "@local/shared";
import { PageContainer } from "components/containers/PageContainer/PageContainer";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { ApiIcon, BotIcon, CommentIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "icons";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { SvgComponent } from "types";
import { CreateType, getCookie, setCookie } from "utils/cookies";
import { CreateViewProps } from "../types";

type CreateInfo = {
    objectType: CreateType;
    description: CommonKey,
    Icon: SvgComponent,
    id: string,
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
        objectType: "Organization",
        description: "CreateOrganizationDescription",
        Icon: OrganizationIcon,
        id: "create-organization-card",
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
    },
    {
        objectType: "Standard",
        description: "CreateStandardDescription",
        Icon: StandardIcon,
        id: "create-standard-card",
    },
    {
        objectType: "SmartContract",
        description: "CreateSmartContractDescription",
        Icon: SmartContractIcon,
        id: "create-smart-contract-card",
    },
    {
        objectType: "Api",
        description: "CreateApiDescription",
        Icon: ApiIcon,
        id: "create-api-card",
    },
];

const sortCardsByUsageHistory = (cards: CreateInfo[]) => {
    const history = getCookie("CreateOrder");
    cards.sort((a, b) => {
        const aIndex = history.indexOf(a.objectType);
        const bIndex = history.indexOf(b.objectType);
        if (aIndex === -1) return 1;
        if (bIndex === -1) return -1;
        return aIndex - bIndex;
    });
};

export const CreateView = ({
    display,
    onClose,
}: CreateViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const sortedCards = useMemo(() => {
        const cardsCopy = [...createCards];
        sortCardsByUsageHistory(cardsCopy);
        return cardsCopy;
    }, []);

    const onSelect = useCallback((objectType: CreateType) => {
        // Update location
        setLocation(`${LINKS[objectType === "Bot" ? "User" : objectType]}/add`);
        // Update usage history
        const history = getCookie("CreateOrder");
        const index = history.indexOf(objectType);
        if (index !== -1) {
            history.splice(index, 1);
        }
        history.unshift(objectType);
        setCookie("CreateOrder", history);
    }, [setLocation]);

    return (
        <PageContainer>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Create")}
            />
            <CardGrid minWidth={300}>
                {sortedCards.map(({ objectType, description, Icon, id }, index) => (
                    <TIDCard
                        buttonText={t("Create")}
                        description={t(description)}
                        Icon={Icon}
                        id={id}
                        key={index}
                        onClick={() => onSelect(objectType)}
                        title={t(objectType, { count: 1 })}
                    />
                ))}
            </CardGrid>
        </PageContainer>
    );
};
