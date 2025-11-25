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
    if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)} minutes ago`;
    if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)} hours ago`;
    if (diffInSeconds < 604800) return `${Math.floor(diffInSeconds / 86400)} days ago`;

    return date.toLocaleDateString();
  };

  return (
    <Card className="cursor-pointer hover:scale-[1.02]" onClick={() => onView(campaign.id)}>
      <CardHeader>
        <div className="space-y-1">
          <div className="flex items-start justify-between gap-3">
            <CardTitle className="line-clamp-1">{campaign.name}</CardTitle>
            <Badge variant={campaign.status}>{campaign.status}</Badge>
          </div>
          <div className="flex items-center gap-1.5 text-xs text-slate-400">
            <User className="h-3 w-3" />
            <span>{campaign.from_agent}</span>
          </div>
        </div>
      </CardHeader>

      <CardContent>
        {campaign.description && (
          <CardDescription className="mb-4 line-clamp-2">{campaign.description}</CardDescription>
        )}

        <div className="grid grid-cols-3 gap-4 text-center">
          <div>
            <div className="text-2xl font-semibold text-slate-50">{campaign.total_files || 0}</div>
            <div className="text-xs uppercase tracking-wider text-slate-500">Files</div>
          </div>
          <div>
            <div className="text-2xl font-semibold text-slate-50">{campaign.visited_files || 0}</div>
            <div className="text-xs uppercase tracking-wider text-slate-500">Visited</div>
          </div>
          <div>
            <div className="text-2xl font-semibold text-slate-50">{coveragePercent}%</div>
            <div className="text-xs uppercase tracking-wider text-slate-500">Coverage</div>
          </div>
        </div>

        <div className="mt-4 space-y-2">
          <div className="h-2 w-full overflow-hidden rounded-full bg-slate-800">
            <div
              className="h-full rounded-full bg-gradient-to-r from-green-500 to-blue-500 transition-all duration-500"
              style={{ width: `${coveragePercent}%` }}
            />
          </div>

          {campaign.patterns && campaign.patterns.length > 0 && (
            <div className="flex flex-wrap gap-1.5">
              {campaign.patterns.slice(0, 3).map((pattern, i) => (
                <span
                  key={i}
                  className="rounded bg-slate-800/60 px-2 py-0.5 font-mono text-xs text-slate-400"
                >
                  {pattern}
                </span>
              ))}
              {campaign.patterns.length > 3 && (
                <span className="rounded bg-slate-800/60 px-2 py-0.5 font-mono text-xs text-slate-400">
                  +{campaign.patterns.length - 3} more
                </span>
              )}
            </div>
          )}
        </div>

        <div className="mt-3 text-xs text-slate-500">
          Created {formatDate(campaign.created_at)}
        </div>
      </CardContent>

      <CardFooter className="justify-end gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            onView(campaign.id);
          }}
        >
          <BarChart3 className="mr-1.5 h-3.5 w-3.5" />
          View
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="border-red-500/30 text-red-400 hover:bg-red-500/10"
          onClick={(e) => {
            e.stopPropagation();
            onDelete(campaign.id, campaign.name);
          }}
        >
          <Trash2 className="mr-1.5 h-3.5 w-3.5" />
          Delete
        </Button>
      </CardFooter>
    </Card>
  );
}
