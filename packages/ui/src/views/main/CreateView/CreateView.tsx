import { ApiIcon, BotIcon, CommonKey, HelpIcon, LINKS, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from "@local/shared";
import { PageContainer } from "components/containers/PageContainer/PageContainer";
import { CardGrid } from "components/lists/CardGrid/CardGrid";
import { TIDCard } from "components/lists/TIDCard/TIDCard";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { CreateViewProps } from "../types";

type CreateType = "Api" | "Bot" | "Note" | "Organization" | "Project" | "Question" | "Reminder" | "Routine" | "SmartContract" | "Standard";

type CreateInfo = {
    objectType: CreateType;
    description: CommonKey,
    Icon: SvgComponent,
    id: string,
}

const createCards: CreateInfo[] = [
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

export const CreateView = ({
    display = "page",
    onClose,
    zIndex,
}: CreateViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const onSelect = useCallback((objectType: CreateType) => {
        setLocation(`${LINKS[objectType === "Bot" ? "User" : objectType]}/add`);
    }, [setLocation]);

    return (
        <PageContainer>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Create")}
                zIndex={zIndex}
            />
            <CardGrid minWidth={300}>
                {createCards.map(({ objectType, description, Icon, id }, index) => (
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
