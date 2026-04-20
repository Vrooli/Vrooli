import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { getProfileLPBSConfig, saveProfileLPBSConfig } from "../lib/api";
import type { LPBSReleaseConfig } from "../lib/api";
import { getErrorMessage } from "../lib/utils";

export interface LPBSReleaseConfigCardProps {
  profileId: string;
}

const EMPTY: LPBSReleaseConfig = {
  profile_id: "",
  lpbs_domain: "",
  lpbs_remote_profile: "",
  lpbs_app_key: "",
  default_channel: "stable",
  update_url: "",
};

export function LPBSReleaseConfigCard({ profileId }: LPBSReleaseConfigCardProps) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<LPBSReleaseConfig>(EMPTY);
  const [editing, setEditing] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ["lpbs-config", profileId],
    queryFn: () => getProfileLPBSConfig(profileId),
    enabled: Boolean(profileId),
  });

  useEffect(() => {
    if (data) {
      setDraft(data);
    }
  }, [data]);

  const save = useMutation({
    mutationFn: (cfg: LPBSReleaseConfig) => saveProfileLPBSConfig(profileId, cfg),
    onSuccess: (saved) => {
      setDraft(saved);
      setEditing(false);
      queryClient.invalidateQueries({ queryKey: ["lpbs-config", profileId] });
    },
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>LPBS Release Config</CardTitle>
        <CardDescription>
          Coordinates deployment-manager uses to publish desktop releases through landing-page-business-suite.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading && <p data-testid="lpbs-config-loading">Loading…</p>}
        {error && (
          <p role="alert" data-testid="lpbs-config-error">
            {getErrorMessage(error)}
          </p>
        )}
        {!isLoading && !error && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              save.mutate(draft);
            }}
            className="space-y-3"
          >
            <Field
              label="LPBS Domain"
              name="lpbs_domain"
              value={draft.lpbs_domain}
              disabled={!editing}
              onChange={(v) => setDraft({ ...draft, lpbs_domain: v })}
            />
            <Field
              label="Remote Profile Tag"
              name="lpbs_remote_profile"
              value={draft.lpbs_remote_profile}
              disabled={!editing}
              onChange={(v) => setDraft({ ...draft, lpbs_remote_profile: v })}
            />
            <Field
              label="App Key"
              name="lpbs_app_key"
              value={draft.lpbs_app_key}
              disabled={!editing}
              onChange={(v) => setDraft({ ...draft, lpbs_app_key: v })}
            />
            <Field
              label="Default Channel"
              name="default_channel"
              value={draft.default_channel}
              disabled={!editing}
              onChange={(v) => setDraft({ ...draft, default_channel: v })}
            />
            <Field
              label="Update URL (optional)"
              name="update_url"
              value={draft.update_url}
              disabled={!editing}
              onChange={(v) => setDraft({ ...draft, update_url: v })}
            />
            <div className="flex gap-2">
              {!editing && (
                <Button type="button" variant="outline" onClick={() => setEditing(true)}>
                  Edit
                </Button>
              )}
              {editing && (
                <>
                  <Button type="submit" disabled={save.isPending}>
                    {save.isPending ? "Saving…" : "Save"}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setDraft(data ?? EMPTY);
                      setEditing(false);
                    }}
                  >
                    Cancel
                  </Button>
                </>
              )}
            </div>
            {save.error && (
              <p role="alert" data-testid="lpbs-config-save-error">
                {getErrorMessage(save.error)}
              </p>
            )}
          </form>
        )}
      </CardContent>
    </Card>
  );
}

interface FieldProps {
  label: string;
  name: string;
  value: string;
  disabled: boolean;
  onChange: (v: string) => void;
}

function Field({ label, name, value, disabled, onChange }: FieldProps) {
  return (
    <div className="space-y-1">
      <Label htmlFor={name}>{label}</Label>
      <Input
        id={name}
        name={name}
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}
