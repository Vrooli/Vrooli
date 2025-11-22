import { useNavigate } from "react-router-dom";
import { Code2, Tag, Package } from "lucide-react";
import { useComponents } from "../lib/api-client";
import { useUIStore } from "../store/ui-store";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";

export function Library() {
  const navigate = useNavigate();
  const { searchQuery, selectedCategory } = useUIStore();
  const { data: components, isLoading, error } = useComponents(searchQuery, selectedCategory || undefined);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mb-4 inline-flex h-12 w-12 animate-spin items-center justify-center rounded-full border-4 border-blue-600 border-t-transparent" />
          <p className="text-slate-400">Loading components...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <Code2 className="mx-auto mb-4 h-12 w-12 text-red-500" />
          <h3 className="mb-2 text-lg font-semibold">Failed to load components</h3>
          <p className="text-sm text-slate-400">{(error as Error).message}</p>
        </div>
      </div>
    );
  }

  if (!components || components.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <Package className="mx-auto mb-4 h-12 w-12 text-slate-600" />
          <h3 className="mb-2 text-lg font-semibold">No components found</h3>
          <p className="mb-4 text-sm text-slate-400">
            {searchQuery
              ? `No components match "${searchQuery}"`
              : "Get started by creating your first component"}
          </p>
          <Button onClick={() => navigate("/editor/new")}>Create Component</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-y-auto p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Component Library</h2>
          <p className="text-sm text-slate-400">
            {components.length} component{components.length !== 1 ? "s" : ""} available
          </p>
        </div>
        <Button onClick={() => navigate("/editor/new")}>
          <Code2 className="mr-2 h-4 w-4" />
          New Component
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {components.map((component) => (
          <Card
            key={component.id}
            className="cursor-pointer transition-all hover:border-blue-500/50 hover:shadow-lg hover:shadow-blue-500/10"
            onClick={() => navigate(`/editor/${component.id}`)}
          >
            <CardHeader>
              <CardTitle className="flex items-center justify-between">
                <span className="truncate">{component.displayName}</span>
                <Badge variant="secondary" className="ml-2 shrink-0">
                  v{component.version}
                </Badge>
              </CardTitle>
              <CardDescription className="line-clamp-2">
                {component.description}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {component.category && (
                  <div className="flex items-center text-xs text-slate-400">
                    <Tag className="mr-1 h-3 w-3" />
                    {component.category}
                  </div>
                )}
                {component.tags && component.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {component.tags.slice(0, 3).map((tag) => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                    {component.tags.length > 3 && (
                      <Badge variant="outline" className="text-xs">
                        +{component.tags.length - 3}
                      </Badge>
                    )}
                  </div>
                )}
                <div className="flex items-center space-x-2 text-xs text-slate-500">
                  <Package className="h-3 w-3" />
                  <span className="truncate">{component.libraryId}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
