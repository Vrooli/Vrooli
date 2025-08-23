# 🎯 **Target-Aware Automation System Implementation Plan**

## **Overview**
Comprehensive plan to implement target-aware automation system for Agent S2, enabling reliable application-specific automation and high-level workflow commands.

## **Phase 1: Window Manager Integration** 
*Enhanced Low-Level Commands with Target Awareness*

---

### **🏗️ Infrastructure & Dependencies**

#### **Container Updates Required:**
```dockerfile
# Add to Agent S2 Dockerfile
RUN apt-get update && apt-get install -y \
    wmctrl \           # Window management control
    xdotool \          # Advanced X11 automation  
    x11-utils \        # Window property inspection (xprop, xwininfo)
    xvfb-run \         # Virtual display utilities
    && rm -rf /var/lib/apt/lists/*
```

#### **X11 Permissions & Configuration:**
```bash
# Ensure proper X11 access in startup.sh
export DISPLAY=:99
xauth add $HOST/unix:99 . $(xxd -l 16 -p /dev/urandom)
```

---

### **🔧 Core Services Implementation**

#### **1. WindowManager Service**
```python
# File: agent_s2/server/services/window_manager.py

class WindowInfo:
    window_id: str
    title: str 
    app_name: str
    process_id: int
    geometry: Dict[str, int]  # x, y, width, height
    is_focused: bool
    last_active: datetime

class WindowManager:
    def __init__(self):
        self.known_apps = {
            "firefox": ["Firefox", "Mozilla Firefox"],
            "terminal": ["xterm", "Terminal", "Xterm"],
            "calculator": ["Calculator", "gnome-calculator"]
        }
    
    def get_running_applications(self) -> List[str]:
        """Get list of running GUI applications"""
    
    def get_application_windows(self, app_name: str) -> List[WindowInfo]:
        """Get all windows for specific application"""
    
    def get_focused_window(self) -> Optional[WindowInfo]:
        """Get currently focused window"""
    
    def focus_application(self, app_name: str, window_criteria: Dict = None) -> bool:
        """Focus specific application with optional window selection"""
    
    def focus_window_by_id(self, window_id: str) -> bool:
        """Focus specific window by ID"""
    
    def start_application(self, app_name: str, args: List[str] = None) -> bool:
        """Start application if not running"""
    
    def verify_focus(self, app_name: str, timeout: float = 2.0) -> bool:
        """Verify application gained focus within timeout"""
```

#### **2. Enhanced AutomationService**
```python
# File: agent_s2/server/services/automation.py

class AutomationService:
    def __init__(self):
        self.window_manager = WindowManager()
        # ... existing init code ...
    
    def _ensure_target_focus(self, target_app: str, window_criteria: Dict = None) -> bool:
        """Ensure target application has focus before automation"""
        if not target_app:
            return True  # No target specified, proceed
            
        # Check if already focused
        focused = self.window_manager.get_focused_window()
        if focused and self._matches_target(focused, target_app):
            return True
            
        # Need to focus target app
        return self.window_manager.focus_application(target_app, window_criteria)
    
    def click_targeted(self, x: int, y: int, target_app: str = None, 
                      button: str = "left", window_criteria: Dict = None):
        """Enhanced click with target awareness"""
        if target_app and not self._ensure_target_focus(target_app, window_criteria):
            raise AutomationError(f"Could not focus target application: {target_app}")
            
        # Proceed with existing click logic
        self.click(x, y, button)
    
    def type_text_targeted(self, text: str, target_app: str = None, 
                          interval: float = 0.01, window_criteria: Dict = None):
        """Enhanced typing with target awareness"""
        if target_app and not self._ensure_target_focus(target_app, window_criteria):
            raise AutomationError(f"Could not focus target application: {target_app}")
            
        self.type_text(text, interval)
    
    def press_key_targeted(self, keys: List[str], target_app: str = None,
                          window_criteria: Dict = None):
        """Enhanced key press with target awareness"""
        if target_app and not self._ensure_target_focus(target_app, window_criteria):
            raise AutomationError(f"Could not focus target application: {target_app}")
            
        self.press_key(keys)
```

