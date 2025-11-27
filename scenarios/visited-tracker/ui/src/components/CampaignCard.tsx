import { Trash2, BarChart3, User } from "lucide-react";
import { Card, CardContent, CardFooter, CardHeader, CardTitle, CardDescription } from "./ui/card";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import type { Campaign } from "../lib/api";

interface CampaignCardProps {
  campaign: Campaign;
  onView: (id: string) => void;
  onDelete: (id: string, name: string) => void;
}

export function CampaignCard({ campaign, onView, onDelete }: CampaignCardProps) {
  const coveragePercent = Math.round(campaign.coverage_percent || 0);

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInSeconds = (now.getTime() - date.getTime()) / 1000;

    if (diffInSeconds < 60) return "just now";
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
    if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)}d ago`;
    if (diffInSeconds < 2592000) return `${Math.floor(diffInSeconds / 604800)}w ago`;

    return date.toLocaleDateString();
  };

  return (
    <Card
      className="cursor-pointer transition-all duration-200 hover:shadow-xl hover:shadow-slate-900/50 hover:scale-[1.02] hover:border-white/20 focus-within:ring-2 focus-within:ring-blue-500 active:scale-[0.98]"
      onClick={() => onView(campaign.id)}
      role="article"
      aria-label={`Campaign: ${campaign.name}`}
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onView(campaign.id);
        }
      }}
    >
      <CardHeader className="pb-3">
        <div className="space-y-1.5">
          <div className="flex items-start justify-between gap-2">
            <CardTitle className="line-clamp-1 text-base sm:text-lg">{campaign.name}</CardTitle>
            <Badge variant={campaign.status} aria-label={`Status: ${campaign.status}`} className="shrink-0">
              {campaign.status}
            </Badge>
          </div>
          <div className="flex items-center gap-1.5 text-[10px] sm:text-xs text-slate-400">
            <User className="h-3 w-3 shrink-0" aria-hidden="true" />
            <span className="truncate" title={campaign.from_agent}>{campaign.from_agent}</span>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        {campaign.description && (
          <CardDescription className="mb-3 sm:mb-4 line-clamp-2 text-xs sm:text-sm">{campaign.description}</CardDescription>
        )}

        <div className="grid grid-cols-3 gap-2 sm:gap-3 md:gap-4 text-center">
          <div className="rounded-lg bg-white/[0.02] p-2 hover:bg-white/[0.04] transition-colors">
            <div className="text-lg sm:text-xl md:text-2xl font-bold text-slate-50" aria-label={`${campaign.total_files || 0} total files`}>
              {campaign.total_files || 0}
            </div>
            <div className="text-[10px] sm:text-xs uppercase tracking-wider text-slate-400 mt-0.5">Files</div>
          </div>
          <div className="rounded-lg bg-white/[0.02] p-2 hover:bg-white/[0.04] transition-colors">
            <div className="text-lg sm:text-xl md:text-2xl font-bold text-green-400" aria-label={`${campaign.visited_files || 0} visited files`}>
              {campaign.visited_files || 0}
            </div>
            <div className="text-[10px] sm:text-xs uppercase tracking-wider text-slate-400 mt-0.5">Visited</div>
          </div>
          <div className="rounded-lg bg-white/[0.02] p-2 hover:bg-white/[0.04] transition-colors">
            <div className="text-lg sm:text-xl md:text-2xl font-bold text-blue-400" aria-label={`${coveragePercent}% coverage`}>
              {coveragePercent}%
            </div>
            <div className="text-[10px] sm:text-xs uppercase tracking-wider text-slate-400 mt-0.5">Cover</div>
          </div>
        </div>

        <div className="mt-3 sm:mt-4 space-y-2">
          <div
            className="h-1.5 sm:h-2 w-full overflow-hidden rounded-full bg-slate-800/80"
            role="progressbar"
            aria-valuenow={coveragePercent}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-label={`Coverage progress: ${coveragePercent}%`}
          >
            <div
              className="h-full rounded-full bg-gradient-to-r from-green-500 to-blue-500 transition-all duration-500 ease-out"
              style={{ width: `${coveragePercent}%` }}
            />
          </div>

          {campaign.patterns && campaign.patterns.length > 0 && (
            <div className="flex flex-wrap gap-1 sm:gap-1.5">
              {campaign.patterns.slice(0, 2).map((pattern, i) => (
                <span
                  key={i}
                  className="rounded bg-slate-800/60 px-1.5 sm:px-2 py-0.5 font-mono text-[10px] sm:text-xs text-slate-400 truncate max-w-[120px]"
                  title={pattern}
                >
                  {pattern}
                </span>
              ))}
              {campaign.patterns.length > 2 && (
                <span className="rounded bg-slate-800/60 px-1.5 sm:px-2 py-0.5 font-mono text-[10px] sm:text-xs text-slate-400">
                  +{campaign.patterns.length - 2}
                </span>
              )}
            </div>
          )}
        </div>

        <div className="mt-2 sm:mt-3 text-[10px] sm:text-xs text-slate-500">
          Created {formatDate(campaign.created_at)}
        </div>
      </CardContent>

      <CardFooter className="justify-end gap-1.5 sm:gap-2 pt-3">
        <Button
          variant="outline"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            onView(campaign.id);
          }}
          aria-label={`View campaign ${campaign.name}`}
          className="transition-all hover:scale-105 active:scale-95"
        >
          <BarChart3 className="mr-1 sm:mr-1.5 h-3 w-3 sm:h-3.5 sm:w-3.5" aria-hidden="true" />
          <span className="text-xs sm:text-sm">View</span>
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="border-red-500/30 text-red-400 hover:bg-red-500/10 hover:border-red-500/50 transition-all hover:scale-105 active:scale-95"
          onClick={(e) => {
            e.stopPropagation();
            onDelete(campaign.id, campaign.name);
          }}
          aria-label={`Delete campaign ${campaign.name}`}
        >
          <Trash2 className="mr-1 sm:mr-1.5 h-3 w-3 sm:h-3.5 sm:w-3.5" aria-hidden="true" />
          <span className="text-xs sm:text-sm hidden xs:inline">Delete</span>
          <span className="text-xs xs:hidden">Del</span>
        </Button>
      </CardFooter>
    </Card>
  );
}
