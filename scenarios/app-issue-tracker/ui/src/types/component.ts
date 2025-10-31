export interface Component {
  type: "scenario" | "resource";
  id: string;
  name: string;
  display_name: string;
  status: "running" | "stopped" | "unknown";
  description?: string;
}

export interface ComponentsResponse {
  scenarios: Component[];
  resources: Component[];
}