---

### **📝 Enhanced API Models**

#### **Request Models with Target Awareness:**
```python
# File: agent_s2/server/models/requests.py

class WindowCriteria(BaseModel):
    """Criteria for selecting specific window of an application"""
    title_contains: Optional[str] = None
    title_matches: Optional[str] = None  # Regex pattern
    url_contains: Optional[str] = None   # For browser windows
    window_id: Optional[str] = None      # Explicit window ID
    prefer_recent: bool = True           # Prefer most recently active

class TargetedAutomationRequest(BaseModel):
    """Base class for target-aware automation requests"""
    target_app: Optional[str] = Field(default=None, description="Target application name (firefox, terminal, calculator)")
    window_criteria: Optional[WindowCriteria] = Field(default=None, description="Criteria for window selection")
    ensure_focus: bool = Field(default=True, description="Ensure target has focus before action")
    focus_timeout: float = Field(default=2.0, description="Timeout for focus operations")

class KeyboardPressRequest(TargetedAutomationRequest):
    """Enhanced keyboard press request with targeting"""
    key: str = Field(..., description="Key to press")
    modifiers: Optional[List[str]] = Field(default=None, description="Modifier keys")

class KeyboardTypeRequest(TargetedAutomationRequest):
    """Enhanced keyboard type request with targeting"""
    text: str = Field(..., description="Text to type")
    interval: float = Field(default=0.01, description="Interval between keystrokes")

class MouseClickRequest(TargetedAutomationRequest):
    """Enhanced mouse click request with targeting"""
    x: int = Field(..., description="X coordinate")
    y: int = Field(..., description="Y coordinate") 
    button: str = Field(default="left", description="Mouse button")
```

#### **Response Models:**
```python
class TargetedActionResponse(BaseModel):
    """Response for targeted automation actions"""
    success: bool
    action: str
    target_app: Optional[str] = None
    focused_window: Optional[str] = None  # Window that was focused
    focus_time: Optional[float] = None    # Time taken to focus
    execution_time: float
    message: str
    error: Optional[str] = None
```

---

### **🌐 Enhanced API Routes**

#### **Updated Keyboard Routes:**
```python
# File: agent_s2/server/routes/keyboard.py

@router.post("/press", response_model=TargetedActionResponse)
async def press_key_targeted(request: KeyboardPressRequest):
    """Press key with optional target application focus"""
    start_time = time.time()
    focus_time = None
    focused_window = None
    
    try:
        # Handle targeting if specified
        if request.target_app:
            focus_start = time.time()
            success = automation_service.window_manager.focus_application(
                request.target_app, 
                request.window_criteria.dict() if request.window_criteria else None
            )
            focus_time = time.time() - focus_start
            
            if not success and request.ensure_focus:
                raise HTTPException(
                    status_code=400, 
                    detail=f"Could not focus target application: {request.target_app}"
                )
                
            focused_window = automation_service.window_manager.get_focused_window()
        
        # Execute keyboard action
        automation_service.press_key_targeted(
            [request.key] + (request.modifiers or []),
            request.target_app,
            request.window_criteria.dict() if request.window_criteria else None
        )
        
        execution_time = time.time() - start_time
        
        return TargetedActionResponse(
            success=True,
            action="press",
            target_app=request.target_app,
            focused_window=focused_window.title if focused_window else None,
            focus_time=focus_time,
            execution_time=execution_time,
            message=f"Pressed {'+'.join([request.key] + (request.modifiers or []))}"
        )
        
    except Exception as e:
        logger.error(f"Targeted key press failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
```

---

### **🤖 AI Agent Integration**

