import { useState, useEffect } from "react";
import type { DistributionTarget } from "../../lib/api";
import { Button } from "../ui/button";
import { Label } from "../ui/label";
import { Select } from "../ui/select";
import { Checkbox } from "../ui/checkbox";

interface DistributionTargetFormProps {
  initialTarget?: DistributionTarget | null;
  onSubmit: (target: DistributionTarget) => void;
  onCancel: () => void;
  isSubmitting?: boolean;
  error?: string;
}

const PROVIDERS = [
  { value: "s3", label: "AWS S3" },
  { value: "r2", label: "Cloudflare R2" },
  { value: "s3-compatible", label: "S3-Compatible" },
] as const;

export function DistributionTargetForm({
  initialTarget,
  onSubmit,
  onCancel,
  isSubmitting,
  error,
}: DistributionTargetFormProps) {
  const [name, setName] = useState(initialTarget?.name || "");
  const [enabled, setEnabled] = useState(initialTarget?.enabled ?? true);
  const [provider, setProvider] = useState<"s3" | "r2" | "s3-compatible">(
    initialTarget?.provider || "s3"
  );
  const [bucket, setBucket] = useState(initialTarget?.bucket || "");
  const [endpoint, setEndpoint] = useState(initialTarget?.endpoint || "");
  const [region, setRegion] = useState(initialTarget?.region || "");
  const [pathPrefix, setPathPrefix] = useState(initialTarget?.path_prefix || "");
  const [accessKeyIdEnv, setAccessKeyIdEnv] = useState(
    initialTarget?.access_key_id_env || ""
  );
  const [secretAccessKeyEnv, setSecretAccessKeyEnv] = useState(
    initialTarget?.secret_access_key_env || ""
  );
  const [acl, setAcl] = useState(initialTarget?.acl || "");
  const [cdnUrl, setCdnUrl] = useState(initialTarget?.cdn_url || "");

  const isEditing = !!initialTarget;

  // Update endpoint placeholder based on provider
  useEffect(() => {
    if (!initialTarget) {
      if (provider === "r2") {
        setEndpoint((prev) =>
          prev.includes("r2.cloudflarestorage.com") || prev === ""
            ? "https://<account-id>.r2.cloudflarestorage.com"
            : prev
        );
      }
    }
  }, [provider, initialTarget]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const target: DistributionTarget = {
      name: name.trim(),
      enabled,
      provider,
      bucket: bucket.trim(),
      endpoint: endpoint.trim() || undefined,
      region: region.trim() || undefined,
      path_prefix: pathPrefix.trim() || undefined,
      access_key_id_env: accessKeyIdEnv.trim(),
      secret_access_key_env: secretAccessKeyEnv.trim(),
      acl: acl.trim() || undefined,
      cdn_url: cdnUrl.trim() || undefined,
    };

    onSubmit(target);
  };

  const requiresEndpoint = provider === "r2" || provider === "s3-compatible";
  const requiresRegion = provider === "s3";

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        {/* Name */}
        <div className="space-y-2">
          <Label htmlFor="target-name">Target Name *</Label>
          <input
            id="target-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="production-r2"
            required
            disabled={isEditing}
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm disabled:opacity-50"
          />
          {isEditing && (
            <p className="text-xs text-slate-500">Name cannot be changed after creation</p>
          )}
        </div>

        {/* Provider */}
        <div className="space-y-2">
          <Label htmlFor="target-provider">Provider *</Label>
          <Select
            id="target-provider"
            value={provider}
            onChange={(e) => setProvider(e.target.value as "s3" | "r2" | "s3-compatible")}
            required
          >
            {PROVIDERS.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </Select>
        </div>

        {/* Bucket */}
        <div className="space-y-2">
          <Label htmlFor="target-bucket">Bucket *</Label>
          <input
            id="target-bucket"
            type="text"
            value={bucket}
            onChange={(e) => setBucket(e.target.value)}
            placeholder="my-releases"
            required
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
        </div>

        {/* Endpoint */}
        <div className="space-y-2">
          <Label htmlFor="target-endpoint">
            Endpoint {requiresEndpoint && "*"}
          </Label>
          <input
            id="target-endpoint"
            type="url"
            value={endpoint}
            onChange={(e) => setEndpoint(e.target.value)}
            placeholder={
              provider === "r2"
                ? "https://<account-id>.r2.cloudflarestorage.com"
                : "https://s3.example.com"
            }
            required={requiresEndpoint}
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
          {provider === "r2" && (
            <p className="text-xs text-slate-500">
              Find this in Cloudflare R2 dashboard under bucket settings
            </p>
          )}
        </div>

        {/* Region */}
        <div className="space-y-2">
          <Label htmlFor="target-region">
            Region {requiresRegion && "(recommended)"}
          </Label>
          <input
            id="target-region"
            type="text"
            value={region}
            onChange={(e) => setRegion(e.target.value)}
            placeholder={provider === "s3" ? "us-east-1" : "auto"}
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
        </div>

        {/* Path Prefix */}
        <div className="space-y-2">
          <Label htmlFor="target-path-prefix">Path Prefix</Label>
          <input
            id="target-path-prefix"
            type="text"
            value={pathPrefix}
            onChange={(e) => setPathPrefix(e.target.value)}
            placeholder="desktop/releases"
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
          <p className="text-xs text-slate-500">
            Uploads go to: bucket/{pathPrefix}/scenario/version/platform/
          </p>
        </div>

        {/* Access Key ID Env */}
        <div className="space-y-2">
          <Label htmlFor="target-access-key-env">Access Key ID Env *</Label>
          <input
            id="target-access-key-env"
            type="text"
            value={accessKeyIdEnv}
            onChange={(e) => setAccessKeyIdEnv(e.target.value)}
            placeholder={provider === "r2" ? "R2_ACCESS_KEY_ID" : "AWS_ACCESS_KEY_ID"}
            required
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm font-mono"
          />
        </div>

        {/* Secret Access Key Env */}
        <div className="space-y-2">
          <Label htmlFor="target-secret-key-env">Secret Access Key Env *</Label>
          <input
            id="target-secret-key-env"
            type="text"
            value={secretAccessKeyEnv}
            onChange={(e) => setSecretAccessKeyEnv(e.target.value)}
            placeholder={provider === "r2" ? "R2_SECRET_ACCESS_KEY" : "AWS_SECRET_ACCESS_KEY"}
            required
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm font-mono"
          />
        </div>

        {/* ACL */}
        <div className="space-y-2">
          <Label htmlFor="target-acl">ACL</Label>
          <Select
            id="target-acl"
            value={acl}
            onChange={(e) => setAcl(e.target.value)}
          >
            <option value="">None (use bucket default)</option>
            <option value="private">private</option>
            <option value="public-read">public-read</option>
            <option value="authenticated-read">authenticated-read</option>
          </Select>
        </div>

        {/* CDN URL */}
        <div className="space-y-2">
          <Label htmlFor="target-cdn-url">CDN URL</Label>
          <input
            id="target-cdn-url"
            type="url"
            value={cdnUrl}
            onChange={(e) => setCdnUrl(e.target.value)}
            placeholder="https://downloads.example.com"
            className="w-full rounded border border-slate-700 bg-slate-950 px-3 py-2 text-sm"
          />
          <p className="text-xs text-slate-500">
            Optional: Public URL for downloads (e.g., CloudFront, Cloudflare CDN)
          </p>
        </div>
      </div>

      {/* Enabled Checkbox */}
      <div className="flex items-center gap-2 pt-2">
        <Checkbox
          id="target-enabled"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
        />
        <Label htmlFor="target-enabled" className="text-sm font-medium">
          Enable this target for uploads
        </Label>
      </div>

      {/* Error Display */}
      {error && (
        <div className="p-3 rounded-lg bg-red-950/50 border border-red-800 text-red-200 text-sm">
          {error}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center justify-end gap-2 pt-4 border-t border-slate-800">
        <Button type="button" variant="outline" onClick={onCancel} disabled={isSubmitting}>
          Cancel
        </Button>
        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting ? "Saving..." : isEditing ? "Update Target" : "Create Target"}
        </Button>
      </div>
    </form>
  );
}
