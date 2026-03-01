#!/usr/bin/env python3
"""
Test script to verify Python service with a real audio file

Usage:
    python test_transcribe.py /path/to/audio.mp3
"""

import sys
import json
import requests
import time
from pathlib import Path

BASE_URL = "http://localhost:5000"


def test_health():
    """Test health endpoint"""
    print("Testing health endpoint...")
    try:
        response = requests.get(f"{BASE_URL}/health")
        response.raise_for_status()
        data = response.json()
        print(f"✓ Health check passed: {data['message']}")
        return True
    except Exception as e:
        print(f"✗ Health check failed: {e}")
        return False


def test_transcribe(audio_path: str):
    """Test transcription with a real audio file"""
    print(f"\nTesting transcription with: {audio_path}")
    
    # Check file exists
    if not Path(audio_path).exists():
        print(f"✗ File not found: {audio_path}")
        return False
    
    # Get absolute path
    abs_path = str(Path(audio_path).absolute())
    print(f"Absolute path: {abs_path}")
    
    # Make request
    print("\nSending transcription request...")
    print("This will take 30-90 seconds depending on file length...")
    
    try:
        start_time = time.time()
        
        response = requests.post(
            f"{BASE_URL}/transcribe",
            json={"file_path": abs_path},
            timeout=300  # 5 minute timeout
        )
        response.raise_for_status()
        
        elapsed = time.time() - start_time
        print(f"\n✓ Request completed in {elapsed:.1f} seconds")
        
        # Parse response
        data = response.json()
        
        if not data['success']:
            print(f"✗ Transcription failed: {data['error']}")
            return False
        
        # Display results
        print("\n" + "="*60)
        print("TRANSCRIPTION RESULTS")
        print("="*60)
        
        metadata = data['metadata']
        print(f"\nMetadata:")
        print(f"  Original duration: {metadata['original_duration']:.2f} seconds")
        print(f"  Sample rate: {metadata['sample_rate']} Hz")
        print(f"  Processing time: {metadata['processing_time']:.2f} seconds")
        print(f"  Total notes detected: {metadata['total_notes']}")
        
        print(f"\nGuitar stem saved to: {data['guitar_stem_path']}")
        
        # Show first 10 notes
        notes = data['notes']
        print(f"\nFirst 10 notes (of {len(notes)}):")
        print(f"{'Time':<8} {'Pitch':<6} {'Duration':<10} {'Confidence':<10}")
        print("-" * 40)
        
        for note in notes[:10]:
            print(f"{note['time']:<8.2f} {note['pitch']:<6} {note['duration']:<10.3f} {note['confidence']:<10.2f}")
        
        if len(notes) > 10:
            print(f"... and {len(notes) - 10} more notes")
        
        # Note statistics
        if notes:
            pitches = [n['pitch'] for n in notes]
            confidences = [n['confidence'] for n in notes]
            
            print(f"\nNote Statistics:")
            print(f"  Pitch range: {min(pitches)} to {max(pitches)} (MIDI)")
            print(f"  Average confidence: {sum(confidences)/len(confidences):.2f}")
            print(f"  Low confidence notes (<0.7): {sum(1 for c in confidences if c < 0.7)}")
        
        print("\n" + "="*60)
        print("✅ TRANSCRIPTION SUCCESSFUL")
        print("="*60)
        
        # Save full results to JSON
        output_file = "transcription_result.json"
        with open(output_file, 'w') as f:
            json.dump(data, f, indent=2)
        print(f"\nFull results saved to: {output_file}")
        
        return True
        
    except requests.Timeout:
        print("\n✗ Request timed out (>5 minutes)")
        print("   Try a shorter audio file")
        return False
    except requests.RequestException as e:
        print(f"\n✗ Request failed: {e}")
        return False
    except Exception as e:
        print(f"\n✗ Unexpected error: {e}")
        return False


def main():
    if len(sys.argv) < 2:
        print("Usage: python test_transcribe.py /path/to/audio.mp3")
        print("\nExample:")
        print("  python test_transcribe.py ~/Music/song.mp3")
        sys.exit(1)
    
    audio_path = sys.argv[1]
    
    print("="*60)
    print("Python Service Transcription Test")
    print("="*60)
    
    # Test health first
    if not test_health():
        print("\n❌ Service is not running or not healthy")
        print("   Make sure to start it with: python main.py")
        sys.exit(1)
    
    # Test transcription
    if not test_transcribe(audio_path):
        print("\n❌ Transcription test failed")
        sys.exit(1)
    
    print("\n✅ All tests passed!")


if __name__ == "__main__":
    main()