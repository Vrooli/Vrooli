import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  X,
  ChevronRight,
  ChevronLeft,
  GripHorizontal,
  Sparkles,
  Play,
  CheckCircle2,
  Circle,
  SkipForward,
  Globe,
  Plus,
  Bot,
  Palette,
  FileText,
  FolderTree,
  Clock,
  Loader2,
  ArrowRight,
  RotateCcw,
} from "lucide-react";

const TOUR_STORAGE_KEY = "browser-automation-studio-tour-completed";
const TOUR_STEP_KEY = "browser-automation-studio-tour-step";
const TOUR_ACTIVE_KEY = "browser-automation-studio-tour-active";
const TOUR_PAUSED_KEY = "browser-automation-studio-tour-paused";
const TOUR_VERSION = "4"; // Bump to show tour again after major updates

// Custom events for controlling the tour from anywhere
const TOUR_RESET_EVENT = "guided-tour-reset";
const TOUR_OPEN_EVENT = "guided-tour-open";

export interface TourStep {
  id: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  /** CSS selector for the element to anchor to */
  anchorSelector?: string;
  /** Which side of the anchor element to position on */
  anchorPosition?: "top" | "bottom" | "left" | "right";
  /** Action hint shown at the bottom of the step */
  actionHint?: string;
  /** If true, wait for user to interact with the target before auto-advancing */
  waitForInteraction?: boolean;
  /** Selector to watch for click to advance */
  advanceOnClick?: string;
  /** Called when this step becomes active */
  onEnter?: () => void;
  /** Called when leaving this step (for cleanup like closing modals) */
  onExit?: () => void;
  /** Action to auto-perform if user presses Next without completing the required interaction */
  autoAction?: () => Promise<void>;
  /** URL pattern this step requires (string for startsWith, RegExp for pattern match) */
  requiredUrl?: string | RegExp;
  /** Auto-navigate to this URL on step entry if not already there */
  navigateTo?: string;
}

// ============================================================================
// Default Tour Steps (17 steps focused on recording workflow)
// ============================================================================

let cachedDefaultTourSteps: TourStep[] | null = null;

