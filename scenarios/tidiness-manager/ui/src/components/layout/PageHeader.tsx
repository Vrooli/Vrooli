import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { Button } from "../ui/button";
import { cn } from "../../lib/utils";

interface PageHeaderProps {
  title: string;
  description?: string;
  backTo?: string;
  actions?: React.ReactNode;
  className?: string;
}

export function PageHeader({ title, description, backTo, actions, className }: PageHeaderProps) {
  const navigate = useNavigate();

  return (
    <header className={cn("flex flex-col gap-4 mb-6", className)}>
      {backTo && (
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate(backTo)}
          className="text-slate-400 hover:text-slate-50 w-fit"
          aria-label="Go back to previous page"
        >
          <ArrowLeft className="h-4 w-4 mr-2" aria-hidden="true" />
          Back
        </Button>
      )}

      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="flex-1 min-w-0">
          <h1 className="text-3xl font-bold tracking-tight">{title}</h1>
          {description && (
            <p className="text-slate-400 mt-2 max-w-2xl">{description}</p>
          )}
        </div>

        {actions && (
          <div className="flex items-center gap-2 shrink-0">
            {actions}
          </div>
        )}
      </div>
    </header>
  );
}
