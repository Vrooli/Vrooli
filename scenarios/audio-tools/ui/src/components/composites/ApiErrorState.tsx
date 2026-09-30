import { AlertTriangle, RefreshCw } from "lucide-react";
import type { ApiError } from "../../api/client";
import { Button } from "../ui/button";
import { EmptyState } from "../ui/empty-state";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

interface Props {
  error: ApiError | Error;
  onRetry?: () => void;
  title?: string;
}

export function ApiErrorState({ error, onRetry, title }: Props) {
  const { t } = useTranslation();
  const code = "code" in error ? error.code : "internal";
  const isUnimplemented = code === "unimplemented" || code === "501";
  const headline =
    title ??
    (isUnimplemented
      ? t(strings.apiError.unimplementedTitle)
      : t(strings.apiError.loadFailedTitle));
  const description = isUnimplemented ? t(strings.apiError.unimplementedDescription) : error.message;
  return (
    <EmptyState
      tone={isUnimplemented ? "neutral" : "error"}
      icon={<AlertTriangle className="h-5 w-5" aria-hidden="true" />}
      title={headline}
      description={description}
      action={
        onRetry ? (
          <Button variant="outline" size="sm" onClick={onRetry}>
            <RefreshCw className="h-3.5 w-3.5" aria-hidden="true" />
            {t(strings.common.retry)}
          </Button>
        ) : undefined
      }
    />
  );
}