#### **Enhanced AI Handler:**
```python
# File: agent_s2/server/services/ai_handler.py

class AIHandler:
    def __init__(self):
        # ... existing init code ...
        self.window_manager = WindowManager()
    
    async def execute_command(self, command: str, context: Optional[str] = None, 
                            target_app: Optional[str] = None) -> Dict[str, Any]:
        """Execute command with optional target application"""
        
        # Enhanced AI prompt with target awareness
        system_prompt = f"""You are an automation assistant that can control applications.

Target Application: {target_app or "auto-detect"}
Available Applications: {self.window_manager.get_running_applications()}

When target_app is specified, all actions will be focused on that application.
Include "target_app": "{target_app}" in all automation actions when specified.

IMPORTANT: Always specify target_app in actions when provided:
{{"type": "click", "x": 100, "y": 200, "target_app": "{target_app}", "description": "..."}}
{{"type": "type", "text": "hello", "target_app": "{target_app}", "description": "..."}}
"""

        # ... rest of execution logic with target_app support ...
    
    def _parse_ai_actions(self, ai_response: str, target_app: str = None) -> List[Dict]:
        """Parse AI response and inject target_app if needed"""
        actions = self._extract_actions_from_response(ai_response)
        
        # Inject target_app into all actions if specified
        if target_app:
            for action in actions:
                if action.get("type") in ["click", "type", "key"]:
                    action["target_app"] = target_app
                    
        return actions
```

---

## **Phase 2: Smart Application Commands**
*High-Level Task-Based Automation*

---

### **🌟 Browser Service Implementation**

#### **Core Browser Service:**
```python
# File: agent_s2/server/services/browser_service.py

class BrowserService:
    def __init__(self, automation_service: AutomationService):
        self.automation = automation_service
        self.window_manager = automation_service.window_manager
        
    async def ensure_browser_running(self, browser: str = "firefox") -> bool:
        """Ensure browser is running, start if needed"""
        if not self.window_manager.get_application_windows(browser):
            return self.window_manager.start_application(browser)
        return True
    
    async def navigate_to_url(self, url: str, window: str = "current", 
                            wait_for_load: bool = True) -> Dict[str, Any]:
        """Navigate to URL with intelligent window handling"""
        
        # 1. Ensure browser is running
        if not await self.ensure_browser_running():
            raise BrowserError("Could not start browser")
        
        # 2. Handle window selection
        target_window = await self._select_or_create_window(window)
        
        # 3. Focus address bar and navigate
        await self._focus_address_bar(target_window)
        await self._clear_address_bar()
        await self._type_url(url)
        await self._press_enter()
        
        # 4. Wait for navigation if requested
        if wait_for_load:
            await self._wait_for_navigation(timeout=10.0)
            
        return {
            "success": True,
            "url": url,
            "window": target_window,
            "load_time": self._last_navigation_time
        }
    
    async def new_tab(self, url: Optional[str] = None) -> Dict[str, Any]:
        """Open new tab with optional URL"""
        
        # Focus browser
        await self._ensure_browser_focused()
        
        # Open new tab (Ctrl+T)
        await self.automation.press_key_targeted(["ctrl", "t"], "firefox")
        await asyncio.sleep(0.5)
        
        # Navigate if URL provided
        if url:
            await self._type_url(url)
            await self._press_enter()
            
        return {"success": True, "action": "new_tab", "url": url}
    
    async def close_tab(self, tab_criteria: Dict = None) -> Dict[str, Any]:
        """Close current or specified tab"""
        
        if tab_criteria:
            await self._focus_tab_by_criteria(tab_criteria)
            
        await self.automation.press_key_targeted(["ctrl", "w"], "firefox")
        return {"success": True, "action": "close_tab"}
    
    async def _select_or_create_window(self, window: str) -> str:
        """Smart window selection logic"""
        if window == "new":
            return await self._create_new_window()
        elif window == "current":
            return await self._get_current_browser_window()
        else:
            # Interpret as window criteria
            return await self._find_window_by_criteria(window)
```

---

### **🔄 Session Workflow Commands**

