"""
Audio Transcriber Module

Handles:
1. Audio source separation (Spleeter via subprocess for isolation)
2. MIDI transcription (Basic Pitch)
"""
import os
# Force TensorFlow to ignore the GPU and use the Ryzen CPU
os.environ['CUDA_VISIBLE_DEVICES'] = '-1'
# Suppress the "libcudart" and "numa" warnings
os.environ['TF_CPP_MIN_LOG_LEVEL'] = '3' 
import logging
import tempfile
import subprocess
from pathlib import Path
from typing import Dict, List

import numpy as np
import librosa
import soundfile as sf
from basic_pitch.inference import predict
from basic_pitch import ICASSP_2022_MODEL_PATH
from mido import MidiFile, MidiTrack, Message, MetaMessage

logger = logging.getLogger(__name__)


class AudioTranscriber:
    """
    Handles audio processing pipeline with GPU memory isolation
    """
    
    def __init__(self):
        """Initialize Basic Pitch model path"""
        logger.info("Initializing AudioTranscriber...")
        
        # Basic Pitch model path
        self.basic_pitch_model_path = ICASSP_2022_MODEL_PATH
        logger.info(f"Basic Pitch model ready: {self.basic_pitch_model_path}")
        
        # Create temp directory for processing
        self.temp_dir = tempfile.mkdtemp(prefix="guitar_transcribe_")
        logger.info(f"Temp directory: {self.temp_dir}")
        
        logger.info("AudioTranscriber initialized (Spleeter will run via subprocess)")
    
    def transcribe(self, audio_path: str) -> Dict:
        """Full transcription pipeline"""
        logger.info(f"Starting transcription for: {audio_path}")
        
        # Step 1: Separate audio using Spleeter subprocess
        logger.info("Step 1/3: Separating audio with Spleeter (subprocess)...")
        guitar_stem_path, duration, sample_rate = self._separate_guitar(audio_path)
        
        # Step 2: Transcribe guitar audio to MIDI
        logger.info("Step 2/3: Transcribing to MIDI with Basic Pitch...")
        notes = self._transcribe_to_midi(guitar_stem_path)
        
        logger.info(f"Transcription complete. {len(notes)} notes detected.")
        
        return {
            'notes': notes,
            'guitar_stem_path': guitar_stem_path,
            'original_duration': duration,
            'sample_rate': sample_rate
        }
    
    def _separate_guitar(self, audio_path: str) -> tuple:
        """
        Separate guitar using Spleeter CLI (subprocess for GPU memory isolation)
        
        Returns:
            (guitar_stem_path, duration, sample_rate)
        """
        try:
            # Get audio duration first
            audio, sr = librosa.load(audio_path, sr=None, mono=False, duration=1.0)
            full_duration = librosa.get_duration(path=audio_path)
            
            logger.info(f"Audio duration: {full_duration:.2f}s")
            
            # Run Spleeter via subprocess (isolates GPU memory)
            logger.info("Running Spleeter separation (this may take 30-60 seconds)...")
            
            cmd = [
                "spleeter", "separate",
                "-p", "spleeter:2stems",
                "-o", self.temp_dir,
                audio_path
            ]
            
            result = subprocess.run(
                cmd,
                check=True,
                capture_output=True,
                text=True
            )
            
            if result.stderr:
                logger.warning(f"Spleeter stderr: {result.stderr}")
            
            logger.info("Spleeter separation complete")
            
            # Spleeter creates: temp_dir/filename/accompaniment.wav
            filename = Path(audio_path).stem
            accompaniment_path = os.path.join(self.temp_dir, filename, "accompaniment.wav")
            
            if not os.path.exists(accompaniment_path):
                raise FileNotFoundError(f"Spleeter output not found: {accompaniment_path}")
            
            # Create final output path in project's processed directory
            # Use absolute path to project root
            project_root = Path(__file__).parent.parent
            output_dir = project_root / "processed"
            output_dir.mkdir(exist_ok=True)
            
            # Extract job ID from filename
            input_path = Path(audio_path)
            filename_parts = input_path.stem.split('_', 1)
            if len(filename_parts) >= 1 and filename_parts[0].isdigit():
                job_id = filename_parts[0]
            else:
                job_id = "unknown"
            
            job_dir = output_dir / job_id
            job_dir.mkdir(exist_ok=True)
            
            final_path = job_dir / "guitar_stem.wav"
            
            # Load and save as mono for Basic Pitch
            guitar_audio, sr = librosa.load(accompaniment_path, sr=44100, mono=True)
            sf.write(str(final_path), guitar_audio, sr)
            
            logger.info(f"Guitar stem saved to: {final_path}")
            
            return str(final_path), full_duration, sr
            
        except subprocess.CalledProcessError as e:
            logger.error(f"Spleeter subprocess failed: {e.stderr}")
            raise
        except Exception as e:
            logger.error(f"Separation failed: {str(e)}")
            raise
    
    def _transcribe_to_midi(self, audio_path: str) -> List[Dict]:
        """
        Transcribe audio to MIDI notes using Basic Pitch
        
        Returns:
            List of note dictionaries
        """
        try:
            logger.info("Running Basic Pitch inference...")
            
            # Basic Pitch predict function
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
            
            for note in note_events:
                start_time = float(note[0])
                end_time = float(note[1])
                pitch = int(note[2])
                amplitude = float(note[3])
                duration = end_time - start_time
                
                # Guitar range filter (E2 to E6: MIDI 40-88)
                if 40 <= pitch <= 88 and duration > 0:
                    notes.append({
                        'time': start_time,
                        'pitch': pitch,
                        'duration': duration,
                        'confidence': min(amplitude, 1.0)
                    })
            
            notes.sort(key=lambda x: x['time'])
            
            logger.info(f"Basic Pitch detected {len(notes)} notes in guitar range")
            
            # Save as MIDI file
            midi_path = audio_path.replace('.wav', '.mid')
            self._save_midi(notes, midi_path)
            logger.info(f"MIDI file saved to: {midi_path}")
            
            return notes
            
        except Exception as e:
            logger.error(f"MIDI transcription failed: {str(e)}")
            raise
    
    def _save_midi(self, notes: List[Dict], output_path: str) -> None:
        """Save notes as a MIDI file (Handles Polyphonic Chords correctly)"""
        try:
            logger.info(f"Creating MIDI file with {len(notes)} notes")
            
            mid = MidiFile()
            track = MidiTrack()
            mid.tracks.append(track)
            
            track.append(MetaMessage('set_tempo', tempo=500000))
            track.append(MetaMessage('track_name', name='Guitar'))
            
            ticks_per_beat = 480
            ticks_per_second = ticks_per_beat * 2  # 120 BPM
            
            # 1. FLATTEN NOTES INTO DISCRETE EVENTS
            events = []
            for note in notes:
                velocity = max(40, min(int(note['confidence'] * 127), 127))
                
                # Note ON event
                events.append({
                    'type': 'note_on',
                    'time': note['time'],
                    'pitch': note['pitch'],
                    'velocity': velocity
                })
                
                # Note OFF event
                events.append({
                    'type': 'note_off',
                    'time': note['time'] + note['duration'],
                    'pitch': note['pitch'],
                    'velocity': 0
                })
                
            # 2. SORT ALL EVENTS BY ABSOLUTE TIME
            # If times are identical, process note_off before note_on to prevent voice stealing
            events.sort(key=lambda x: (x['time'], 0 if x['type'] == 'note_off' else 1))
            
            # 3. CALCULATE DELTAS AND WRITE
            last_time_ticks = 0
            
            for event in events:
                # Convert absolute seconds to absolute ticks
                current_time_ticks = int(event['time'] * ticks_per_second)
                
                # Delta is the gap between this event and the previous one
                delta_ticks = current_time_ticks - last_time_ticks
                
                # Failsafe: Prevent negative deltas caused by floating-point rounding
                delta_ticks = max(0, delta_ticks)
                
                track.append(Message(
                    event['type'], 
                    note=event['pitch'], 
                    velocity=event['velocity'], 
                    time=delta_ticks
                ))
                
                last_time_ticks = current_time_ticks
            
            mid.save(output_path)
            logger.info(f"Successfully saved MIDI: {output_path} ({len(events)} events)")
            
        except Exception as e:
            logger.error(f"Failed to save MIDI: {str(e)}", exc_info=True)
    
    def __del__(self):
        """Cleanup temp directory"""
        try:
            import shutil
            if hasattr(self, 'temp_dir') and os.path.exists(self.temp_dir):
                shutil.rmtree(self.temp_dir)
                logger.info(f"Cleaned up temp directory: {self.temp_dir}")
        except:
            pass