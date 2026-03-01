"""
Guitar Transcriber Python Service

This service handles:
1. Audio source separation using Spleeter (extract guitar stem)
2. Audio-to-MIDI transcription using Basic Pitch
"""

import os
import time
import logging
from pathlib import Path
from typing import List

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='[%(asctime)s] %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S'
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Guitar Transcriber Python Service",
    description="Audio processing service for guitar transcription",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global transcriber instance (initialized on startup)
transcriber = None


# Request/Response models
class TranscribeRequest(BaseModel):
    file_path: str  # Absolute path to the uploaded audio file


class RawNote(BaseModel):
    time: float
    pitch: int
    duration: float
    confidence: float


class AudioMetadata(BaseModel):
    original_duration: float
    sample_rate: int
    total_notes: int
    processing_time: float


class TranscribeResponse(BaseModel):
    success: bool
    notes: List[RawNote]
    guitar_stem_path: str
    error: str = ""
    metadata: AudioMetadata


class HealthResponse(BaseModel):
    status: str
    message: str


@app.on_event("startup")
async def startup_event():
    """Initialize transcriber on startup"""
    global transcriber
    logger.info("Initializing AudioTranscriber...")
    
    from transcriber import AudioTranscriber
    transcriber = AudioTranscriber()
    
    logger.info("AudioTranscriber initialized successfully")


# Endpoints
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    if transcriber is None:
        return HealthResponse(
            status="starting",
            message="Service is starting up..."
        )
    
    return HealthResponse(
        status="ok",
        message="Python service is running. Spleeter and Basic Pitch models loaded."
    )


@app.post("/transcribe", response_model=TranscribeResponse)
async def transcribe_audio(request: TranscribeRequest):
    """
    Transcribe audio file to MIDI notes
    
    Process:
    1. Load audio file
    2. Separate guitar stem using Spleeter
    3. Transcribe guitar stem to MIDI using Basic Pitch
    4. Return notes as JSON
    """
    if transcriber is None:
        raise HTTPException(status_code=503, detail="Service is still starting up")
    
    logger.info(f"Transcription request received for: {request.file_path}")
    
    # Validate file exists
    if not os.path.exists(request.file_path):
        logger.error(f"File not found: {request.file_path}")
        raise HTTPException(status_code=404, detail=f"File not found: {request.file_path}")
    
    # Validate file extension
    valid_extensions = ['.mp3', '.wav', '.m4a', '.flac', '.ogg']
    file_ext = Path(request.file_path).suffix.lower()
    if file_ext not in valid_extensions:
        logger.error(f"Invalid file type: {file_ext}")
        raise HTTPException(
            status_code=400, 
            detail=f"Invalid file type. Supported: {', '.join(valid_extensions)}"
        )
    
    try:
        start_time = time.time()
        
        # Run the transcription pipeline
        result = transcriber.transcribe(request.file_path)
        
        processing_time = time.time() - start_time
        logger.info(f"Transcription completed in {processing_time:.2f}s. Notes detected: {len(result['notes'])}")
        
        # Build response
        notes = [
            RawNote(
                time=note['time'],
                pitch=note['pitch'],
                duration=note['duration'],
                confidence=note['confidence']
            )
            for note in result['notes']
        ]
        
        metadata = AudioMetadata(
            original_duration=result['original_duration'],
            sample_rate=result['sample_rate'],
            total_notes=len(notes),
            processing_time=processing_time
        )
        
        return TranscribeResponse(
            success=True,
            notes=notes,
            guitar_stem_path=result['guitar_stem_path'],
            metadata=metadata
        )
        
    except Exception as e:
        logger.error(f"Transcription failed: {str(e)}", exc_info=True)
        return TranscribeResponse(
            success=False,
            notes=[],
            guitar_stem_path="",
            error=str(e),
            metadata=AudioMetadata(
                original_duration=0.0,
                sample_rate=0,
                total_notes=0,
                processing_time=0.0
            )
        )


if __name__ == "__main__":
    import uvicorn
    
    port = int(os.getenv("PORT", "5000"))
    logger.info(f"Starting Python service on port {port}")
    
    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=port,
        log_level="info",
        reload=False
    )