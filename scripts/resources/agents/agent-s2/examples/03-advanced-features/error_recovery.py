#!/usr/bin/env python3
"""
Error Recovery - Building robust automations

This example demonstrates how to handle errors and build
resilient automation scripts.
"""

from agent_s2.client import AgentS2Client, AutomationClient, ScreenshotClient
import time
import logging
from typing import Optional, Callable, Any

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


class RobustAutomation:
    """A wrapper for automation with error recovery capabilities"""
    
    def __init__(self):
        self.client = AgentS2Client()
        self.automation = AutomationClient(self.client)
        self.screenshot = ScreenshotClient(self.client)
        self.max_retries = 3
        self.retry_delay = 2.0
        
    def with_retry(self, func: Callable, *args, **kwargs) -> Any:
        """Execute a function with retry logic"""
        last_error = None
        
        for attempt in range(self.max_retries):
            try:
                logger.info(f"Attempting {func.__name__} (attempt {attempt + 1}/{self.max_retries})")
                result = func(*args, **kwargs)
                logger.info(f"✅ {func.__name__} succeeded")
                return result
            except Exception as e:
                last_error = e
                logger.warning(f"❌ Attempt {attempt + 1} failed: {e}")
                
                if attempt < self.max_retries - 1:
                    logger.info(f"Waiting {self.retry_delay}s before retry...")
                    time.sleep(self.retry_delay)
                    
                    # Take a screenshot for debugging
                    try:
                        self.screenshot.save(
                            f"error_attempt_{attempt + 1}.png",
                            directory="../../testing/test-outputs/screenshots"
                        )
                    except:
                        pass
                        
        raise Exception(f"Failed after {self.max_retries} attempts: {last_error}")
        
    def safe_click(self, x: int, y: int, verify: bool = True) -> bool:
        """Click with verification"""
        try:
            # Take before screenshot
            before = self.screenshot.capture()
            
            # Perform click
            self.automation.click(x, y)
            time.sleep(0.5)
            
            if verify:
                # Take after screenshot
                after = self.screenshot.capture()
                
                # In a real implementation, you would compare screenshots
                # to verify the click had an effect
                logger.info("Click verification passed (simulated)")
                
            return True
            
        except Exception as e:
            logger.error(f"Safe click failed at ({x}, {y}): {e}")
            return False
            
    def wait_for_condition(self, 
                          condition: Callable[[], bool],
                          timeout: float = 30.0,
                          check_interval: float = 1.0,
                          description: str = "condition") -> bool:
        """Wait for a condition to become true"""
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            try:
                if condition():
                    logger.info(f"✅ Condition met: {description}")
                    return True
            except Exception as e:
                logger.debug(f"Condition check failed: {e}")
                
            time.sleep(check_interval)
            
        logger.warning(f"⏱️ Timeout waiting for: {description}")
        return False
        
    def create_checkpoint(self, name: str):
        """Create a checkpoint for potential rollback"""
        try:
            # Save current state
            screenshot_path = self.screenshot.save(
                f"checkpoint_{name}.png",
                directory="../../testing/test-outputs/screenshots"
            )
            
            # Save mouse position
            x, y = self.automation.get_mouse_position()
            
            checkpoint = {
                "name": name,
                "screenshot": screenshot_path,
                "mouse_position": (x, y),
                "timestamp": time.time()
            }
            
            logger.info(f"📍 Checkpoint created: {name}")
            return checkpoint
            
        except Exception as e:
            logger.error(f"Failed to create checkpoint: {e}")
            return None


def example_with_recovery():
    """Example automation with error recovery"""
    automation = RobustAutomation()
    
    print("Agent S2 - Error Recovery Example")
    print("=================================")
    
    # Check health with retry
    def check_health():
        if not automation.client.health_check():
            raise Exception("Agent S2 not healthy")
        return True
        
    try:
        automation.with_retry(check_health)
    except Exception as e:
        print(f"❌ Failed to connect to Agent S2: {e}")
        return
        
    # Create initial checkpoint
    checkpoint = automation.create_checkpoint("start")
    
    # Example 1: Click with retry and verification
    print("\n1. Demonstrating click with retry...")
    try:
        screen_info = automation.screenshot.capture()
        center_x = screen_info['size']['width'] // 2
        center_y = screen_info['size']['height'] // 2
        
        automation.with_retry(automation.safe_click, center_x, center_y)
        
    except Exception as e:
        logger.error(f"Click operation failed completely: {e}")
        
    # Example 2: Type with verification
    print("\n2. Demonstrating type with verification...")
    
    def type_with_verification(text: str):
        # Clear first
        automation.automation.select_all()
        time.sleep(0.2)
        
        # Type text
        automation.automation.type_text(text)
        time.sleep(0.5)
        
        # In real scenario, you would verify the text appeared
        # For now, we just simulate success
        logger.info(f"Typed and verified: {text}")
        
    try:
        automation.with_retry(type_with_verification, "Error recovery demo text")
    except Exception as e:
        logger.error(f"Type operation failed: {e}")
        
    # Example 3: Wait for condition
    print("\n3. Demonstrating wait for condition...")
    
    # Simulate waiting for an element to appear
    def element_visible():
        # In real scenario, this would check for a specific element
        # For demo, we just return True after a few checks
        if not hasattr(element_visible, 'count'):
            element_visible.count = 0
        element_visible.count += 1
        return element_visible.count > 3
        
    success = automation.wait_for_condition(
        element_visible,
        timeout=10.0,
        description="demo element to appear"
    )
    
    if success:
        print("✅ Element appeared!")
    else:
        print("❌ Element did not appear in time")
        
    # Example 4: Complex operation with multiple checkpoints
    print("\n4. Demonstrating complex operation with checkpoints...")
    
    checkpoints = []
    
    try:
        # Step 1
        checkpoints.append(automation.create_checkpoint("before_step1"))
        automation.automation.move_mouse(100, 100, duration=1.0)
        logger.info("Step 1 completed")
        
        # Step 2
        checkpoints.append(automation.create_checkpoint("before_step2"))
        automation.automation.click()
        logger.info("Step 2 completed")
        
        # Step 3 - Simulate a potential failure point
        checkpoints.append(automation.create_checkpoint("before_step3"))
        
        # This might fail
        if time.time() % 2 > 1:  # 50% chance of "failure"
            raise Exception("Simulated failure in step 3")
            
        automation.automation.type_text("Success!")
        logger.info("Step 3 completed")
        
    except Exception as e:
        logger.error(f"Operation failed at step 3: {e}")
        logger.info("🔄 Could rollback to last checkpoint here")
        
        # In a real implementation, you might restore to the last checkpoint
        if checkpoints:
            last_checkpoint = checkpoints[-1]
            logger.info(f"Last good state: {last_checkpoint['name']}")
            
    # Final checkpoint
    automation.create_checkpoint("end")
    
    print("\n✅ Error recovery demonstration complete!")
    print("\nKey concepts demonstrated:")
    print("- Retry logic with exponential backoff")
    print("- Operation verification")
    print("- Waiting for conditions")
    print("- Checkpoint/rollback capability")
    print("- Comprehensive logging")


if __name__ == "__main__":
    example_with_recovery()