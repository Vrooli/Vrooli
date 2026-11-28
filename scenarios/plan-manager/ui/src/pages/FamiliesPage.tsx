import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FamilyConsole } from "../features/families/FamilyConsole";
import { useTranslation } from "../i18n";

export function FamiliesPage() {
  const { t } = useTranslation();
  return <section data-testid={selectors.pages.families} aria-labelledby="families-heading" className="flex flex-col gap-4"><header><h2 id="families-heading" className="text-2xl font-semibold">{t(strings.pages.families.title)}</h2><p className="text-app-muted-foreground">{t(strings.pages.families.description)}</p></header><FamilyConsole /></section>;
}
