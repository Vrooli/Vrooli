#!/usr/bin/env python3
"""
Window detection and shortcut matching for Agent-S2
Detects the active window and provides relevant keyboard shortcuts
"""

import json
import os
import subprocess
import re
from typing import Dict, List, Optional, Tuple
from pathlib import Path


class ShortcutDetector:
    def __init__(self, shortcuts_dir: str = None):
        """Initialize the shortcut detector with the shortcuts directory"""
        if shortcuts_dir is None:
            # Get the directory where this script is located
            script_dir = Path(__file__).parent
            self.shortcuts_dir = script_dir
        else:
            self.shortcuts_dir = Path(shortcuts_dir)
        
        self.registry = self._load_registry()
        self.shortcuts_cache = {}
    
    def _load_registry(self) -> Dict:
        """Load the master registry file"""
        registry_path = self.shortcuts_dir / "registry.json"
        if not registry_path.exists():
            raise FileNotFoundError(f"Registry not found at {registry_path}")
        
        with open(registry_path, 'r') as f:
            return json.load(f)
    
    def _load_shortcuts(self, app_file: str) -> Dict:
        """Load shortcuts for a specific application"""
        if app_file in self.shortcuts_cache:
            return self.shortcuts_cache[app_file]
        
        shortcuts_path = self.shortcuts_dir / app_file
        if not shortcuts_path.exists():
            return {}
        
        with open(shortcuts_path, 'r') as f:
            shortcuts = json.load(f)
            self.shortcuts_cache[app_file] = shortcuts
            return shortcuts
    
    def get_active_window(self) -> Tuple[Optional[str], Optional[str]]:
        """Get the active window title and class using xdotool"""
        try:
            # Get active window ID
            window_id = subprocess.check_output(
                ['xdotool', 'getactivewindow'],
                stderr=subprocess.DEVNULL
            ).decode().strip()
            
            # Get window name (title)
            window_name = subprocess.check_output(
                ['xdotool', 'getwindowname', window_id],
                stderr=subprocess.DEVNULL
            ).decode().strip()
            
            # Get window class
            window_class = subprocess.check_output(
                ['xprop', '-id', window_id, 'WM_CLASS'],
                stderr=subprocess.DEVNULL
            ).decode().strip()
            
            # Extract class name from xprop output
            # Format: WM_CLASS(STRING) = "instance", "class"
            match = re.search(r'WM_CLASS\(STRING\) = "([^"]+)", "([^"]+)"', window_class)
            if match:
                instance_name = match.group(1)
                class_name = match.group(2)
                # Prefer class name, fall back to instance
                window_class = class_name or instance_name
            else:
                window_class = None
            
            return window_name, window_class
            
        except subprocess.CalledProcessError:
            return None, None
        except Exception as e:
            print(f"Error detecting window: {e}")
            return None, None
    
    def match_application(self, window_title: str, window_class: str) -> Optional[str]:
        """Match window info to an application in the registry"""
        if not window_title and not window_class:
            return None
        
        # Combine title and class for matching
        window_info = f"{window_title} {window_class}".lower()
        
        # Check each application in registry
        for app_name, app_config in self.registry.get('applications', {}).items():
            if not app_config.get('enabled', True):
                continue
            
            # Check if any pattern matches
            for pattern in app_config.get('patterns', []):
                if pattern == '*':  # Skip wildcard for specific matching
                    continue
                if pattern.lower() in window_info:
                    return app_name
        
        # If no specific match, return common
        return 'common'
    
    def get_shortcuts_for_window(self, window_title: str = None, window_class: str = None) -> Dict:
        """Get relevant shortcuts for the current or specified window"""
        # If no window info provided, detect current window
        if window_title is None and window_class is None:
            window_title, window_class = self.get_active_window()
        
        # Match to application
        app_name = self.match_application(window_title or '', window_class or '')
        
        # Load common shortcuts first
        common_config = self.registry.get('applications', {}).get('common', {})
        common_shortcuts = self._load_shortcuts(common_config.get('file', ''))
        
        # Load app-specific shortcuts
        app_shortcuts = {}
        if app_name and app_name != 'common':
            app_config = self.registry.get('applications', {}).get(app_name, {})
            app_shortcuts = self._load_shortcuts(app_config.get('file', ''))
        
        # Merge shortcuts (app-specific overrides common)
        merged = {
            'detected_app': app_name,
            'window_title': window_title,
            'window_class': window_class,
            'shortcuts': {}
        }
        
        # Start with common shortcuts
        if common_shortcuts:
            merged['shortcuts'].update(common_shortcuts.get('shortcuts', {}))
            merged['common_hints'] = common_shortcuts.get('context_hints', {})
        
        # Override with app-specific shortcuts
        if app_shortcuts:
            merged['shortcuts'].update(app_shortcuts.get('shortcuts', {}))
            merged['app_name'] = app_shortcuts.get('app_name', app_name)
            merged['priority_actions'] = app_shortcuts.get('priority_actions', {})
            merged['context_hints'] = app_shortcuts.get('context_hints', {})
        
        return merged
    
    def get_shortcut_for_action(self, action: str, app_name: str = None) -> Optional[List[str]]:
        """Get keyboard shortcuts for a specific action"""
        shortcuts_data = self.get_shortcuts_for_window()
        
        # Search through all categories for the action
        for category, shortcuts in shortcuts_data.get('shortcuts', {}).items():
            if action in shortcuts:
                shortcut_info = shortcuts[action]
                if isinstance(shortcut_info, dict):
                    return shortcut_info.get('keys', [])
                elif isinstance(shortcut_info, list):
                    return shortcut_info
        
        return None
    
    def format_shortcuts_context(self, task_description: str = None) -> str:
        """Format shortcuts into context string for AI"""
        shortcuts_data = self.get_shortcuts_for_window()
        
        context_parts = []
        context_parts.append(f"\n=== KEYBOARD SHORTCUTS AVAILABLE ===")
        context_parts.append(f"Active Application: {shortcuts_data.get('app_name', 'Unknown')}")
        
        # Add priority shortcuts first
        priority_actions = shortcuts_data.get('priority_actions', {})
        if priority_actions:
            context_parts.append("\nRECOMMENDED SHORTCUTS FOR COMMON TASKS:")
            for task, actions in priority_actions.items():
                task_formatted = task.replace('_', ' ').title()
                shortcuts_list = []
                for action in actions:
                    for category, shortcuts in shortcuts_data.get('shortcuts', {}).items():
                        if action in shortcuts:
                            shortcut_info = shortcuts[action]
                            if isinstance(shortcut_info, dict):
                                keys = shortcut_info.get('keys', [])
                                if keys:
                                    shortcuts_list.append(f"{keys[0]}")
                if shortcuts_list:
                    context_parts.append(f"- {task_formatted}: {', '.join(shortcuts_list)}")
        
        # Add high-priority shortcuts
        context_parts.append("\nHIGH-PRIORITY SHORTCUTS:")
        for category, shortcuts in shortcuts_data.get('shortcuts', {}).items():
            for action, info in shortcuts.items():
                if isinstance(info, dict) and info.get('priority', 0) == 1:
                    keys = info.get('keys', [])
                    if keys:
                        context_parts.append(f"- {info['description']}: {keys[0]}")
        
        # Add context hints
        hints = shortcuts_data.get('context_hints', {})
        if hints:
            context_parts.append("\nCONTEXT-SPECIFIC TIPS:")
            for hint_key, hint_text in hints.items():
                context_parts.append(f"- {hint_text}")
        
        # Add general preference
        context_parts.append("\nIMPORTANT: Prefer keyboard shortcuts over mouse clicks when possible for efficiency and reliability.")
        
        return '\n'.join(context_parts)


