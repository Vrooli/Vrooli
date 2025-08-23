"""Screenshot capture service for Agent S2"""

import os
import io
import base64
import logging
import time
import hashlib
from typing import Optional, List, Dict, Any, Tuple
from concurrent.futures import ThreadPoolExecutor

import pyautogui
from PIL import Image, ImageChops, ImageDraw

from ...config import Config
from .window_manager import WindowManager, WindowManagerError

logger = logging.getLogger(__name__)


class ScreenshotService:
    """Service for handling screenshot operations"""
    
    def __init__(self):
        """Initialize screenshot service"""
        # Configure pyautogui
        pyautogui.FAILSAFE = False  # Disable failsafe in container
        pyautogui.PAUSE = 0.1  # Small pause between actions
        
        # Initialize window manager for window capture
        self.window_manager = WindowManager()
        
        # Change detection state
        self._last_screenshot = None
        self._last_screenshot_hash = None
        self._monitoring_active = False
        
    def capture(self, 
                format: str = "png",
                quality: int = 95,
                region: Optional[List[int]] = None,
                max_dimension: int = 4096) -> Dict[str, Any]:
        """Capture a screenshot
        
        Args:
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            max_dimension: Maximum width or height (to prevent memory issues)
            
        Returns:
            Dictionary with screenshot data and metadata
            
        Raises:
            ValueError: If parameters are invalid
            MemoryError: If screenshot would be too large
        """
        # Validate format
        format = format.lower()
        if format not in ["png", "jpeg", "jpg"]:
            raise ValueError(f"Invalid format: {format}. Must be png or jpeg")
            
        # Normalize jpeg/jpg
        if format == "jpg":
            format = "jpeg"
            
        # Validate region dimensions if provided
        if region and len(region) == 4:
            x, y, width, height = region
            if width > max_dimension or height > max_dimension:
                raise ValueError(f"Region dimensions too large. Maximum: {max_dimension}x{max_dimension}")
            if width <= 0 or height <= 0:
                raise ValueError("Region width and height must be positive")
            
        # Take screenshot
        if region and len(region) == 4:
            # Convert [x, y, width, height] to (x, y, width, height) for pyautogui
            x, y, width, height = region
            screenshot = pyautogui.screenshot(region=(x, y, width, height))
        else:
            screenshot = pyautogui.screenshot()
            
        # Check screenshot dimensions
        if screenshot.width > max_dimension or screenshot.height > max_dimension:
            # Resize if too large
            logger.warning(f"Screenshot too large ({screenshot.width}x{screenshot.height}), resizing to fit {max_dimension}")
            screenshot.thumbnail((max_dimension, max_dimension), Image.Resampling.LANCZOS)
        
        # Convert to bytes
        img_buffer = io.BytesIO()
        try:
            if format == "jpeg":
                # Convert RGBA to RGB for JPEG
                screenshot = screenshot.convert('RGB')
                screenshot.save(img_buffer, format="JPEG", quality=quality, optimize=True)
            else:
                screenshot.save(img_buffer, format="PNG", optimize=True)
                
            img_buffer.seek(0)
            
            # Check buffer size
            buffer_size = img_buffer.tell()
            if buffer_size > 50 * 1024 * 1024:  # 50MB limit
                raise MemoryError(f"Screenshot buffer too large: {buffer_size / 1024 / 1024:.2f}MB")
            
            # Encode to base64
            img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
            
            # Verify data URI is properly formatted
            data_uri = f"data:image/{format};base64,{img_base64}"
            if not data_uri.startswith("data:image/"):
                raise ValueError("Failed to create valid data URI")
            
            return {
                "format": format,
                "size": {"width": screenshot.width, "height": screenshot.height},
                "data": data_uri,
                "file_size_mb": buffer_size / (1024 * 1024)
            }
        finally:
            img_buffer.close()
        
    def capture_to_file(self, 
                       filename: str,
                       format: Optional[str] = None,
                       quality: int = 95,
                       region: Optional[List[int]] = None) -> str:
        """Capture screenshot and save to file
        
        Args:
            filename: Output filename
            format: Image format (auto-detected from filename if None)
            quality: JPEG quality (1-100)
            region: Optional region [x, y, width, height]
            
        Returns:
            Path to saved file
        """
        # Auto-detect format from filename
        if format is None:
            ext = os.path.splitext(filename)[1].lower()
            format = ext[1:] if ext else "png"
            
        # Capture screenshot
        data = self.capture(format=format, quality=quality, region=region)
        
        # Decode base64
        img_data = data["data"].split(",")[1]
        img_bytes = base64.b64decode(img_data)
        
        # Save to file
        with open(filename, "wb") as f:
            f.write(img_bytes)
            
        return filename
        
    def get_screen_info(self) -> Dict[str, Any]:
        """Get screen information
        
        Returns:
            Dictionary with screen information
        """
        width, height = pyautogui.size()
        
        return {
            "size": {"width": width, "height": height},
            "display": os.environ.get("DISPLAY", ":99"),
            "color_depth": Config.SCREEN_DEPTH
        }
        
    def find_on_screen(self, 
                      image_path: str,
                      confidence: float = 0.8,
                      grayscale: bool = False) -> Optional[Tuple[int, int, int, int]]:
        """Find an image on screen
        
        Args:
            image_path: Path to image to find
            confidence: Matching confidence (0-1)
            grayscale: Use grayscale matching
            
        Returns:
            Tuple of (x, y, width, height) if found, None otherwise
        """
        try:
            location = pyautogui.locateOnScreen(
                image_path,
                confidence=confidence,
                grayscale=grayscale
            )
            return location
        except Exception as e:
            logger.error(f"Image search failed: {e}")
            return None
            
    def find_all_on_screen(self,
                          image_path: str,
                          confidence: float = 0.8,
                          grayscale: bool = False) -> List[Tuple[int, int, int, int]]:
        """Find all instances of an image on screen
        
        Args:
            image_path: Path to image to find
            confidence: Matching confidence (0-1)
            grayscale: Use grayscale matching
            
        Returns:
            List of (x, y, width, height) tuples
        """
        try:
            locations = list(pyautogui.locateAllOnScreen(
                image_path,
                confidence=confidence,
                grayscale=grayscale
            ))
            return locations
        except Exception as e:
            logger.error(f"Image search failed: {e}")
            return []
            
    def wait_for_image(self,
                      image_path: str,
                      timeout: float = 30.0,
                      confidence: float = 0.8) -> Optional[Tuple[int, int, int, int]]:
        """Wait for an image to appear on screen
        
        Args:
            image_path: Path to image to find
            timeout: Maximum wait time in seconds
            confidence: Matching confidence (0-1)
            
        Returns:
            Tuple of (x, y, width, height) if found, None if timeout
        """
        import time
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            location = self.find_on_screen(image_path, confidence)
            if location:
                return location
            time.sleep(0.5)
            
        return None
    
    def capture_window(self, 
                      window_title: str,
                      format: str = "png",
                      quality: int = 95) -> Dict[str, Any]:
        """Capture specific window by title
        
        Args:
            window_title: Window title to capture
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            
        Returns:
            Dictionary with screenshot data and metadata
            
        Raises:
            WindowManagerError: If window not found or capture fails
        """
        try:
            # Find windows matching the title
            all_windows = self.window_manager.list_all_windows()
            matching_windows = [
                w for w in all_windows 
                if window_title.lower() in w.title.lower()
            ]
            
            if not matching_windows:
                raise WindowManagerError(f"No window found with title containing: {window_title}")
            
            # Use the first matching window (or most recently active)
            window = matching_windows[0]
            
            # Focus the window first to ensure it's on top
            if not self.window_manager.focus_window_by_id(window.window_id):
                logger.warning(f"Could not focus window {window.window_id}, capturing anyway")
            
            # Small delay to ensure focus change takes effect
            time.sleep(0.2)
            
            # Capture the window region
            geometry = window.geometry
            region = [geometry['x'], geometry['y'], geometry['width'], geometry['height']]
            
            result = self.capture(format=format, quality=quality, region=region)
            
            # Add window information to the result
            result['window_info'] = {
                'window_id': window.window_id,
                'title': window.title,
                'app_name': window.app_name,
                'geometry': geometry
            }
            
            return result
            
        except Exception as e:
            logger.error(f"Window capture failed: {e}")
            raise WindowManagerError(f"Failed to capture window '{window_title}': {str(e)}")
    
    def capture_window_by_id(self,
                            window_id: str,
                            format: str = "png",
                            quality: int = 95) -> Dict[str, Any]:
        """Capture specific window by ID
        
        Args:
            window_id: Window ID to capture
            format: Image format (png or jpeg)
            quality: JPEG quality (1-100)
            
        Returns:
            Dictionary with screenshot data and metadata
            
        Raises:
            WindowManagerError: If window not found or capture fails
        """
        try:
            window = self.window_manager.get_window_info(window_id)
            if not window:
                raise WindowManagerError(f"Window with ID {window_id} not found")
            
            # Focus the window first
            if not self.window_manager.focus_window_by_id(window.window_id):
                logger.warning(f"Could not focus window {window.window_id}, capturing anyway")
            
            time.sleep(0.2)
            
            # Capture the window region
            geometry = window.geometry
            region = [geometry['x'], geometry['y'], geometry['width'], geometry['height']]
            
            result = self.capture(format=format, quality=quality, region=region)
            
            # Add window information
            result['window_info'] = {
                'window_id': window.window_id,
                'title': window.title,
                'app_name': window.app_name,
                'geometry': geometry
            }
            
            return result
            
        except Exception as e:
            logger.error(f"Window capture by ID failed: {e}")
            raise WindowManagerError(f"Failed to capture window ID '{window_id}': {str(e)}")
    
    def compare_screenshots(self, 
                           screenshot1: Dict[str, Any], 
                           screenshot2: Dict[str, Any],
                           method: str = "mse") -> Dict[str, Any]:
        """Compare two screenshots for similarity
        
        Args:
            screenshot1: First screenshot data
            screenshot2: Second screenshot data
            method: Comparison method (mse, ssim, histogram)
            
        Returns:
            Dictionary with comparison results
        """
        try:
            # Extract images from screenshot data
            img1 = self._extract_image_from_data(screenshot1)
            img2 = self._extract_image_from_data(screenshot2)
            
            # Ensure images are same size
            if img1.size != img2.size:
                # Resize to smaller dimensions
                min_width = min(img1.width, img2.width)
                min_height = min(img1.height, img2.height)
                img1 = img1.resize((min_width, min_height), Image.Resampling.LANCZOS)
                img2 = img2.resize((min_width, min_height), Image.Resampling.LANCZOS)
            
            # Convert to numpy arrays for comparison
            arr1 = np.array(img1)
            arr2 = np.array(img2)
            
            result = {
                'method': method,
                'image_size': img1.size,
                'timestamp': time.time()
            }
            
            if method == "mse":
                # Mean Squared Error
                mse = np.mean((arr1 - arr2) ** 2)
                similarity = 1.0 / (1.0 + mse / 1000.0)  # Normalize to 0-1 range
                result.update({
                    'mse': float(mse),
                    'similarity': float(similarity)
                })
            
            elif method == "histogram":
                # Histogram comparison
                hist1 = img1.histogram()
                hist2 = img2.histogram()
                
                # Calculate correlation coefficient
                correlation = self._calculate_histogram_correlation(hist1, hist2)
                result.update({
                    'histogram_correlation': float(correlation),
                    'similarity': float(correlation)
                })
            
            elif method == "pixel_diff":
                # Pixel-by-pixel difference
                diff = ImageChops.difference(img1, img2)
                diff_arr = np.array(diff)
                total_pixels = diff_arr.size
                changed_pixels = np.count_nonzero(diff_arr)
                
                similarity = 1.0 - (changed_pixels / total_pixels)
                result.update({
                    'changed_pixels': int(changed_pixels),
                    'total_pixels': int(total_pixels),
                    'change_percentage': float(changed_pixels / total_pixels * 100),
                    'similarity': float(similarity)
                })
            
            else:
                # Default to MSE
                mse = np.mean((arr1 - arr2) ** 2)
                similarity = 1.0 / (1.0 + mse / 1000.0)
                result.update({
                    'mse': float(mse),
                    'similarity': float(similarity)
                })
            
            return result
            
        except Exception as e:
            logger.error(f"Screenshot comparison failed: {e}")
            raise ValueError(f"Failed to compare screenshots: {str(e)}")
    
    def _extract_image_from_data(self, screenshot_data: Dict[str, Any]) -> Image.Image:
        """Extract PIL Image from screenshot data"""
        data = screenshot_data.get('data', '')
        if data.startswith('data:image'):
            # Extract base64 part
            base64_data = data.split(',')[1]
        else:
            base64_data = data
            
        # Decode and create Image
        img_bytes = base64.b64decode(base64_data)
        return Image.open(io.BytesIO(img_bytes))
    
    def _calculate_histogram_correlation(self, hist1: List[int], hist2: List[int]) -> float:
        """Calculate correlation coefficient between histograms"""
        try:
            # Convert to numpy arrays
            h1 = np.array(hist1, dtype=np.float64)
            h2 = np.array(hist2, dtype=np.float64)
            
            # Calculate correlation coefficient
            correlation_matrix = np.corrcoef(h1, h2)
            correlation = correlation_matrix[0, 1]
            
            # Handle NaN case (identical constant images)
            if np.isnan(correlation):
                return 1.0 if np.array_equal(h1, h2) else 0.0
                
            return max(0.0, correlation)  # Ensure non-negative
            
        except Exception:
            return 0.0
    
    def detect_changes(self, 
                      interval: float = 1.0,
                      timeout: float = 30.0,
                      threshold: float = 0.05) -> List[Dict[str, Any]]:
        """Monitor screen for changes
        
        Args:
            interval: Check interval in seconds
            timeout: Maximum monitoring time in seconds
            threshold: Change detection threshold (0-1, lower = more sensitive)
            
        Returns:
            List of detected changes with timestamps and details
        """
        changes = []
        start_time = time.time()
        last_screenshot = None
        
        try:
            self._monitoring_active = True
            
            while time.time() - start_time < timeout and self._monitoring_active:
                current_screenshot = self.capture()
                current_time = time.time()
                
                if last_screenshot:
                    # Compare with previous screenshot
                    comparison = self.compare_screenshots(
                        last_screenshot, 
                        current_screenshot, 
                        method="pixel_diff"
                    )
                    
                    # Check if change exceeds threshold
                    change_percentage = comparison.get('change_percentage', 0)
                    if change_percentage > (threshold * 100):
                        change_info = {
                            'timestamp': current_time,
                            'change_percentage': change_percentage,
                            'similarity': comparison.get('similarity', 1.0),
                            'changed_pixels': comparison.get('changed_pixels', 0),
                            'screenshot_before': last_screenshot,
                            'screenshot_after': current_screenshot,
                            'detection_method': 'pixel_diff'
                        }
                        changes.append(change_info)
                        logger.info(f"Change detected: {change_percentage:.2f}% of pixels changed")
                
                last_screenshot = current_screenshot
                
                # Wait for next check
                if self._monitoring_active:
                    time.sleep(interval)
            
            return changes
            
        except Exception as e:
            logger.error(f"Change detection failed: {e}")
            raise ValueError(f"Failed to detect changes: {str(e)}")
        finally:
            self._monitoring_active = False
    
    def stop_change_detection(self) -> None:
        """Stop active change detection"""
        self._monitoring_active = False
    
    def get_pixel_color(self, x: int, y: int) -> Tuple[int, int, int]:
        """Get color of specific pixel
        
        Args:
            x: X coordinate
            y: Y coordinate
            
        Returns:
            RGB color tuple
            
        Raises:
            ValueError: If coordinates are invalid
        """
        try:
            # Get screen dimensions
            screen_width, screen_height = pyautogui.size()
            
            # Validate coordinates
            if not (0 <= x < screen_width and 0 <= y < screen_height):
                raise ValueError(f"Coordinates ({x}, {y}) are out of screen bounds (0, 0, {screen_width}, {screen_height})")
            
            # Capture 1x1 region at the specified coordinates
            screenshot = pyautogui.screenshot(region=(x, y, 1, 1))
            
            # Get the pixel color
            pixel_color = screenshot.getpixel((0, 0))
            
            # Convert RGBA to RGB if necessary
            if len(pixel_color) == 4:
                pixel_color = pixel_color[:3]
            
            return pixel_color
            
        except Exception as e:
            logger.error(f"Failed to get pixel color at ({x}, {y}): {e}")
            raise ValueError(f"Failed to get pixel color: {str(e)}")
    
    def get_pixel_colors_region(self, 
                               x: int, y: int, 
                               width: int, height: int) -> List[List[Tuple[int, int, int]]]:
        """Get colors of all pixels in a region
        
        Args:
            x: Starting X coordinate
            y: Starting Y coordinate  
            width: Region width
            height: Region height
            
        Returns:
            2D list of RGB color tuples [row][col] = (r, g, b)
        """
        try:
            # Capture the region
            screenshot = pyautogui.screenshot(region=(x, y, width, height))
            
            # Convert to RGB if needed
            if screenshot.mode != 'RGB':
                screenshot = screenshot.convert('RGB')
            
            # Extract pixel colors
            colors = []
            for row in range(height):
                row_colors = []
                for col in range(width):
                    pixel_color = screenshot.getpixel((col, row))
                    row_colors.append(pixel_color)
                colors.append(row_colors)
            
            return colors
            
        except Exception as e:
            logger.error(f"Failed to get pixel colors for region ({x}, {y}, {width}, {height}): {e}")
            raise ValueError(f"Failed to get pixel colors: {str(e)}")
    
    def create_difference_image(self, 
                               screenshot1: Dict[str, Any], 
                               screenshot2: Dict[str, Any]) -> Dict[str, Any]:
        """Create a visual difference image between two screenshots
        
        Args:
            screenshot1: First screenshot data
            screenshot2: Second screenshot data
            
        Returns:
            Dictionary with difference image data
        """
        try:
            # Extract images
            img1 = self._extract_image_from_data(screenshot1)
            img2 = self._extract_image_from_data(screenshot2)
            
            # Ensure same size
            if img1.size != img2.size:
                min_width = min(img1.width, img2.width)
                min_height = min(img1.height, img2.height)
                img1 = img1.resize((min_width, min_height), Image.Resampling.LANCZOS)
                img2 = img2.resize((min_width, min_height), Image.Resampling.LANCZOS)
            
            # Create difference image
            diff_img = ImageChops.difference(img1, img2)
            
            # Enhance the difference for better visibility
            enhanced_diff = ImageChops.multiply(diff_img, 3)  # Make differences more visible
            
            # Convert to base64
            img_buffer = io.BytesIO()
            enhanced_diff.save(img_buffer, format="PNG")
            img_buffer.seek(0)
            
            img_base64 = base64.b64encode(img_buffer.getvalue()).decode('utf-8')
            
            return {
                'format': 'png',
                'size': {'width': enhanced_diff.width, 'height': enhanced_diff.height},
                'data': f"data:image/png;base64,{img_base64}",
                'type': 'difference_image'
            }
            
        except Exception as e:
            logger.error(f"Failed to create difference image: {e}")
            raise ValueError(f"Failed to create difference image: {str(e)}")