#### **Session Management Service:**
```python
# File: agent_s2/server/services/session_workflows.py

class SessionWorkflowService:
    def __init__(self, browser_service: BrowserService, 
                 session_manager: SessionManager):
        self.browser = browser_service
        self.session = session_manager
        
    async def clean_browse_session(self, url: str, 
                                 clear_data: bool = True) -> Dict[str, Any]:
        """Ultimate workflow: Reset session + navigate in one command"""
        
        workflow_start = time.time()
        steps_completed = []
        
        try:
            # Step 1: Clear session state 
            if clear_data:
                await self.session.delete_session_state(
                    automation_service=self.browser.automation
                )
                steps_completed.append("session_state_cleared")
            
            # Step 2: Clear session data (restart browser)
            await self.session.delete_session_data(
                automation_service=self.browser.automation,
                restart_browser=True
            )
            steps_completed.append("session_data_cleared")
            steps_completed.append("browser_restarted")
            
            # Step 3: Wait for clean browser startup
            await asyncio.sleep(3.0)
            steps_completed.append("browser_ready")
            
            # Step 4: Navigate to target URL
            nav_result = await self.browser.navigate_to_url(url)
            steps_completed.append("navigation_completed")
            
            total_time = time.time() - workflow_start
            
            return {
                "success": True,
                "workflow": "clean_browse_session",
                "url": url,
                "steps_completed": steps_completed,
                "total_time": total_time,
                "browser_info": nav_result
            }
            
        except Exception as e:
            return {
                "success": False,
                "workflow": "clean_browse_session", 
                "error": str(e),
                "steps_completed": steps_completed,
                "partial_time": time.time() - workflow_start
            }
    
    async def reset_and_navigate(self, url: str) -> Dict[str, Any]:
        """Simplified: Reset + navigate workflow"""
        return await self.clean_browse_session(url, clear_data=True)
    
    async def quick_navigate(self, url: str, new_tab: bool = False) -> Dict[str, Any]:
        """Quick navigation without session reset"""
        if new_tab:
            return await self.browser.new_tab(url)
        else:
            return await self.browser.navigate_to_url(url)
```

---

### **🌐 Phase 2 API Routes**

#### **Browser Routes:**
```python
# File: agent_s2/server/routes/browser.py

@router.post("/navigate", response_model=BrowserActionResponse)
async def navigate_to_url(request: NavigateRequest):
    """Navigate to URL with smart window handling"""
    
    result = await browser_service.navigate_to_url(
        url=request.url,
        window=request.window or "current",
        wait_for_load=request.wait_for_load
    )
    
    return BrowserActionResponse(**result)

@router.post("/new-tab", response_model=BrowserActionResponse)  
async def create_new_tab(request: NewTabRequest):
    """Create new tab with optional URL"""
    
    result = await browser_service.new_tab(request.url)
    return BrowserActionResponse(**result)

@router.delete("/tab", response_model=BrowserActionResponse)
async def close_tab(request: CloseTabRequest):
    """Close current or specified tab"""
    
    result = await browser_service.close_tab(request.tab_criteria)
    return BrowserActionResponse(**result)
```

#### **Workflow Routes:**
```python
# File: agent_s2/server/routes/workflows.py

@router.post("/clean-browse", response_model=WorkflowResponse)
async def clean_browse_session(request: CleanBrowseRequest):
    """Reset session and navigate to URL in one command"""
    
    result = await workflow_service.clean_browse_session(
        url=request.url,
        clear_data=request.clear_data
    )
    
    return WorkflowResponse(**result)

@router.post("/reset-and-navigate", response_model=WorkflowResponse)
async def reset_and_navigate(request: ResetNavigateRequest):
    """Simple reset + navigate workflow"""
    
    result = await workflow_service.reset_and_navigate(request.url)
    return WorkflowResponse(**result)
```

---

### **🤖 Enhanced AI Agent Integration**

