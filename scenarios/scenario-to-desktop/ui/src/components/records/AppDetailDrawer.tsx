import { useState } from "react";
import {
  Monitor,
  FolderOutput,
  FolderInput,
  Shield,
  LayoutTemplate,
  Trash2,
  MoveRight,
  Video,
  Settings,
  Hammer,
  Info,
} from "lucide-react";
import type { DesktopRecordResponse } from "../../lib/api";
import { buildUrl } from "../../lib/api";
import { formatBytes } from "../../domain/download";
import { Drawer, DrawerBody } from "../ui/drawer";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { SigningBadge } from "./SigningBadge";
import { pathLabel } from "./utils";

type RecordItem = DesktopRecordResponse["records"][number];

interface AppDetailDrawerProps {
  item: RecordItem | null;
  open: boolean;
  onClose: () => void;
  onMove: (recordId: string, target: "destination" | "custom", customPath?: string) => void;
  onDelete: (scenarioName: string) => void;
  movePending: boolean;
  onSwitchTemplate?: (scenarioName: string, templateType?: string) => void;
  onEditSigning?: (scenarioName: string) => void;
  onRebuildWithSigning?: (scenarioName: string) => void;
}

function SectionTitle({ icon: Icon, children }: { icon: typeof Monitor; children: React.ReactNode }) {
  return (
    <h3 className="flex items-center gap-2 text-sm font-semibold text-slate-100">
      <Icon className="h-4 w-4 text-slate-400" />
      {children}
    </h3>
  );
}

function ActionRow({
  icon: Icon,
  title,
  subtitle,
  children,
}: {
  icon: typeof Monitor;
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-slate-800 bg-slate-900/50 px-3 py-2.5">
      <div className="flex items-center gap-2.5 min-w-0">
        <Icon className="h-4 w-4 shrink-0 text-slate-400" />
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-200">{title}</p>
          <p className="text-xs text-slate-500">{subtitle}</p>
        </div>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="space-y-0.5">
      <p className="text-xs text-slate-500">{label}</p>
      <p className="text-sm text-slate-200">{value || "—"}</p>
    </div>
  );
}

