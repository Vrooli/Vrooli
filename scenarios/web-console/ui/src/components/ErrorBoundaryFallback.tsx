import { AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";

interface ErrorBoundaryFallbackProps {
  region: string;
  message: string;
  onReset: () => void;
}

export default function ErrorBoundaryFallback({
  region,
  message,
  onReset,
}: ErrorBoundaryFallbackProps) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={`error-boundary-${region}`}
      className="flex flex-col items-center justify-center gap-3 rounded-md border border-wc-error bg-wc-error-surface p-6 text-sm text-wc-error-text"
    >
      <AlertTriangle className="h-6 w-6 text-wc-error-detail" />
      <p className="font-medium">{t(strings.errorBoundary.somethingWentWrong, { region })}</p>
      <p className="max-w-md text-center text-xs text-wc-error-detail/70">{message}</p>
      <Button variant="outline" size="sm" onClick={onReset} className="mt-2">
        <RefreshCw className="me-1.5 h-3.5 w-3.5" />
        {t(strings.errorBoundary.tryAgain)}
      </Button>
    </div>
  );
}
