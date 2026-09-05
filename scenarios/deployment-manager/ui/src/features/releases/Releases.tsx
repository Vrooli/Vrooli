import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, RefreshCw } from "lucide-react";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { listProfiles, listProfileReleases } from "../../lib/api";
import type { DeploymentProfile } from "../../lib/api";

export function Releases() {
  const [profileID, setProfileID] = useState("");
  const profilesQuery = useQuery({ queryKey: ["profiles"], queryFn: listProfiles });
  const releasesQuery = useQuery({ queryKey: ["releases", profileID], queryFn: () => listProfileReleases(profileID, 50), enabled: profileID.length > 0 });
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 data-testid="releases-heading" className="text-3xl font-bold">Releases</h1><p data-testid="releases-records-intro" className="mt-1 text-slate-400">Real release records and their platform outcomes.</p></div>{profileID && <Button variant="outline" onClick={() => void releasesQuery.refetch()}><RefreshCw className="mr-2 h-4 w-4" />Refresh</Button>}</div>
      <Card><CardContent className="pt-6"><label className="text-sm text-slate-300">Profile<select className="mt-1 block w-full max-w-md rounded-md border border-white/10 bg-white/5 px-3 py-2" value={profileID} onChange={(event) => setProfileID(event.target.value)}><option value="">Select profile</option>{profilesQuery.data?.map((profile: DeploymentProfile) => <option key={profile.id} value={profile.id}>{profile.name}</option>)}</select></label></CardContent></Card>
      {releasesQuery.isLoading && <Loader2 className="h-8 w-8 animate-spin text-cyan-400" />}
      {releasesQuery.data?.releases.map((release) => <Card key={release.id}><CardHeader><CardTitle className="flex items-center justify-between"><span>{release.release_version} · {release.channel}</span><Badge>{release.status}</Badge></CardTitle></CardHeader><CardContent className="grid gap-2 text-sm text-slate-300 md:grid-cols-3"><span>Commit: <code>{release.git_commit_hash}</code></span><span>Created: {new Date(release.created_at).toLocaleString()}</span><span>Platforms: {release.platforms?.length ?? 0}</span></CardContent></Card>)}
      {profileID && releasesQuery.data?.releases.length === 0 && <p className="text-slate-400">No release records for this profile.</p>}
    </div>
  );
}
