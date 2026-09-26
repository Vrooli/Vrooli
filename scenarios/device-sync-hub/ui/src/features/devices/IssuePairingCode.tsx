import { useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { QrCode } from "../../components/QrCode";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { useIssuePairingCodeMutation } from "./queries";

/**
 * Owner-gated "add a device" surface: issue a short-TTL pairing code and show it
 * both as large copyable text and as a QR (rendered dependency-free) so another
 * device can join by typing or scanning it.
 */
export function IssuePairingCode() {
  const { t } = useTranslation();
  const [deviceName, setDeviceName] = useState("");
  const issue = useIssuePairingCodeMutation();
  const code = issue.data?.pairingCode;

  return (
    <div data-testid={selectors.devices.issuePanel} className="flex flex-col gap-3">
      <div>
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.devices.issue.heading)}
        </h3>
        <p className="text-xs text-app-muted-foreground">{t(strings.devices.issue.description)}</p>
      </div>
      <div className="flex flex-wrap items-end gap-2">
        <div className="flex flex-col gap-1">
          <label htmlFor="issue-name" className="text-xs text-app-muted-foreground">
            {t(strings.devices.issue.deviceNameLabel)}
          </label>
          <Input
            id="issue-name"
            data-testid={selectors.devices.issueNameInput}
            value={deviceName}
            onChange={(e) => setDeviceName(e.target.value)}
            placeholder={t(strings.devices.issue.deviceNamePlaceholder)}
            className="h-9 w-48"
          />
        </div>
        <Button
          data-testid={selectors.devices.issueButton}
          onClick={() => issue.mutate(deviceName.trim())}
          disabled={issue.isPending}
        >
          {t(strings.devices.issue.issue)}
        </Button>
      </div>

      {issue.error && <p className="text-sm text-app-danger">{errorMessage(issue.error, t)}</p>}

      {code && (
        <div className="flex flex-wrap items-center gap-4 rounded-panel border border-app-border bg-app-background p-4">
          <QrCode
            value={code.code}
            data-testid={selectors.devices.issuedQr}
            aria-label={t(strings.devices.issue.qrLabel)}
          />
          <div className="flex flex-col gap-1">
            <p className="text-xs uppercase text-app-muted-foreground">
              {t(strings.devices.issue.codeHeading)}
            </p>
            <p data-testid={selectors.devices.issuedCode} className="select-all font-mono text-2xl font-semibold tracking-widest text-app-foreground">
              {code.code}
            </p>
            <p className="text-xs text-app-muted-foreground">{t(strings.devices.issue.codeHint)}</p>
            {code.expiresAt && (
              <p className="text-xs text-app-muted-foreground">
                {t(strings.devices.issue.expiresLabel, {
                  when: formatDate(timestampDate(code.expiresAt), { dateStyle: "short", timeStyle: "short" }),
                })}
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