export function getDefaultTourSteps(): TourStep[] {
  if (cachedDefaultTourSteps) {
    return cachedDefaultTourSteps;
  }

  cachedDefaultTourSteps = [
    // Step 1: Welcome / Getting Started
    {
      id: "welcome",
      title: "Welcome to Browser Automation Studio",
      description:
        "Let's take a quick tour to help you create your first automated workflow. You can drag this panel anywhere on the screen.",
      icon: <Sparkles size={24} className="text-amber-400" />,
    },

    // Step 2: Explain workflows and 3 creation methods
    // Note: Feature cards only show in WelcomeHero when no projects exist,
    // but the step still works - just without the highlight if cards aren't visible.
    {
      id: "explain-methods",
      title: "Three Ways to Create Workflows",
      description:
        "You can create workflows by Recording browser actions, using AI to navigate for you, or building visually with drag-and-drop. We'll start with Recording - the easiest way to get started.",
      icon: <Sparkles size={24} className="text-purple-400" />,
      anchorSelector: "[data-testid='dashboard-feature-cards']",
      anchorPosition: "bottom",
    },

    // Step 3: Open Record Mode
    // Recording sessions start automatically when navigating to /record/new
    {
      id: "open-record-mode",
      title: "Open Record Mode",
      description:
        "Let's open Record Mode to start capturing browser actions. When you navigate to a page, every click, scroll, and input will be recorded automatically.",
      icon: <Circle size={24} className="text-red-400 fill-red-400" />,
      actionHint: "Press Next to open Record Mode",
      requiredUrl: "/",
      navigateTo: "/record/new",
    },

    // Step 4: Enter URL
    {
      id: "enter-url",
      title: "Enter a URL",
      description:
        "Type or paste the URL of the website you want to automate. We'll use vrooli.com as an example - it's our landing page!",
      icon: <Globe size={24} className="text-blue-400" />,
      anchorSelector: "[data-testid='browser-url-input']",
      anchorPosition: "bottom",
      actionHint: "Enter a URL (or we'll use vrooli.com)",
      requiredUrl: /^\/record/,
      waitForInteraction: true,
      autoAction: async () => {
        const input = document.querySelector(
          "[data-testid='browser-url-input']"
        ) as HTMLInputElement;
        if (input && !input.value.trim()) {
          // Focus the input first
          input.focus();
          await new Promise((r) => setTimeout(r, 100));

          // For React controlled inputs, we need to use the native value setter
          // to properly trigger React's onChange handler
          const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
            window.HTMLInputElement.prototype,
            "value"
          )?.set;
          if (nativeInputValueSetter) {
            nativeInputValueSetter.call(input, "vrooli.com");
          } else {
            // Fallback for older browsers
            input.value = "vrooli.com";
          }

          // Dispatch input event to trigger React state update
          input.dispatchEvent(new Event("input", { bubbles: true }));

          // Wait for React to process the input
          await new Promise((r) => setTimeout(r, 200));

          // Press Enter to navigate
          input.dispatchEvent(
            new KeyboardEvent("keydown", {
              key: "Enter",
              code: "Enter",
              bubbles: true,
            })
          );
        }
      },
    },

    // Step 5: Observe timeline
    {
      id: "observe-timeline",
      title: "Watch the Timeline",
      description:
        "As you scroll and click around the page, each action appears in the timeline on the left. Try interacting with the page to see how actions are captured!",
      icon: <FileText size={24} className="text-green-400" />,
      anchorSelector: "[data-testid='sidebar-timeline-tab']",
      anchorPosition: "right",
      actionHint: "Scroll or click on the page to see actions appear",
      requiredUrl: /^\/record/,
    },

    // Step 6: Add Step button
    {
      id: "add-step",
      title: "Add Advanced Steps",
      description:
        "Use 'Add Step' to insert complex behaviors like data extraction, assertions, and branching. These are great for building composable workflows and testing website behavior.",
      icon: <Plus size={24} className="text-blue-400" />,
      anchorSelector: "[data-testid='timeline-add-step-button']",
      anchorPosition: "right",
      actionHint: "Click to see available step types",
      requiredUrl: /^\/record/,
      onExit: () => {
        // Close the InsertNodeModal if it's open
        const overlay = document.querySelector(
          "[data-testid='responsive-dialog-overlay']"
        );
        if (overlay) {
          (overlay as HTMLElement).click();
        }
        // Note: Don't dispatch Escape key here - it triggers the tour's
        // own keyboard handler and causes an infinite loop
      },
    },

    // Step 7: AI Navigation button
    {
      id: "ai-navigation",
      title: "AI Navigation",
      description:
        "Let AI navigate for you! Describe what you want to accomplish in natural language, and AI will perform the actions automatically.",
      icon: <Bot size={24} className="text-purple-400" />,
      anchorSelector: "[data-testid='sidebar-auto-tab']",
      anchorPosition: "right",
      actionHint: "Click to see AI navigation",
      requiredUrl: /^\/record/,
      onEnter: () => {
        // Switch to Auto tab
        const autoTab = document.querySelector(
          "[data-testid='sidebar-auto-tab']"
        );
        if (autoTab) {
          (autoTab as HTMLElement).click();
        }
      },
      onExit: () => {
        // Switch back to Timeline tab
        const timelineTab = document.querySelector(
          "[data-testid='sidebar-timeline-tab']"
        );
        if (timelineTab) {
          (timelineTab as HTMLElement).click();
        }
      },
    },

    // Step 8: Replay Style button
    {
      id: "replay-style",
      title: "Replay Style Settings",
      description:
        "Enable replay styling to create polished exports for marketing, demos, and documentation. You can customize the style in Settings anytime.",
      icon: <Palette size={24} className="text-pink-400" />,
      anchorSelector: "[data-testid='browser-replay-style-button']",
      anchorPosition: "bottom",
      actionHint: "Click to toggle replay styling",
      requiredUrl: /^\/record/,
      waitForInteraction: true,
      advanceOnClick: "[data-testid='browser-replay-style-button']",
      autoAction: async () => {
        const button = document.querySelector(
          "[data-testid='browser-replay-style-button']"
        ) as HTMLButtonElement;
        if (button) {
          const wasOn = button.getAttribute("aria-checked") === "true";
          // Always click to trigger advanceOnClick and advance the step
          button.click();
          // If it was already on, clicking turned it off - click again to restore
          if (wasOn) {
            await new Promise((r) => setTimeout(r, 50));
            button.click();
          }
        }
      },
    },

    // Step 9: Create Workflow button
    {
      id: "create-workflow",
      title: "Create Your Workflow",
      description:
        "When you're done recording, click 'Create Workflow' to save your recorded actions as a reusable workflow.",
      icon: <ArrowRight size={24} className="text-blue-400" />,
      anchorSelector: "[data-testid='timeline-create-workflow-button']",
      anchorPosition: "top",
      actionHint: "Click 'Create Workflow' to continue",
      requiredUrl: /^\/record/,
      waitForInteraction: true,
      advanceOnClick: "[data-testid='timeline-create-workflow-button']",
      autoAction: async () => {
        const button = document.querySelector(
          "[data-testid='timeline-create-workflow-button']"
        );
        if (button) {
          (button as HTMLElement).click();
        }
      },
    },

    // Step 10: Fill form and submit
    {
      id: "fill-form",
      title: "Name Your Workflow",
      description:
        "Give your workflow a descriptive name and select a project to save it in. Then click 'Generate Workflow' to create it.",
      icon: <FileText size={24} className="text-blue-400" />,
      anchorSelector: "[data-testid='workflow-creation-name-input']",
      anchorPosition: "left",
      actionHint: "Enter a name and click 'Generate Workflow'",
      requiredUrl: /^\/record/,
      waitForInteraction: true,
      advanceOnClick: "[data-testid='workflow-creation-submit-button']",
      autoAction: async () => {
        // Fill in a random name if empty
        const nameInput = document.querySelector(
          "[data-testid='workflow-creation-name-input']"
        ) as HTMLInputElement;
        if (nameInput && !nameInput.value.trim()) {
          nameInput.focus();
          await new Promise((r) => setTimeout(r, 100));

          // For React controlled inputs, use native value setter
          const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
            window.HTMLInputElement.prototype,
            "value"
          )?.set;
          const workflowName = `My Workflow ${Date.now().toString().slice(-4)}`;
          if (nativeInputValueSetter) {
            nativeInputValueSetter.call(nameInput, workflowName);
          } else {
            nameInput.value = workflowName;
          }
          nameInput.dispatchEvent(new Event("input", { bubbles: true }));
        }

        // Wait for React to process, then click submit
        await new Promise((r) => setTimeout(r, 300));
        const submitButton = document.querySelector(
          "[data-testid='workflow-creation-submit-button']"
        );
        if (submitButton && !(submitButton as HTMLButtonElement).disabled) {
          (submitButton as HTMLElement).click();
        }
      },
    },

    // Step 11: Project page - file tree
    {
      id: "project-page",
      title: "Your Project",
      description:
        "Welcome to your project! The file tree on the left shows all your workflows organized in folders. Click any workflow to see its details.",
      icon: <FolderTree size={24} className="text-amber-400" />,
      anchorSelector: "[data-testid='project-file-tree']",
      anchorPosition: "right",
      requiredUrl: /^\/projects/,
    },

    // Step 12: Workflow details panel
    {
      id: "workflow-panel",
      title: "Workflow Details",
      description:
        "This panel shows the selected workflow's details - its steps, execution stats, and settings. You can quickly run, edit, or schedule workflows from here.",
      icon: <FileText size={24} className="text-blue-400" />,
      anchorSelector: "[data-testid='workflow-preview-pane']",
      anchorPosition: "left",
      requiredUrl: /^\/projects/,
    },

    // Step 13: Run button - introduction
    {
      id: "run-button",
      title: "Run Your Workflow",
      description:
        "Click 'Run' to execute your workflow. You'll see a live preview of each step as it runs, just like during recording!",
      icon: <Play size={24} className="text-green-400" />,
      anchorSelector: "[data-testid='workflow-preview-run-button']",
      anchorPosition: "left",
      actionHint: "We'll try it after a few more tips",
      requiredUrl: /^\/projects/,
    },

    // Step 14: Open Editor
    {
      id: "open-editor",
      title: "Visual Editor",
      description:
        "For more advanced editing, click 'Open Editor' to use the visual workflow builder. Add conditional logic, loops, and complex branching.",
      icon: <Sparkles size={24} className="text-purple-400" />,
      anchorSelector: "[data-testid='workflow-preview-open-editor-button']",
      anchorPosition: "left",
      requiredUrl: /^\/projects/,
    },

    // Step 15: Schedule section
    {
      id: "schedule-section",
      title: "Schedule Automation",
      description:
        "Set up schedules to run your workflows automatically - daily reports, hourly checks, or custom cron expressions. Perfect for monitoring and automation.",
      icon: <Clock size={24} className="text-cyan-400" />,
      anchorSelector: "[data-testid='workflow-preview-schedule-section']",
      anchorPosition: "left",
      requiredUrl: /^\/projects/,
    },

    // Step 16: Run the workflow
    {
      id: "run-workflow",
      title: "Try Running It!",
      description:
        "Go ahead and click 'Run' to see your workflow in action. Watch as each recorded step is replayed automatically. After running, you can export the replay as video, GIF, or JSON!",
      icon: <Play size={24} className="text-green-400" />,
      anchorSelector: "[data-testid='workflow-preview-run-button']",
      anchorPosition: "left",
      actionHint: "Click 'Run' to execute your workflow",
      requiredUrl: /^\/projects/,
      waitForInteraction: true,
      advanceOnClick: "[data-testid='workflow-preview-run-button']",
    },

    // Step 17: Complete
    {
      id: "complete",
      title: "You're All Set!",
      description:
        "Congratulations! You now know the basics of Browser Automation Studio. Explore AI-assisted recording, visual editing, and scheduling to unlock even more power. Happy automating!",
      icon: <CheckCircle2 size={24} className="text-emerald-400" />,
    },
  ];

  return cachedDefaultTourSteps;
}

