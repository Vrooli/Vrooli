import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";

export function ContactsPage() {
  const { t } = useTranslation();
  return <section aria-labelledby="contacts-heading" className="flex flex-col gap-4"><h2 id="contacts-heading" className="text-2xl font-semibold">{t(strings.console.contacts.title)}</h2><p className="text-app-muted-foreground">{t(strings.console.contacts.description)}</p><ExperienceSurface surfaceId="contact-region" state="empty" className="rounded-lg border p-6">{t(strings.console.contacts.empty)}</ExperienceSurface></section>;
}
