import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { listProfileReleases, reverifyRelease } from "../../lib/api";
import type { Release } from "../../lib/api";
import { getErrorMessage } from "../../lib/utils";

export interface ReleasesPanelProps {
  profileId: string;
  limit?: number;
}

function statusVariant(status: Release["status"]): "success" | "warning" | "destructive" | "secondary" {
  switch (status) {
    case "published":
      return "success";
    case "publishing":
    case "pending":
      return "warning";
    case "failed":
    case "verify_failed":
      return "destructive";
    default:
      return "secondary";
  }
}

export function ReleasesPanel({ profileId, limit = 10 }: ReleasesPanelProps) {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["profile-releases", profileId, limit],
    queryFn: () => listProfileReleases(profileId, limit),
    enabled: Boolean(profileId),
  });

  const reverify = useMutation({
    mutationFn: (releaseId: string) => reverifyRelease(releaseId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["profile-releases", profileId, limit] });
    },
  });

  const releases = data?.releases ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Releases</CardTitle>
        <CardDescription>
          Latest desktop releases driven by deployment-manager. Per-platform verification status reflects the LPBS update endpoint.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading && <p data-testid="releases-loading">Loading releases…</p>}
        {error && (
          <p role="alert" data-testid="releases-error">
            {getErrorMessage(error)}
          </p>
        )}
        {!isLoading && !error && releases.length === 0 && (
          <p data-testid="releases-empty">No releases recorded yet for this profile.</p>
        )}
        {!isLoading && !error && releases.length > 0 && (
          <ul className="space-y-3">
            {releases.map((rel) => (
              <li key={rel.id} className="border rounded p-3" data-testid={`release-${rel.id}`}>
                <div className="flex items-center justify-between gap-2">
                  <div>
                    <div className="font-mono text-sm">{rel.id.substring(0, 12)}…</div>
                    <div className="text-sm text-muted-foreground">
                      {rel.release_version} on {rel.channel}
                    </div>
                  </div>
                  <Badge variant={statusVariant(rel.status)} data-testid={`release-status-${rel.id}`}>
                    {rel.status}
                  </Badge>
                </div>

                {rel.platforms && rel.platforms.length > 0 && (
                  <ul className="mt-2 text-sm">
                    {rel.platforms.map((p) => (
                      <li key={p.platform} className="flex justify-between">
                        <span>{p.platform}</span>
                        <Badge variant={p.status === "published" ? "success" : p.status === "verify_failed" ? "destructive" : "secondary"}>
                          {p.status}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                )}

                <div className="mt-2 flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => reverify.mutate(rel.id)}
                    disabled={reverify.isPending}
                    data-testid={`release-reverify-${rel.id}`}
                  >
                    {reverify.isPending && reverify.variables === rel.id ? "Re-verifying…" : "Re-verify"}
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
        {reverify.error && (
          <p role="alert" data-testid="releases-reverify-error">
            {getErrorMessage(reverify.error)}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
