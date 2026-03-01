"""
Audio Transcriber Module

Handles:
1. Audio source separation (Spleeter)
2. MIDI transcription (Basic Pitch)
"""

import os
import logging
import tempfile
from pathlib import Path
from typing import Dict, List

import numpy as np
import librosa
import soundfile as sf
from spleeter.separator import Separator
from basic_pitch.inference import predict
from basic_pitch import ICASSP_2022_MODEL_PATH

logger = logging.getLogger(__name__)


class AudioTranscriber:
    """
    Handles audio processing pipeline:
    1. Load audio
    2. Separate guitar using Spleeter
    3. Transcribe to MIDI using Basic Pitch
    """
    
    def __init__(self):
        """Initialize Spleeter and Basic Pitch models"""
        logger.info("Loading Spleeter model (2stems - vocals/accompaniment)...")
        
        # Initialize Spleeter with 2stems model
        # This separates into vocals and accompaniment (guitar is in accompaniment)
        self.separator = Separator('spleeter:2stems')
        
        logger.info("Spleeter model loaded")
        
        # Basic Pitch model path
        self.basic_pitch_model_path = ICASSP_2022_MODEL_PATH
        logger.info(f"Basic Pitch model ready: {self.basic_pitch_model_path}")
        
        # Create temp directory for processing
        self.temp_dir = tempfile.mkdtemp(prefix="guitar_transcribe_")
        logger.info(f"Temp directory: {self.temp_dir}")
    
    def transcribe(self, audio_path: str) -> Dict:
        """
        Full transcription pipeline
        
        Args:
            audio_path: Path to input audio file
            
        Returns:
            Dictionary with:
                - notes: List of detected notes
                - guitar_stem_path: Path to extracted guitar audio
                - original_duration: Duration in seconds
                - sample_rate: Sample rate in Hz
        """
        logger.info(f"Starting transcription for: {audio_path}")
        
        # Step 1: Separate audio to extract guitar/accompaniment
        logger.info("Step 1/3: Separating audio with Spleeter...")
        guitar_audio, sample_rate, duration = self._separate_guitar(audio_path)
        
        # Step 2: Save guitar stem
        logger.info("Step 2/3: Saving guitar stem...")
        guitar_stem_path = self._save_guitar_stem(audio_path, guitar_audio, sample_rate)
        
        # Step 3: Transcribe guitar audio to MIDI
        logger.info("Step 3/3: Transcribing to MIDI with Basic Pitch...")
        notes = self._transcribe_to_midi(guitar_stem_path, sample_rate)
        
        logger.info(f"Transcription complete. {len(notes)} notes detected.")
        
        return {
            'notes': notes,
            'guitar_stem_path': guitar_stem_path,
            'original_duration': duration,
            'sample_rate': sample_rate
        }
    
    def _separate_guitar(self, audio_path: str) -> tuple:
        """
        Separate guitar/accompaniment from vocals using Spleeter
        
        Returns:
            (audio_array, sample_rate, duration)
        """
        try:
            # Load audio with librosa
            audio, sr = librosa.load(audio_path, sr=44100, mono=False)
            
            # If mono, convert to stereo for Spleeter
            if audio.ndim == 1:
                audio = np.stack([audio, audio])
            
            # Spleeter expects shape (samples, channels)
            if audio.shape[0] == 2:  # (2, samples) -> (samples, 2)
                audio = audio.T
            
            duration = audio.shape[0] / sr
            logger.info(f"Loaded audio: {duration:.2f}s, {sr}Hz, shape={audio.shape}")
            
            # Separate using Spleeter
            # Returns dict with 'vocals' and 'accompaniment'
            prediction = self.separator.separate(audio)
            
            # Get accompaniment (contains guitar, bass, drums, etc.)
            # In many songs, guitar is the dominant melodic instrument in accompaniment
            accompaniment = prediction['accompaniment']
            
            # Convert back to mono for Basic Pitch (average L+R channels)
            guitar_mono = np.mean(accompaniment, axis=1)
            
            logger.info(f"Separation complete. Accompaniment shape: {accompaniment.shape}")
            
            return guitar_mono, sr, duration
            
        except Exception as e:
            logger.error(f"Separation failed: {str(e)}")
            raise
    
    def _save_guitar_stem(self, original_path: str, audio: np.ndarray, sr: int) -> str:
        """
        Save the extracted guitar stem to disk
        
        Returns:
            Path to saved guitar stem
        """
        try:
            # Create output path based on input filename
            input_path = Path(original_path)
            output_dir = input_path.parent.parent / "processed"
            output_dir.mkdir(exist_ok=True)
            
            # Extract job ID from filename (format: jobid_filename.ext)
            filename_parts = input_path.stem.split('_', 1)
            if len(filename_parts) >= 1 and filename_parts[0].isdigit():
                job_id = filename_parts[0]
            else:
                job_id = "unknown"
            
            # Create job-specific directory
            job_dir = output_dir / job_id
            job_dir.mkdir(exist_ok=True)
            
            # Save as WAV
            output_path = job_dir / "guitar_stem.wav"
            sf.write(str(output_path), audio, sr)
            
            logger.info(f"Guitar stem saved to: {output_path}")
            return str(output_path)
            
        except Exception as e:
            logger.error(f"Failed to save guitar stem: {str(e)}")
            raise
    
    def _transcribe_to_midi(self, audio_path: str, sr: int) -> List[Dict]:
        """
        Transcribe audio to MIDI notes using Basic Pitch
        
        Args:
            audio_path: Path to the guitar stem audio file
            sr: Sample rate (unused, Basic Pitch loads audio itself)
        
        Returns:
            List of note dictionaries with time, pitch, duration, confidence
        """
        try:
            # Basic Pitch expects a file path, not raw audio
            # It will load and resample the audio internally
            
            # Run Basic Pitch inference
            model_output, midi_data, note_events = predict(
                audio_path,
                self.basic_pitch_model_path,
                onset_threshold=0.5,
                frame_threshold=0.3,
                minimum_note_length=58,
                minimum_frequency=None,
                maximum_frequency=None,
                melodia_trick=True,
            )
            
            # Convert note events to our format
            notes = []
            
            # note_events is a list of tuples: (start_time, end_time, pitch, amplitude, [bends])
            for note in note_events:
                start_time = float(note[0])
                end_time = float(note[1])
                pitch = int(note[2])
                amplitude = float(note[3])  # Use as confidence proxy
                
                # Calculate duration
                duration = end_time - start_time
                
                # Only include notes in guitar range (E2 to E6: MIDI 40-88)
                # This filters out bass notes and very high harmonics
                if 40 <= pitch <= 88 and duration > 0:
                    notes.append({
                        'time': start_time,
                        'pitch': pitch,
                        'duration': duration,
                        'confidence': min(amplitude, 1.0)  # Clamp to [0, 1]
                    })
            
            # Sort notes by time
            notes.sort(key=lambda x: x['time'])
            
            logger.info(f"Basic Pitch detected {len(notes)} notes in guitar range")
            
            return notes
            
        except Exception as e:
            logger.error(f"MIDI transcription failed: {str(e)}")
            raise
    
    def __del__(self):
        """Cleanup temp directory"""
        try:
            import shutil
            if hasattr(self, 'temp_dir') and os.path.exists(self.temp_dir):
                shutil.rmtree(self.temp_dir)
                logger.info(f"Cleaned up temp directory: {self.temp_dir}")
        except:
            pass