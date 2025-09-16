#!/usr/bin/env python3
"""
Real-time Stream Processing for Splink Record Linkage
Enables continuous record matching for streaming data
"""

import os
import json
import asyncio
import logging
from datetime import datetime
from typing import Dict, List, Optional, Any, AsyncGenerator
from uuid import uuid4
import time
from collections import deque
from threading import Lock

import redis.asyncio as redis
from pydantic import BaseModel, Field

# Import Splink engine
from api.splink_engine import get_splink_engine

# Configure logging
logger = logging.getLogger(__name__)

# Configuration
REDIS_HOST = os.getenv("REDIS_HOST", "localhost")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6380"))
STREAM_BUFFER_SIZE = int(os.getenv("STREAM_BUFFER_SIZE", "1000"))
STREAM_BATCH_SIZE = int(os.getenv("STREAM_BATCH_SIZE", "100"))
STREAM_BATCH_TIMEOUT = float(os.getenv("STREAM_BATCH_TIMEOUT", "5.0"))  # seconds

class StreamConfig(BaseModel):
    """Configuration for stream processing"""
    stream_name: str
    source_type: str = Field(default="redis", pattern="^(redis|kafka|rabbitmq|webhook)$")
    match_threshold: float = Field(default=0.85, ge=0, le=1)
    buffer_size: int = Field(default=STREAM_BUFFER_SIZE, ge=10, le=10000)
    batch_size: int = Field(default=STREAM_BATCH_SIZE, ge=1, le=1000)
    batch_timeout: float = Field(default=STREAM_BATCH_TIMEOUT, ge=0.1, le=60)
    comparison_columns: List[str] = Field(default_factory=list)
    blocking_rules: List[str] = Field(default_factory=list)
    deduplication_window: int = Field(default=3600, ge=60, le=86400)  # seconds

class StreamRecord(BaseModel):
    """Individual record in the stream"""
    id: str
    data: Dict[str, Any]
    timestamp: Optional[datetime] = None
    metadata: Optional[Dict[str, Any]] = None

class StreamMatch(BaseModel):
    """Result of a stream matching operation"""
    incoming_id: str
    matched_id: str
    confidence: float
    match_type: str  # 'exact', 'probable', 'possible'
    timestamp: datetime
    metadata: Optional[Dict[str, Any]] = None

