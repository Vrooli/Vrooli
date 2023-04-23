import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { ApiIcon, HelpIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon } from "@local/icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/containers/PageContainer/PageContainer";
import { CardGrid } from "../../../components/lists/CardGrid/CardGrid";
import { TIDCard } from "../../../components/lists/TIDCard/TIDCard";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { useLocation } from "../../../utils/route";
const createCards = [
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
export const CreateView = ({ display = "page", }) => {
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const onSelect = useCallback((objectType) => {
        setLocation(`${LINKS[objectType]}/add`);
    }, [setLocation]);
    return (_jsxs(PageContainer, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Create",
                } }), _jsx(CardGrid, { minWidth: 300, children: createCards.map(({ objectType, description, Icon }, index) => (_jsx(TIDCard, { buttonText: t("Create"), description: t(description), Icon: Icon, onClick: () => onSelect(objectType), title: t(objectType, { count: 1 }) }, index))) })] }));
};
//# sourceMappingURL=CreateView.js.map