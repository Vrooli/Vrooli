import { Settings } from "lucide-react";
import { PageHeader } from "../components/layout/PageHeader";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Button } from "../components/ui/button";

export default function SettingsView() {
  return (
    <div>
      <PageHeader
        title="Settings"
        description="Configure tidiness manager thresholds and preferences"
      />

      <div className="space-y-6">
        {/* Thresholds */}
        <Card>
          <CardHeader>
            <CardTitle>File Thresholds</CardTitle>
            <CardDescription>
              Configure line count thresholds for flagging long files
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Long File Threshold (lines)
              </label>
              <Input type="number" defaultValue={500} />
              <p className="text-xs text-slate-500 mt-1">
                Files exceeding this line count will be flagged as long
              </p>
            </div>

            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Max Scans Per File
              </label>
              <Input type="number" defaultValue={10} />
              <p className="text-xs text-slate-500 mt-1">
                Maximum number of times a file can be scanned in a campaign
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Campaign Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Campaign Settings</CardTitle>
            <CardDescription>
              Configure automated campaign behavior
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Max Concurrent Campaigns
              </label>
              <Input type="number" defaultValue={3} />
              <p className="text-xs text-slate-500 mt-1">
                Maximum number of campaigns that can run simultaneously
              </p>
            </div>

            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Default Max Sessions
              </label>
              <Input type="number" defaultValue={10} />
              <p className="text-xs text-slate-500 mt-1">
                Default maximum sessions for new campaigns
              </p>
            </div>

            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Default Files Per Session
              </label>
              <Input type="number" defaultValue={5} />
              <p className="text-xs text-slate-500 mt-1">
                Default number of files to analyze per session
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Scan Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Scan Settings</CardTitle>
            <CardDescription>
              Configure scan behavior and timeouts
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                Light Scan Timeout (seconds)
              </label>
              <Input type="number" defaultValue={120} />
              <p className="text-xs text-slate-500 mt-1">
                Maximum time to wait for light scan completion
              </p>
            </div>

            <div>
              <label className="text-sm text-slate-400 mb-2 block">
                AI Scan Batch Size
              </label>
              <Input type="number" defaultValue={10} />
              <p className="text-xs text-slate-500 mt-1">
                Maximum files to process in a single AI scan batch
              </p>
            </div>
          </CardContent>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Button variant="outline">Reset to Defaults</Button>
          <Button variant="primary">Save Changes</Button>
        </div>
      </div>
    </div>
  );
}
