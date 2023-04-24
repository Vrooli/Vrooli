import { LINKS } from ":local/consts";
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, SvgComponent } from ":local/icons";
import { CommonKey } from ":local/translations";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/containers/PageContainer/PageContainer";
import { CardGrid } from "../../../components/lists/CardGrid/CardGrid";
import { TIDCard } from "../../../components/lists/TIDCard/TIDCard";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { useLocation } from "../../../utils/route";
import { CreateViewProps } from "../types";

type CreateType = "Api" | "Note" | "Organization" | "Project" | "Question" | "Reminder" | "Routine" | "SmartContract" | "Standard";

type CreateInfo = {
    objectType: CreateType;
    description: CommonKey,
    Icon: SvgComponent,
}

const createCards: CreateInfo[] = [
    {
        objectType: "Reminder",
        description: "CreateReminderDescription",
        Icon: ReminderIcon,
    },
    {
        objectType: "Note",
        description: "CreateNoteDescription",
        Icon: NoteIcon,
    },
    {
        objectType: "Routine",
        description: "CreateRoutineDescription",
        Icon: RoutineIcon,
    },
    {
        objectType: "Project",
        description: "CreateProjectDescription",
        Icon: ProjectIcon,
    },
    {
        objectType: "Organization",
        description: "CreateOrganizationDescription",
        Icon: OrganizationIcon,
    },
    {
        objectType: "Question",
        description: "CreateQuestionDescription",
        Icon: HelpIcon,
    },
    {
        objectType: "Standard",
        description: "CreateStandardDescription",
        Icon: StandardIcon,
    },
    {
        objectType: "SmartContract",
        description: "CreateSmartContractDescription",
        Icon: SmartContractIcon,
    },
    {
        objectType: "Api",
        description: "CreateApiDescription",
        Icon: ApiIcon,
    },
];

export const CreateView = ({
    display = "page",
}: CreateViewProps) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();

    const onSelect = useCallback((objectType: CreateType) => {
        setLocation(`${LINKS[objectType]}/add`);
    }, [setLocation]);

    return (
        <PageContainer>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "Create",
                }}
            />
            <CardGrid minWidth={300}>
                {createCards.map(({ objectType, description, Icon }, index) => (
                    <TIDCard
                        buttonText={t("Create")}
                        description={t(description)}
                        Icon={Icon}
                        key={index}
                        onClick={() => onSelect(objectType)}
                        title={t(objectType, { count: 1 })}
                    />
                ))}
            </CardGrid>
        </PageContainer>
    );
};
