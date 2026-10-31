import type { ReactNode, ComponentType } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { Checkbox } from "../ui/checkbox";
import { Label } from "../ui/label";

interface SigningFormWrapperProps {
  platform: string;
  platformId: string;
  icon: ComponentType<{ className?: string }>;
  iconClassName?: string;
  isConfigured: boolean;
  onToggle: (enabled: boolean) => void;
  headerActions?: ReactNode;
  disabledMessage: string;
  children: ReactNode;
  testId?: string;
}

export function SigningFormWrapper({
  platform,
  platformId,
  icon: Icon,
  iconClassName,
  isConfigured,
  onToggle,
  headerActions,
  disabledMessage,
  children,
  testId,
}: SigningFormWrapperProps) {
  return (
    <Card data-testid={testId} className="border-slate-700/50 bg-slate-950/40">
      <CardHeader className="pb-3">
        <CardTitle className="text-base flex items-center justify-between">
          <span className="flex items-center gap-2">
            <Icon className={iconClassName ?? "h-4 w-4"} />
            {platform}
          </span>
          <div className="flex items-center gap-2">
            {headerActions}
            <Checkbox
              id={`${platformId}-enabled`}
              checked={isConfigured}
              onChange={(e) => {
                onToggle(e.target.checked);
              }}
            />
            <Label
              htmlFor={`${platformId}-enabled`}
              className="text-xs text-slate-400"
            >
              Configure
            </Label>
          </div>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {isConfigured ? (
          children
        ) : (
          <p className="text-xs text-slate-500 text-center py-4">
            {disabledMessage}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