#### **Smart Command Recognition:**
```python
# Enhanced AI Handler with workflow awareness

async def execute_command(self, command: str, context: Optional[str] = None) -> Dict[str, Any]:
    """Enhanced command execution with workflow detection"""
    
    # Detect high-level workflows
    workflow_patterns = {
        r"(?:go to|navigate to|open)\s+(.+?)(?:\s+(?:in new tab|new tab))?": "navigate",
        r"(?:reset.*?(?:go to|navigate)|clean.*?browse)\s+(.+)": "clean_browse",
        r"(?:new tab|open new tab)(?:\s+(?:with|to)\s+(.+))?": "new_tab",
        r"close.*?tab": "close_tab"
    }
    
    # Try to match workflow patterns first
    for pattern, workflow_type in workflow_patterns.items():
        match = re.search(pattern, command.lower())
        if match:
            return await self._execute_workflow(workflow_type, match, command)
    
    # Fall back to enhanced low-level automation
    return await self._execute_low_level_automation(command, context)

async def _execute_workflow(self, workflow_type: str, match, original_command: str):
    """Execute high-level workflow"""
    
    if workflow_type == "clean_browse":
        url = self._extract_url(match.group(1))
        if url:
            return await workflow_service.clean_browse_session(url)
    
    elif workflow_type == "navigate":
        url = self._extract_url(match.group(1))
        new_tab = "new tab" in original_command.lower()
        
        if new_tab:
            return await browser_service.new_tab(url)
        else:
            return await browser_service.navigate_to_url(url)
    
    # ... handle other workflow types ...
```

---

## **🧪 Testing Strategy**

### **Phase 1 Testing:**
```python
# Test target awareness
def test_keyboard_with_target():
    # Test focusing Firefox before typing
    response = client.post("/keyboard/type", json={
        "text": "hello world",
        "target_app": "firefox"
    })
    assert response.status_code == 200
    assert "firefox" in response.json()["focused_window"].lower()

def test_focus_fallback():
    # Test behavior when target app not running
    response = client.post("/keyboard/press", json={
        "key": "l",
        "modifiers": ["ctrl"],
        "target_app": "nonexistent_app",
        "ensure_focus": False  # Should not fail
    })
    assert response.status_code == 200
```

### **Phase 2 Testing:**
```python
def test_clean_browse_workflow():
    # Test end-to-end workflow
    response = client.post("/workflows/clean-browse", json={
        "url": "https://example.com"
    })
    assert response.status_code == 200
    result = response.json()
    assert result["success"] == True
    assert "navigation_completed" in result["steps_completed"]
```

---

## **📋 Implementation Roadmap**

### **Phase 1 Implementation Order:**
1. ✅ **Container Dependencies** (wmctrl, xdotool)
2. ✅ **WindowManager Service** (core window operations)
3. ✅ **Enhanced Request Models** (add target_app fields)
4. ✅ **Enhanced AutomationService** (target awareness)
5. ✅ **Updated API Routes** (keyboard, mouse with targeting)
6. ✅ **AI Agent Integration** (target_app support)
7. ✅ **Testing & Validation**

### **Phase 2 Implementation Order:**
1. ✅ **BrowserService** (high-level browser automation)
2. ✅ **SessionWorkflowService** (workflow orchestration)
3. ✅ **Browser API Routes** (navigate, new-tab, etc.)
4. ✅ **Workflow API Routes** (clean-browse, reset-and-navigate)
5. ✅ **Enhanced AI Recognition** (workflow pattern matching)
6. ✅ **Integration Testing** (end-to-end workflows)

### **Success Criteria:**
- **Phase 1**: `navigate_to_url(..., target_app="firefox")` works reliably
- **Phase 2**: Single command `clean_browse_session("https://wikipedia.org")` does complete reset + navigate

**This architecture provides both surgical precision and powerful workflows while maintaining backward compatibility!**

---

## **Key Benefits**

### **Phase 1 Benefits:**
- ✅ Commands always reach intended application
- ✅ Auto-recovery if focus is lost
- ✅ Backwards compatible with existing code
- ✅ Foundation for sophisticated workflows

### **Phase 2 Benefits:**
- ✅ Single command for complex workflows
- ✅ Built-in error handling and recovery
- ✅ Fast execution (no network round-trips)
- ✅ Natural task-based interface

### **Overall Impact:**
- 🎯 **Reliability**: No more "blind" automation
- 🚀 **Productivity**: Complex tasks in single commands
- 🔧 **Flexibility**: Both low-level and high-level control
- 🛡️ **Robustness**: Automatic error handling and recovery