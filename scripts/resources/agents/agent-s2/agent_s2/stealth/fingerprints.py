"""Browser fingerprint generation and randomization"""

import random
import hashlib
import base64
from typing import Dict, Any, List, Tuple
from datetime import datetime, timedelta


class FingerprintGenerator:
    """Generates randomized browser fingerprints for anti-detection"""
    
    def __init__(self):
        """Initialize fingerprint generator"""
        self._seed = None
        
    def set_seed(self, seed: str) -> None:
        """Set seed for consistent fingerprints
        
        Args:
            seed: Seed string for randomization
        """
        self._seed = seed
        random.seed(hashlib.md5(seed.encode()).hexdigest())
        
    def generate_canvas_noise(self) -> str:
        """Generate canvas fingerprint noise
        
        Returns:
            Base64 encoded noise data
        """
        # Generate deterministic but unique canvas data
        width, height = 280, 60
        noise_data = []
        
        for i in range(width * height):
            # Add subtle noise to pixel values
            r = random.randint(0, 255)
            g = random.randint(0, 255) 
            b = random.randint(0, 255)
            a = 255  # Full opacity
            
            noise_data.extend([r, g, b, a])
            
        # Convert to base64
        noise_bytes = bytes(noise_data)
        return base64.b64encode(noise_bytes).decode('utf-8')
        
    def generate_webgl_noise(self) -> Dict[str, Any]:
        """Generate WebGL fingerprint noise
        
        Returns:
            WebGL parameters with noise
        """
        return {
            "MAX_TEXTURE_SIZE": random.choice([4096, 8192, 16384]),
            "MAX_VIEWPORT_DIMS": [
                random.choice([8192, 16384, 32767]),
                random.choice([8192, 16384, 32767])
            ],
            "ALIASED_LINE_WIDTH_RANGE": [1, random.choice([7, 10, 15])],
            "ALIASED_POINT_SIZE_RANGE": [1, random.choice([255, 511, 1023])],
            "MAX_COMBINED_TEXTURE_IMAGE_UNITS": random.choice([32, 64, 80]),
            "MAX_CUBE_MAP_TEXTURE_SIZE": random.choice([4096, 8192, 16384]),
            "MAX_FRAGMENT_UNIFORM_VECTORS": random.choice([256, 512, 1024]),
            "MAX_RENDERBUFFER_SIZE": random.choice([8192, 16384]),
            "MAX_TEXTURE_IMAGE_UNITS": random.choice([16, 32]),
            "MAX_VARYING_VECTORS": random.choice([15, 30, 32]),
            "MAX_VERTEX_ATTRIBS": random.choice([16, 32]),
            "MAX_VERTEX_TEXTURE_IMAGE_UNITS": random.choice([16, 32]),
            "MAX_VERTEX_UNIFORM_VECTORS": random.choice([256, 512, 1024]),
            "SHADING_LANGUAGE_VERSION": random.choice([
                "WebGL GLSL ES 1.0",
                "WebGL GLSL ES 3.0"
            ]),
            "RENDERER": self._generate_renderer(),
            "VENDOR": self._generate_vendor(),
            "VERSION": random.choice(["WebGL 1.0", "WebGL 2.0"]),
            "extensions": self._generate_webgl_extensions()
        }
        
    def _generate_renderer(self) -> str:
        """Generate WebGL renderer string"""
        renderers = [
            "ANGLE (Intel, Intel(R) HD Graphics 620 Direct3D11 vs_5_0 ps_5_0, D3D11-27.20.100.8935)",
            "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 Direct3D11 vs_5_0 ps_5_0, D3D11-31.0.15.2647)",
            "ANGLE (AMD, AMD Radeon(TM) Vega 8 Graphics Direct3D11 vs_5_0 ps_5_0, D3D11-30.0.13044.0)",
            "Mesa DRI Intel(R) HD Graphics 620 (KBL GT2)",
            "Apple GPU",
            "Mali-G76 MP12",
            "Adreno (TM) 640",
        ]
        return random.choice(renderers)
        
    def _generate_vendor(self) -> str:
        """Generate WebGL vendor string"""
        vendors = [
            "Google Inc. (Intel)",
            "Google Inc. (NVIDIA)",
            "Google Inc. (AMD)",
            "Intel Open Source Technology Center",
            "Apple Inc.",
            "ARM",
            "Qualcomm",
        ]
        return random.choice(vendors)
        
    def _generate_webgl_extensions(self) -> List[str]:
        """Generate list of WebGL extensions"""
        all_extensions = [
            "ANGLE_instanced_arrays",
            "EXT_blend_minmax",
            "EXT_color_buffer_float",
            "EXT_color_buffer_half_float",
            "EXT_disjoint_timer_query",
            "EXT_float_blend",
            "EXT_frag_depth",
            "EXT_shader_texture_lod",
            "EXT_texture_compression_bptc",
            "EXT_texture_compression_rgtc",
            "EXT_texture_filter_anisotropic",
            "EXT_sRGB",
            "KHR_parallel_shader_compile",
            "OES_element_index_uint",
            "OES_fbo_render_mipmap",
            "OES_standard_derivatives",
            "OES_texture_float",
            "OES_texture_float_linear",
            "OES_texture_half_float",
            "OES_texture_half_float_linear",
            "OES_vertex_array_object",
            "WEBGL_color_buffer_float",
            "WEBGL_compressed_texture_s3tc",
            "WEBGL_compressed_texture_s3tc_srgb",
            "WEBGL_debug_renderer_info",
            "WEBGL_debug_shaders",
            "WEBGL_depth_texture",
            "WEBGL_draw_buffers",
            "WEBGL_lose_context",
            "WEBGL_multi_draw",
        ]
        
        # Select random subset
        num_extensions = random.randint(20, len(all_extensions))
        return random.sample(all_extensions, num_extensions)
        
    def generate_audio_noise(self) -> Dict[str, float]:
        """Generate audio context fingerprint noise
        
        Returns:
            Audio parameters with noise
        """
        base_frequency = 44100
        
        return {
            "sampleRate": base_frequency + random.randint(-10, 10),
            "channelCount": random.choice([2, 6]),
            "channelCountMode": "explicit",
            "channelInterpretation": "speakers",
            "maxChannelCount": random.choice([2, 6, 8]),
            "numberOfInputs": 1,
            "numberOfOutputs": 1,
            "latency": round(random.uniform(0.01, 0.05), 4),
            "baseLatency": round(random.uniform(0.005, 0.02), 4),
        }
        
    def generate_speech_synthesis_noise(self) -> List[Dict[str, str]]:
        """Generate speech synthesis voices
        
        Returns:
            List of available voices
        """
        voice_templates = [
            {"name": "Microsoft David - English (United States)", "lang": "en-US", "local": True},
            {"name": "Microsoft Mark - English (United States)", "lang": "en-US", "local": True},
            {"name": "Microsoft Zira - English (United States)", "lang": "en-US", "local": True},
            {"name": "Google US English", "lang": "en-US", "local": False},
            {"name": "Google UK English Female", "lang": "en-GB", "local": False},
            {"name": "Google UK English Male", "lang": "en-GB", "local": False},
            {"name": "Google español", "lang": "es-ES", "local": False},
            {"name": "Google français", "lang": "fr-FR", "local": False},
            {"name": "Google Deutsch", "lang": "de-DE", "local": False},
            {"name": "Google italiano", "lang": "it-IT", "local": False},
            {"name": "Google 日本語", "lang": "ja-JP", "local": False},
            {"name": "Google 한국의", "lang": "ko-KR", "local": False},
            {"name": "Google 中文", "lang": "zh-CN", "local": False},
        ]
        
        # Select random subset
        num_voices = random.randint(3, len(voice_templates))
        return random.sample(voice_templates, num_voices)
        
    def generate_client_rects_noise(self) -> float:
        """Generate DOMRect noise for getClientRects
        
        Returns:
            Noise factor for client rects
        """
        # Add tiny variations to rect measurements
        return round(random.uniform(-0.125, 0.125), 3)
        
    def generate_screen_noise(self) -> Dict[str, Any]:
        """Generate screen property noise
        
        Returns:
            Screen properties with noise
        """
        base_width = random.choice([1920, 1366, 1440, 1536])
        base_height = random.choice([1080, 768, 900, 864])
        
        return {
            "width": base_width,
            "height": base_height,
            "availWidth": base_width,
            "availHeight": base_height - random.choice([40, 72, 100]),  # Taskbar
            "colorDepth": random.choice([24, 32]),
            "pixelDepth": random.choice([24, 32]),
            "availLeft": 0,
            "availTop": 0,
            "orientation": {
                "angle": 0,
                "type": "landscape-primary"
            }
        }
        
    def generate_permission_noise(self) -> Dict[str, str]:
        """Generate permission states
        
        Returns:
            Permission states
        """
        states = ["granted", "denied", "prompt"]
        
        return {
            "geolocation": random.choice(states),
            "notifications": random.choice(states),
            "push": random.choice(["denied", "prompt"]),
            "midi": random.choice(["denied", "prompt"]),
            "camera": random.choice(states),
            "microphone": random.choice(states),
            "speaker": random.choice(["granted", "denied"]),
            "device-info": random.choice(["denied", "prompt"]),
            "background-sync": random.choice(["granted", "denied"]),
            "bluetooth": random.choice(["denied", "prompt"]),
            "persistent-storage": random.choice(["granted", "prompt"]),
            "ambient-light-sensor": "denied",
            "accelerometer": "denied",
            "gyroscope": "denied",
            "magnetometer": "denied",
            "clipboard": random.choice(["granted", "prompt"]),
            "accessibility-events": "denied"
        }
        
    def generate_media_devices(self) -> List[Dict[str, Any]]:
        """Generate media device list
        
        Returns:
            List of media devices
        """
        devices = []
        
        # Add random cameras
        num_cameras = random.choice([0, 1, 2])
        for i in range(num_cameras):
            devices.append({
                "deviceId": self._generate_device_id(),
                "kind": "videoinput",
                "label": f"Camera {i+1}" if i > 0 else "Integrated Camera",
                "groupId": self._generate_device_id()
            })
            
        # Add random microphones
        num_mics = random.choice([1, 2, 3])
        for i in range(num_mics):
            devices.append({
                "deviceId": self._generate_device_id(),
                "kind": "audioinput",
                "label": f"Microphone {i+1}" if i > 0 else "Default Microphone",
                "groupId": self._generate_device_id()
            })
            
        # Add audio outputs
        devices.append({
            "deviceId": self._generate_device_id(),
            "kind": "audiooutput",
            "label": "Default Speaker",
            "groupId": self._generate_device_id()
        })
        
        return devices
        
    def _generate_device_id(self) -> str:
        """Generate random device ID"""
        chars = "0123456789abcdef"
        return "".join(random.choice(chars) for _ in range(64))
        
    def generate_battery_info(self) -> Dict[str, Any]:
        """Generate battery information
        
        Returns:
            Battery status
        """
        charging = random.choice([True, False])
        level = round(random.uniform(0.2, 1.0), 2)
        
        return {
            "charging": charging,
            "chargingTime": random.choice([0, float('inf')]) if charging else float('inf'),
            "dischargingTime": float('inf') if charging else random.randint(3600, 28800),
            "level": level
        }
        
    def generate_connection_info(self) -> Dict[str, Any]:
        """Generate network connection information
        
        Returns:
            Connection properties
        """
        return {
            "downlink": round(random.uniform(1.0, 10.0), 2),
            "effectiveType": random.choice(["4g", "3g", "2g", "slow-2g"]),
            "rtt": random.choice([50, 100, 150, 200, 300]),
            "saveData": False,
            "type": random.choice(["wifi", "cellular", "ethernet", "bluetooth"])
        }
        
    def generate_timing_noise(self) -> float:
        """Generate timing noise for performance APIs
        
        Returns:
            Timing offset in milliseconds
        """
        # Add small random delays to timing measurements
        return round(random.uniform(0.001, 0.1), 3)