import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Panel, PanelGroup, PanelResizeHandle } from "react-resizable-panels";
import {
  Save,
  Play,
  LayoutGrid,
  MousePointer2,
  Sparkles,
  ArrowLeft,
  Plus,
} from "lucide-react";
import { useComponent, useComponentContent, useUpdateComponentContent } from "../lib/api-client";
import { useUIStore } from "../store/ui-store";
import { CodeEditor } from "../components/CodeEditor";
import { EmulatorFrame } from "../components/EmulatorFrame";
import { AIChat } from "../components/AIChat";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "../components/ui/dialog";
import { VIEWPORT_PRESETS } from "../types";

export function Editor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const {
    emulatorFrames,
    addEmulatorFrame,
    removeEmulatorFrame,
    selectionMode,
    toggleSelectionMode,
    addSelectedElement,
    toggleAIPanel,
  } = useUIStore();

  const [code, setCode] = useState("");
  const [hasChanges, setHasChanges] = useState(false);

  const { data: component, isLoading: componentLoading } = useComponent(id || "");
  const { data: contentData, isLoading: contentLoading } = useComponentContent(id || "");
  const updateContent = useUpdateComponentContent();

  useEffect(() => {
    if (contentData?.content) {
      setCode(contentData.content);
      setHasChanges(false);
    }
  }, [contentData]);

  const handleSave = async () => {
    if (!id) return;
    try {
      await updateContent.mutateAsync({ id, content: code });
      setHasChanges(false);
    } catch (error) {
      console.error("Failed to save:", error);
    }
  };

  const handleCodeChange = (value: string | undefined) => {
    setCode(value || "");
    setHasChanges(true);
  };

  const handleElementSelect = (selector: string) => {
    addSelectedElement({
      frameId: emulatorFrames[0]?.id || "",
      selector,
      xpath: "",
      tagName: selector.split(/[#.]/)[0] || "div",
      attributes: {},
    });
  };

  if (componentLoading || contentLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mb-4 inline-flex h-12 w-12 animate-spin items-center justify-center rounded-full border-4 border-blue-600 border-t-transparent" />
          <p className="text-slate-400">Loading component...</p>
        </div>
      </div>
    );
  }

  if (!component && id !== "new") {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <h3 className="mb-2 text-lg font-semibold">Component not found</h3>
          <Button onClick={() => navigate("/")}>Back to Library</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      {/* Toolbar */}
      <div className="flex items-center justify-between border-b border-white/10 bg-slate-900/50 px-6 py-3">
        <div className="flex items-center space-x-4">
          <Button variant="ghost" size="icon" onClick={() => navigate("/")}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <h2 className="text-lg font-semibold">
              {component?.displayName || "New Component"}
            </h2>
            {component && (
              <p className="text-xs text-slate-400">
                {component.libraryId} • v{component.version}
              </p>
            )}
          </div>
          {hasChanges && (
            <Badge variant="warning" className="animate-pulse">
              Unsaved Changes
            </Badge>
          )}
        </div>

        <div className="flex items-center space-x-2">
          <Button
            variant="outline"
            size="sm"
            onClick={toggleSelectionMode}
            className={selectionMode ? "bg-blue-600 text-white" : ""}
          >
            <MousePointer2 className="mr-2 h-4 w-4" />
            {selectionMode ? "Selection Active" : "Select Elements"}
          </Button>

          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline" size="sm">
                <LayoutGrid className="mr-2 h-4 w-4" />
                Add Frame
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add Viewport Frame</DialogTitle>
                <DialogDescription>
                  Choose a viewport preset to add to the emulator
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-3">
                {VIEWPORT_PRESETS.map((preset) => (
                  <Button
                    key={preset.id}
                    variant="outline"
                    className="justify-start"
                    onClick={() => {
                      addEmulatorFrame(preset);
                    }}
                  >
                    <Plus className="mr-2 h-4 w-4" />
                    {preset.name} ({preset.width} × {preset.height})
                  </Button>
                ))}
              </div>
            </DialogContent>
          </Dialog>

          <Button variant="outline" size="sm" onClick={toggleAIPanel}>
            <Sparkles className="mr-2 h-4 w-4" />
            AI Assistant
          </Button>

          <Button size="sm" onClick={handleSave} disabled={!hasChanges || updateContent.isPending}>
            <Save className="mr-2 h-4 w-4" />
            {updateContent.isPending ? "Saving..." : "Save"}
          </Button>
        </div>
      </div>

      {/* Editor + Preview */}
      <div className="flex-1 overflow-hidden">
        <PanelGroup direction="horizontal">
          {/* Code Editor */}
          <Panel defaultSize={50} minSize={30}>
            <div className="h-full border-r border-white/10">
              <CodeEditor value={code} onChange={handleCodeChange} language="typescript" />
            </div>
          </Panel>

          <PanelResizeHandle className="w-1 bg-white/5 hover:bg-blue-500/50 transition-colors" />

          {/* Preview Panels */}
          <Panel defaultSize={50} minSize={30}>
            <div className="h-full overflow-auto bg-slate-950 p-6">
              <div className="grid gap-6">
                {emulatorFrames.length === 0 ? (
                  <div className="flex h-full items-center justify-center text-center">
                    <div>
                      <LayoutGrid className="mx-auto mb-4 h-12 w-12 text-slate-600" />
                      <h3 className="mb-2 font-semibold">No Preview Frames</h3>
                      <p className="mb-4 text-sm text-slate-400">
                        Add a viewport frame to see your component in action
                      </p>
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button>
                            <Plus className="mr-2 h-4 w-4" />
                            Add Frame
                          </Button>
                        </DialogTrigger>
                        <DialogContent>
                          <DialogHeader>
                            <DialogTitle>Add Viewport Frame</DialogTitle>
                            <DialogDescription>
                              Choose a viewport preset to add to the emulator
                            </DialogDescription>
                          </DialogHeader>
                          <div className="grid gap-3">
                            {VIEWPORT_PRESETS.map((preset) => (
                              <Button
                                key={preset.id}
                                variant="outline"
                                className="justify-start"
                                onClick={() => {
                                  addEmulatorFrame(preset);
                                }}
                              >
                                <Plus className="mr-2 h-4 w-4" />
                                {preset.name} ({preset.width} × {preset.height})
                              </Button>
                            ))}
                          </div>
                        </DialogContent>
                      </Dialog>
                    </div>
                  </div>
                ) : (
                  emulatorFrames.map((frame) => (
                    <EmulatorFrame
                      key={frame.id}
                      frame={frame}
                      componentCode={code}
                      onRemove={() => removeEmulatorFrame(frame.id)}
                      onElementSelect={selectionMode ? handleElementSelect : undefined}
                    />
                  ))
                )}
              </div>
            </div>
          </Panel>
        </PanelGroup>
      </div>

      {/* AI Chat */}
      <AIChat
        componentCode={code}
        onCodeSuggestion={(suggestedCode) => {
          setCode(suggestedCode);
          setHasChanges(true);
        }}
      />
    </div>
  );
}
