import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import identity from "../launch-screen.html?raw";

/** The same local, trusted markup is inlined into the pre-JavaScript shell. */
export default function LaunchScreen() {
  const { t } = useTranslation();
  return (
    <div
      className="wc-launch"
      data-wc-launch="true"
      data-testid="app-launch"
      role="status"
      aria-label={t(strings.app.loading)}
      dangerouslySetInnerHTML={{ __html: identity }}
    />
  );
}
