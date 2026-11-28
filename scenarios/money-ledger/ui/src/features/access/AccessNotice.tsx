import { useState } from "react";
import { LockKeyhole } from "lucide-react";
import { BannerRegion } from "@vrooli/react-component-library/Banner/1";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { SignInDialog } from "./SignInDialog";
import { ACCESS_NOTICE_ID } from "./WriteFieldset";
import { refusalKey, signInCanHelp, useWriteAccess } from "./useWriteAccess";

/**
 * The page-wide notice that this browser is viewing only, why, and what to do.
 * Reading never needs a sign-in, so the notice informs rather than blocks; it
 * offers the sign-in when signing in could change the answer.
 */
export function AccessNotice() {
  const { t } = useTranslation();
  const access = useWriteAccess();
  const [signingIn, setSigningIn] = useState(false);
  const data = access.data;
  const viewingOnly = data && !data.canWrite;
  return (
    <div id={ACCESS_NOTICE_ID}>
      <BannerRegion
        ariaLabel={t(strings.access.noticeLabel)}
        tintStatusBar={false}
        testId={selectors.access.noticeRegion}
        banners={[
          viewingOnly && {
            id: "write-access",
            tone: "info",
            priority: 10,
            icon: LockKeyhole,
            title: t(strings.access.viewingOnly),
            description: t(refusalKey(data.refusal), { person: data.signedInAs }),
            detail: t(strings.access.cliHint),
            actions: signInCanHelp(data) ? [{ id: "sign-in", label: t(strings.access.signInAction), primary: true, onSelect: () => setSigningIn(true) }] : [],
            testId: selectors.access.notice,
            // The answer is the server's decision, not a flapping signal: show
            // it at once, and clear it at once when a sign-in changes it.
            damping: { enterAfterMs: 0, minVisibleMs: 0, exitAfterMs: 0 },
          },
        ]}
      />
      <SignInDialog open={signingIn} onClose={() => setSigningIn(false)} />
    </div>
  );
}