export function AppDetailDrawer({
  item,
  open,
  onClose,
  onMove,
  onDelete,
  movePending,
  onSwitchTemplate,
  onEditSigning,
  onRebuildWithSigning,
}: AppDetailDrawerProps) {
  const [showCustomMove, setShowCustomMove] = useState(false);
  const [customPath, setCustomPath] = useState("");

  if (!item) return <Drawer open={open} onClose={onClose} title="App Details"><DrawerBody /></Drawer>;

  const rec = item.record;
  const locationPath = pathLabel(item);
  const destinationPath = rec.destination_path;
  const showDestination = destinationPath && destinationPath !== locationPath;
  const hasSmokeVideo = item.smoke_test_id && item.screen_recording?.recorded;

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={rec.app_display_name || rec.scenario_name}
      description={rec.scenario_name}
    >
      <DrawerBody className="space-y-6">
        {/* Smoke Test Video */}
        {hasSmokeVideo && item.smoke_test_id && (
          <section className="space-y-2">
            <SectionTitle icon={Video}>Smoke Test Video</SectionTitle>
            <video
              controls
              className="w-full rounded-lg border border-slate-700"
              src={buildUrl(`/smoketest/${encodeURIComponent(item.smoke_test_id)}/video`)}
            >
              Your browser does not support the video tag.
            </video>
            <div className="flex gap-4 text-xs text-slate-500">
              {item.screen_recording?.duration_ms != null && (
                <span>Duration: {(item.screen_recording.duration_ms / 1000).toFixed(1)}s</span>
              )}
              {item.screen_recording?.file_size_bytes != null && (
                <span>Size: {formatBytes(item.screen_recording.file_size_bytes)}</span>
              )}
            </div>
          </section>
        )}

        {item.screen_recording?.error && (
          <section className="space-y-2">
            <SectionTitle icon={Video}>Smoke Test Video</SectionTitle>
            <p className="text-xs text-red-300">Recording error: {item.screen_recording.error}</p>
          </section>
        )}

        {/* Status Overview */}
        <section className="space-y-2">
          <SectionTitle icon={Info}>Status</SectionTitle>
          <div className="grid grid-cols-2 gap-3 rounded-lg border border-slate-800 bg-slate-900/50 p-3">
            <InfoItem
              label="Build Status"
              value={item.build_state || item.build_status?.status || "unknown"}
            />
            <div className="space-y-0.5">
              <p className="text-xs text-slate-500">Signing</p>
              <SigningBadge scenarioName={rec.scenario_name} />
            </div>
            <InfoItem label="Deployment" value={rec.deployment_mode || "local"} />
            <InfoItem
              label="Template"
              value={rec.template_type || item.build_status?.template_type || "Unknown"}
            />
            <InfoItem label="Framework" value={rec.framework || item.build_status?.framework} />
            <InfoItem label="Location Mode" value={rec.location_mode || "proper"} />
          </div>
        </section>

        {/* File Location */}
        <section className="space-y-2">
          <SectionTitle icon={FolderOutput}>File Location</SectionTitle>
          <div className="space-y-1.5">
            <p className="text-xs text-slate-500">Current path</p>
            <code className="block rounded-md border border-slate-800 bg-slate-950/70 px-3 py-2 text-xs text-slate-100 break-all">
              {locationPath}
            </code>
          </div>
          {showDestination && (
            <div className="space-y-1.5">
              <p className="text-xs text-slate-500">Destination path</p>
              <code className="block rounded-md border border-slate-800 bg-slate-950/70 px-3 py-2 text-xs text-slate-100 break-all">
                {destinationPath}
              </code>
            </div>
          )}
          <div className="space-y-2">
            <ActionRow
              icon={MoveRight}
              title="Move to destination"
              subtitle="Move your build to its configured destination folder."
            >
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={movePending || (!rec.destination_path) || locationPath === rec.destination_path}
                onClick={() => onMove(rec.id, "destination")}
              >
                Move
              </Button>
            </ActionRow>
            <ActionRow
              icon={FolderInput}
              title="Move to custom path"
              subtitle="Choose a specific folder."
            >
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setShowCustomMove(!showCustomMove)}
              >
                {showCustomMove ? "Cancel" : "Choose"}
              </Button>
            </ActionRow>
            {showCustomMove && (
              <div className="flex gap-2 pl-6">
                <Input
                  value={customPath}
                  onChange={(e) => setCustomPath(e.target.value)}
                  placeholder="/path/to/destination"
                  className="text-xs flex-1"
                />
                <Button
                  type="button"
                  size="sm"
                  disabled={movePending || !customPath}
                  onClick={() => {
                    onMove(rec.id, "custom", customPath);
                    setShowCustomMove(false);
                    setCustomPath("");
                  }}
                >
                  Move
                </Button>
              </div>
            )}
          </div>
        </section>

        {/* Code Signing */}
        {(onEditSigning || onRebuildWithSigning) && (
          <section className="space-y-2">
            <SectionTitle icon={Shield}>Code Signing</SectionTitle>
            <div className="space-y-2">
              {onEditSigning && (
                <ActionRow
                  icon={Settings}
                  title="Configure Signing"
                  subtitle="Set up certificates for Windows, macOS, or Linux."
                >
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => onEditSigning(rec.scenario_name)}
                  >
                    Configure
                  </Button>
                </ActionRow>
              )}
              {onRebuildWithSigning && (
                <ActionRow
                  icon={Hammer}
                  title="Rebuild with Signing"
                  subtitle="Re-run the build with signing applied."
                >
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => onRebuildWithSigning(rec.scenario_name)}
                  >
                    Rebuild
                  </Button>
                </ActionRow>
              )}
            </div>
          </section>
        )}

        {/* Template */}
        {onSwitchTemplate && (
          <section className="space-y-2">
            <SectionTitle icon={LayoutTemplate}>Template</SectionTitle>
            <div className="flex items-center gap-2 mb-2">
              <Badge variant="secondary" className="capitalize">
                {rec.template_type || item.build_status?.template_type || "Unknown"}
              </Badge>
            </div>
            <ActionRow
              icon={LayoutTemplate}
              title="Change Template"
              subtitle="Switch to a different template and rebuild."
            >
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() =>
                  onSwitchTemplate(
                    rec.scenario_name,
                    rec.template_type || item.build_status?.template_type,
                  )
                }
              >
                Change
              </Button>
            </ActionRow>
          </section>
        )}

        {/* Danger Zone */}
        <section className="space-y-2">
          <div className="rounded-lg border border-red-900/50 bg-red-950/10 p-3 space-y-2">
            <h3 className="text-sm font-semibold text-red-200">Danger Zone</h3>
            <ActionRow
              icon={Trash2}
              title="Delete Wrapper"
              subtitle="Permanently removes the Electron platform files. The scenario itself is not affected."
            >
              <Button
                type="button"
                size="sm"
                variant="destructive"
                onClick={() => {
                  if (
                    window.confirm(
                      `Delete desktop build for "${rec.scenario_name}"? This removes platforms/electron for that scenario.`,
                    )
                  ) {
                    onDelete(rec.scenario_name);
                  }
                }}
              >
                Delete
              </Button>
            </ActionRow>
          </div>
        </section>
      </DrawerBody>
    </Drawer>
  );
}
