import { useState, useRef, useEffect } from "react";
import { Send, Bot, User, X, Sparkles } from "lucide-react";
import { useUIStore } from "../store/ui-store";
import { useAIChat } from "../lib/api-client";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Card } from "./ui/card";
import { Badge } from "./ui/badge";

interface AIChatProps {
  componentCode?: string;
  onCodeSuggestion?: (code: string) => void;
}

export function AIChat({ componentCode, onCodeSuggestion }: AIChatProps) {
  const {
    aiPanelOpen,
    toggleAIPanel,
    aiMessages,
    addAIMessage,
    selectedElements,
    clearSelectedElements,
  } = useUIStore();

  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const chatMutation = useAIChat();

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [aiMessages]);

  const handleSend = async () => {
    if (!input.trim()) return;

    const userMessage = {
      id: `msg-${Date.now()}`,
      role: "user" as const,
      content: input,
      timestamp: new Date().toISOString(),
      attachedElements: selectedElements.map((el) => el.selector),
    };

    addAIMessage(userMessage);
    setInput("");
    clearSelectedElements();

    try {
      const response = await chatMutation.mutateAsync({
        message: input,
        context: {
          componentCode,
          selectedElements: selectedElements,
        },
      });

      const assistantMessage = {
        id: `msg-${Date.now()}-ai`,
        role: "assistant" as const,
        content: response.response,
        timestamp: new Date().toISOString(),
        codeSnippet: response.suggestions?.[0],
      };

      addAIMessage(assistantMessage);

      if (response.suggestions?.[0] && onCodeSuggestion) {
        onCodeSuggestion(response.suggestions[0]);
      }
    } catch (error) {
      const errorMessage = {
        id: `msg-${Date.now()}-error`,
        role: "assistant" as const,
        content: `Error: ${(error as Error).message}`,
        timestamp: new Date().toISOString(),
      };
      addAIMessage(errorMessage);
    }
  };

  if (!aiPanelOpen) {
    return (
      <Button
        onClick={toggleAIPanel}
        className="fixed bottom-6 right-6 h-14 w-14 rounded-full shadow-lg"
        size="icon"
      >
        <Sparkles className="h-6 w-6" />
      </Button>
    );
  }

  return (
    <Card className="fixed bottom-6 right-6 flex h-[600px] w-96 flex-col shadow-2xl">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-white/10 p-4">
        <div className="flex items-center space-x-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-600">
            <Bot className="h-4 w-4" />
          </div>
          <div>
            <h3 className="font-semibold">AI Assistant</h3>
            <p className="text-xs text-slate-400">Powered by OpenRouter</p>
          </div>
        </div>
        <Button variant="ghost" size="icon" onClick={toggleAIPanel}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* Selected Elements */}
      {selectedElements.length > 0 && (
        <div className="border-b border-white/10 p-3">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-xs font-medium text-slate-400">Selected Elements</span>
            <Button
              variant="ghost"
              size="sm"
              className="h-6 text-xs"
              onClick={clearSelectedElements}
            >
              Clear
            </Button>
          </div>
          <div className="flex flex-wrap gap-1">
            {selectedElements.map((el, idx) => (
              <Badge key={idx} variant="secondary" className="text-xs">
                {el.selector}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {aiMessages.length === 0 && (
          <div className="flex h-full items-center justify-center text-center">
            <div className="text-sm text-slate-400">
              <Bot className="mx-auto mb-2 h-8 w-8 text-slate-600" />
              <p>Ask me to help refactor, style, or improve your component!</p>
            </div>
          </div>
        )}

        {aiMessages.map((message) => (
          <div
            key={message.id}
            className={`flex ${message.role === "user" ? "justify-end" : "justify-start"}`}
          >
            <div
              className={`max-w-[80%] rounded-lg p-3 ${
                message.role === "user"
                  ? "bg-blue-600 text-white"
                  : "bg-slate-800 text-slate-100"
              }`}
            >
              <div className="mb-1 flex items-center space-x-2">
                {message.role === "assistant" ? (
                  <Bot className="h-4 w-4" />
                ) : (
                  <User className="h-4 w-4" />
                )}
                <span className="text-xs opacity-70">
                  {new Date(message.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <p className="text-sm">{message.content}</p>
              {message.codeSnippet && (
                <pre className="mt-2 overflow-x-auto rounded bg-black/20 p-2 text-xs">
                  <code>{message.codeSnippet}</code>
                </pre>
              )}
            </div>
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="border-t border-white/10 p-4">
        <div className="flex space-x-2">
          <Input
            placeholder="Ask for help with your component..."
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
            disabled={chatMutation.isPending}
          />
          <Button
            onClick={handleSend}
            disabled={!input.trim() || chatMutation.isPending}
            size="icon"
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </Card>
  );
}