// ============================================================================
// Component Types
// ============================================================================

interface GuidedTourProps {
  isOpen: boolean;
  onClose: () => void;
  onComplete?: () => void;
  steps?: TourStep[];
  startAtStep?: number;
}

interface Position {
  x: number;
  y: number;
}

type AnchorPosition = "top" | "bottom" | "left" | "right";

// ============================================================================
// Helper Functions
// ============================================================================

/** Check if a position overlaps with the anchor element */
function overlapsAnchor(
  panelRect: { x: number; y: number; width: number; height: number },
  anchorRect: DOMRect
): boolean {
  const panelRight = panelRect.x + panelRect.width;
  const panelBottom = panelRect.y + panelRect.height;

  return !(
    panelRect.x > anchorRect.right ||
    panelRight < anchorRect.left ||
    panelRect.y > anchorRect.bottom ||
    panelBottom < anchorRect.top
  );
}

/** Compute panel position for a given anchor position */
function computePositionForSide(
  anchorRect: DOMRect,
  panelWidth: number,
  panelHeight: number,
  side: AnchorPosition,
  gap: number = 12
): Position {
  switch (side) {
    case "top":
      return {
        x: anchorRect.left + anchorRect.width / 2 - panelWidth / 2,
        y: anchorRect.top - panelHeight - gap,
      };
    case "bottom":
      return {
        x: anchorRect.left + anchorRect.width / 2 - panelWidth / 2,
        y: anchorRect.bottom + gap,
      };
    case "left":
      return {
        x: anchorRect.left - panelWidth - gap,
        y: anchorRect.top + anchorRect.height / 2 - panelHeight / 2,
      };
    case "right":
    default:
      return {
        x: anchorRect.right + gap,
        y: anchorRect.top + anchorRect.height / 2 - panelHeight / 2,
      };
  }
}

