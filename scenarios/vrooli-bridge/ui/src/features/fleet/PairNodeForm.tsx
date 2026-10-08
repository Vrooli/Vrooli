import { useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { KeyRound } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { useIssuePairingCodeMutation } from "./queries";

/**
 * One-touch pairing surface (OT-P0-002): the owner types a node label and mints
 * a single-use code. The plaintext code + control-plane public key are shown
 * ONCE (the server only stores the hash) for out-of-band delivery to the node's
 * bootstrap installer. Handles loading (submitting) / error / result states.
 */
export function PairNodeForm() {
  const { t } = useTranslation();
  const issue = useIssuePairingCodeMutation();
  const [name, setName] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = name.trim();
    if (!trimmed) return;
    issue.mutate(trimmed);
  };

  const result = issue.data;

  const handleCopy = () => {
    // `navigator.clipboard` is typed as always-present but is undefined in some
    // runtimes (insecure contexts, jsdom); read it through an optional access.
    const clipboard = navigator.clipboard as Clipboard | undefined;
    if (result?.code && clipboard) {
      void clipboard.writeText(result.code);
    }
  };

  return (
    <section
      aria-labelledby="fleet-pairing-heading"
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 id="fleet-pairing-heading" className="text-sm font-semibold text-app-foreground">
        {t(strings.fleet.pairing.heading)}
      </h3>
      <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.fleet.pairing.description)}</p>

      <form
        data-testid={selectors.fleet.pairing.form}
        onSubmit={handleSubmit}
        className="mt-3 flex flex-wrap items-end gap-2"
      >
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <label htmlFor="fleet-pairing-name-input" className="text-xs text-app-muted-foreground">
            {t(strings.fleet.pairing.nameLabel)}
          </label>
          <Input
            id="fleet-pairing-name-input"
            data-testid={selectors.fleet.pairing.nameInput}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t(strings.fleet.pairing.namePlaceholder)}
            disabled={issue.isPending}
          />
        </div>
        <Button
          type="submit"
          size="sm"
          data-testid={selectors.fleet.pairing.submit}
          disabled={issue.isPending || name.trim().length === 0}
        >
          <KeyRound aria-hidden="true" className="mr-2 h-4 w-4" />
          {issue.isPending ? t(strings.fleet.pairing.submitting) : t(strings.fleet.pairing.submit)}
        </Button>
      </form>

      {issue.error && (
        <p
          data-testid={selectors.fleet.pairing.error}
          role="alert"
          className="mt-3 text-sm text-app-danger"
        >
          {errorMessage(issue.error, t)}
        </p>
      )}

      {result && (
        <div
          data-testid={selectors.fleet.pairing.result}
          className="mt-3 flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="min-w-0">
              <p className="text-xs font-semibold text-app-foreground">
                {t(strings.fleet.pairing.codeHeading)}
              </p>
              <code
                data-testid={selectors.fleet.pairing.code}
                className="block break-all font-mono text-sm text-app-foreground"
              >
                {result.code}
              </code>
            </div>
            <Button
              type="button"
              size="sm"
              variant="secondary"
              data-testid={selectors.fleet.pairing.copy}
              onClick={handleCopy}
            >
              {t(strings.fleet.pairing.copy)}
            </Button>
          </div>
          <p className="text-xs text-app-muted-foreground">{t(strings.fleet.pairing.codeHelp)}</p>
          <p className="break-all text-xs text-app-muted-foreground">
            {t(strings.fleet.pairing.publicKeyLabel)}: <code className="font-mono">{result.controlPlanePublicKey}</code>
          </p>
          {result.expiresAt && (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.fleet.pairing.expiresLabel)}:{" "}
              {formatDate(timestampDate(result.expiresAt), { dateStyle: "short", timeStyle: "short" })}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
