import { Button } from "@vrooli/react-component-library/Button/2";
import { useNavigate } from "react-router-dom";

import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Launch for a packet that is already built: opens its launch page, where its
 * check runs and one confirmation starts it. Never launches here.
 */
export function LaunchLink({ id }: { id: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  return (
    <div>
      <Button variant="primary" onClick={() => navigate(`/launch/${encodeURIComponent(id)}`)}>{t(strings.launch.launchPrepared)}</Button>
    </div>
  );
}
