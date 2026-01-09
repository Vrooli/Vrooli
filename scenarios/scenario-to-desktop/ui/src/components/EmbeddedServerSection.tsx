import { Input } from "./ui/input";
import { Label } from "./ui/label";

interface EmbeddedServerSectionProps {
  serverPort: number;
  onServerPortChange: (port: number) => void;
  localServerPath: string;
  onLocalServerPathChange: (path: string) => void;
  localApiEndpoint: string;
  onLocalApiEndpointChange: (endpoint: string) => void;
}

export function EmbeddedServerSection({
  serverPort,
  onServerPortChange,
  localServerPath,
  onLocalServerPathChange,
  localApiEndpoint,
  onLocalApiEndpointChange
}: EmbeddedServerSectionProps) {
  return (
    <div className="rounded-lg border border-yellow-800 bg-yellow-950/10 p-4 space-y-3">
      <p className="text-sm text-yellow-200">
        Embedded servers require more manual work. Make sure the scenario's API can run within the wrapper (Node script or executable) and that resource usage fits the target machine.
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <Label htmlFor="serverPort">Server Port</Label>
          <Input
            id="serverPort"
            type="number"
            value={serverPort}
            onChange={(e) => onServerPortChange(Number(e.target.value))}
            min={1}
          />
        </div>
        <div>
          <Label htmlFor="localServerPath">Server Entry</Label>
          <Input
            id="localServerPath"
            value={localServerPath}
            onChange={(e) => onLocalServerPathChange(e.target.value)}
            placeholder="ui/server.js"
          />
        </div>
      </div>
      <div>
        <Label htmlFor="localApiEndpoint">API Endpoint</Label>
        <Input
          id="localApiEndpoint"
          value={localApiEndpoint}
          onChange={(e) => onLocalApiEndpointChange(e.target.value)}
          placeholder="http://localhost:3001/api"
        />
      </div>
    </div>
  );
}