class StreamProcessor:
    """
    Real-time stream processor for continuous record matching
    """
    
    def __init__(self, config: StreamConfig):
        self.config = config
        self.redis_client = None
        self.buffer = deque(maxlen=config.buffer_size)
        self.buffer_lock = Lock()
        self.reference_data = {}  # In-memory cache of recent records
        self.splink_engine = None
        self.processing = False
        self.stats = {
            "records_processed": 0,
            "matches_found": 0,
            "exact_matches": 0,
            "probable_matches": 0,
            "possible_matches": 0,
            "processing_time_avg": 0,
            "last_processed": None
        }
    
    async def initialize(self):
        """Initialize the stream processor"""
        try:
            # Connect to Redis for stream management with timeout
            self.redis_client = await asyncio.wait_for(
                redis.from_url(
                    f"redis://{REDIS_HOST}:{REDIS_PORT}",
                    decode_responses=True,
                    socket_connect_timeout=2,
                    retry_on_error=False
                ),
                timeout=3.0
            )
            
            # Initialize Splink engine
            self.splink_engine = get_splink_engine()
            
            # Create stream if it doesn't exist
            try:
                await self.redis_client.xadd(
                    self.config.stream_name,
                    {"init": "stream"},
                    id="0-1"
                )
            except:
                pass  # Stream already exists
            
            logger.info(f"Stream processor initialized for {self.config.stream_name}")
            return True
            
        except (ConnectionError, asyncio.TimeoutError) as e:
            logger.warning(f"Redis not available for stream processing: {e}")
            # Continue without Redis - use in-memory processing only
            self.redis_client = None
            self.splink_engine = get_splink_engine()
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize stream processor: {e}")
            return False
    
    async def start(self):
        """Start processing the stream"""
        if not self.redis_client:
            await self.initialize()
        
        self.processing = True
        logger.info(f"Starting stream processing for {self.config.stream_name}")
        
        # Start background tasks
        await asyncio.gather(
            self._consume_stream(),
            self._process_batches()
        )
    
    async def stop(self):
        """Stop processing the stream"""
        self.processing = False
        
        # Process remaining buffer
        if self.buffer:
            await self._process_buffer()
        
        # Close connections
        if self.redis_client:
            await self.redis_client.close()
        
        logger.info(f"Stopped stream processing for {self.config.stream_name}")
    
    async def _consume_stream(self):
        """Consume records from the stream"""
        if not self.redis_client:
            # If Redis is not available, just sleep
            while self.processing:
                await asyncio.sleep(1)
            return
            
        last_id = "0"
        
        while self.processing:
            try:
                # Read from Redis stream
                messages = await self.redis_client.xread(
                    {self.config.stream_name: last_id},
                    count=self.config.batch_size,
                    block=1000  # 1 second timeout
                )
                
                if messages:
                    stream_name, stream_messages = messages[0]
                    
                    for message_id, data in stream_messages:
                        # Parse record
                        record = StreamRecord(
                            id=message_id,
                            data=data,
                            timestamp=datetime.utcnow()
                        )
                        
                        # Add to buffer
                        with self.buffer_lock:
                            self.buffer.append(record)
                        
                        last_id = message_id
                        
                        # Trigger immediate processing if buffer is full
                        if len(self.buffer) >= self.config.batch_size:
                            asyncio.create_task(self._process_buffer())
                
            except Exception as e:
                logger.error(f"Error consuming stream: {e}")
                await asyncio.sleep(1)
    
    async def _process_batches(self):
        """Process batches periodically based on timeout"""
        while self.processing:
            await asyncio.sleep(self.config.batch_timeout)
            
            if self.buffer:
                await self._process_buffer()
    
    async def _process_buffer(self):
        """Process records in the buffer"""
        with self.buffer_lock:
            if not self.buffer:
                return
            
            # Extract batch
            batch_size = min(len(self.buffer), self.config.batch_size)
            batch = [self.buffer.popleft() for _ in range(batch_size)]
        
        start_time = time.time()
        matches = []
        
        try:
            for record in batch:
                # Find matches against reference data
                record_matches = await self._find_matches(record)
                matches.extend(record_matches)
                
                # Add to reference data for future matching
                self._update_reference_data(record)
                
                # Update stats
                self.stats["records_processed"] += 1
                self.stats["last_processed"] = datetime.utcnow().isoformat()
            
            # Store matches
            if matches:
                await self._store_matches(matches)
                self.stats["matches_found"] += len(matches)
            
            # Update processing time
            processing_time = time.time() - start_time
            self.stats["processing_time_avg"] = (
                (self.stats["processing_time_avg"] * 0.9) + (processing_time * 0.1)
            )
            
            logger.debug(f"Processed batch of {len(batch)} records in {processing_time:.2f}s")
            
        except Exception as e:
            logger.error(f"Error processing batch: {e}")
    
    async def _find_matches(self, record: StreamRecord) -> List[StreamMatch]:
        """Find matches for a record against reference data"""
        matches = []
        
        # Quick exact match check
        for ref_id, ref_data in self.reference_data.items():
            if self._is_exact_match(record.data, ref_data["data"]):
                match = StreamMatch(
                    incoming_id=record.id,
                    matched_id=ref_id,
                    confidence=1.0,
                    match_type="exact",
                    timestamp=datetime.utcnow()
                )
                matches.append(match)
                self.stats["exact_matches"] += 1
                continue
            
            # Probabilistic matching using simplified algorithm
            confidence = self._calculate_match_confidence(record.data, ref_data["data"])
            
            if confidence >= self.config.match_threshold:
                match_type = "probable" if confidence >= 0.9 else "possible"
                match = StreamMatch(
                    incoming_id=record.id,
                    matched_id=ref_id,
                    confidence=confidence,
                    match_type=match_type,
                    timestamp=datetime.utcnow()
                )
                matches.append(match)
                
                if match_type == "probable":
                    self.stats["probable_matches"] += 1
                else:
                    self.stats["possible_matches"] += 1
        
        return matches
    
    def _is_exact_match(self, data1: Dict, data2: Dict) -> bool:
        """Check if two records are exact matches"""
        if not self.config.comparison_columns:
            return False
        
        for col in self.config.comparison_columns:
            if col in data1 and col in data2:
                if data1[col] != data2[col]:
                    return False
        return True
    
    def _calculate_match_confidence(self, data1: Dict, data2: Dict) -> float:
        """Calculate match confidence using simplified algorithm"""
        if not self.config.comparison_columns:
            return 0.0
        
        scores = []
        for col in self.config.comparison_columns:
            if col in data1 and col in data2:
                # Simple string similarity
                val1 = str(data1[col]).lower()
                val2 = str(data2[col]).lower()
                
                if val1 == val2:
                    scores.append(1.0)
                elif val1 in val2 or val2 in val1:
                    scores.append(0.7)
                else:
                    # Calculate character overlap
                    common = len(set(val1) & set(val2))
                    total = len(set(val1) | set(val2))
                    scores.append(common / total if total > 0 else 0)
        
        return sum(scores) / len(scores) if scores else 0.0
    
    def _update_reference_data(self, record: StreamRecord):
        """Update reference data with new record"""
        # Add record to reference data
        self.reference_data[record.id] = {
            "data": record.data,
            "timestamp": time.time()
        }
        
        # Clean old records outside deduplication window
        current_time = time.time()
        cutoff_time = current_time - self.config.deduplication_window
        
        to_remove = [
            ref_id for ref_id, ref_data in self.reference_data.items()
            if ref_data["timestamp"] < cutoff_time
        ]
        
        for ref_id in to_remove:
            del self.reference_data[ref_id]
    
    async def _store_matches(self, matches: List[StreamMatch]):
        """Store matches in Redis"""
        pipeline = self.redis_client.pipeline()
        
        for match in matches:
            # Store in sorted set by timestamp
            pipeline.zadd(
                f"{self.config.stream_name}:matches",
                {json.dumps(match.dict()): match.timestamp.timestamp()}
            )
            
            # Store in hash for quick lookup
            pipeline.hset(
                f"{self.config.stream_name}:match_lookup",
                match.incoming_id,
                json.dumps(match.dict())
            )
        
        await pipeline.execute()
    
    async def get_stats(self) -> Dict:
        """Get stream processing statistics"""
        return self.stats
    
    async def get_recent_matches(self, limit: int = 100) -> List[StreamMatch]:
        """Get recent matches from the stream"""
        matches = await self.redis_client.zrevrange(
            f"{self.config.stream_name}:matches",
            0,
            limit - 1
        )
        
        return [StreamMatch(**json.loads(m)) for m in matches]
    
    async def add_record(self, record: Dict[str, Any]) -> str:
        """Add a record to the stream"""
        message_id = await self.redis_client.xadd(
            self.config.stream_name,
            record
        )
        return message_id

# Singleton stream processors
stream_processors: Dict[str, StreamProcessor] = {}

async def get_stream_processor(config: StreamConfig) -> StreamProcessor:
    """Get or create a stream processor"""
    if config.stream_name not in stream_processors:
        processor = StreamProcessor(config)
        await processor.initialize()
        stream_processors[config.stream_name] = processor
    
    return stream_processors[config.stream_name]

async def cleanup_stream_processors():
    """Cleanup all stream processors"""
    for processor in stream_processors.values():
        await processor.stop()
    stream_processors.clear()