/** Clamp position to viewport bounds */
function clampToViewport(
  pos: Position,
  panelWidth: number,
  panelHeight: number,
  padding: number = 16
): Position {
  return {
    x: Math.max(
      padding,
      Math.min(pos.x, window.innerWidth - panelWidth - padding)
    ),
    y: Math.max(
      padding,
      Math.min(pos.y, window.innerHeight - panelHeight - padding)
    ),
  };
}

/** Find a non-overlapping position for the panel */
function computeNonOverlappingPosition(
  anchorRect: DOMRect,
  panelWidth: number,
  panelHeight: number,
  preferredPosition: AnchorPosition
): Position {
  const positions: AnchorPosition[] = [
    preferredPosition,
    "right",
    "left",
    "bottom",
    "top",
  ];

  for (const pos of positions) {
    const candidate = computePositionForSide(
      anchorRect,
      panelWidth,
      panelHeight,
      pos
    );
    const clamped = clampToViewport(candidate, panelWidth, panelHeight);

    if (
      !overlapsAnchor(
        { ...clamped, width: panelWidth, height: panelHeight },
        anchorRect
      )
    ) {
      return clamped;
    }
  }

  // Fallback: corner position
  return { x: 20, y: 80 };
}

/** Check if current URL matches the required pattern */
function urlMatches(
  currentPath: string,
  requiredUrl: string | RegExp | undefined
): boolean {
  if (!requiredUrl) return true;

  if (typeof requiredUrl === "string") {
    return currentPath.startsWith(requiredUrl);
  }

  return requiredUrl.test(currentPath);
}

// ============================================================================
// Main Component
// ============================================================================