def main():
    """CLI interface for testing the detector"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Detect window and show available shortcuts')
    parser.add_argument('--action', choices=['detect', 'shortcuts', 'context', 'action'],
                       default='context', help='Action to perform')
    parser.add_argument('--action-name', help='Name of action to look up shortcuts for')
    parser.add_argument('--format', choices=['json', 'text'], default='text',
                       help='Output format')
    
    args = parser.parse_args()
    
    detector = ShortcutDetector()
    
    if args.action == 'detect':
        title, window_class = detector.get_active_window()
        if args.format == 'json':
            print(json.dumps({'title': title, 'class': window_class}, indent=2))
        else:
            print(f"Window Title: {title}")
            print(f"Window Class: {window_class}")
    
    elif args.action == 'shortcuts':
        shortcuts = detector.get_shortcuts_for_window()
        if args.format == 'json':
            print(json.dumps(shortcuts, indent=2))
        else:
            print(f"Detected App: {shortcuts.get('detected_app')}")
            print(f"Window: {shortcuts.get('window_title')}")
            print("\nAvailable Shortcuts:")
            for category, items in shortcuts.get('shortcuts', {}).items():
                print(f"\n{category.upper()}:")
                for action, info in items.items():
                    if isinstance(info, dict):
                        keys = info.get('keys', [])
                        desc = info.get('description', '')
                        print(f"  - {desc}: {', '.join(keys)}")
    
    elif args.action == 'context':
        context = detector.format_shortcuts_context()
        print(context)
    
    elif args.action == 'action' and args.action_name:
        shortcuts = detector.get_shortcut_for_action(args.action_name)
        if shortcuts:
            print(f"Shortcuts for '{args.action_name}': {', '.join(shortcuts)}")
        else:
            print(f"No shortcuts found for '{args.action_name}'")


if __name__ == '__main__':
    main()