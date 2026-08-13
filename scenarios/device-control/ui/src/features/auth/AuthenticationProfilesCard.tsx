import type { AuthProfile, ProviderStatus } from "../../api/authentication";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

type Props = { profiles: AuthProfile[]; providers: Record<string, ProviderStatus> };

/** Reference-only profile view: it intentionally has no credential-value prop. */
export function AuthenticationProfilesCard({ profiles, providers }: Props) {
  const { t } = useTranslation();
  return (
    <Card data-testid="authentication-profiles-card">
      <CardHeader><CardTitle>{t(strings.pages.auth.title)}</CardTitle></CardHeader>
      <CardContent>
        {profiles.length === 0 ? <p className="text-app-muted-foreground">{t(strings.pages.auth.none)}</p> : (
          <div className="flex flex-col gap-3">
            {profiles.map((profile) => {
              const provider = providers[profile.id];
              return <div key={profile.id} className="rounded-md border p-3">
                <div className="flex items-center justify-between gap-3">
                  <span className="font-medium">{profile.id}</span>
                  <span className="rounded-full border px-2 py-1 text-xs">{profile.status}</span>
                </div>
                <p className="mt-1 text-sm text-app-muted-foreground">{profile.device_id} · {profile.method} · {profile.verification}</p>
                <p className="text-xs text-app-muted-foreground">{t(strings.pages.auth.provider, { state: provider?.provider_state || "not checked" })} · {t(strings.pages.auth.lastOutcome, { outcome: profile.last_outcome || "never tested" })}</p>
                <p className="text-xs text-app-muted-foreground">{t(strings.pages.auth.reference, { identity: profile.credential_identity, field: profile.credential_field })}</p>
                <p className="mt-2 text-sm">{t(provider?.configured ? strings.pages.auth.configured : strings.pages.auth.unconfigured)}</p>
              </div>;
            })}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