export function GuidedTour({
  isOpen,
  onClose,
  onComplete,
  steps,
  startAtStep = 0,
}: GuidedTourProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const resolvedSteps = steps ?? getDefaultTourSteps();

  // State
  const [currentStepIndex, setCurrentStepIndex] = useState(() => {
    try {
      const savedStep = sessionStorage.getItem(TOUR_STEP_KEY);
      if (savedStep !== null) {
        const parsed = parseInt(savedStep, 10);
        if (!isNaN(parsed) && parsed >= 0 && parsed < resolvedSteps.length) {
          return parsed;
        }
      }
    } catch {
      // sessionStorage unavailable
    }
    return startAtStep;
  });
  const [position, setPosition] = useState<Position>({ x: 20, y: 80 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragOffset, setDragOffset] = useState<Position>({ x: 0, y: 0 });
  const [anchorRect, setAnchorRect] = useState<DOMRect | null>(null);
  const [pendingAutoAction, setPendingAutoAction] = useState(false);
  const [isPaused, setIsPaused] = useState(false);

  // Refs
  const panelRef = useRef<HTMLDivElement>(null);
  const hasInitializedRef = useRef(false);
  const previousStepRef = useRef<number | null>(null);

  // Derived state
  const currentStep = resolvedSteps[currentStepIndex];
  const isFirstStep = currentStepIndex === 0;
  const isLastStep = currentStepIndex === resolvedSteps.length - 1;
  const isOffTrack =
    currentStep?.requiredUrl &&
    !urlMatches(location.pathname, currentStep.requiredUrl);

  // ============================================================================
  // Persistence
  // ============================================================================

  useEffect(() => {
    if (!isOpen) return;
    try {
      sessionStorage.setItem(TOUR_STEP_KEY, currentStepIndex.toString());
      sessionStorage.setItem(TOUR_ACTIVE_KEY, "true");
    } catch {
      // sessionStorage unavailable
    }
  }, [currentStepIndex, isOpen]);

  // ============================================================================
  // Listen for Reset Event (allows resetting from any component)
  // ============================================================================

  useEffect(() => {
    const handleReset = () => {
      setCurrentStepIndex(0);
      setIsPaused(false);
      previousStepRef.current = null;
    };

    window.addEventListener(TOUR_RESET_EVENT, handleReset);
    return () => window.removeEventListener(TOUR_RESET_EVENT, handleReset);
  }, []);

  // ============================================================================
  // Lifecycle Hooks (onEnter / onExit)
  // ============================================================================

  useEffect(() => {
    if (!isOpen || isPaused) return;

    const prevStep = previousStepRef.current;
    const newStep = currentStepIndex;

    // Call onExit for previous step
    if (prevStep !== null && prevStep !== newStep) {
      const previousStepObj = resolvedSteps[prevStep];
      previousStepObj?.onExit?.();
    }

    // Call onEnter for current step
    if (prevStep !== newStep) {
      currentStep?.onEnter?.();
    }

    previousStepRef.current = currentStepIndex;
  }, [currentStepIndex, isOpen, isPaused, currentStep, resolvedSteps]);

  // ============================================================================
  // Anchor Element Tracking
  // ============================================================================

  useEffect(() => {
    if (!isOpen || !currentStep?.anchorSelector || isPaused) {
      setAnchorRect(null);
      return;
    }

    const findAnchor = () => {
      const el = document.querySelector(currentStep.anchorSelector!);
      if (el) {
        setAnchorRect(el.getBoundingClientRect());
      } else {
        setAnchorRect(null);
      }
    };

    findAnchor();

    const handleUpdate = () => findAnchor();
    window.addEventListener("scroll", handleUpdate, true);
    window.addEventListener("resize", handleUpdate);

    const observer = new MutationObserver(findAnchor);
    observer.observe(document.body, { childList: true, subtree: true });

    return () => {
      window.removeEventListener("scroll", handleUpdate, true);
      window.removeEventListener("resize", handleUpdate);
      observer.disconnect();
    };
  }, [isOpen, currentStep?.anchorSelector, currentStepIndex, isPaused]);

  // ============================================================================
  // Panel Positioning (with auto-repositioning to avoid overlap)
  // ============================================================================

  useEffect(() => {
    if (!anchorRect || !panelRef.current || isDragging) return;

    const panelWidth = 360;
    const panelHeight = panelRef.current.offsetHeight || 320;
    const preferredPosition = currentStep?.anchorPosition || "right";

    const newPos = computeNonOverlappingPosition(
      anchorRect,
      panelWidth,
      panelHeight,
      preferredPosition
    );

    setPosition(newPos);
  }, [anchorRect, isDragging, currentStep?.anchorPosition]);

  // ============================================================================
  // Advance on Click
  // ============================================================================

  useEffect(() => {
    if (!isOpen || !currentStep?.advanceOnClick || isPaused) return;

    const handleClick = (e: MouseEvent) => {
      const target = e.target as Element;
      const selectors = currentStep
        .advanceOnClick!.split(",")
        .map((s) => s.trim());

      for (const selector of selectors) {
        if (target.closest(selector)) {
          const nextStep = currentStepIndex + 1;

          // Persist immediately before navigation
          if (!isLastStep) {
            try {
              sessionStorage.setItem(TOUR_STEP_KEY, nextStep.toString());
              sessionStorage.setItem(TOUR_ACTIVE_KEY, "true");
            } catch {
              // sessionStorage unavailable
            }
          }

          // Delay to let action complete
          setTimeout(() => {
            if (isLastStep) {
              handleComplete();
            } else {
              setCurrentStepIndex(nextStep);
            }
          }, 300);
          break;
        }
      }
    };

    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  }, [isOpen, currentStep?.advanceOnClick, isLastStep, currentStepIndex, isPaused]);

  // ============================================================================
  // Dragging
  // ============================================================================

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (!panelRef.current) return;
      e.preventDefault();
      setIsDragging(true);
      setDragOffset({
        x: e.clientX - position.x,
        y: e.clientY - position.y,
      });
    },
    [position]
  );

  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      const panelWidth = panelRef.current?.offsetWidth || 360;
      const panelHeight = panelRef.current?.offsetHeight || 320;
      const padding = 8;

      let newX = e.clientX - dragOffset.x;
      let newY = e.clientY - dragOffset.y;

      newX = Math.max(
        padding,
        Math.min(newX, window.innerWidth - panelWidth - padding)
      );
      newY = Math.max(
        padding,
        Math.min(newY, window.innerHeight - panelHeight - padding)
      );

      setPosition({ x: newX, y: newY });
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
    document.body.style.userSelect = "none";

    return () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.userSelect = "";
    };
  }, [isDragging, dragOffset]);

  // ============================================================================
  // Keyboard Navigation
  // ============================================================================

  useEffect(() => {
    if (!isOpen || isPaused) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        handleSkip();
      } else if (e.key === "ArrowRight") {
        handleNext();
      } else if (e.key === "ArrowLeft") {
        handlePrevious();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, currentStepIndex, isPaused]);

  // ============================================================================
  // Initialization
  // ============================================================================

  useEffect(() => {
    if (isOpen && !hasInitializedRef.current) {
      hasInitializedRef.current = true;
      try {
        const wasActive = sessionStorage.getItem(TOUR_ACTIVE_KEY);
        const savedStep = sessionStorage.getItem(TOUR_STEP_KEY);
        const wasPaused = sessionStorage.getItem(TOUR_PAUSED_KEY);

        if (wasPaused === "true") {
          setIsPaused(true);
        }

        if (wasActive === "true" && savedStep !== null) {
          setPosition({ x: 20, y: 80 });
          return;
        }
      } catch {
        // sessionStorage unavailable
      }
      setCurrentStepIndex(startAtStep);
      setPosition({ x: 20, y: 80 });
    }
  }, [isOpen, startAtStep]);

  // ============================================================================
  // Navigation Handlers
  // ============================================================================

  const handleNext = useCallback(async () => {
    if (pendingAutoAction) return;

    const step = resolvedSteps[currentStepIndex];

    // If step requires interaction and has autoAction, perform it
    if (step.waitForInteraction && step.autoAction) {
      setPendingAutoAction(true);
      await new Promise((r) => setTimeout(r, 500)); // Visual delay
      await step.autoAction();
      setPendingAutoAction(false);

      // If step has advanceOnClick, wait for that to trigger advancement
      // Otherwise, advance now since autoAction completed
      if (step.advanceOnClick) {
        return;
      }
      // Fall through to advance logic below
    }

    // Call onExit before advancing
    step?.onExit?.();

    if (isLastStep) {
      handleComplete();
    } else {
      const nextStep = resolvedSteps[currentStepIndex + 1];

      // Handle navigation if needed
      if (
        nextStep.navigateTo &&
        !location.pathname.startsWith(nextStep.navigateTo)
      ) {
        navigate(nextStep.navigateTo);
      }

      setCurrentStepIndex((prev) => prev + 1);
    }
  }, [
    isLastStep,
    currentStepIndex,
    resolvedSteps,
    pendingAutoAction,
    location.pathname,
    navigate,
  ]);

  const handlePrevious = useCallback(() => {
    if (!isFirstStep && !pendingAutoAction) {
      currentStep?.onExit?.();
      setCurrentStepIndex((prev) => prev - 1);
    }
  }, [isFirstStep, pendingAutoAction, currentStep]);

  const handleComplete = useCallback(() => {
    currentStep?.onExit?.();
    try {
      localStorage.setItem(TOUR_STORAGE_KEY, TOUR_VERSION);
      sessionStorage.removeItem(TOUR_STEP_KEY);
      sessionStorage.removeItem(TOUR_ACTIVE_KEY);
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
    } catch {
      // storage might be unavailable
    }
    hasInitializedRef.current = false;
    onComplete?.();
    onClose();
  }, [onComplete, onClose, currentStep]);

  const handleSkip = useCallback(() => {
    currentStep?.onExit?.();
    try {
      localStorage.setItem(TOUR_STORAGE_KEY, TOUR_VERSION);
      sessionStorage.removeItem(TOUR_STEP_KEY);
      sessionStorage.removeItem(TOUR_ACTIVE_KEY);
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
    } catch {
      // storage might be unavailable
    }
    hasInitializedRef.current = false;
    onClose();
  }, [onClose, currentStep]);

  const handlePause = useCallback(() => {
    try {
      sessionStorage.setItem(TOUR_PAUSED_KEY, "true");
    } catch {
      // storage unavailable
    }
    setIsPaused(true);
  }, []);

  const handleResume = useCallback(() => {
    try {
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
    } catch {
      // storage unavailable
    }
    setIsPaused(false);

    // Navigate to the required URL if needed
    if (currentStep?.navigateTo) {
      navigate(currentStep.navigateTo);
    } else if (currentStep?.requiredUrl) {
      if (typeof currentStep.requiredUrl === "string") {
        navigate(currentStep.requiredUrl);
      } else {
        // Extract base path from regex pattern (e.g., /^\/record/ -> /record/new)
        const pattern = currentStep.requiredUrl.source;
        if (pattern.startsWith("^\\/record")) {
          // For record mode, we can start a new session
          navigate("/record/new");
        }
        // For /projects patterns, we can't know which project, so don't navigate.
        // The user will need to manually return to their project.
        // The banner will remain visible until they're back on a matching URL.
      }
    }
  }, [currentStep, navigate]);

  const handleRestart = useCallback(() => {
    // Reset to step 0
    setCurrentStepIndex(0);
    setIsPaused(false);

    // Clear paused state from storage
    try {
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
      sessionStorage.setItem(TOUR_STEP_KEY, "0");
    } catch {
      // storage unavailable
    }

    // Navigate to the first step's URL
    const firstStep = resolvedSteps[0];
    if (firstStep?.navigateTo) {
      navigate(firstStep.navigateTo);
    } else if (firstStep?.requiredUrl && typeof firstStep.requiredUrl === "string") {
      navigate(firstStep.requiredUrl);
    } else {
      // Default to dashboard
      navigate("/");
    }
  }, [resolvedSteps, navigate]);

  // ============================================================================
  // Highlight Styles (with pulsing glow effect)
  // ============================================================================

  const highlightStyles = useMemo(() => {
    if (!anchorRect || isPaused) return null;

    const padding = 6;
    return {
      position: "fixed" as const,
      left: anchorRect.left - padding,
      top: anchorRect.top - padding,
      width: anchorRect.width + padding * 2,
      height: anchorRect.height + padding * 2,
      borderRadius: 12,
      border: "3px solid rgba(59, 130, 246, 0.9)",
      boxShadow: `
        0 0 0 4000px rgba(0, 0, 0, 0.25),
        0 0 0 6px rgba(59, 130, 246, 0.3),
        0 0 20px rgba(59, 130, 246, 0.5),
        0 0 40px rgba(59, 130, 246, 0.3)
      `,
      pointerEvents: "none" as const,
      zIndex: 9998,
      transition: "all 0.3s ease-out",
      animation: "tour-pulse 2s ease-in-out infinite",
    };
  }, [anchorRect, isPaused]);

  // ============================================================================
  // Render
  // ============================================================================

  if (!isOpen) return null;

  // Show resume banner when paused or off-track
  if (isPaused || isOffTrack) {
    return (
      <>
        <style>{`
          @keyframes tour-slide-in {
            from {
              transform: translateY(100%);
              opacity: 0;
            }
            to {
              transform: translateY(0);
              opacity: 1;
            }
          }
        `}</style>
        <div
          className="fixed bottom-4 right-4 z-[9999] bg-gray-900/95 backdrop-blur-md border border-gray-700 rounded-xl p-4 shadow-2xl max-w-sm"
          style={{ animation: "tour-slide-in 0.3s ease-out" }}
        >
          <div className="flex items-start gap-3">
            <div className="p-2 bg-blue-500/20 rounded-lg">
              <Sparkles size={20} className="text-blue-400" />
            </div>
            <div className="flex-1">
              <p className="text-sm font-medium text-white">Continue tutorial?</p>
              <p className="text-xs text-gray-400 mt-0.5">
                Step {currentStepIndex + 1}: {currentStep.title}
              </p>
            </div>
          </div>
          <div className="flex gap-2 mt-3">
            <button
              onClick={handleResume}
              className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors"
            >
              <Play size={14} />
              Resume
            </button>
            <button
              onClick={handleRestart}
              className="flex items-center justify-center gap-1.5 px-3 py-2 text-sm text-gray-300 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
              title="Start tutorial from the beginning"
            >
              <RotateCcw size={14} />
              Restart
            </button>
            <button
              onClick={handleSkip}
              className="px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
            >
              Skip
            </button>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      {/* CSS for pulsing animation */}
      <style>{`
        @keyframes tour-pulse {
          0%, 100% {
            box-shadow:
              0 0 0 4000px rgba(0, 0, 0, 0.25),
              0 0 0 6px rgba(59, 130, 246, 0.3),
              0 0 20px rgba(59, 130, 246, 0.5),
              0 0 40px rgba(59, 130, 246, 0.3);
          }
          50% {
            box-shadow:
              0 0 0 4000px rgba(0, 0, 0, 0.25),
              0 0 0 8px rgba(59, 130, 246, 0.5),
              0 0 30px rgba(59, 130, 246, 0.7),
              0 0 60px rgba(59, 130, 246, 0.4);
          }
        }
      `}</style>

      {/* Highlight around anchor element */}
      {highlightStyles && <div style={highlightStyles} aria-hidden="true" />}

      {/* Tour panel */}
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="false"
        aria-label="Guided Tour"
        className={`fixed z-[9999] w-[360px] bg-gray-900/95 backdrop-blur-md border border-gray-700 rounded-xl shadow-2xl ${
          isDragging ? "cursor-grabbing" : ""
        }`}
        style={{
          left: position.x,
          top: position.y,
          transition: isDragging ? "none" : "left 0.3s ease-out, top 0.3s ease-out",
        }}
      >
        {/* Drag handle */}
        <div
          onMouseDown={handleMouseDown}
          className="flex items-center justify-between px-4 py-2.5 border-b border-gray-800 cursor-grab active:cursor-grabbing bg-gray-800/50 rounded-t-xl"
        >
          <div className="flex items-center gap-2 text-gray-400">
            <GripHorizontal size={16} className="opacity-50" />
            <span className="text-xs font-medium">Tutorial</span>
          </div>
          <div className="flex items-center gap-1">
            <button
              onClick={handlePause}
              className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded-md transition-colors"
              aria-label="Pause tour"
              title="Pause tour"
            >
              <SkipForward size={14} />
            </button>
            <button
              onClick={handleSkip}
              className="p-1.5 text-gray-500 hover:text-gray-300 hover:bg-gray-700 rounded-md transition-colors"
              aria-label="Close tour"
            >
              <X size={14} />
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="p-5">
          {/* Icon and title */}
          <div className="flex items-start gap-3 mb-3">
            <div className="flex-shrink-0 p-2.5 bg-gray-800 rounded-lg border border-gray-700">
              {currentStep.icon}
            </div>
            <div className="flex-1 min-w-0">
              <h3 className="text-base font-semibold text-white leading-tight">
                {currentStep.title}
              </h3>
              <p className="text-xs text-gray-400 mt-0.5">
                Step {currentStepIndex + 1} of {resolvedSteps.length}
              </p>
            </div>
          </div>

          {/* Description */}
          <p className="text-sm text-gray-300 leading-relaxed mb-4">
            {currentStep.description}
          </p>

          {/* Action hint */}
          {currentStep.actionHint && (
            <div className="flex items-center gap-2 px-3 py-2 bg-blue-500/10 border border-blue-500/30 rounded-lg mb-4">
              <Circle size={10} className="text-blue-400 fill-blue-400 flex-shrink-0" />
              <span className="text-xs text-blue-300">{currentStep.actionHint}</span>
            </div>
          )}

          {/* Progress bar */}
          <div className="h-1.5 bg-gray-800 rounded-full mb-4 overflow-hidden">
            <div
              className="h-full bg-gradient-to-r from-blue-500 to-purple-500 rounded-full transition-all duration-300"
              style={{
                width: `${((currentStepIndex + 1) / resolvedSteps.length) * 100}%`,
              }}
            />
          </div>

          {/* Navigation buttons */}
          <div className="flex items-center justify-between">
            <button
              onClick={handlePrevious}
              disabled={isFirstStep || pendingAutoAction}
              className={`flex items-center gap-1 px-3 py-1.5 text-sm rounded-lg transition-colors ${
                isFirstStep || pendingAutoAction
                  ? "text-gray-600 cursor-not-allowed"
                  : "text-gray-400 hover:text-white hover:bg-gray-800"
              }`}
            >
              <ChevronLeft size={16} />
              <span>Back</span>
            </button>

            <div className="flex items-center gap-2">
              {/* Restart button on last step */}
              {isLastStep && (
                <button
                  onClick={handleRestart}
                  disabled={pendingAutoAction}
                  className="flex items-center gap-1 px-3 py-1.5 text-sm text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors"
                  title="Start the tutorial again"
                >
                  <RotateCcw size={14} />
                  <span>Restart</span>
                </button>
              )}

              <button
                onClick={handleNext}
                disabled={pendingAutoAction}
                className="flex items-center gap-1 px-4 py-1.5 text-sm font-medium bg-blue-600 hover:bg-blue-500 disabled:bg-blue-600/50 text-white rounded-lg transition-colors min-w-[100px] justify-center"
              >
                {pendingAutoAction ? (
                  <>
                    <Loader2 size={14} className="animate-spin" />
                    <span>Working...</span>
                  </>
                ) : isLastStep ? (
                  <>
                    <span>Finish</span>
                    <CheckCircle2 size={16} />
                  </>
                ) : (
                  <>
                    <span>Next</span>
                    <ChevronRight size={16} />
                  </>
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Keyboard hint */}
        <div className="px-4 py-2 border-t border-gray-800 text-center">
          <span className="text-[10px] text-gray-500">
            Arrow keys to navigate · Esc to close · Drag to move
          </span>
        </div>
      </div>
    </>
  );
}

// ============================================================================
// Hook: useGuidedTour
// ============================================================================

export function useGuidedTour() {
  const [showTour, setShowTour] = useState(false);
  const [hasCheckedStorage, setHasCheckedStorage] = useState(false);
  // Key that changes when resetTour is called, forcing GuidedTour to remount
  const [tourKey, setTourKey] = useState(0);

  useEffect(() => {
    try {
      // Suppress auto-opening during automated runs
      const isAutomatedRun =
        typeof navigator !== "undefined" &&
        (navigator.webdriver === true ||
          /lighthouse/i.test(navigator.userAgent) ||
          /HeadlessChrome/i.test(navigator.userAgent));
      if (isAutomatedRun) {
        setHasCheckedStorage(true);
        return;
      }

      const completed = localStorage.getItem(TOUR_STORAGE_KEY);
      const wasActive = sessionStorage.getItem(TOUR_ACTIVE_KEY);

      if (completed !== TOUR_VERSION || wasActive === "true") {
        setShowTour(true);
      }
    } catch {
      // storage unavailable
    }
    setHasCheckedStorage(true);
  }, []);

  // Listen for global open event (allows opening from any component)
  useEffect(() => {
    const handleOpen = () => {
      setShowTour(true);
      setTourKey((k) => k + 1);
    };

    window.addEventListener(TOUR_OPEN_EVENT, handleOpen);
    return () => window.removeEventListener(TOUR_OPEN_EVENT, handleOpen);
  }, []);

  const openTour = useCallback(() => {
    setShowTour(true);
  }, []);

  const closeTour = useCallback(() => {
    setShowTour(false);
  }, []);

  const resetTour = useCallback(() => {
    try {
      localStorage.removeItem(TOUR_STORAGE_KEY);
      sessionStorage.removeItem(TOUR_STEP_KEY);
      sessionStorage.removeItem(TOUR_ACTIVE_KEY);
      sessionStorage.removeItem(TOUR_PAUSED_KEY);
    } catch {
      // storage unavailable
    }
    // Dispatch events to reset and open the tour from any hook instance
    window.dispatchEvent(new CustomEvent(TOUR_RESET_EVENT));
    window.dispatchEvent(new CustomEvent(TOUR_OPEN_EVENT));
  }, []);

  return {
    showTour,
    hasCheckedStorage,
    openTour,
    closeTour,
    resetTour,
    tourKey,
  };
}

export default GuidedTour;
